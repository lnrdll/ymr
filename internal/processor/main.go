package processor

import (
	"bytes"
	"fmt"
	"log/slog"
	"regexp"
	"text/template"

	"gopkg.in/yaml.v3"
)

// Regex captures the directive and capture **everything** after the colon to the end of the line.
var paramCommentRegex = regexp.MustCompile(`(from-param|from-param-merge):\s*(.+)`)

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
// map keys and values correctly.
func traverse(node *yaml.Node, params map[string]any) {
	switch node.Kind {
	case yaml.DocumentNode:
		// Recurse into the document's content
		for _, child := range node.Content {
			traverse(child, params)
		}

	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			if keyNode.LineComment != "" {
				directive, templateString := parseParamFromComment(keyNode.LineComment)
				if templateString != "" {
					renderedValue, err := executeTemplate(templateString, params)
					if err == nil {
						updateNodeValue(valueNode, renderedValue, directive)
					} else {
						slog.Debug("Template key missing, preserving default", "template", templateString, "error", err)
					}

					// Always clear the comment
					keyNode.LineComment = ""
				}
			}

			if valueNode.LineComment != "" {
				directive, templateString := parseParamFromComment(valueNode.LineComment)
				if templateString != "" {
					renderedValue, err := executeTemplate(templateString, params)
					if err == nil {
						updateNodeValue(valueNode, renderedValue, directive)
					} else {
						slog.Debug("Template value missing, preserving default", "template", templateString, "error", err)
					}

					// Always clear the comment
					valueNode.LineComment = ""
				}
			}

			traverse(keyNode, params)
			traverse(valueNode, params)
		}
	case yaml.SequenceNode:
		// Handle sequences (arrays)
		for _, child := range node.Content {
			if child.LineComment != "" {
				directive, templateString := parseParamFromComment(child.LineComment)
				if templateString != "" {
					renderedValue, err := executeTemplate(templateString, params)
					if err == nil {
						updateNodeValue(child, renderedValue, directive)
					} else {
						slog.Debug("Template key missing, preserving default", "template", templateString, "error", err)
					}

					// Always clear the comment
					child.LineComment = ""
				}
			}
			traverse(child, params)
		}

		// The leaves of the tree
		// Scalar nodes (strings, numbers, etc.) have no children, so recursion stops.
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
	var replacementNode *yaml.Node

	// 'newValue' is the *rendered string* (e.g., "1", "true", "my-string").
	// We need to parse this string as YAML to get the correct type (number, bool, string).
	if strVal, ok := newValue.(string); ok {
		var tempNode yaml.Node
		// Unmarshal the string value *as* a YAML scalar
		if err := yaml.Unmarshal([]byte(strVal), &tempNode); err == nil && tempNode.Kind == yaml.DocumentNode {
			replacementNode = tempNode.Content[0] // Get the scalar node
		}
	}

	// If the above failed or newValue was not a string (e.g., complex map from merge),
	// fall back to the old method.
	if replacementNode == nil {
		tempYAML, err := yaml.Marshal(newValue)
		if err != nil {
			node.Tag = "!!str"
			node.Value = fmt.Sprintf("ERROR_MARSHALING:%v", err)
			return
		}
		var tempRootNode yaml.Node
		err = yaml.Unmarshal(tempYAML, &tempRootNode)
		if err != nil {
			node.Tag = "!!str"
			node.Value = fmt.Sprintf("ERROR_UNMARSHALING:%v", err)
			return
		}
		if tempRootNode.Kind != yaml.DocumentNode || len(tempRootNode.Content) == 0 {
			return
		}
		replacementNode = tempRootNode.Content[0]
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
	tmpl, err := template.New("param").Option("missingkey=error").Parse(tmplStr)
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
