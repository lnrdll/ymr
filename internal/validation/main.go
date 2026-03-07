package validation

import (
	"fmt"
	"log/slog"
	"slices"

	"github.com/lnrdll/ymr/internal/spec"

	"github.com/google/cel-go/cel"
)

type compiledValidation struct {
	rule      string
	message   string
	targetIDs []string
	program   cel.Program
}

type Engine struct {
	rules []compiledValidation
}

func NewEngine(validations []spec.Validation) (*Engine, error) {
	slog.Debug("Compiling validation policies", "validations", len(validations))

	if len(validations) == 0 {
		return &Engine{rules: nil}, nil
	}

	env, err := cel.NewEnv(
		cel.Variable("params", cel.MapType(cel.StringType, cel.DynType)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create validation environment: %w", err)
	}

	compiled := make([]compiledValidation, 0, len(validations))
	for _, v := range validations {
		ast, issues := env.Compile(v.Rule)
		if issues.Err() != nil {
			return nil, fmt.Errorf("invalid validation rule '%s': %w", v.Rule, issues.Err())
		}

		if ast.OutputType() != cel.BoolType {
			return nil, fmt.Errorf("rule '%s' must evaluate to a boolean, got %s", v.Rule, ast.OutputType())
		}

		prg, err := env.Program(ast)
		if err != nil {
			return nil, fmt.Errorf("validation creation error: %w", err)
		}

		compiled = append(compiled, compiledValidation{
			rule:      v.Rule,
			message:   v.Message,
			targetIDs: v.TargetId,
			program:   prg,
		})
	}

	return &Engine{rules: compiled}, nil
}

func Check(params map[string]any, validations []spec.Validation) error {
	engine, err := NewEngine(validations)
	if err != nil {
		return err
	}

	return engine.CheckAll(params)
}

func (e *Engine) CheckAll(params map[string]any) error {
	slog.Debug("Executing validation policies", "validations", len(e.rules), "param_count", len(params))

	for _, rule := range e.rules {
		if err := runValidationRule(rule, params); err != nil {
			return err
		}
	}

	return nil
}

func (e *Engine) CheckTarget(targetID string, params map[string]any) error {
	slog.Debug("Executing validation policies", "validations", len(e.rules), "param_count", len(params), "target", targetID)

	if len(e.rules) == 0 {
		return nil
	}

	for _, rule := range e.rules {
		if len(rule.targetIDs) == 0 || !slices.Contains(rule.targetIDs, targetID) {
			continue
		}

		if err := runValidationRule(rule, params); err != nil {
			return err
		}
	}

	return nil
}

func runValidationRule(rule compiledValidation, params map[string]any) error {
	out, _, err := rule.program.Eval(map[string]any{"params": params})
	if err != nil {
		return fmt.Errorf("validation error for rule '%s': %w", rule.rule, err)
	}

	val, ok := out.Value().(bool)
	if !ok {
		return fmt.Errorf("rule '%s' did not return a boolean value", rule.rule)
	}

	if !val {
		if rule.message == "" {
			return fmt.Errorf("validation failed: %s", rule.rule)
		}
		return fmt.Errorf("%s", rule.message)
	}

	return nil
}
