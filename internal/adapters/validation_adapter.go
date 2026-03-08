package adapters

import (
	validation "github.com/lnrdll/ymr/internal/adapters/validation"
	config "github.com/lnrdll/ymr/internal/domain/config"
	"github.com/lnrdll/ymr/internal/ports"
)

type validationAdapter struct{}

func NewValidationAdapter() ports.ValidationPort {
	return validationAdapter{}
}

func (validationAdapter) NewEngine(validations []config.Validation) (ports.ValidationEnginePort, error) {
	return validation.NewEngine(validations)
}
