package spec

import "testing"

func TestParseCliParams(t *testing.T) {
	t.Parallel()

	got, err := ParseCliParams([]string{"a=1", "b=true", "c=hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["a"] != int64(1) {
		t.Fatalf("a: expected int64(1), got %T(%v)", got["a"], got["a"])
	}
	if got["b"] != true {
		t.Fatalf("b: expected true, got %T(%v)", got["b"], got["b"])
	}
	if got["c"] != "hello" {
		t.Fatalf("c: expected 'hello', got %T(%v)", got["c"], got["c"])
	}
}

func TestParseCliParams_InvalidFormat(t *testing.T) {
	t.Parallel()

	_, err := ParseCliParams([]string{"noequals"})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestParseCliParams_EmptyKey(t *testing.T) {
	t.Parallel()

	_, err := ParseCliParams([]string{"=x"})
	if err == nil {
		t.Fatalf("expected error")
	}
}
