package adapters

import (
	source "github.com/lnrdll/ymr/internal/adapters/source"
	config "github.com/lnrdll/ymr/internal/domain/config"
	"github.com/lnrdll/ymr/internal/ports"
)

type sourceAdapter struct{}

func NewSourceAdapter() ports.SourcePort {
	return sourceAdapter{}
}

func (sourceAdapter) LoadSpec(specLocation string, token string) (*config.SpecConfig, error) {
	return source.LoadSpec(specLocation, token)
}

func (sourceAdapter) LoadSpecTemplate(specLocation string, templatePath string, token string) ([]byte, error) {
	return source.LoadSpecTemplate(specLocation, templatePath, token)
}

func (sourceAdapter) LoadTemplate(templateLocation string, token string) ([]byte, error) {
	return source.LoadTemplate(templateLocation, token)
}

func (sourceAdapter) LoadParams(location string, token string) (map[string]any, error) {
	return source.LoadParams(location, token)
}

func (sourceAdapter) LoadValidations(location string, token string) ([]config.Validation, error) {
	return source.LoadValidations(location, token)
}
