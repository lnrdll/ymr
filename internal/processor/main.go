package processor

import (
	"bytes"
	"fmt"
	"regexp"

	"gopkg.in/yaml.v3"
)

var paramCommentRegex = regexp.MustCompile(`(from-param|from-param-merge):\s*(\S+)`)

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
	// The "trunk" of the tree
	// Recurse into the document's content
	case yaml.DocumentNode:
		for _, child := range node.Content {
			traverse(child, params)
		}

	// A branch that splits into more key-value pairs
	// Handle maps: [key1, value1, key2, value2]
	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			// Check for comment on the KEY node
			if keyNode.LineComment != "" {
				directive, paramName := parseParamFromComment(keyNode.LineComment)
				if paramName != "" {
					if value, ok := params[paramName]; ok {
						// Apply substitution to the VALUE node
						updateNodeValue(valueNode, value, directive)
					}
					keyNode.LineComment = "" // Clear comment from key
				}
			}

			// Check for comment on the VALUE node
			if valueNode.LineComment != "" {
				directive, paramName := parseParamFromComment(valueNode.LineComment)
				if paramName != "" {
					if value, ok := params[paramName]; ok {
						// Apply substitution to the VALUE node
						updateNodeValue(valueNode, value, directive)
					}
					valueNode.LineComment = "" // Clear comment from value
				}
			}

			// Recurse into children
			traverse(keyNode, params)
			traverse(valueNode, params)
		}

	// A branch that splits into a list of items.
	// Handle sequences (arrays)
	case yaml.SequenceNode:
		for _, child := range node.Content {
			// Check for comment on the child node itself
			if child.LineComment != "" {
				directive, paramName := parseParamFromComment(child.LineComment)
				if paramName != "" {
					if value, ok := params[paramName]; ok {
						updateNodeValue(child, value, directive)
					}
					child.LineComment = "" // Clear comment
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
	tempYaml, err := yaml.Marshal(newValue)
	if err != nil {
		node.Tag = "!!str"
		node.Value = fmt.Sprintf("error marshaling:%v", err)
		return
	}

	var tempRootNode yaml.Node
	err = yaml.Unmarshal(tempYaml, &tempRootNode)
	if err != nil {
		node.Tag = "!!str"
		node.Value = fmt.Sprintf("error marshaling:%v", err)
		return
	}

	if tempRootNode.Kind != yaml.DocumentNode || len(tempRootNode.Content) == 0 {
		return
	}

	replacementNode := tempRootNode.Content[0]

	// 2. To merge or To replace
	isMerge := directive == "from-param-merge"
	isNodeSequence := node.Kind == yaml.SequenceNode
	isReplacementSequence := replacementNode.Kind == yaml.SequenceNode
	isNodeMap := node.Kind == yaml.MappingNode
	isReplacementMap := replacementNode.Kind == yaml.MappingNode

	if isMerge && isNodeSequence && isReplacementSequence {
		// Merge Array/Lists: Append new items
		node.Content = append(node.Content, replacementNode.Content...)
	} else if isMerge && isNodeMap && isReplacementMap {
		// Merge object/maps
		node.Content = append(node.Content, replacementNode.Content...)
	} else {
		// Replace
		node.Kind = replacementNode.Kind
		node.Tag = replacementNode.Tag
		node.Value = replacementNode.Value
		node.Content = replacementNode.Content
	}
}
