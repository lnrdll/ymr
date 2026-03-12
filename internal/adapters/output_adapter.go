package adapters

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lnrdll/ymr/internal/ports"
)

type outputAdapter struct{}

func NewOutputAdapter() ports.OutputPort {
	return outputAdapter{}
}

func (outputAdapter) Write(outputs []ports.RenderedOutput, terminalOutput bool, outputDir string) error {
	if terminalOutput {
		for _, output := range outputs {
			fmt.Print(output.Content)
		}
		return nil
	}

	if outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("creating output directory '%s': %w", outputDir, err)
		}
	}

	var writeErrs []error
	for _, output := range outputs {
		outPath := filepath.Join(outputDir, output.TargetFile)
		err := os.WriteFile(outPath, []byte(output.Content), 0644)
		if err != nil {
			writeErrs = append(writeErrs, fmt.Errorf("writing output file '%s': %w", outPath, err))
		}
	}

	if len(writeErrs) > 0 {
		return errors.Join(writeErrs...)
	}

	return nil
}
