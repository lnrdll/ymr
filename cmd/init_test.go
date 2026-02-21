package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteSpecFile_NoForce_FailsIfExists(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	specPath := filepath.Join(dir, "spec.yaml")

	if err := os.WriteFile(specPath, []byte("old"), 0644); err != nil {
		t.Fatalf("seed file: %v", err)
	}

	if err := writeSpecFile(specPath, []byte("new"), false); err == nil {
		t.Fatalf("expected error")
	}
}

func TestWriteSpecFile_Force_OverwritesIfExists(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	specPath := filepath.Join(dir, "spec.yaml")

	if err := os.WriteFile(specPath, []byte("old"), 0644); err != nil {
		t.Fatalf("seed file: %v", err)
	}

	if err := writeSpecFile(specPath, []byte("new"), true); err != nil {
		t.Fatalf("writeSpecFile: %v", err)
	}

	b, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(b) != "new" {
		t.Fatalf("expected overwrite, got %q", string(b))
	}
}

func TestWriteSpecFile_WritesIfMissing(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	specPath := filepath.Join(dir, "spec.yaml")

	if err := writeSpecFile(specPath, []byte("content"), false); err != nil {
		t.Fatalf("writeSpecFile: %v", err)
	}
}
