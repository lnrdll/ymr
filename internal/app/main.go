package app

import (
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"ymr/internal/processor"
	"ymr/internal/source"
	"ymr/internal/spec"
)

// applyCliOverrides applies CLI provided parameter overrides to a given parameter map.
func applyCliOverrides(paramMap map[string]any, cliOverrides map[string]any) {
	maps.Copy(paramMap, cliOverrides)
}

// Run is the main entrypoint for the application logic.
func Run(cfg Config) error {
	token := cfg.GithubToken
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}

	// Handle output dir/'-o -' logic
	terminalOutput := false
	outputDir := cfg.OutputDir
	if outputDir == "-" {
		terminalOutput = true
		outputDir = ""
	}

	// Load the specs
	var specConfig *spec.SpecConfig
	var loader source.SourceLoader
	var err error

	if cfg.SpecFile != "" {
		// Spec-file mode
		slog.Debug("Using source loader", "source", cfg.SpecFile)

		loader, err = source.NewSourceLoader(cfg.SpecFile, token)
		if err != nil {
			return err
		}

		specConfig, err = loader.LoadSpec(token)
		if err != nil {
			return fmt.Errorf("loading spec file: %w", err)
		}
	} else {
		// Spec-Less mode
		slog.Debug("Running in spec-less mode (no spec file found or provided)")
		specConfig = &spec.SpecConfig{
			Templates:  []string{cfg.OverrideTemplate},
			TargetIds:  cfg.OverrideTargets,
			Parameters: []spec.ParamSet{},
		}

		// Create a LocalLoader based on the current directory
		cwd, _ := os.Getwd()
		loader = &source.LocalLoader{BaseDir: cwd, SpecPath: ""}
	}

	// (Override) Handle CLI template override
	if cfg.OverrideTemplate != "" {
		specConfig.Templates = []string{cfg.OverrideTemplate}
		slog.Debug("Overriding templates", "template", cfg.OverrideTemplate)
	}

	// Build the parameter map
	paramLookup := spec.BuildParamLookup(specConfig)

	// (Override) Parse and apply CLI parameters
	cliOverrides, err := spec.ParseCliParams(cfg.OverrideParams)
	if err != nil {
		return fmt.Errorf("parsing override parameters: %w", err)
	}

	if len(cliOverrides) > 0 {
		slog.Debug("Applying CLI parameter overrides", "count", len(cliOverrides))
		for targetId, paramMap := range paramLookup {
			applyCliOverrides(paramMap, cliOverrides)
			paramLookup[targetId] = paramMap
		}
	}

	// (Override) Filter targets
	targetsToRender := specConfig.TargetIds
	if len(cfg.OverrideTargets) > 0 {
		filteredTargets := make([]string, 0)
		cliTargetSet := make(map[string]bool)
		for _, t := range cfg.OverrideTargets {
			cliTargetSet[t] = true
		}
		for _, specTargetId := range specConfig.TargetIds {
			if _, ok := cliTargetSet[specTargetId]; ok {
				filteredTargets = append(filteredTargets, specTargetId)
			}
		}
		targetsToRender = filteredTargets
		slog.Debug("Rendering specific targets", "targets", targetsToRender)
	} else {
		slog.Debug("Rendering all targets", "count", len(targetsToRender))
	}

	// Process each template against each target
	allOutputs := []processor.RenderedOutput{}
	for _, templatePath := range specConfig.Templates {
		var templateContent []byte
		var err error

		// If a template is specified via CLI flag, it should be loaded relative to the CWD.
		// Passing a nil loader to LoadTemplate signals that it should use the CWD as the base path.
		if cfg.OverrideTemplate != "" && templatePath == cfg.OverrideTemplate {
			templateContent, err = source.LoadTemplate(nil, templatePath, token)
		} else {
			templateContent, err = source.LoadTemplate(loader, templatePath, token)
		}

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

			if len(cliOverrides) > 0 && !ok {
				applyCliOverrides(params, cliOverrides)
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

	// Handle Output
	return handleOutput(allOutputs, terminalOutput, outputDir)
}

// handleOutput writes rendered content to files or to the console.
// It creates the output directory if it doesn't exist.
func handleOutput(outputs []processor.RenderedOutput, terminalOutput bool, outputDir string) error {
	if terminalOutput {
		for _, output := range outputs {
			fmt.Println(output.Content)
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
