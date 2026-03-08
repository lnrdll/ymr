package app

import (
	source "github.com/lnrdll/ymr/internal/adapters/source"
	config "github.com/lnrdll/ymr/internal/domain/config"
)

type executionPlan struct {
	Token          string
	SpecConfig     *config.SpecConfig
	Loader         source.SourceLoader
	ParamLookup    map[string]map[string]any
	ParamsOverride map[string]any
	Targets        []string
	Validations    []config.Validation
}

func preparePlan(cfg Config) (*executionPlan, error) {
	return preparePlanWithDeps(cfg, newDefaultDeps())
}

func preparePlanWithDeps(cfg Config, deps appDeps) (*executionPlan, error) {
	token := getGithubTokenWithRuntime(cfg.GithubToken, deps.runtime)

	specConfig, loader, err := loadSpecConfigWithDeps(cfg, token, deps)
	if err != nil {
		return nil, err
	}

	applyTemplateOverride(specConfig, cfg.OverrideTemplate)

	paramLookup := config.BuildParamLookup(specConfig)
	paramsOverride, err := applyParamsOverrides(deps.source, cfg.OverrideParams, cfg.OverrideParamFiles, cfg.OverrideParamYAML, token)
	if err != nil {
		return nil, err
	}

	targets := filterTargets(specConfig.TargetIds, cfg.OverrideTargets)

	validations := specConfig.Validations
	if cfg.ValidationFile != "" {
		validations, err = deps.source.LoadValidations(cfg.ValidationFile, token)
		if err != nil {
			return nil, err
		}
	}

	return &executionPlan{
		Token:          token,
		SpecConfig:     specConfig,
		Loader:         loader,
		ParamLookup:    paramLookup,
		ParamsOverride: paramsOverride,
		Targets:        targets,
		Validations:    validations,
	}, nil
}
