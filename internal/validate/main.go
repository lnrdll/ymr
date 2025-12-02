package validate

import (
	"fmt"
	"log/slog"
	"ymr/internal/spec"

	"github.com/google/cel-go/cel"
)

func Check(params map[string]any, validations []spec.Validation) error {
	slog.Debug("running validations", "params", params, "validations", validations)

	if len(validations) == 0 {
		return nil
	}

	// CEL environment
	env, err := cel.NewEnv(
		cel.Variable("params", cel.MapType(cel.StringType, cel.DynType)),
	)
	if err != nil {
		return fmt.Errorf("failed to create CEL environment: %w", err)
	}

	// Iterate over rules
	for _, v := range validations {
		ast, issues := env.Compile(v.Rule)
		if issues.Err() != nil {
			return fmt.Errorf("invalid rule '%s': %w", v.Rule, issues.Err())
		}

		// Ensure the rule actually returns a boolean type before evaluating
		if ast.OutputType() != cel.BoolType {
			return fmt.Errorf("rule '%s' must evaluate to a boolean, got %s", v.Rule, ast.OutputType())
		}

		prg, err := env.Program(ast)
		if err != nil {
			return fmt.Errorf("cel program creation error: %w", err)
		}

		out, _, err := prg.Eval(map[string]any{"params": params})
		if err != nil {
			return fmt.Errorf("evaluation error for rule '%s': %w", v.Rule, err)
		}

		// out.Value() returns `any`. We assert it is a boolean.
		// If conversion fails OR the result is false, validation fails.
		val, ok := out.Value().(bool)

		// Handle cases where the output is strictly not a boolean (though the compilation check helps)
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

		slog.Debug("validation rule passed", "rule", v.Rule)
	}

	return nil
}
