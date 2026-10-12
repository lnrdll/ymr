package source

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

func (r resolver) LoadParams(filePath string, token string) (map[string]any, error) {
	data, err := r.acquire(filePath, token)
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
