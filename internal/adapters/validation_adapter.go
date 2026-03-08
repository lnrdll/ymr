package adapters

import (
	"github.com/lnrdll/ymr/internal/ports"
	"github.com/lnrdll/ymr/internal/spec"
	"github.com/lnrdll/ymr/internal/validation"
)

type validationAdapter struct{}

func NewValidationAdapter() ports.ValidationPort {
	return validationAdapter{}
}

func (validationAdapter) NewEngine(validations []spec.Validation) (ports.ValidationEnginePort, error) {
	return validation.NewEngine(validations)
}
