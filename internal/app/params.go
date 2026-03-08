package app

import (
	"fmt"
	"log/slog"
	"maps"
	"slices"

	config "github.com/lnrdll/ymr/internal/domain/config"
	"gopkg.in/yaml.v3"
)

func applyParamsOverrides(
	sourcePort SourcePort,
	overrideParams []string,
	overrideParamFiles []string,
	overrideParamYAML []string,
	token string,
) (map[string]any, error) {
	paramsOverride, err := buildParamsOverride(sourcePort, overrideParams, overrideParamFiles, overrideParamYAML, token)
	if err != nil {
		return nil, fmt.Errorf("parsing override parameters: %w", err)
	}

	if len(paramsOverride) > 0 {
		slog.Debug("Overriding parameters", "count", len(paramsOverride), "keys", mapKeys(paramsOverride))
	}

	return paramsOverride, nil
}

func resolveParamsForTarget(
	paramLookup map[string]map[string]any,
	targetId string,
	paramsOverride map[string]any,
) map[string]any {
	resolved := make(map[string]any)

	if base, ok := paramLookup[targetId]; ok {
		maps.Copy(resolved, base)
	}

	if len(paramsOverride) > 0 {
		maps.Copy(resolved, paramsOverride)
	}

	return resolved
}

func buildParamsOverride(
	sourcePort SourcePort,
	overrideParams []string,
	overrideParamFiles []string,
	overrideParamYAML []string,
	token string,
) (map[string]any, error) {
	overrides := make(map[string]any)

	for _, p := range overrideParamFiles {
		m, err := sourcePort.LoadParams(p, token)
		if err != nil {
			return nil, err
		}
		maps.Copy(overrides, m)
	}

	for _, s := range overrideParamYAML {
		var m map[string]any
		if err := yaml.Unmarshal([]byte(s), &m); err != nil {
			return nil, fmt.Errorf("parsing --param-yaml: %w", err)
		}
		if m == nil {
			m = make(map[string]any)
		}
		maps.Copy(overrides, m)
	}

	m, err := config.ParseCliParams(overrideParams)
	if err != nil {
		return nil, err
	}
	maps.Copy(overrides, m)

	return overrides, nil
}

func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
