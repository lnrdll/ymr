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
	token := getGithubToken(cfg.GithubToken)

	specConfig, loader, err := loadSpecConfig(cfg, token)
	if err != nil {
		return err
	}

	applyTemplateOverride(specConfig, cfg.OverrideTemplate)

	paramLookup := spec.BuildParamLookup(specConfig)
	paramsOverride, err := applyParamsOverrides(paramLookup, cfg.OverrideParams, cfg.OverrideParamFiles, cfg.OverrideParamYAML, token)
	if err != nil {
		return err
	}

	targetsToValidate := filterTargets(specConfig.TargetIds, cfg.OverrideTargets)
	if cfg.Strict && len(cfg.OverrideTargets) > 0 && len(targetsToValidate) == 0 {
		return fmt.Errorf("no requested targets matched spec targetIds")
	}

	validations := specConfig.Validations
	if cfg.ValidationFile != "" {
		validations, err = source.LoadValidations(cfg.ValidationFile, token)
		if err != nil {
			return err
		}
	}

	if err := validateTargets(targetsToValidate, paramLookup, paramsOverride, validations); err != nil {
		return err
	}

	if err := validateTemplates(specConfig, loader, token, targetsToValidate, paramLookup, paramsOverride, cfg.OverrideTemplate, cfg.Strict); err != nil {
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
			params, ok := paramLookup[targetId]
			if !ok {
				params = make(map[string]any)
			}
			if len(paramsOverride) > 0 && !ok {
				applyParamsOverride(params, paramsOverride)
			}

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
