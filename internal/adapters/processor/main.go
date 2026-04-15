package processor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

var paramCommentRegex = regexp.MustCompile(`^\s*(?:#\s*)?(from-param|from-param-merge):\s*(.+)\s*$`)
var simpleTemplateRegex = regexp.MustCompile(`^\s*{{\s*\.([a-zA-Z0-9_.-]+)\s*}}\s*$`)

const structuredTemplateResultPrefix = "__ymr_structured__:"

type structuredTemplateResult struct {
	Kind      string   `json:"kind"`
	Items     []string `json:"items,omitempty"`
	Delimiter string   `json:"delimiter,omitempty"`
}

func ProcessContent(templateContent []byte, params map[string]any, strict bool) (string, error) {
	if len(bytes.TrimSpace(templateContent)) == 0 {
		return "", nil
	}

	dec := yaml.NewDecoder(bytes.NewReader(templateContent))
	docs := make([]yaml.Node, 0)

	for {
		var doc yaml.Node
		err := dec.Decode(&doc)
		if err != nil {
			if errorsIsEOF(err) {
				break
			}
			return "", fmt.Errorf("failed to parse template yaml: %w", err)
		}
		docs = append(docs, doc)
	}

	for i := range docs {
		if err := traverse(&docs[i], params, strict); err != nil {
			return "", err
		}
	}

	var out bytes.Buffer
	for i := range docs {
		if i > 0 {
			out.WriteString("---\n")
		}

		var b bytes.Buffer
		enc := yaml.NewEncoder(&b)
		enc.SetIndent(2)

		nodeToEncode := &docs[i]
		if docs[i].Kind == yaml.DocumentNode && len(docs[i].Content) == 1 {
			nodeToEncode = docs[i].Content[0]
		}

		if err := enc.Encode(nodeToEncode); err != nil {
			_ = enc.Close()
			return "", fmt.Errorf("failed to re-marshal yaml: %w", err)
		}
		if err := enc.Close(); err != nil {
			return "", fmt.Errorf("failed to re-marshal yaml: %w", err)
		}

		out.Write(b.Bytes())
	}

	return out.String(), nil
}

func errorsIsEOF(err error) bool {
	return err == io.EOF
}

func resolveParam(params map[string]any, key string) (any, bool) {
	// Back-compat: allow literal keys with dots.
	if v, ok := params[key]; ok {
		return v, true
	}

	if !strings.Contains(key, ".") {
		return nil, false
	}

	segments := strings.Split(key, ".")
	var cur any = params

	for _, seg := range segments {
		switch typed := cur.(type) {
		case map[string]any:
			v, ok := typed[seg]
			if !ok {
				return nil, false
			}
			cur = v
		case map[any]any:
			v, ok := typed[seg]
			if !ok {
				return nil, false
			}
			cur = v
		case []any:
			idx, err := strconv.Atoi(seg)
			if err != nil || idx < 0 || idx >= len(typed) {
				return nil, false
			}
			cur = typed[idx]
		default:
			return nil, false
		}
	}

	return cur, true
}

func traverse(node *yaml.Node, params map[string]any, strict bool) error {
	slog.Debug("Traversing node", "kind", node.Kind, "tag", node.Tag)

	switch node.Kind {
	case yaml.DocumentNode:
		for _, child := range node.Content {
			if err := traverse(child, params, strict); err != nil {
				return err
			}
		}

	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			if keyNode.LineComment != "" {
				directive, rawString := parseParamFromComment(keyNode.LineComment)
				if rawString != "" {
					if err := processDirective(valueNode, directive, rawString, params, strict); err != nil {
						return fmt.Errorf("directive %q failed at line %d: %w", directive, keyNode.Line, err)
					}
					keyNode.LineComment = ""
				}
			}

			if valueNode.LineComment != "" {
				directive, rawString := parseParamFromComment(valueNode.LineComment)
				if rawString != "" {
					if err := processDirective(valueNode, directive, rawString, params, strict); err != nil {
						return fmt.Errorf("directive %q failed at line %d: %w", directive, valueNode.Line, err)
					}
					valueNode.LineComment = ""
				}
			}

			if err := traverse(keyNode, params, strict); err != nil {
				return err
			}
			if err := traverse(valueNode, params, strict); err != nil {
				return err
			}
		}

	case yaml.SequenceNode:
		for _, child := range node.Content {
			if child.LineComment != "" {
				directive, rawString := parseParamFromComment(child.LineComment)
				if rawString != "" {
					if err := processDirective(child, directive, rawString, params, strict); err != nil {
						return fmt.Errorf("directive %q failed at line %d: %w", directive, child.Line, err)
					}
					child.LineComment = ""
				}
			}
			if err := traverse(child, params, strict); err != nil {
				return err
			}
		}
	}

	return nil
}

func processDirective(node *yaml.Node, directive, rawString string, params map[string]any, strict bool) error {
	slog.Debug("Processing directive", "directive", directive, "rawString", rawString)

	if matches := simpleTemplateRegex.FindStringSubmatch(rawString); len(matches) > 1 {
		paramName := matches[1]
		if value, ok := resolveParam(params, paramName); ok {
			return updateNodeValue(node, value, directive)
		}

		if strict {
			return fmt.Errorf("missing parameter %q", paramName)
		}

		slog.Debug("Template key missing, preserving default", "key", paramName)
		return nil
	}

	renderedValue, err := executeTemplate(rawString, params)
	if err == nil {
		decodedValue, handled, err := decodeStructuredTemplateResult(renderedValue, node)
		if err != nil {
			if strict {
				return err
			}

			slog.Debug("Structured template result decode failed, preserving default", "template", rawString, "error", err)
			return nil
		}

		if handled {
			return updateNodeValue(node, decodedValue, directive)
		}

		return updateNodeValue(node, renderedValue, directive)
	}
	if strict {
		return err
	}

	slog.Debug("Template failed, preserving default", "template", rawString, "error", err)

	return nil
}

