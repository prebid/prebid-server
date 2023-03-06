package processor

import (
	"github.com/prebid/prebid-server/config"
)

type Processor interface {
	// Replace the macros and returns replaced string
	// if any error the error will be returned
	Replace(url string, macroProvider Provider) (string, error)
}

var processor Processor

// NewProcessor will return instance of macro processor
// Defaults to emtpy processor, in which case the macros will not be replace in url.
// Supports string based and template based implementation
func NewProcessor(cfg config.MacroProcessorConfig) Processor {

	if cfg.Delimiter == "" {
		cfg.Delimiter = "##"
	}

	switch cfg.ProcessorType {
	case config.StringBasedProcessor:
		processor = newStringBasedProcessor(cfg)
	case config.TemplateBasedProcessor:
		processor = newtemplateBasedProcessor(cfg)
	default:
		processor = &emptyProcessor{}
	}

	return processor
}

func GetMacroProcessor() Processor {
	return processor
}
