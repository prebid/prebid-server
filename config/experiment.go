package config

// Experiment defines if experimental features are available
type Experiment struct {
	AdCerts ExperimentAdsCert `mapstructure:"adscert"`
}

// ExperimentAdsCert configures and enables functionality to generate and send Ads Cert Auth header to bidders
type ExperimentAdsCert struct {
	Enabled   bool             `mapstructure:"enabled"`
	InProcess AdsCertInProcess `mapstructure:"inprocess"`
	Remote    AdsCertRemote    `mapstructure:"remote"`
}

// AdsCertInProcess configures data to sign requests using ads certs library in core PBS logic
type AdsCertInProcess struct {
	//Origin is ads.cert hostname for the originating party
	Origin string `mapstructure:"origin"`
	//PrivateKey is a base-64 encoded private key.
	PrivateKey string `mapstructure:"key"`
	//DNSCheckIntervalInSeconds specifies frequency to check origin _delivery._adscert and _adscert subdomains, used for indexing data, default: 30
	DNSCheckIntervalInSeconds int `mapstructure:"domain_check_interval_seconds"`
	//DNSRenewalIntervalInSeconds specifies frequency to renew origin _delivery._adscert and _adscert subdomains, used for indexing data, default: 30
	DNSRenewalIntervalInSeconds int `mapstructure:"domain_renewal_interval_seconds"`
}

// AdsCertRemote configures data to sign requests using remote signatory service
type AdsCertRemote struct {
	//Url - address of gRPC server that will create a call signature
	Url string `mapstructure:"url"`
	//SigningTimeoutMs specifies how long this client will wait for signing to finish before abandoning
	SigningTimeoutMs int `mapstructure:"signing_timeout_ms"`
}
