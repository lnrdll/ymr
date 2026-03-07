package app

import (
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/lnrdll/ymr/internal/processor"
	"github.com/lnrdll/ymr/internal/source"
	"github.com/lnrdll/ymr/internal/spec"
	"github.com/lnrdll/ymr/internal/validation"
	"gopkg.in/yaml.v3"
)

// Run is the main entrypoint for the application logic.
func Run(cfg Config) error {
	// Handle output dir/'-o -' logic
	outputDir, terminalOutput := prepareOutputDir(cfg.OutputDir)

	plan, err := preparePlan(cfg)
	if err != nil {
		return err
	}

	// Validate Rules
	if err := validateTargets(plan.Targets, plan.ParamLookup, plan.ParamsOverride, plan.Validations); err != nil {
		return err
	}

	// Process each template against each target
	allOutputs, renderErr := processTemplates(
		plan.SpecConfig,
		plan.Loader,
		plan.Token,
		plan.Targets,
		plan.ParamLookup,
		plan.ParamsOverride,
		cfg.OverrideTemplate,
		cfg.Strict,
	)
	if cfg.Strict && renderErr != nil {
		return renderErr
	}

	// Handle Output
	return handleOutput(allOutputs, terminalOutput, outputDir)
}

func validateTargets(
	targetsToRender []string,
	paramLookup map[string]map[string]any,
	paramsOverride map[string]any,
	validations []spec.Validation,
) error {
	engine, err := validation.NewEngine(validations)
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
func handleOutput(
	outputs []processor.RenderedOutput,
	terminalOutput bool,
	outputDir string,
) error {
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
			slog.Debug("Failed to write output file", "path", outPath, "error", err)
			writeErrs = append(writeErrs, fmt.Errorf("writing output file '%s': %w", outPath, err))
			continue
		} else {
			slog.Debug("Generated file", "path", outPath)
		}
	}

	if len(writeErrs) > 0 {
		return errors.Join(writeErrs...)
	}

	return nil
}

// prepareOutputDir determines the output directory and whether to print to the terminal.
func prepareOutputDir(cfgOutputDir string) (string, bool) {
	if cfgOutputDir == "-" {
		return "", true // Terminal output
	}
	return cfgOutputDir, false
}

func applyParamsOverrides(
	overrideParams []string,
	overrideParamFiles []string,
	overrideParamYAML []string,
	token string,
) (map[string]any, error) {
	paramsOverride, err := buildParamsOverride(overrideParams, overrideParamFiles, overrideParamYAML, token)
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
	overrideParams []string,
	overrideParamFiles []string,
	overrideParamYAML []string,
	token string,
) (map[string]any, error) {
	overrides := make(map[string]any)

	for _, p := range overrideParamFiles {
		m, err := source.LoadParams(p, token)
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
			cwd, _ := os.Getwd()
			loaderToUse = &source.LocalLoader{BaseDir: cwd, SpecPath: ""}
		}

		templateContent, err = source.LoadTemplate(loaderToUse, templatePath, token)
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

			renderedYaml, err := processor.ProcessContent(templateContent, params)
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
	if cfg.IsSpecFile {
		slog.Debug("Using source loader", "source", cfg.SpecFile)

		loader, err := source.NewSourceLoader(cfg.SpecFile, token)
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

	cwd, _ := os.Getwd()
	loader := &source.LocalLoader{BaseDir: cwd, SpecPath: ""}

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
	if t == "" {
		return os.Getenv("GITHUB_TOKEN")
	}

	return t
}
