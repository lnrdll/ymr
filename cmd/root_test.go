package cmd

import (
	"errors"
	"testing"
)

func TestFormatCLIError(t *testing.T) {
	t.Parallel()

	err := errors.New("validation failed for target 'dev': replicas must be >= 2")
	got := formatCLIError(err)
	want := "ERROR validation failed for target 'dev': replicas must be >= 2"
	if got != want {
		t.Fatalf("formatCLIError() = %q, want %q", got, want)
	}
}

func TestFormatGitHubActionsError_EscapesSpecialCharacters(t *testing.T) {
	t.Parallel()

	err := errors.New("bad 100% value\nnext line\rreturn")
	got := formatGitHubActionsError(err)
	want := "::error::bad 100%25 value%0Anext line%0Dreturn"
	if got != want {
		t.Fatalf("formatGitHubActionsError() = %q, want %q", got, want)
	}
}
