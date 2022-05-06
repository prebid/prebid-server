package config

type Experiment struct {
	AdCerts ExperimentAdCerts `mapstructure:"adscert"`
}

type ExperimentAdCerts struct {
	Enabled bool `mapstructure:"enabled"`
	// InProcess configures data to sign requests using ads certs library in core PBS logic
	InProcess InProcess `mapstructure:"in-process"`
	// Remote configures remote signatory server
	Remote Remote `mapstructure:"remote"`
}

type InProcess struct {
	// ads.cert hostname for the originating party
	Origin string `mapstructure:"origin"`
	// PrivateKey is a base-64 encoded private key.
	PrivateKey string `mapstructure:"key"`
	// default: 30
	DNSCheckIntervalInSeconds int `mapstructure:"domain_check_interval_seconds"`
	// default: 30
	DNSRenewalIntervalInSeconds int `mapstructure:"domain_renewal_interval_seconds"`

	//domainCheckInterval   = flag.Duration("domain_check_interval", time.Duration(utils.GetEnvVarInt("DOMAIN_CHECK_INTERVAL", 30))*time.Second, "interval for checking domain records")
	//domainRenewalInterval = flag.Duration("domain_renewal_interval", time.Duration(utils.GetEnvVarInt("DOMAIN_RENEWAL_INTERVAL", 300))*time.Second, "interval before considering domain records for renewal")
}

type Remote struct {
	Url            string `mapstructure:"url"`
	TimeoutSeconds int    `mapstructure:"timeout_seconds"`
}

// aoriginCallsign string, ds.cert hostname for the originating party")
//"private_key", utils.GetEnvVarString("PRIVATE_KEY", ""), "base-64 encoded private key")
//domainCheckInterval time.Duration,
//domainRenewalInterval time.Duration,
