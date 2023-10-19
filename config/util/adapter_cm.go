package config

//Adapter level Commerce Specific parameters
type AdapterCommerce struct {
	ImpTracker  string `mapstructure:"impurl"`
	ClickTracker  string `mapstructure:"clickurl"`
	ConversionTracker  string `mapstructure:"conversionurl"`
}
