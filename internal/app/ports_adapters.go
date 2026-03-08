package app

import (
	"github.com/lnrdll/ymr/internal/adapters"
	"github.com/lnrdll/ymr/internal/ports"
)

type SourcePort = ports.SourcePort
type ProcessorPort = ports.ProcessorPort
type ValidationPort = ports.ValidationPort
type ValidationEnginePort = ports.ValidationEnginePort
type OutputPort = ports.OutputPort
type RuntimePort = ports.RuntimePort

type appDeps struct {
	source     ports.SourcePort
	processor  ports.ProcessorPort
	validation ports.ValidationPort
	output     ports.OutputPort
	runtime    ports.RuntimePort
}

func newDefaultDeps() appDeps {
	return appDeps{
		source:     adapters.NewSourceAdapter(),
		processor:  adapters.NewProcessorAdapter(),
		validation: adapters.NewValidationAdapter(),
		output:     adapters.NewOutputAdapter(),
		runtime:    adapters.NewRuntimeAdapter(),
	}
}
