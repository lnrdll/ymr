package app

import "fmt"

func validateRunConfig(cfg Config) (Config, error) {
	if cfg.IsSpecFile {
		if cfg.SpecFile == "" {
			return cfg, fmt.Errorf("--spec requires a non-empty path")
		}
		return cfg, nil
	}

	cfg.SpecFile = ""
	if cfg.OverrideTemplate == "" {
		return cfg, fmt.Errorf("in spec-less mode (no -s flag), --template (-T) is required")
	}
	if len(cfg.OverrideParams) == 0 && len(cfg.OverrideParamFiles) == 0 && len(cfg.OverrideParamYAML) == 0 {
		return cfg, fmt.Errorf("in spec-less mode (no -s flag), at least one of --param, --param-file, or --param-yaml is required")
	}

	return cfg, nil
}
