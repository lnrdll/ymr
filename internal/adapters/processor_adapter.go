package adapters

import (
	processor "github.com/lnrdll/ymr/internal/adapters/processor"
	"github.com/lnrdll/ymr/internal/ports"
)

type processorAdapter struct{}

func NewProcessorAdapter() ports.ProcessorPort {
	return processorAdapter{}
}

func (processorAdapter) ProcessContent(templateContent []byte, params map[string]any) (string, error) {
	return processor.ProcessContent(templateContent, params)
}
