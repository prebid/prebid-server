package config

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExperimentValidate(t *testing.T) {
	testCases := []struct {
		desc           string
		data           Experiment
		expectErrors   bool
		expectedErrors []error
	}{
		{
			desc: "Remote signer config: invalid remote url passed to config",
			data: Experiment{
				AdCerts: ExperimentAdsCert{Mode: AdCertsSignerModeRemote, Remote: AdsCertRemote{Url: "test@com", SigningTimeoutMs: 5}},
			},
			expectErrors:   true,
			expectedErrors: []error{errors.New("invalid url for remote signer: test@com")},
		},
		{
			desc: "Remote signer config: invalid SigningTimeoutMs passed to config",
			data: Experiment{
				AdCerts: ExperimentAdsCert{Mode: AdCertsSignerModeRemote, Remote: AdsCertRemote{Url: "http://test.com", SigningTimeoutMs: 0}},
			},
			expectErrors:   true,
			expectedErrors: []error{errors.New("invalid signing timeout for remote signer: 0")},
		},
		{
			desc: "Remote signer config: invalid URL and SigningTimeoutMs passed to config",
			data: Experiment{
				AdCerts: ExperimentAdsCert{Mode: AdCertsSignerModeRemote, Remote: AdsCertRemote{Url: "test@com", SigningTimeoutMs: 0}},
			},
			expectErrors: true,
			expectedErrors: []error{errors.New("invalid url for remote signer: test@com"),
				errors.New("invalid signing timeout for remote signer: 0")},
		},
		{
			desc: "Remote signer config: valid URL and SigningTimeoutMs passed to config",
			data: Experiment{
				AdCerts: ExperimentAdsCert{Mode: AdCertsSignerModeRemote, Remote: AdsCertRemote{Url: "http://test.com", SigningTimeoutMs: 5}},
			},
			expectErrors:   false,
			expectedErrors: []error{},
		},
		{
			desc: "Experiment config: experiment config is empty",
			data: Experiment{
				AdCerts: ExperimentAdsCert{Mode: ""},
			},
			expectErrors:   false,
			expectedErrors: []error{},
		},
		{
			desc: "Experiment config: experiment config is off",
			data: Experiment{
				AdCerts: ExperimentAdsCert{Mode: AdCertsSignerModeOff},
			},
			expectErrors:   false,
			expectedErrors: []error{},
		},
		{
			desc: "Experiment config: experiment config is init with a wrong value",
			data: Experiment{
				AdCerts: ExperimentAdsCert{Mode: "test"},
			},
			expectErrors:   true,
			expectedErrors: []error{ErrSignerModeIncorrect},
		},
		{
			desc: "Inprocess signer config: valid config",
			data: Experiment{
				AdCerts: ExperimentAdsCert{Mode: AdCertsSignerModeInprocess, InProcess: AdsCertInProcess{Origin: "http://test.com", PrivateKey: "pk", DNSCheckIntervalInSeconds: 10, DNSRenewalIntervalInSeconds: 10}},
			},
			expectErrors:   false,
			expectedErrors: []error{},
		},
		{
			desc: "Inprocess signer config: invaild origin url passed to config",
			data: Experiment{
				AdCerts: ExperimentAdsCert{Mode: AdCertsSignerModeInprocess, InProcess: AdsCertInProcess{Origin: "test@com", PrivateKey: "pk", DNSCheckIntervalInSeconds: 10, DNSRenewalIntervalInSeconds: 10}},
			},
			expectErrors:   true,
			expectedErrors: []error{errors.New("invalid url for inprocess signer: test@com")},
		},
		{
			desc: "Inprocess signer config: empty PK passed to config",
			data: Experiment{
				AdCerts: ExperimentAdsCert{Mode: AdCertsSignerModeInprocess, InProcess: AdsCertInProcess{Origin: "http://test.com", PrivateKey: "", DNSCheckIntervalInSeconds: 10, DNSRenewalIntervalInSeconds: 10}},
			},
			expectErrors:   true,
			expectedErrors: []error{ErrInProcessSignerInvalidPrivateKey},
		},
		{
			desc: "Inprocess signer config: negative dns check interval passed to config",
			data: Experiment{
				AdCerts: ExperimentAdsCert{Mode: AdCertsSignerModeInprocess, InProcess: AdsCertInProcess{Origin: "http://test.com", PrivateKey: "pk", DNSCheckIntervalInSeconds: -10, DNSRenewalIntervalInSeconds: 10}},
			},
			expectErrors:   true,
			expectedErrors: []error{errors.New("invalid dns check interval for inprocess signer: -10")},
		},
		{
			desc: "Inprocess signer config: zero dns check interval passed to config",
			data: Experiment{
				AdCerts: ExperimentAdsCert{Mode: AdCertsSignerModeInprocess, InProcess: AdsCertInProcess{Origin: "http://test.com", PrivateKey: "pk", DNSCheckIntervalInSeconds: 10, DNSRenewalIntervalInSeconds: 0}},
			},
			expectErrors:   true,
			expectedErrors: []error{errors.New("invalid dns renewal interval for inprocess signer: 0")},
		},
		{
			desc: "Inprocess signer config: all config parameters are invalid",
			data: Experiment{
				AdCerts: ExperimentAdsCert{Mode: AdCertsSignerModeInprocess, InProcess: AdsCertInProcess{Origin: "test@com", PrivateKey: "", DNSCheckIntervalInSeconds: -10, DNSRenewalIntervalInSeconds: 0}},
			},
			expectErrors: true,
			expectedErrors: []error{
				errors.New("invalid url for inprocess signer: test@com"),
				ErrInProcessSignerInvalidPrivateKey,
				errors.New("invalid dns check interval for inprocess signer: -10"),
				errors.New("invalid dns renewal interval for inprocess signer: 0")},
		},
	}
	for _, test := range testCases {
		errs := test.data.validate([]error{})
		if test.expectErrors {
			assert.ElementsMatch(t, test.expectedErrors, errs, "Test case threw unexpected errors. Desc: %s  \n", test.desc)
		} else {
			assert.Empty(t, test.expectedErrors, "Test case should not return errors. Desc: %s  \n", test.desc)
		}
	}
}
