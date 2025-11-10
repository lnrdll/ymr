package spec

// SpecConfig defines the structure of the spec.yaml file
type SpecConfig struct {
	Templates  []string   `yaml:"templates"`
	TargetIds  []string   `yaml:"targetIds"`
	Parameters []ParamSet `yaml:"parameters"`
}

// ParamSet defines a set of values for one or more targets
type ParamSet struct {
	Values   map[string]any `yaml:"values"`
	TargetId []string       `yaml:"targetId"`
}
