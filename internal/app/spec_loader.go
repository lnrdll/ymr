package app

import (
	"fmt"
	"log/slog"

	config "github.com/lnrdll/ymr/internal/domain/config"
)

func loadSpecConfigWithDeps(cfg Config, token string, deps appDeps) (*config.SpecConfig, error) {
	if cfg.IsSpecFile {
		slog.Debug("Loading spec", "source", cfg.SpecFile)

		specConfig, err := deps.source.LoadSpec(cfg.SpecFile, token)
		if err != nil {
			return nil, fmt.Errorf("loading spec file: %w", err)
		}

		return specConfig, nil
	}

	targetIds := []string{""}
	if len(cfg.OverrideTargets) > 0 {
		targetIds = cfg.OverrideTargets
	}

	specConfig := &config.SpecConfig{
		Templates:  []string{cfg.OverrideTemplate},
		TargetIds:  targetIds,
		Parameters: []config.ParamSet{},
	}

	slog.Debug("Running in spec-less mode", "template", cfg.OverrideTemplate)

	return specConfig, nil
}

func applyTemplateOverride(specConfig *config.SpecConfig, overrideTemplate string) {
	if overrideTemplate != "" {
		slog.Debug("Overriding template", "template", overrideTemplate)
		specConfig.Templates = []string{overrideTemplate}
	}
}
