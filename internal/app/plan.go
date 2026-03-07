package app

import (
	"github.com/lnrdll/ymr/internal/source"
	"github.com/lnrdll/ymr/internal/spec"
)

type executionPlan struct {
	Token          string
	SpecConfig     *spec.SpecConfig
	Loader         source.SourceLoader
	ParamLookup    map[string]map[string]any
	ParamsOverride map[string]any
	Targets        []string
	Validations    []spec.Validation
}

func preparePlan(cfg Config) (*executionPlan, error) {
	token := getGithubToken(cfg.GithubToken)

	specConfig, loader, err := loadSpecConfig(cfg, token)
	if err != nil {
		return nil, err
	}

	applyTemplateOverride(specConfig, cfg.OverrideTemplate)

	paramLookup := spec.BuildParamLookup(specConfig)
	paramsOverride, err := applyParamsOverrides(paramLookup, cfg.OverrideParams, cfg.OverrideParamFiles, cfg.OverrideParamYAML, token)
	if err != nil {
		return nil, err
	}

	targets := filterTargets(specConfig.TargetIds, cfg.OverrideTargets)

	validations := specConfig.Validations
	if cfg.ValidationFile != "" {
		validations, err = source.LoadValidations(cfg.ValidationFile, token)
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
