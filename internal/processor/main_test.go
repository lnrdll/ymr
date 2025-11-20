package processor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestProcessContent(t *testing.T) {
	tests := []struct {
		name          string
		inputYAML     string
		params        map[string]any
		expectedYAML  string
		expectedError bool
		desc          string
	}{
		{
			name:         "Simple String Replacement",
			inputYAML:    "key: default_value # from-param: {{ .MyVal }}",
			params:       map[string]any{"MyVal": "new_value"},
			expectedYAML: "key: new_value",
			desc:         "Should replace the value of the key with the param",
		},
		{
			name:         "Template Function - Lower",
			inputYAML:    "environment: PROD # from-param: {{ .Env | lower }}",
			params:       map[string]any{"Env": "STAGING"},
			expectedYAML: "environment: staging",
			desc:         "Should allow use of the 'lower' function",
		},
		{
			name:         "Template Function - Replace",
			inputYAML:    `image: nginx # from-param: {{ .Img | replace "nginx" "alpine" }}`,
			params:       map[string]any{"Img": "my-nginx-image"},
			expectedYAML: "image: my-alpine-image",
			desc:         "Should correctly handle the replace function order when using pipes",
		},
		{
			name:         "List Replacement",
			inputYAML:    `items: ["default"] # from-param: {{ .List }}`,
			params:       map[string]any{"List": []string{"item1", "item2"}},
			expectedYAML: `items: ["item1", "item2"]`,
			desc:         "Should completely replace a list node",
		},
		{
			name: "List Merge",
			inputYAML: `
items: # from-param-merge: {{ .NewItems }}
  - existing
`,
			params:       map[string]any{"NewItems": []string{"added1", "added2"}},
			expectedYAML: `items: ["existing", "added1", "added2"]`,
			desc:         "Should append new items to the existing list",
		},
		{
			name: "Map Merge",
			inputYAML: `
labels: # from-param-merge: {{ .ExtraLabels }}
  app: main
`,
			params: map[string]any{"ExtraLabels": map[string]string{"tier": "backend"}},
			expectedYAML: `
labels:
  app: main
  tier: backend
`,
			desc: "Should merge new keys into an existing map",
		},
		{
			name: "Nested Structure",
			inputYAML: `
database:
  host: localhost
  port: 5432 # from-param: {{ .DBPort }}
`,
			params: map[string]any{"DBPort": 3306},
			expectedYAML: `
database:
  host: localhost
  port: 3306
`,
			desc: "Should handle nested YAML keys",
		},
		{
			name:         "Missing Param (Failure Fallback)",
			inputYAML:    `key: original # from-param: {{ .MissingKey }}`,
			params:       map[string]any{},
			expectedYAML: `key: original`,
			desc:         "Should preserve original value if template execution fails (missing key)",
		},
		{
			name:         "Type Mismatch in Template Func",
			inputYAML:    `key: original # from-param: {{ .IntVal | lower }}`,
			params:       map[string]any{"IntVal": 123},
			expectedYAML: `key: original`,
			desc:         "Should preserve original value if strict type check fails in template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputBytes := []byte(tt.inputYAML)

			got, err := ProcessContent(inputBytes, tt.params)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			var gotNode, wantNode map[string]any
			err1 := yaml.Unmarshal([]byte(got), &gotNode)
			err2 := yaml.Unmarshal([]byte(tt.expectedYAML), &wantNode)
			require.NoError(t, err1)
			require.NoError(t, err2)

			assert.Equal(t, wantNode, gotNode, tt.desc)
		})
	}
}

func TestInvalidYAML(t *testing.T) {
	input := []byte(`: - invalid yaml`)
	_, err := ProcessContent(input, nil)
	assert.Error(t, err, "Should return error for invalid YAML input")
}
