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
	// 1. Parse into a yaml.Node tree
	var rootNode yaml.Node
	err := yaml.Unmarshal(templateContent, &rootNode)
	if err != nil {
		return "", fmt.Errorf("failed to parse template yaml: %w", err)
	}

	// 2. Traverse the tree and substitute values
	traverse(&rootNode, params)

	// 3. Marshal the modified tree back into a string
	var b bytes.Buffer
	encoder := yaml.NewEncoder(&b)
	encoder.SetIndent(2)
	err = encoder.Encode(&rootNode)
	if err != nil {
		return "", fmt.Errorf("failed to re-marshal yaml: %w", err)
	}

	return b.String(), nil
}

// traverse recursivelly visits every node in the tree
func traverse(node *yaml.Node, params map[string]any) {
	if node.LineComment != "" {
		directive, paramName := parseParamFromComment(node.LineComment)
		if paramName != "" {
			if value, ok := params[paramName]; ok {
				updateNodeValue(node, value, directive)
			}
			node.LineComment = ""
		}
	}

	for _, child := range node.Content {
		traverse(child, params)
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
	// 1. Convert new value into a yaml.Node
	tempYaml, err := yaml.Marshal(newValue)
	if err != nil {
		node.Tag = "!!str"
		node.Value = fmt.Sprintf("ERROR_MARSHALING:%v", err)
		return
	}

	var tempRootNode yaml.Node
	err = yaml.Unmarshal(tempYaml, &tempRootNode)
	if err != nil {
		node.Tag = "!!str"
		node.Value = fmt.Sprintf("ERROR_MARSHALING:%v", err)
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
