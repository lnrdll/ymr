package processor

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

// The "full" regex for capturing the entire template string
var paramCommentRegex = regexp.MustCompile(`(from-param|from-param-merge):\s*(.+)`)

// A regex to detect if the template is a key lookup. It matches "{{ .key }}" or "{{.key}}"
var simpleTemplateRegex = regexp.MustCompile(`^\s*{{\s*\.([a-zA-Z0-9_.-]+)\s*}}\s*$`)

// ProcessContent takes template content and substitutes params.
func ProcessContent(templateContent []byte, params map[string]any) (string, error) {
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
		traverse(&docs[i], params)
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

// traverse recursively visits every node in the YAML tree, handling map keys and values.
func traverse(node *yaml.Node, params map[string]any) {
	slog.Debug("Traversing node", "kind", node.Kind, "tag", node.Tag)

	switch node.Kind {
	case yaml.DocumentNode:
		for _, child := range node.Content {
			traverse(child, params)
		}

	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			if keyNode.LineComment != "" {
				directive, rawString := parseParamFromComment(keyNode.LineComment)
				if rawString != "" {
					processDirective(valueNode, directive, rawString, params)
					keyNode.LineComment = ""
				}
			}

			if valueNode.LineComment != "" {
				directive, rawString := parseParamFromComment(valueNode.LineComment)
				if rawString != "" {
					processDirective(valueNode, directive, rawString, params)
					valueNode.LineComment = ""
				}
			}

			traverse(keyNode, params)
			traverse(valueNode, params)
		}

	case yaml.SequenceNode:
		for _, child := range node.Content {
			if child.LineComment != "" {
				directive, rawString := parseParamFromComment(child.LineComment)
				if rawString != "" {
					processDirective(child, directive, rawString, params)
					child.LineComment = ""
				}
			}
			traverse(child, params)
		}
	}
}

// processDirective performs the directive substitution.
// It updates the provided yaml.Node with the processed value.
func processDirective(node *yaml.Node, directive, rawString string, params map[string]any) {
	slog.Debug("Processing directive", "directive", directive, "rawString", rawString)

	// This preserves arrays/maps/objects
	if matches := simpleTemplateRegex.FindStringSubmatch(rawString); len(matches) > 1 {
		paramName := matches[1]
		if value, ok := params[paramName]; ok {
			// It's a simple key. Pass the raw interface{} value.
			updateNodeValue(node, value, directive)
			return
		} else {
			slog.Debug("Template key missing, preserving default", "key", paramName)
			return
		}
	}

	renderedValue, err := executeTemplate(rawString, params)
	if err == nil {
		updateNodeValue(node, renderedValue, directive)
	} else {
		slog.Debug("Template failed, preserving default", "template", rawString, "error", err)
	}
}

// parseParamFromComment checks a comment string for the directive and param name.
func parseParamFromComment(comment string) (directive, paramName string) {
	matches := paramCommentRegex.FindStringSubmatch(comment)
	if len(matches) > 2 {
		return matches[1], matches[2]
	}
	return "", ""
}

// updateNodeValue updates a yaml.Node with a new value, handling merge/replace logic.
func updateNodeValue(node *yaml.Node, newValue any, directive string) {
	var replacementNode yaml.Node

	err := replacementNode.Encode(newValue)
	if err != nil {
		slog.Debug("Failed to encode replacement node", "error", err)
		node.Tag = "!!str"
		node.Value = fmt.Sprintf("ERROR_ENCONDING:%v", err)
	}

	// The merge/replace logic
	isMerge := directive == "from-param-merge"
	isNodeSequence := node.Kind == yaml.SequenceNode
	isReplacementSequence := replacementNode.Kind == yaml.SequenceNode
	isNodeMap := node.Kind == yaml.MappingNode
	isReplacementMap := replacementNode.Kind == yaml.MappingNode

	if isMerge && isNodeSequence && isReplacementSequence {
		node.Content = append(node.Content, replacementNode.Content...)
	} else if isMerge && isNodeMap && isReplacementMap {
		node.Content = append(node.Content, replacementNode.Content...)
	} else {
		node.Kind = replacementNode.Kind
		node.Tag = replacementNode.Tag
		node.Value = replacementNode.Value
		node.Content = replacementNode.Content
	}
}

// executeTemplate renders a Go template string with the provided parameters.
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
