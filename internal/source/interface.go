package source

import "ymr/internal/spec"

// SourceLoader defines the interface for any configuration source
type SourceLoader interface {
	// LoadSpec fetches and parses the spec.yaml file.
	LoadSpec(token string) (*spec.SpecConfig, error)

	// LoadTemplate fetches a template file.
	LoadTemplate(templatePath string, token string) ([]byte, error)
}
