package app

// Config holds all configuration flags passed from the CLI
type Config struct {
	Debug              bool
	Strict             bool
	Plan               bool
	GithubToken        string
	OutputDir          string
	OverrideParams     []string
	OverrideParamYAML  []string
	OverrideParamFiles []string
	OverrideTargets    []string
	OverrideTemplate   string
	SpecFile           string
	ValidationFile     string
	IsSpecFile         bool
}
