package source

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func LoadParams(filePath string, token string) (map[string]any, error) {
	var data []byte
	var err error
	if isRemotePath(filePath) {
		data, err = fetch(filePath, token, false)
	} else if _, ok := ParseGitHubURL(filePath); ok {
		data, err = fetch(filePath, token, false)
	} else {
		data, err = os.ReadFile(filePath)
	}
	if err != nil {
		return nil, fmt.Errorf("reading params file '%s': %w", filePath, err)
	}

	var params map[string]any
	if err := yaml.Unmarshal(data, &params); err != nil {
		return nil, fmt.Errorf("unmarshaling params file '%s': %w", filePath, err)
	}

	if params == nil {
		params = make(map[string]any)
	}

	return params, nil
}
