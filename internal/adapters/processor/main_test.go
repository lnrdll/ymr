package processor

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestProcessContent_NonStrict_MissingParam_PreservesDefaultAndRemovesDirective(t *testing.T) {
	t.Parallel()

	tpl := []byte(strings.TrimSpace(`
key: default # from-param: {{ .missing }}
`) + "\n")

	out, err := ProcessContent(tpl, map[string]any{}, false)
	if err != nil {
		t.Fatalf("ProcessContent err = %v", err)
	}
	if strings.Contains(out, "from-param") {
		t.Fatalf("expected directive removed; got:\n%s", out)
	}
	if !strings.Contains(out, "key: default") {
		t.Fatalf("expected default preserved; got:\n%s", out)
	}
}

func TestProcessContent_Strict_MissingParam_ReturnsError(t *testing.T) {
	t.Parallel()

	tpl := []byte("key: default # from-param: {{ .missing }}\n")

	_, err := ProcessContent(tpl, map[string]any{}, true)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestProcessContent_NonStrict_TemplateExecFailure_PreservesDefault(t *testing.T) {
	t.Parallel()

	// replace expects 3 args; piping just .name will cause an execution error.
	tpl := []byte("key: default # from-param: {{ .name | replace }}\n")

	out, err := ProcessContent(tpl, map[string]any{"name": "abc"}, false)
	if err != nil {
		t.Fatalf("ProcessContent err = %v", err)
	}
	if !strings.Contains(out, "key: default") {
		t.Fatalf("expected default preserved; got:\n%s", out)
	}
}

func TestProcessContent_Strict_TemplateExecFailure_ReturnsError(t *testing.T) {
	t.Parallel()

	tpl := []byte("key: default # from-param: {{ .name | replace }}\n")

	_, err := ProcessContent(tpl, map[string]any{"name": "abc"}, true)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestProcessContent_NonStrict_TemplateParseFailure_PreservesDefault(t *testing.T) {
	t.Parallel()

	tpl := []byte("key: default # from-param: {{ .name\n")

	out, err := ProcessContent(tpl, map[string]any{"name": "abc"}, false)
	if err != nil {
		t.Fatalf("ProcessContent err = %v", err)
	}
	if !strings.Contains(out, "key: default") {
		t.Fatalf("expected default preserved; got:\n%s", out)
	}
}

func TestProcessContent_Strict_TemplateParseFailure_ReturnsError(t *testing.T) {
	t.Parallel()

	tpl := []byte("key: default # from-param: {{ .name\n")

	_, err := ProcessContent(tpl, map[string]any{"name": "abc"}, true)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestProcessContent_MergeMap_AppendsKeys(t *testing.T) {
	t.Parallel()

	tpl := []byte(strings.TrimSpace(`
metadata:
  labels: # from-param-merge: {{ .commonLabels }}
    app: my-app
    environment: dev
`) + "\n")

	out, err := ProcessContent(tpl, map[string]any{"commonLabels": map[string]any{"environment": "prod", "version": "v1"}}, true)
	if err != nil {
		t.Fatalf("ProcessContent err = %v", err)
	}
	if strings.Contains(out, "from-param-merge") {
		t.Fatalf("expected directive removed; got:\n%s", out)
	}
	if !strings.Contains(out, "app: my-app") {
		t.Fatalf("expected existing key preserved; got:\n%s", out)
	}
	if strings.Count(out, "environment:") != 1 {
		t.Fatalf("expected environment key to appear once; got:\n%s", out)
	}

	var got map[string]any
	if err := yaml.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal output: %v\noutput:\n%s", err, out)
	}
	metadata := got["metadata"].(map[string]any)
	labels := metadata["labels"].(map[string]any)
	if labels["environment"] != "prod" {
		t.Fatalf("environment = %#v, want prod", labels["environment"])
	}
	if labels["version"] != "v1" {
		t.Fatalf("version = %#v, want v1", labels["version"])
	}
}

func TestProcessContent_SimpleParam_SupportsNestedPathAndPreservesType(t *testing.T) {
	t.Parallel()

	tpl := []byte("ports: [0] # from-param: {{ .app.ports }}\n")

	out, err := ProcessContent(tpl, map[string]any{"app": map[string]any{"ports": []any{80, 443}}}, true)
	if err != nil {
		t.Fatalf("ProcessContent err = %v", err)
	}

	var got map[string]any
	if err := yaml.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal output: %v\noutput:\n%s", err, out)
	}
	ports := got["ports"].([]any)
	if len(ports) != 2 || ports[0] != 80 || ports[1] != 443 {
		t.Fatalf("ports = %#v, want [80 443]", ports)
	}
}

func TestProcessContent_SimpleParam_ArrayIndexing_Works(t *testing.T) {
	t.Parallel()

	tpl := []byte("first: 0 # from-param: {{ .ports.0 }}\n")

	out, err := ProcessContent(tpl, map[string]any{"ports": []any{80, 443}}, true)
	if err != nil {
		t.Fatalf("ProcessContent err = %v", err)
	}
	if !strings.Contains(out, "first: 80") {
		t.Fatalf("expected first=80; got:\n%s", out)
	}
}

func TestProcessContent_Strict_ArrayIndexOutOfRange_ReturnsError(t *testing.T) {
	t.Parallel()

	tpl := []byte("first: 0 # from-param: {{ .ports.2 }}\n")

	_, err := ProcessContent(tpl, map[string]any{"ports": []any{80, 443}}, true)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestProcessContent_NonStrict_ArrayIndexOutOfRange_PreservesDefault(t *testing.T) {
	t.Parallel()

	tpl := []byte("first: 0 # from-param: {{ .ports.2 }}\n")

	out, err := ProcessContent(tpl, map[string]any{"ports": []any{80, 443}}, false)
	if err != nil {
		t.Fatalf("ProcessContent err = %v", err)
	}
	if !strings.Contains(out, "first: 0") {
		t.Fatalf("expected default preserved; got:\n%s", out)
	}
}

func TestProcessContent_SimpleParam_PrefersFlatDottedKeyOverNestedPath(t *testing.T) {
	t.Parallel()

	tpl := []byte("key: default # from-param: {{ .a.b }}\n")

	params := map[string]any{
		"a.b": "flat",
		"a":   map[string]any{"b": "nested"},
	}

	out, err := ProcessContent(tpl, params, true)
	if err != nil {
		t.Fatalf("ProcessContent err = %v", err)
	}
	if !strings.Contains(out, "key: flat") {
		t.Fatalf("expected flat dotted key to win; got:\n%s", out)
	}
}

func TestProcessContent_SimpleParam_SupportsMapAnyAnyTraversal(t *testing.T) {
	t.Parallel()

	tpl := []byte("val: 0 # from-param: {{ .app.ports.1 }}\n")

	params := map[string]any{
		"app": map[any]any{
			"ports": []any{80, 443},
		},
	}

	out, err := ProcessContent(tpl, params, true)
	if err != nil {
		t.Fatalf("ProcessContent err = %v", err)
	}
	if !strings.Contains(out, "val: 443") {
		t.Fatalf("expected val=443; got:\n%s", out)
	}
}

func TestProcessContent_KeyNodeLineCommentDirective_Works(t *testing.T) {
	t.Parallel()

	// Directive is attached to the mapping key node (comment after 'name:').
	tpl := []byte(strings.TrimSpace(`
name: # from-param: {{ .name }}
  default
`) + "\n")

	out, err := ProcessContent(tpl, map[string]any{"name": "myapp"}, true)
	if err != nil {
		t.Fatalf("ProcessContent err = %v", err)
	}
	if strings.Contains(out, "from-param") {
		t.Fatalf("expected directive removed; got:\n%s", out)
	}
	if !strings.Contains(out, "name: myapp") {
		t.Fatalf("expected replacement; got:\n%s", out)
	}
}

func TestProcessContent_SequenceItemDirective_Works(t *testing.T) {
	t.Parallel()

	tpl := []byte(strings.TrimSpace(`
items:
  - default # from-param: {{ .item }}
`) + "\n")

	out, err := ProcessContent(tpl, map[string]any{"item": "x"}, true)
	if err != nil {
		t.Fatalf("ProcessContent err = %v", err)
	}
	if strings.Contains(out, "from-param") {
		t.Fatalf("expected directive removed; got:\n%s", out)
	}
	if !strings.Contains(out, "- x") {
		t.Fatalf("expected item replacement; got:\n%s", out)
	}
}

func TestProcessContent_MultiDocument_AppliesBothDocs(t *testing.T) {
	t.Parallel()

	tpl := []byte(strings.TrimSpace(`
key1: default # from-param: {{ .v1 }}
---
key2: default # from-param: {{ .v2 }}
`) + "\n")

	out, err := ProcessContent(tpl, map[string]any{"v1": "a", "v2": "b"}, true)
	if err != nil {
		t.Fatalf("ProcessContent err = %v", err)
	}
	if strings.Count(out, "---\n") != 1 {
		t.Fatalf("expected multi-doc separator; got:\n%s", out)
	}
	if strings.Contains(out, "from-param") {
		t.Fatalf("expected directives removed; got:\n%s", out)
	}
	if !strings.Contains(out, "key1: a") || !strings.Contains(out, "key2: b") {
		t.Fatalf("expected both docs rendered; got:\n%s", out)
	}
}

func TestProcessContent_TemplateFunctions_Succeed(t *testing.T) {
	t.Parallel()

	tpl := []byte(strings.TrimSpace(`
lowered: default # from-param: {{ .name | lower }}
uppered: default # from-param: {{ .name | upper }}
replaced: default # from-param: {{ .name | replace "." "-" }}
`) + "\n")

	out, err := ProcessContent(tpl, map[string]any{"name": "Ab.C"}, true)
	if err != nil {
		t.Fatalf("ProcessContent err = %v", err)
	}
	if !strings.Contains(out, "lowered: ab.c") {
		t.Fatalf("expected lower; got:\n%s", out)
	}
	if !strings.Contains(out, "uppered: AB.C") {
		t.Fatalf("expected upper; got:\n%s", out)
	}
	if !strings.Contains(out, "replaced: Ab-C") {
		t.Fatalf("expected replace; got:\n%s", out)
	}
}

func TestProcessContent_TemplateFunctions_ForWithJoin_Succeeds(t *testing.T) {
	t.Parallel()

	tpl := []byte("email: x # from-param: {{ .ldaps | for \"$i@example.com\" \",\" }}\n")

	out, err := ProcessContent(tpl, map[string]any{"ldaps": []any{"Sponge", "bob"}}, true)
	if err != nil {
		t.Fatalf("ProcessContent err = %v", err)
	}
	if !strings.Contains(out, "email: Sponge@example.com,bob@example.com") {
		t.Fatalf("expected joined emails; got:\n%s", out)
	}
}

func TestProcessContent_TemplateFunctions_ForOnSequence_IgnoresDelimiterAndPreservesArrayType(t *testing.T) {
	t.Parallel()

	tpl := []byte("emails: [] # from-param: {{ .ldaps | for \"$i@example.com\" \",\" }}\n")

	out, err := ProcessContent(tpl, map[string]any{"ldaps": []any{"Sponge", "bob"}}, true)
	if err != nil {
		t.Fatalf("ProcessContent err = %v", err)
	}

	var got map[string]any
	if err := yaml.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal output: %v\noutput:\n%s", err, out)
	}

	emails, ok := got["emails"].([]any)
	if !ok {
		t.Fatalf("emails type = %T, want []any; got %#v", got["emails"], got)
	}
	if len(emails) != 2 || emails[0] != "Sponge@example.com" || emails[1] != "bob@example.com" {
		t.Fatalf("emails = %#v, want [Sponge@example.com bob@example.com]", emails)
	}
}

func TestProcessContent_TemplateFunctions_ForWithEmptyDelimiter_JoinsAsScalar(t *testing.T) {
	t.Parallel()

	tpl := []byte("email: x # from-param: {{ .ldaps | for \"$i@example.com\" \"\" }}\n")

	out, err := ProcessContent(tpl, map[string]any{"ldaps": []any{"Sponge", "bob"}}, true)
	if err != nil {
		t.Fatalf("ProcessContent err = %v", err)
	}
	if !strings.Contains(out, "email: Sponge@example.combob@example.com") {
		t.Fatalf("expected joined scalar with empty delimiter; got:\n%s", out)
	}

	var got map[string]any
	if err := yaml.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal output: %v\noutput:\n%s", err, out)
	}
	if _, ok := got["email"].([]any); ok {
		t.Fatalf("email unexpectedly decoded as sequence: %#v", got["email"])
	}
}

