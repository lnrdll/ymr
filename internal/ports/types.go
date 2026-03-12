package ports

import config "github.com/lnrdll/ymr/internal/domain/config"

type SourceLoader interface {
	LoadSpec(token string) (*config.SpecConfig, error)
	GetBasePath() string
}

type RenderedOutput struct {
	TargetFile   string
	TemplateUsed string
	Content      string
}
