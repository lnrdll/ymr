package validation

import (
	"testing"

	"github.com/lnrdll/ymr/internal/spec"
)

func TestEngineCheckTarget_UsesExplicitTargetIDsOnly(t *testing.T) {
	t.Parallel()

	engine, err := NewEngine([]spec.Validation{
		{Rule: "params.minScale >= 1", Message: "minScale must be >= 1", TargetId: []string{"dev"}},
		{Rule: "params.maxScale >= 5", Message: "maxScale must be >= 5", TargetId: []string{}},
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	err = engine.CheckTarget("prd", map[string]any{"minScale": 0, "maxScale": 0})
	if err != nil {
		t.Fatalf("expected no validation for prd target, got: %v", err)
	}

	err = engine.CheckTarget("dev", map[string]any{"minScale": 0})
	if err == nil {
		t.Fatalf("expected dev target validation failure")
	}
}

func TestCheck_EvaluatesAllProvidedValidations(t *testing.T) {
	t.Parallel()

	err := Check(
		map[string]any{"minScale": 0},
		[]spec.Validation{{Rule: "params.minScale >= 1", Message: "minScale must be >= 1"}},
	)
	if err == nil {
		t.Fatalf("expected validation failure")
	}
}
