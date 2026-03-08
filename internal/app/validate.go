package app

func Validate(cfg Config) error {
	return NewValidateCommand(cfg).Execute()
}
