package app

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/lnrdll/ymr/internal/processor"
	"github.com/lnrdll/ymr/internal/source"
	"github.com/lnrdll/ymr/internal/spec"
)

func Validate(cfg Config) error {
	plan, err := preparePlan(cfg)
	if err != nil {
		return err
	}

	targetsToValidate := plan.Targets
	if cfg.Strict && len(cfg.OverrideTargets) > 0 && len(targetsToValidate) == 0 {
		return fmt.Errorf("no requested targets matched spec targetIds")
	}

	if err := validateTargets(targetsToValidate, plan.ParamLookup, plan.ParamsOverride, plan.Validations); err != nil {
		return err
	}

	if err := validateTemplates(plan.SpecConfig, plan.Loader, plan.Token, targetsToValidate, plan.ParamLookup, plan.ParamsOverride, cfg.OverrideTemplate, cfg.Strict); err != nil {
		return err
	}

	return nil
}

func validateTemplates(
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
			cwd, _ := os.Getwd()
			loaderToUse = &source.LocalLoader{BaseDir: cwd, SpecPath: ""}
		}

		content, err := source.LoadTemplate(loaderToUse, templatePath, token)
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

			_, err := processor.ProcessContent(content, params)
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
