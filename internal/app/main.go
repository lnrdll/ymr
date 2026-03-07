package app

import (
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/lnrdll/ymr/internal/processor"
	"github.com/lnrdll/ymr/internal/source"
	"github.com/lnrdll/ymr/internal/spec"
	"gopkg.in/yaml.v3"
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
	validations []spec.Validation,
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

// handleOutput writes rendered content to files or to the console.
// It creates the output directory if it doesn't exist.
func handleOutput(outputPort OutputPort, outputs []processor.RenderedOutput, terminalOutput bool, outputDir string) error {
	return outputPort.Write(outputs, terminalOutput, outputDir)
}

// prepareOutputDir determines the output directory and whether to print to the terminal.
func prepareOutputDir(cfgOutputDir string) (string, bool) {
	if cfgOutputDir == "-" {
		return "", true // Terminal output
	}
	return cfgOutputDir, false
}

func applyParamsOverrides(
	sourcePort SourcePort,
	overrideParams []string,
	overrideParamFiles []string,
	overrideParamYAML []string,
	token string,
) (map[string]any, error) {
	paramsOverride, err := buildParamsOverride(sourcePort, overrideParams, overrideParamFiles, overrideParamYAML, token)
	if err != nil {
		return nil, fmt.Errorf("parsing override parameters: %w", err)
	}

	if len(paramsOverride) > 0 {
		slog.Debug("Overriding parameters", "count", len(paramsOverride), "keys", mapKeys(paramsOverride))
	}
	return paramsOverride, nil
}

func resolveParamsForTarget(
	paramLookup map[string]map[string]any,
	targetId string,
	paramsOverride map[string]any,
) map[string]any {
	resolved := make(map[string]any)

	if base, ok := paramLookup[targetId]; ok {
		maps.Copy(resolved, base)
	}

	if len(paramsOverride) > 0 {
		maps.Copy(resolved, paramsOverride)
	}

	return resolved
}

func buildParamsOverride(
	sourcePort SourcePort,
	overrideParams []string,
	overrideParamFiles []string,
	overrideParamYAML []string,
	token string,
) (map[string]any, error) {
	overrides := make(map[string]any)

	for _, p := range overrideParamFiles {
		m, err := sourcePort.LoadParams(p, token)
		if err != nil {
			return nil, err
		}
		maps.Copy(overrides, m)
	}

	for _, s := range overrideParamYAML {
		var m map[string]any
		if err := yaml.Unmarshal([]byte(s), &m); err != nil {
			return nil, fmt.Errorf("parsing --param-yaml: %w", err)
		}
		if m == nil {
			m = make(map[string]any)
		}
		maps.Copy(overrides, m)
	}

	m, err := spec.ParseCliParams(overrideParams)
	if err != nil {
		return nil, err
	}
	maps.Copy(overrides, m)

	return overrides, nil
}

func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// filterTargets now performs strict explicit matching.
// If overrideTargets is provided, only targets existing in BOTH slices are returned.
func filterTargets(specTargetIds []string, overrideTargets []string) []string {
	if len(overrideTargets) == 0 {
		slog.Debug("Rendering all targets", "count", len(specTargetIds))
		return specTargetIds
	}

	filtered := make([]string, 0)
	for _, t := range overrideTargets {
		if slices.Contains(specTargetIds, t) {
			filtered = append(filtered, t)
		} else {
			slog.Warn("Requested target not found in spec", "targetId", t)
		}
	}

	slog.Debug("Rendering specific targets", "targets", filtered)
	return filtered
}

// processTemplates processes each template against each target and returns the rendered outputs.
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
	var renderErrs []error

	for _, templatePath := range specConfig.Templates {
		var templateContent []byte
		var err error

		loaderToUse := loader
		if overrideTemplate != "" {
			cwd, _ := deps.runtime.Getwd()
			loaderToUse = deps.source.NewLocalLoader(cwd)
		}

		templateContent, err = deps.source.LoadTemplate(loaderToUse, templatePath, token)
		if err != nil {
			if strict {
				renderErrs = append(renderErrs, fmt.Errorf("loading template '%s': %w", templatePath, err))
			}
			slog.Warn("Skipping template due to load error", "template", templatePath, "error", err)
			continue
		}

		templateBaseName := filepath.Base(templatePath)
		templateExt := filepath.Ext(templateBaseName)
		templateNameOnly := strings.TrimSuffix(templateBaseName, templateExt)

		for _, targetId := range targetsToRender {
			params := resolveParamsForTarget(paramLookup, targetId, paramsOverride)

			renderedYaml, err := deps.processor.ProcessContent(templateContent, params)
			if err != nil {
				if strict {
					renderErrs = append(renderErrs, fmt.Errorf("processing template '%s' for target '%s': %w", templatePath, targetId, err))
				}
				slog.Warn("Skipping template/target due to processing error", "template", templatePath, "target", targetId, "error", err)
				continue
			}

			outputFileName := fmt.Sprintf("%s-%s%s", targetId, templateNameOnly, templateExt)

			allOutputs = append(allOutputs, processor.RenderedOutput{
				TargetFile:   outputFileName,
				TemplateUsed: templatePath,
				Content:      renderedYaml,
			})
		}
	}

	if strict && len(renderErrs) > 0 {
		return allOutputs, errors.Join(renderErrs...)
	}
	return allOutputs, nil
}

// loadSpecConfig loads the specification configuration based on the provided application config.
// It handles both spec file-based and spec-less modes.
func loadSpecConfig(cfg Config, token string) (*spec.SpecConfig, source.SourceLoader, error) {
	return loadSpecConfigWithDeps(cfg, token, newDefaultDeps())
}

func loadSpecConfigWithDeps(cfg Config, token string, deps appDeps) (*spec.SpecConfig, source.SourceLoader, error) {
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

	// Spec-less mode
	targetIds := []string{""}
	if len(cfg.OverrideTargets) > 0 {
		targetIds = cfg.OverrideTargets
	}

	specConfig := &spec.SpecConfig{
		Templates:  []string{cfg.OverrideTemplate},
		TargetIds:  targetIds,
		Parameters: []spec.ParamSet{},
	}

	cwd, _ := deps.runtime.Getwd()
	loader := deps.source.NewLocalLoader(cwd)

	slog.Debug("Running in spec-less mode", "loader", loader)

	return specConfig, loader, nil
}

// applyTemplateOverride overrides the templates in the spec config if an override is provided.
func applyTemplateOverride(specConfig *spec.SpecConfig, overrideTemplate string) {
	if overrideTemplate != "" {
		slog.Debug("Overriding template", "template", overrideTemplate)
		specConfig.Templates = []string{overrideTemplate}
	}
}

// getGithubToken return a github token if provided.
func getGithubToken(t string) string {
	return getGithubTokenWithRuntime(t, newDefaultDeps().runtime)
}

func getGithubTokenWithRuntime(t string, runtimePort RuntimePort) string {
	if t == "" {
		return runtimePort.Getenv("GITHUB_TOKEN")
	}

	return t
}
