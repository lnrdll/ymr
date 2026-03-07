package app

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/lnrdll/ymr/internal/source"
	"github.com/lnrdll/ymr/internal/spec"
)

func Validate(cfg Config) error {
	return NewValidateCommand(cfg).Execute()
}

func validateTemplates(
	deps appDeps,
	specConfig *spec.SpecConfig,
	loader source.SourceLoader,
	token string,
	targets []string,
	paramLookup map[string]map[string]any,
	paramsOverride map[string]any,
	overrideTemplate string,
	strict bool,
) error {
	var errs []error

	for _, templatePath := range specConfig.Templates {
		loaderToUse := loader
		if overrideTemplate != "" {
			cwd, _ := deps.runtime.Getwd()
			loaderToUse = deps.source.NewLocalLoader(cwd)
		}

		content, err := deps.source.LoadTemplate(loaderToUse, templatePath, token)
		if err != nil {
			if strict {
				errs = append(errs, fmt.Errorf("loading template '%s': %w", templatePath, err))
			} else {
				slog.Warn("Skipping template due to load error", "template", templatePath, "error", err)
			}
			continue
		}

		for _, targetId := range targets {
			params := resolveParamsForTarget(paramLookup, targetId, paramsOverride)

			_, err := deps.processor.ProcessContent(content, params)
			if err != nil {
				if strict {
					errs = append(errs, fmt.Errorf("processing template '%s' for target '%s': %w", templatePath, targetId, err))
				} else {
					slog.Warn("Skipping template/target due to processing error", "template", templatePath, "target", targetId, "error", err)
				}
				continue
			}
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
