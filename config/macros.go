package config

type ProcessorType int

const (
	EmptyProcessor         = 0
	StringBasedProcessor   = 1
	TemplateBasedProcessor = 2
)

// MacroProcessorConfig defines the macro processor configuration
type MacroProcessorConfig struct {
	// ProcessorType define the type of macro processor to be used
	// Defaults to emtpy processor, which will not replace the macros.
	ProcessorType ProcessorType
	// Delimiter identifies the start and end of a macro in url.
	// Defaults to ##
	Delimiter string
}
