package adapters

import (
	"github.com/lnrdll/ymr/internal/ports"
	"github.com/lnrdll/ymr/internal/processor"
)

type processorAdapter struct{}

func NewProcessorAdapter() ports.ProcessorPort {
	return processorAdapter{}
}

func (processorAdapter) ProcessContent(templateContent []byte, params map[string]any) (string, error) {
	return processor.ProcessContent(templateContent, params)
}
