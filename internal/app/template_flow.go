package app

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/lnrdll/ymr/internal/processor"
	"github.com/lnrdll/ymr/internal/source"
	"github.com/lnrdll/ymr/internal/spec"
)

func processTemplates(
	deps appDeps,
	specConfig *spec.SpecConfig,
	loader source.SourceLoader,
	token string,
	targetsToRender []string,
	paramLookup map[string]map[string]any,
	paramsOverride map[string]any,
	overrideTemplate string,
	strict bool,
) ([]processor.RenderedOutput, error) {
	allOutputs := []processor.RenderedOutput{}

	err := walkTemplateTargets(
		deps,
		specConfig,
		loader,
		token,
		targetsToRender,
		paramLookup,
		paramsOverride,
		overrideTemplate,
		strict,
		true,
		func(templatePath string, templateNameOnly string, templateExt string, targetId string, renderedYaml string) {
			outputFileName := fmt.Sprintf("%s-%s%s", targetId, templateNameOnly, templateExt)
			allOutputs = append(allOutputs, processor.RenderedOutput{
				TargetFile:   outputFileName,
				TemplateUsed: templatePath,
				Content:      renderedYaml,
			})
		},
	)

	return allOutputs, err
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
	return walkTemplateTargets(
		deps,
		specConfig,
		loader,
		token,
		targets,
		paramLookup,
		paramsOverride,
		overrideTemplate,
		strict,
		false,
		func(string, string, string, string, string) {},
	)
}

func walkTemplateTargets(
	deps appDeps,
	specConfig *spec.SpecConfig,
	loader source.SourceLoader,
	token string,
	targets []string,
	paramLookup map[string]map[string]any,
	paramsOverride map[string]any,
	overrideTemplate string,
	strict bool,
	warnOnStrict bool,
	onRendered func(templatePath string, templateNameOnly string, templateExt string, targetId string, renderedYaml string),
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
			}
			if !strict || warnOnStrict {
				slog.Warn("Skipping template due to load error", "template", templatePath, "error", err)
			}
			continue
		}

		templateBaseName := filepath.Base(templatePath)
		templateExt := filepath.Ext(templateBaseName)
		templateNameOnly := strings.TrimSuffix(templateBaseName, templateExt)

		for _, targetId := range targets {
			params := resolveParamsForTarget(paramLookup, targetId, paramsOverride)

			renderedYaml, err := deps.processor.ProcessContent(content, params)
			if err != nil {
				if strict {
					errs = append(errs, fmt.Errorf("processing template '%s' for target '%s': %w", templatePath, targetId, err))
				}
				if !strict || warnOnStrict {
					slog.Warn("Skipping template/target due to processing error", "template", templatePath, "target", targetId, "error", err)
				}
				continue
			}

			onRendered(templatePath, templateNameOnly, templateExt, targetId, renderedYaml)
		}
	}

	if strict && len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}