func parseParamFromComment(comment string) (directive, paramName string) {
	matches := paramCommentRegex.FindStringSubmatch(comment)
	if len(matches) > 2 {
		return matches[1], strings.TrimSpace(matches[2])
	}
	return "", ""
}

func mergeMappingNode(dst *yaml.Node, src *yaml.Node) {
	// Deterministic merge: override existing keys; append new keys.
	// This avoids duplicate keys in the output where possible.
	index := make(map[string]int, len(dst.Content)/2)
	for i := 0; i+1 < len(dst.Content); i += 2 {
		k := dst.Content[i]
		if k != nil && k.Kind == yaml.ScalarNode {
			index[k.Value] = i
		}
	}

	for i := 0; i+1 < len(src.Content); i += 2 {
		sk := src.Content[i]
		sv := src.Content[i+1]
		if sk != nil && sk.Kind == yaml.ScalarNode {
			if di, ok := index[sk.Value]; ok {
				dst.Content[di+1] = sv
				continue
			}
			index[sk.Value] = len(dst.Content)
		}
		dst.Content = append(dst.Content, sk, sv)
	}
}

func updateNodeValue(node *yaml.Node, newValue any, directive string) error {
	var replacementNode yaml.Node

	if newValue == nil {
		replacementNode.Kind = yaml.ScalarNode
		replacementNode.Tag = "!!null"
		replacementNode.Value = "null"
	} else {
		err := replacementNode.Encode(newValue)
		if err != nil {
			slog.Debug("Failed to encode replacement node", "error", err)
			return fmt.Errorf("failed to encode replacement node: %w", err)
		}
	}

	isMerge := directive == "from-param-merge"
	isNodeSequence := node.Kind == yaml.SequenceNode
	isReplacementSequence := replacementNode.Kind == yaml.SequenceNode
	isNodeMap := node.Kind == yaml.MappingNode
	isReplacementMap := replacementNode.Kind == yaml.MappingNode

	if isMerge && isNodeSequence && isReplacementSequence {
		node.Content = append(node.Content, replacementNode.Content...)
	} else if isMerge && isNodeMap && isReplacementMap {
		mergeMappingNode(node, &replacementNode)
	} else {
		node.Kind = replacementNode.Kind
		node.Tag = replacementNode.Tag
		node.Value = replacementNode.Value
		node.Style = replacementNode.Style
		node.Content = replacementNode.Content
	}

	return nil
}

func executeTemplate(tmplStr string, params map[string]any) (string, error) {
	slog.Debug("executeTemplate", "template", tmplStr)

	tmpl, err := template.New("template").
		Option("missingkey=error").
		Funcs(template.FuncMap{
			"lower": strings.ToLower,
			"upper": strings.ToUpper,
			"replace": func(old, new string, s any) string {
				return strings.ReplaceAll(fmt.Sprint(s), old, new)
			},
			"for": func(itemTemplate string, args ...any) (string, error) {
				return renderForLoop(itemTemplate, args...)
			},
		}).
		Parse(tmplStr)

	if err != nil {
		return "", fmt.Errorf("failed to parse template string %s: %w", tmplStr, err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, params)
	if err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", tmplStr, err)
	}

	return buf.String(), nil
}

func decodeStructuredTemplateResult(rendered string, node *yaml.Node) (any, bool, error) {
	if !strings.HasPrefix(rendered, structuredTemplateResultPrefix) {
		return nil, false, nil
	}

	payload := strings.TrimPrefix(rendered, structuredTemplateResultPrefix)
	var decoded structuredTemplateResult
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		return nil, true, fmt.Errorf("failed to decode structured template result: %w", err)
	}

	switch decoded.Kind {
	case "for":
		if node.Kind == yaml.SequenceNode {
			return decoded.Items, true, nil
		}
		return strings.Join(decoded.Items, decoded.Delimiter), true, nil
	default:
		return nil, true, fmt.Errorf("unknown structured template result kind %q", decoded.Kind)
	}
}

func renderForLoop(itemTemplate string, args ...any) (string, error) {
	var (
		delimiter string
		values    any
	)

	switch len(args) {
	case 1:
		values = args[0]
	case 2:
		delimiter = fmt.Sprint(args[0])
		values = args[1]
	default:
		return "", fmt.Errorf("for expects template, optional delimiter, and slice value")
	}

	rendered, err := mapLoopValues(itemTemplate, values)
	if err != nil {
		return "", err
	}

	payload, err := json.Marshal(structuredTemplateResult{
		Kind:      "for",
		Items:     rendered,
		Delimiter: delimiter,
	})
	if err != nil {
		return "", fmt.Errorf("marshaling for result: %w", err)
	}

	return structuredTemplateResultPrefix + string(payload), nil
}

func mapLoopValues(itemTemplate string, values any) ([]string, error) {
	rv := reflect.ValueOf(values)
	if !rv.IsValid() {
		return nil, fmt.Errorf("for expects slice or array value")
	}

	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return nil, fmt.Errorf("for expects slice or array value")
	}

	rendered := make([]string, 0, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		item := rv.Index(i).Interface()
		rendered = append(rendered, strings.ReplaceAll(itemTemplate, "$i", fmt.Sprint(item)))
	}

	return rendered, nil
}
