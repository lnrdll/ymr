package processor

// RenderedOUtput holds the result of a single template render
type RenderedOutput struct {
	TargetFile   string
	TemplateUsed string
	Content      string
}
