package app

func Validate(cfg Config) error {
	cfg.ValidateOnly = true
	return NewRunCommand(cfg).Execute()
}
