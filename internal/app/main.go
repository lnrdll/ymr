package app

import (
	"fmt"
	"log/slog"

	config "github.com/lnrdll/ymr/internal/domain/config"
)

// Run is the main entrypoint for the application logic.
func Run(cfg Config) error {
	return NewRunCommand(cfg).Execute()
}

func validateTargets(
	validationPort ValidationPort,
	targetsToRender []string,
	paramLookup map[string]map[string]any,
	paramsOverride map[string]any,
	validations []config.Validation,
) error {
	engine, err := validationPort.NewEngine(validations)
	if err != nil {
		return err
	}

	for _, targetId := range targetsToRender {
		params := resolveParamsForTarget(paramLookup, targetId, paramsOverride)

		slog.Debug("Running validations", "target", targetId)
		if err := engine.CheckTarget(targetId, params); err != nil {
			return fmt.Errorf("validation failed for target '%s': %w", targetId, err)
		}
	}

	return nil
}
