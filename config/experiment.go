package config

//Experiment defines if experimental features are available
type Experiment struct {
	AdCerts ExperimentAdCerts `mapstructure:"adscert"`
}

//ExperimentAdCerts configures and enables functionality to generate and send Ads Cert Auth header to bidders
type ExperimentAdCerts struct {
	Enabled   bool      `mapstructure:"enabled"`
	InProcess InProcess `mapstructure:"in-process"`
}

//InProcess configures data to sign requests using ads certs library in core PBS logic
type InProcess struct {
	//Origin is ads.cert hostname for the originating party
	Origin string `mapstructure:"origin"`
	//PrivateKey is a base-64 encoded private key.
	PrivateKey string `mapstructure:"key"`
	//DNSCheckIntervalInSeconds default: 30
	DNSCheckIntervalInSeconds int `mapstructure:"domain_check_interval_seconds"`
	//DNSRenewalIntervalInSeconds default: 30
	DNSRenewalIntervalInSeconds int `mapstructure:"domain_renewal_interval_seconds"`
}
