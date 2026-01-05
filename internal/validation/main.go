package validation

import (
	"fmt"
	"log/slog"

	"github.com/lnrdll/ymr/internal/spec"

	"github.com/google/cel-go/cel"
)

func Check(params map[string]any, validations []spec.Validation) error {
	slog.Debug("Execution validation policies", "params", params, "validations", validations)

	if len(validations) == 0 {
		return nil
	}

	// CEL environment
	env, err := cel.NewEnv(
		cel.Variable("params", cel.MapType(cel.StringType, cel.DynType)),
	)
	if err != nil {
		return fmt.Errorf("failed to create validation environment: %w", err)
	}

	// Iterate over rules
	for _, v := range validations {
		ast, issues := env.Compile(v.Rule)
		if issues.Err() != nil {
			return fmt.Errorf("invalid validation rule '%s': %w", v.Rule, issues.Err())
		}

		if ast.OutputType() != cel.BoolType {
			return fmt.Errorf("rule '%s' must evaluate to a boolean, got %s", v.Rule, ast.OutputType())
		}

		prg, err := env.Program(ast)
		if err != nil {
			return fmt.Errorf("validation creation error: %w", err)
		}

		out, _, err := prg.Eval(map[string]any{"params": params})
		if err != nil {
			return fmt.Errorf("validation error for rule '%s': %w", v.Rule, err)
		}

		// out.Value() returns `any`. We assert it is a boolean.
		// If conversion fails OR the result is false, validation fails.
		val, ok := out.Value().(bool)

		// Handle cases where the output is strictly not a boolean
		if !ok {
			return fmt.Errorf("rule '%s' did not return a boolean value", v.Rule)
		}

		// Logic: If val is false, the validation failed.
		if !val {
			if v.Message == "" {
				return fmt.Errorf("validation failed: %s", v.Rule)
			}
			return fmt.Errorf("%s", v.Message)
		}
	}

	return nil
}