func TestProcessContent_TemplateFunctions_ForWithoutJoin_PreservesArrayType(t *testing.T) {
	t.Parallel()

	tpl := []byte("emails: [] # from-param: {{ .ldaps | for \"$i@example.com\" }}\n")

	out, err := ProcessContent(tpl, map[string]any{"ldaps": []any{"Sponge", "bob"}}, true)
	if err != nil {
		t.Fatalf("ProcessContent err = %v", err)
	}

	var got map[string]any
	if err := yaml.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal output: %v\noutput:\n%s", err, out)
	}

	emails, ok := got["emails"].([]any)
	if !ok {
		t.Fatalf("emails type = %T, want []any; got %#v", got["emails"], got)
	}
	if len(emails) != 2 || emails[0] != "Sponge@example.com" || emails[1] != "bob@example.com" {
		t.Fatalf("emails = %#v, want [Sponge@example.com bob@example.com]", emails)
	}
}

func TestProcessContent_TemplateFunctions_ForWithoutDelimiter_OnScalar_JoinsWithoutSeparator(t *testing.T) {
	t.Parallel()

	tpl := []byte("email: x # from-param: {{ .ldaps | for \"$i@example.com\" }}\n")

	out, err := ProcessContent(tpl, map[string]any{"ldaps": []any{"Sponge", "bob"}}, true)
	if err != nil {
		t.Fatalf("ProcessContent err = %v", err)
	}
	if !strings.Contains(out, "email: Sponge@example.combob@example.com") {
		t.Fatalf("expected scalar join without separator; got:\n%s", out)
	}
}

