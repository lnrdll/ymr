package adapters

import (
	"os"

	"github.com/lnrdll/ymr/internal/ports"
)

type runtimeAdapter struct{}

func NewRuntimeAdapter() ports.RuntimePort {
	return runtimeAdapter{}
}

func (runtimeAdapter) Getwd() (string, error) {
	return os.Getwd()
}

func (runtimeAdapter) Getenv(key string) string {
	return os.Getenv(key)
}
