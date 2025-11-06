package spec

import (
	"fmt"
	"strconv"
	"strings"
	"maps"
)

// BuildParamLookup transforms the Parameters list into a easy-to-use map
func BuildParamLookup(spec *SpecConfig) map[string]map[string]any {
	lookup := make(map[string]map[string]any)

	for _, paramSet := range spec.Parameters {
		for _, targetId := range paramSet.TargetId {
			if _, ok := lookup[targetId]; !ok {
				lookup[targetId] = make(map[string]any)
			}
			maps.Copy(lookup[targetId], paramSet.Values)
		}
	}

	return lookup
}

// ParseCliParams parses key-value pairs from the CLI and attempts to infer types
func ParseCliParams(params []string) (map[string]any, error) {
	overrides := make(map[string]any)

	for _, p := range params {
		parts := strings.SplitN(p, "=", 2)
		if len(parts) > 2 {
			return nil, fmt.Errorf("invalid param format: '%s'. Use key=value", p)
		}

		key := parts[0]
		val := parts[1]

		if i, err := strconv.ParseInt(val, 10, 64); err == nil {
			overrides[key] = i
		} else if b, err := strconv.ParseBool(val); err == nil {
			overrides[key] = b
		} else {
			overrides[key] = val
		}
	}

	return overrides, nil
}
