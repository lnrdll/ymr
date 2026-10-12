package source

import (
	"fmt"

	config "github.com/lnrdll/ymr/internal/domain/config"

	"gopkg.in/yaml.v3"
)

func (r resolver) LoadValidations(filePath string, token string) ([]config.Validation, error) {
	data, err := r.acquire(filePath, token)
	if err != nil {
		return nil, fmt.Errorf("reading validation file '%s': %w", filePath, err)
	}

	var validations []config.Validation
	if err := yaml.Unmarshal(data, &validations); err != nil {
		return nil, fmt.Errorf("unmarshaling validation file '%s': %w", filePath, err)
	}

	return validations, nil
}
