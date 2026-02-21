package app

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/lnrdll/ymr/internal/source"
)

type planOutput struct {
	Template string
	Target   string
	File     string
}

func Plan(cfg Config) (string, error) {
	token := getGithubToken(cfg.GithubToken)

	specConfig, loader, err := loadSpecConfig(cfg, token)
	if err != nil {
		return "", err
	}

	applyTemplateOverride(specConfig, cfg.OverrideTemplate)

	outputDir, terminalOutput := prepareOutputDir(cfg.OutputDir)
	destination := "stdout"
	if !terminalOutput {
		if outputDir == "" {
			destination = "."
		} else {
			destination = outputDir
		}
	}

	mode := "spec-less"
	sourceDesc := fmt.Sprintf("template=%s", cfg.OverrideTemplate)
	if cfg.IsSpecFile {
		mode = "spec"
		sourceDesc = fmt.Sprintf("spec=%s", cfg.SpecFile)
	}

	targets, missingTargets := planTargets(specConfig.TargetIds, cfg.OverrideTargets)

	outputs := make([]planOutput, 0)
	for _, templatePath := range specConfig.Templates {
		base := filepath.Base(templatePath)
		ext := filepath.Ext(base)
		nameOnly := strings.TrimSuffix(base, ext)

		for _, targetId := range targets {
			outFile := fmt.Sprintf("%s-%s%s", targetId, nameOnly, ext)
			outputs = append(outputs, planOutput{Template: templatePath, Target: targetId, File: outFile})
		}
	}

	maxFileLen := 0
	for _, o := range outputs {
		if len(o.File) > maxFileLen {
			maxFileLen = len(o.File)
		}
	}

	var b strings.Builder
	b.WriteString("Plan\n")
	b.WriteString(fmt.Sprintf("Mode: %s (%s)\n", mode, sourceDesc))
	b.WriteString(fmt.Sprintf("Destination: %s\n", destination))
	b.WriteString(fmt.Sprintf("Targets (%d): %s\n", len(targets), joinOrNone(targets)))
	b.WriteString(fmt.Sprintf("Templates (%d): %s\n", len(specConfig.Templates), joinOrNone(specConfig.Templates)))
	b.WriteString(fmt.Sprintf("Outputs (%d):\n", len(outputs)))

	if len(outputs) == 0 {
		b.WriteString("(none)\n")
	}

	for _, o := range outputs {
		b.WriteString(fmt.Sprintf("%-*s  <- %s\n", maxFileLen, o.File, o.Template))
	}

	var notes []string
	for _, o := range outputs {
		if strings.HasPrefix(o.File, "-") {
			notes = append(notes, "Spec-less mode without --target produces filenames starting with '-' (pass -t to make filenames stable)")
			break
		}
	}
	if len(missingTargets) > 0 {
		notes = append(notes, fmt.Sprintf("Requested targets not in spec: %s", strings.Join(missingTargets, ", ")))
	}
	if _, ok := loader.(*source.GithubLoader); ok {
		notes = append(notes, "Templates/spec may be remote; plan does not fetch templates")
	}
	if _, ok := loader.(*source.HTTPLoader); ok {
		notes = append(notes, "Templates/spec may be remote; plan does not fetch templates")
	}

	if len(notes) > 0 {
		b.WriteString("Notes:\n")
		for _, n := range dedupeStrings(notes) {
			b.WriteString("- " + n + "\n")
		}
	}

	return b.String(), nil
}

func planTargets(specTargetIds []string, overrideTargets []string) (targets []string, missing []string) {
	if len(overrideTargets) == 0 {
		return specTargetIds, nil
	}

	for _, t := range overrideTargets {
		if slices.Contains(specTargetIds, t) {
			targets = append(targets, t)
		} else {
			missing = append(missing, t)
		}
	}

	return targets, missing
}

func joinOrNone(items []string) string {
	if len(items) == 0 {
		return "(none)"
	}
	return strings.Join(items, ", ")
}

func dedupeStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
