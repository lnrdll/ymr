package app

import (
	"fmt"
	"log/slog"

	config "github.com/lnrdll/ymr/internal/domain/config"
	"github.com/lnrdll/ymr/internal/ports"
)

func loadSpecConfigWithDeps(cfg Config, token string, deps appDeps) (*config.SpecConfig, ports.SourceLoader, error) {
	if cfg.IsSpecFile {
		slog.Debug("Using source loader", "source", cfg.SpecFile)

		loader, err := deps.source.NewSourceLoader(cfg.SpecFile, token)
		if err != nil {
			return nil, nil, err
		}

		specConfig, err := loader.LoadSpec(token)
		if err != nil {
			return nil, nil, fmt.Errorf("loading spec file: %w", err)
		}

		return specConfig, loader, nil
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

	cwd, err := deps.runtime.Getwd()
	if err != nil {
		return nil, nil, fmt.Errorf("resolving working directory: %w", err)
	}
	loader := deps.source.NewLocalLoader(cwd)

	slog.Debug("Running in spec-less mode", "loader", loader)

	return specConfig, loader, nil
}

func applyTemplateOverride(specConfig *config.SpecConfig, overrideTemplate string) {
	if overrideTemplate != "" {
		slog.Debug("Overriding template", "template", overrideTemplate)
		specConfig.Templates = []string{overrideTemplate}
	}
}
