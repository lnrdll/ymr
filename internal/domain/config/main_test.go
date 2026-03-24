package config

import "testing"

func TestParseCliParams_ParsesIntBoolFloatString(t *testing.T) {
	t.Parallel()

	m, err := ParseCliParams([]string{"i=10", "b=true", "f=1.5", "s=hello"})
	if err != nil {
		t.Fatalf("ParseCliParams err = %v", err)
	}

	if _, ok := m["i"].(int64); !ok {
		t.Fatalf("i type = %T, want int64", m["i"])
	}
	if m["i"].(int64) != 10 {
		t.Fatalf("i = %v, want 10", m["i"])
	}

	if _, ok := m["b"].(bool); !ok {
		t.Fatalf("b type = %T, want bool", m["b"])
	}
	if m["b"].(bool) != true {
		t.Fatalf("b = %v, want true", m["b"])
	}

	if _, ok := m["f"].(float64); !ok {
		t.Fatalf("f type = %T, want float64", m["f"])
	}
	if m["f"].(float64) != 1.5 {
		t.Fatalf("f = %v, want 1.5", m["f"])
	}

	if m["s"].(string) != "hello" {
		t.Fatalf("s = %v, want hello", m["s"])
	}
}

func TestParseCliParams_InvalidFormat(t *testing.T) {
	t.Parallel()

	if _, err := ParseCliParams([]string{"nope"}); err == nil {
		t.Fatalf("expected error")
	}
}
