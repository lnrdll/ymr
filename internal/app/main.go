package app

import (
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"ymr/internal/processor"
	"ymr/internal/source"
	"ymr/internal/spec"
	"ymr/internal/validation"
)

// Run is the main entrypoint for the application logic.
func Run(cfg Config) error {
	token := getGithubToken(cfg.GithubToken)

	// Handle output dir/'-o -' logic
	outputDir, terminalOutput := prepareOutputDir(cfg.OutputDir)

	// Load the specs
	specConfig, loader, err := loadSpecConfig(cfg, token)
	if err != nil {
		return err
	}

	// (Override) Template
	applyTemplateOverride(specConfig, cfg.OverrideTemplate)

	// Build the parameter map
	paramLookup := spec.BuildParamLookup(specConfig)

	// (Override) Parameters
	paramsOverride, err := applyParamsOverrides(paramLookup, cfg.OverrideParams)
	if err != nil {
		return err
	}

	// (Override) Targets
	targetsToRender := filterTargets(specConfig.TargetIds, cfg.OverrideTargets)

	// Load validations
	validations := specConfig.Validations
	if cfg.ValidationFile != "" {
		var err error
		validations, err = source.LoadValidations(cfg.ValidationFile, token)
		if err != nil {
			return err
		}
	}

	// Validate Rules
	if err := validateTargets(targetsToRender, paramLookup, paramsOverride, validations); err != nil {
		return err
	}

	// Process each template against each target
	allOutputs := processTemplates(specConfig, loader, token, targetsToRender, paramLookup, paramsOverride, cfg.OverrideTemplate)

	// Handle Output
	return handleOutput(allOutputs, terminalOutput, outputDir)
}

func validateTargets(
	targetsToRender []string,
	paramLookup map[string]map[string]any,
	paramsOverride map[string]any,
	validations []spec.Validation,
) error {
	for _, targetId := range targetsToRender {
		params, ok := paramLookup[targetId]
		if !ok {
			params = make(map[string]any)
		}

		if len(paramsOverride) > 0 && !ok {
			applyParamsOverride(params, paramsOverride)
		}

		// Filter validations for the current target
		targetValidations := []spec.Validation{}
		for _, v := range validations {
			if len(v.TargetId) == 0 {
				targetValidations = append(targetValidations, v)
				continue
			}

			if slices.Contains(v.TargetId, targetId) {
				targetValidations = append(targetValidations, v)
			}
		}

		if err := validation.Check(params, targetValidations); err != nil {
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

	for _, output := range outputs {
		outPath := filepath.Join(outputDir, output.TargetFile)
		err := os.WriteFile(outPath, []byte(output.Content), 0644)
		if err != nil {
			slog.Debug(fmt.Sprintf("Skipping file '%s' due to error: %v", outPath, err))
			continue
		} else {
			slog.Debug("Generated file", "path", outPath)
		}
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

// applyParamsOverride applies CLI provided parameter overrides to a given parameter map.
func applyParamsOverride(paramMap map[string]any, cliOverrides map[string]any) {
	maps.Copy(paramMap, cliOverrides)
}

// applyParamsOverrides parses and applies CLI parameter overrides to the parameter lookup.
// It returns the parsed override parameters.
func applyParamsOverrides(
	paramLookup map[string]map[string]any,
	overrideParams []string,
) (map[string]any, error) {
	paramsOverride, err := spec.ParseCliParams(overrideParams)
	if err != nil {
		return nil, fmt.Errorf("parsing override parameters: %w", err)
	}

	if len(paramsOverride) > 0 {
		slog.Debug("Overriding parameters", "count", len(paramsOverride), "parameters", paramsOverride)
		for _, paramMap := range paramLookup {
			applyParamsOverride(paramMap, paramsOverride)
		}
	}
	return paramsOverride, nil
}

// filterTargets filters the targets to be rendered based on the provided override targets.
func filterTargets(specTargetIds []string, overrideTargets []string) []string {
	if len(overrideTargets) == 0 {
		slog.Debug("Rendering all targets", "count", len(specTargetIds))
		return specTargetIds
	}

	filteredTargets := make([]string, 0)
	cliTargetSet := make(map[string]bool)

	for _, t := range overrideTargets {
		cliTargetSet[t] = true
	}

	for _, specTargetId := range specTargetIds {
		if _, ok := cliTargetSet[specTargetId]; ok {
			filteredTargets = append(filteredTargets, specTargetId)
		}
	}

	slog.Debug("Rendering specific targets", "targets", filteredTargets)
	return filteredTargets
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
) []processor.RenderedOutput {
	allOutputs := []processor.RenderedOutput{}

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
			slog.Debug(fmt.Sprintf("Skipping template '%s' due to error: %v", templatePath, err))
			continue
		}

		templateBaseName := filepath.Base(templatePath)
		templateExt := filepath.Ext(templateBaseName)
		templateNameOnly := strings.TrimSuffix(templateBaseName, templateExt)

		for _, targetId := range targetsToRender {
			params, ok := paramLookup[targetId]
			if !ok {
				params = make(map[string]any)
			}

			if len(paramsOverride) > 0 && !ok {
				applyParamsOverride(params, paramsOverride)
			}

			renderedYaml, err := processor.ProcessContent(templateContent, params)
			if err != nil {
				slog.Debug(fmt.Sprintf("Skipping template '%s' for target '%s' due to error: %v", templatePath, targetId, err))
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

	return allOutputs
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
	specConfig := &spec.SpecConfig{
		Templates:  []string{cfg.OverrideTemplate},
		TargetIds:  []string{cfg.SpecFile},
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
