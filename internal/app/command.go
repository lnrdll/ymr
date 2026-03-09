package app

import "fmt"

type Command interface {
	Execute() error
}

type runCommand struct {
	cfg  Config
	deps appDeps
}

func NewRunCommand(cfg Config) Command {
	return &runCommand{cfg: cfg, deps: newDefaultDeps()}
}

func (c *runCommand) Execute() error {
	outputDir, terminalOutput := prepareOutputDir(c.cfg.OutputDir)

	plan, err := preparePlanWithDeps(c.cfg, c.deps)
	if err != nil {
		return err
	}

	if err := validateTargets(c.deps.validation, plan.Targets, plan.ParamLookup, plan.ParamsOverride, plan.Validations); err != nil {
		return err
	}

	if c.cfg.ValidateOnly {
		return c.executeValidationOnly(plan)
	}

	allOutputs, renderErr := processTemplates(
		c.deps,
		plan.SpecConfig,
		plan.Loader,
		plan.Token,
		plan.Targets,
		plan.ParamLookup,
		plan.ParamsOverride,
		c.cfg.OverrideTemplate,
		c.cfg.Strict,
	)
	if c.cfg.Strict && renderErr != nil {
		return renderErr
	}

	return handleOutput(c.deps.output, allOutputs, terminalOutput, outputDir)
}

func (c *runCommand) executeValidationOnly(plan *executionPlan) error {
	targetsToValidate := plan.Targets
	if c.cfg.Strict && len(c.cfg.OverrideTargets) > 0 && len(targetsToValidate) == 0 {
		return fmt.Errorf("no requested targets matched spec targetIds")
	}

	return validateTemplates(c.deps, plan.SpecConfig, plan.Loader, plan.Token, targetsToValidate, plan.ParamLookup, plan.ParamsOverride, c.cfg.OverrideTemplate, c.cfg.Strict)
}
