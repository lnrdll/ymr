package app

// Config holds all configuration flags passed from the CLI
type Config struct {
	Debug            bool
	Strict           bool
	GithubToken      string
	OutputDir        string
	OverrideParams   []string
	OverrideTargets  []string
	OverrideTemplate string
	SpecFile         string
	ValidationFile   string
	IsSpecFile       bool
}