func TestProcessContent_Strict_TemplateForWithScalarValue_ReturnsError(t *testing.T) {
	t.Parallel()

	tpl := []byte("email: x # from-param: {{ .ldap | for \"$i@example.com\" \",\" }}\n")

	_, err := ProcessContent(tpl, map[string]any{"ldap": "Sponge"}, true)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestProcessContent_NonStrict_TemplateForWithScalarValue_PreservesDefault(t *testing.T) {
	t.Parallel()

	tpl := []byte("email: default # from-param: {{ .ldap | for \"$i@example.com\" \",\" }}\n")

	out, err := ProcessContent(tpl, map[string]any{"ldap": "Sponge"}, false)
	if err != nil {
		t.Fatalf("ProcessContent err = %v", err)
	}
	if !strings.Contains(out, "email: default") {
		t.Fatalf("expected default preserved; got:\n%s", out)
	}
}

func TestProcessContent_SimpleParam_NullEncodesAsYAMLNull(t *testing.T) {
	t.Parallel()

	tpl := []byte("maybe: x # from-param: {{ .maybe }}\n")

	out, err := ProcessContent(tpl, map[string]any{"maybe": nil}, true)
	if err != nil {
		t.Fatalf("ProcessContent err = %v", err)
	}

	var got map[string]any
	if err := yaml.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal output: %v\noutput:\n%s", err, out)
	}
	if v, ok := got["maybe"]; !ok || v != nil {
		t.Fatalf("maybe = %#v (ok=%v), want nil; output:\n%s", v, ok, out)
	}
}

