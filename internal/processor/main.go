package processor

import (
	"bytes"
	"fmt"
	"log/slog"
	"regexp"
	"text/template"

	"gopkg.in/yaml.v3"
)

// 1. The "full" regex for capturing the entire template string
var paramCommentRegex = regexp.MustCompile(`(from-param|from-param-merge):\s*(.+)`)

//  2. A "simple" regex to detect if the template is *only* a key lookup
//     It matches "{{ .key }}" or "{{.key}}"
var simpleTemplateRegex = regexp.MustCompile(`^\s*{{\s*\.([a-zA-Z0-9_.-]+)\s*}}\s*$`)

// ProcessContent takes template content and substitutes params
func ProcessContent(templateContent []byte, params map[string]any) (string, error) {
	var rootNode yaml.Node
	err := yaml.Unmarshal(templateContent, &rootNode)
	if err != nil {
		return "", fmt.Errorf("failed to parse template yaml: %w", err)
	}

	traverse(&rootNode, params)

	var b bytes.Buffer
	encoder := yaml.NewEncoder(&b)
	encoder.SetIndent(2)
	err = encoder.Encode(&rootNode)
	if err != nil {
		return "", fmt.Errorf("failed to re-marshal yaml: %w", err)
	}

	return b.String(), nil
}

// traverse recursively visits every node in the YAML tree, handling
// map keys and values.
func traverse(node *yaml.Node, params map[string]any) {
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
					child.LineComment = "" // Always clear comment
				}
			}
			traverse(child, params)
		}
	}
}

// This helper function now contains the "smart" logic.
func processDirective(node *yaml.Node, directive, rawString string, params map[string]any) {
	// 1. Check if it's a *simple* template (e.g., "{{ .envVars }}")
	if matches := simpleTemplateRegex.FindStringSubmatch(rawString); len(matches) > 1 {
		paramName := matches[1]
		if value, ok := params[paramName]; ok {
			// It's a simple key. Pass the raw interface{} value.
			// This preserves arrays/maps/objects.
			updateNodeValue(node, value, directive)
			return
		} else {
			slog.Debug("Template key missing, preserving default", "key", paramName)
			return
		}
	}

	// 2. It's a *complex* template (e.g., "image-{{ .env }}")
	renderedValue, err := executeTemplate(rawString, params)
	if err == nil {
		updateNodeValue(node, renderedValue, directive)
	} else {
		slog.Debug("Template failed, preserving default", "template", rawString, "error", err)
	}
}

// parseParamFromContent checks a comment string for the directive and param name
func parseParamFromComment(comment string) (directive, paramName string) {
	matches := paramCommentRegex.FindStringSubmatch(comment)
	if len(matches) > 2 {
		return matches[1], matches[2]
	}
	return "", ""
}

// updateNodeValue updates a yaml.Node with a new value, handling merge/replace
func updateNodeValue(node *yaml.Node, newValue any, directive string) {
	var replacementNode yaml.Node
	err := replacementNode.Encode(newValue)
	if err != nil {
		node.Tag = "!!str"
		node.Value = fmt.Sprintf("ERROR_ENCONDING:%v", err)
	}

	// (The merge/replace logic is unchanged)
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

// Helper function to execute the template string
func executeTemplate(tmplStr string, params map[string]any) (string, error) {
	tmpl, err := template.New("param").
		Option("missingkey=error").
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
