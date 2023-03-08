package processor

// empty Processor provide default implementation for macro processor.
type emptyProcessor struct{}

func (*emptyProcessor) Replace(url string, macroProvider Provider) (string, error) {
	return url, nil
}
