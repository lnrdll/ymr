package app

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	config "github.com/lnrdll/ymr/internal/domain/config"
	"github.com/lnrdll/ymr/internal/ports"
)

func processTemplates(
	deps appDeps,
	specConfig *config.SpecConfig,
	specLocation string,
	token string,
	targetsToRender []string,
	paramLookup map[string]map[string]any,
	paramsOverride map[string]any,
	overrideTemplate string,
	strict bool,
) ([]ports.RenderedOutput, error) {
	allOutputs := []ports.RenderedOutput{}

	err := walkTemplateTargets(
		deps,
		specConfig,
		specLocation,
		token,
		targetsToRender,
		paramLookup,
		paramsOverride,
		overrideTemplate,
		strict,
		true,
		func(templatePath string, templateNameOnly string, templateExt string, targetId string, renderedYaml string) {
			outputFileName := fmt.Sprintf("%s-%s%s", targetId, templateNameOnly, templateExt)
			allOutputs = append(allOutputs, ports.RenderedOutput{
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
	specConfig *config.SpecConfig,
	specLocation string,
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
		specLocation,
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
	specConfig *config.SpecConfig,
	specLocation string,
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
		content, err := loadTemplateContent(deps.source, specLocation, templatePath, overrideTemplate, token)
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

			renderedYaml, err := deps.processor.ProcessContent(content, params, strict)
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

// loadTemplateContent resolves a --template override (spec mode or spec-less)
// against the working directory and a spec-listed template against the spec.
func loadTemplateContent(source ports.SourcePort, specLocation string, templatePath string, overrideTemplate string, token string) ([]byte, error) {
	if overrideTemplate != "" {
		return source.LoadTemplate(templatePath, token)
	}
	return source.LoadSpecTemplate(specLocation, templatePath, token)
}