func TestParseParamFromComment_RequiresExactDirective(t *testing.T) {
	t.Parallel()

	if d, p := parseParamFromComment("blah from-param: {{ .x }}"); d != "" || p != "" {
		t.Fatalf("unexpected match: (%q, %q)", d, p)
	}

	if d, p := parseParamFromComment(" # from-param: {{ .x }} "); d != "from-param" || p != "{{ .x }}" {
		t.Fatalf("unexpected parse: (%q, %q)", d, p)
	}
}

func TestProcessContent_MergeSequence_AppendsItems(t *testing.T) {
	t.Parallel()

	tpl := []byte("ports: [80] # from-param-merge: {{ .extraPorts }}\n")

	out, err := ProcessContent(tpl, map[string]any{"extraPorts": []any{443}}, true)
	if err != nil {
		t.Fatalf("ProcessContent err = %v", err)
	}

	var got map[string]any
	if err := yaml.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal output: %v\noutput:\n%s", err, out)
	}

	ports, ok := got["ports"].([]any)
	if !ok {
		t.Fatalf("ports type = %T, want []any; got %#v", got["ports"], got)
	}
	if len(ports) != 2 {
		t.Fatalf("ports len = %d, want 2; ports=%#v", len(ports), ports)
	}
	if ports[0] != 80 || ports[1] != 443 {
		t.Fatalf("ports = %#v, want [80 443]", ports)
	}
}

func TestProcessContent_MergeMap_WithScalarReplacement_ReplacesNode(t *testing.T) {
	t.Parallel()

	tpl := []byte(strings.TrimSpace(`
labels: # from-param-merge: {{ .x }}
  a: b
`) + "\n")

	out, err := ProcessContent(tpl, map[string]any{"x": "oops"}, true)
	if err != nil {
		t.Fatalf("ProcessContent err = %v", err)
	}

	var got map[string]any
	if err := yaml.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal output: %v\noutput:\n%s", err, out)
	}
	if got["labels"] != "oops" {
		t.Fatalf("labels = %#v, want oops", got["labels"])
	}
}

func TestProcessContent_MergeSequence_WithScalarReplacement_ReplacesNode(t *testing.T) {
	t.Parallel()

	tpl := []byte("ports: [80] # from-param-merge: {{ .x }}\n")

	out, err := ProcessContent(tpl, map[string]any{"x": "oops"}, true)
	if err != nil {
		t.Fatalf("ProcessContent err = %v", err)
	}

	var got map[string]any
	if err := yaml.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal output: %v\noutput:\n%s", err, out)
	}
	if got["ports"] != "oops" {
		t.Fatalf("ports = %#v, want oops", got["ports"])
	}
}
