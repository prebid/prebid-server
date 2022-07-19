package config

import (
	"errors"
	"net/url"
)

var (
	ErrSignerModeIncorrect                      = errors.New("signer mode is not specified, specify 'off', 'inprocess' or 'remote'")
	ErrInProcessSignerInvalidURL                = errors.New("invalid url for inprocess signer")
	ErrInProcessSignerInvalidPrivateKey         = errors.New("invalid private key for inprocess signer")
	ErrInProcessSignerInvalidDNSRenewalInterval = errors.New("invalid dns renewal interval for inprocess signer")
	ErrInProcessSignerInvalidDNSCheckInterval   = errors.New("invalid dns check interval for inprocess signer")
	ErrInvalidRemoteSignerURL                   = errors.New("invalid url for remote signer")
	ErrInvalidRemoteSignerSigningTimeout        = errors.New("invalid signing timeout for remote signer")

	AdCertsSignerModeOff       = "off"
	AdCertsSignerModeInprocess = "inprocess"
	AdCertsSignerModeRemote    = "remote"
)

// Experiment defines if experimental features are available
type Experiment struct {
	AdCerts ExperimentAdsCert `mapstructure:"adscert"`
}

// ExperimentAdsCert configures and enables functionality to generate and send Ads Cert Auth header to bidders
type ExperimentAdsCert struct {
	Mode      string           `mapstructure:"mode"`
	InProcess AdsCertInProcess `mapstructure:"inprocess"`
	Remote    AdsCertRemote    `mapstructure:"remote"`
}

// AdsCertInProcess configures data to sign requests using ads certs library in core PBS logic
type AdsCertInProcess struct {
	// Origin is ads.cert hostname for the originating party
	Origin string `mapstructure:"origin"`
	// PrivateKey is a base-64 encoded private key.
	PrivateKey string `mapstructure:"key"`
	// DNSCheckIntervalInSeconds specifies frequency to check origin _delivery._adscert and _adscert subdomains, used for indexing data, default: 30
	DNSCheckIntervalInSeconds int `mapstructure:"domain_check_interval_seconds"`
	// DNSRenewalIntervalInSeconds specifies frequency to renew origin _delivery._adscert and _adscert subdomains, used for indexing data, default: 30
	DNSRenewalIntervalInSeconds int `mapstructure:"domain_renewal_interval_seconds"`
}

// AdsCertRemote configures data to sign requests using remote signatory service
type AdsCertRemote struct {
	// Url is the address of gRPC server that will create a call signature
	Url string `mapstructure:"url"`
	// SigningTimeoutMs specifies how long this client will wait for signing to finish before abandoning
	SigningTimeoutMs int `mapstructure:"signing_timeout_ms"`
}

func (cfg *Experiment) validate(errs []error) []error {
	if len(cfg.AdCerts.Mode) == 0 {
		return errs
	}
	if !(cfg.AdCerts.Mode == AdCertsSignerModeOff ||
		cfg.AdCerts.Mode == AdCertsSignerModeInprocess ||
		cfg.AdCerts.Mode == AdCertsSignerModeRemote) {
		return append(errs, ErrSignerModeIncorrect)
	}
	if cfg.AdCerts.Mode == AdCertsSignerModeInprocess {
		_, err := url.ParseRequestURI(cfg.AdCerts.InProcess.Origin)
		if err != nil {
			errs = append(errs, ErrInProcessSignerInvalidURL)
		}
		if len(cfg.AdCerts.InProcess.PrivateKey) == 0 {
			errs = append(errs, ErrInProcessSignerInvalidPrivateKey)
		}
		if cfg.AdCerts.InProcess.DNSRenewalIntervalInSeconds <= 0 {
			errs = append(errs, ErrInProcessSignerInvalidDNSRenewalInterval)
		}
		if cfg.AdCerts.InProcess.DNSCheckIntervalInSeconds <= 0 {
			errs = append(errs, ErrInProcessSignerInvalidDNSCheckInterval)
		}
	} else if cfg.AdCerts.Mode == AdCertsSignerModeRemote {
		_, err := url.ParseRequestURI(cfg.AdCerts.Remote.Url)
		if err != nil {
			errs = append(errs, ErrInvalidRemoteSignerURL)
		}
		if cfg.AdCerts.Remote.SigningTimeoutMs <= 0 {
			errs = append(errs, ErrInvalidRemoteSignerSigningTimeout)
		}
	}
	return errs
}
