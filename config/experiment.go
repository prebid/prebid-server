package config

import (
	"errors"
	"fmt"
	"net/url"
)

var (
	ErrSignerModeIncorrect              = errors.New("signer mode is not specified, specify 'off', 'inprocess' or 'remote'")
	ErrInProcessSignerInvalidPrivateKey = errors.New("private key for inprocess signer cannot be empty")

	ErrMsgInProcessSignerInvalidURL                = "invalid url for inprocess signer"
	ErrMsgInProcessSignerInvalidDNSRenewalInterval = "invalid dns renewal interval for inprocess signer"
	ErrMsgInProcessSignerInvalidDNSCheckInterval   = "invalid dns check interval for inprocess signer"
	ErrMsgInvalidRemoteSignerURL                   = "invalid url for remote signer"
	ErrMsgInvalidRemoteSignerSigningTimeout        = "invalid signing timeout for remote signer"
)

const (
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
			errs = append(errs, fmt.Errorf("%s: %s", ErrMsgInProcessSignerInvalidURL, cfg.AdCerts.InProcess.Origin))
		}
		if len(cfg.AdCerts.InProcess.PrivateKey) == 0 {
			errs = append(errs, ErrInProcessSignerInvalidPrivateKey)
		}
		if cfg.AdCerts.InProcess.DNSRenewalIntervalInSeconds <= 0 {
			errs = append(errs, fmt.Errorf("%s: %d", ErrMsgInProcessSignerInvalidDNSRenewalInterval, cfg.AdCerts.InProcess.DNSRenewalIntervalInSeconds))
		}
		if cfg.AdCerts.InProcess.DNSCheckIntervalInSeconds <= 0 {
			errs = append(errs, fmt.Errorf("%s: %d", ErrMsgInProcessSignerInvalidDNSCheckInterval, cfg.AdCerts.InProcess.DNSCheckIntervalInSeconds))
		}
	} else if cfg.AdCerts.Mode == AdCertsSignerModeRemote {
		_, err := url.ParseRequestURI(cfg.AdCerts.Remote.Url)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %s", ErrMsgInvalidRemoteSignerURL, cfg.AdCerts.Remote.Url))
		}
		if cfg.AdCerts.Remote.SigningTimeoutMs <= 0 {
			errs = append(errs, fmt.Errorf("%s: %d", ErrMsgInvalidRemoteSignerSigningTimeout, cfg.AdCerts.Remote.SigningTimeoutMs))
		}
	}
	return errs
}
