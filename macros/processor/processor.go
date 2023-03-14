package processor

type Processor interface {
	// Replace the macros and returns replaced string
	// if any error the error will be returned
	Replace(url string, macroProvider Provider) (string, error)
}

var processor Processor

// NewProcessor will return instance of macro processor
func NewProcessor() Processor {
	return &stringBasedProcessor{
		templates: make(map[string]urlMetaTemplate),
	}
}

func GetMacroProcessor() Processor {
	return processor
}
