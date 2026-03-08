package config

type SpecConfig struct {
	Templates   []string     `yaml:"templates"`
	TargetIds   []string     `yaml:"targetIds"`
	Parameters  []ParamSet   `yaml:"parameters"`
	Validations []Validation `yaml:"validations"`
}

type ParamSet struct {
	Values   map[string]any `yaml:"values"`
	TargetId []string       `yaml:"targetId"`
}

type Validation struct {
	Rule     string   `yaml:"rule"`
	Message  string   `yaml:"message"`
	TargetId []string `yaml:"targetId"`
}
