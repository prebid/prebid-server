package adscert

import (
	"github.com/prebid/prebid-server/config"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInProcessSigner(t *testing.T) {
	type aTest struct {
		desc              string
		generateError     bool
		operationStatusOk bool
	}
	testCases := []aTest{
		{
			desc:              "generate signer error",
			generateError:     true,
			operationStatusOk: false,
		},
		{
			desc:              "generate valid response without signature operation error",
			generateError:     false,
			operationStatusOk: true,
		},
		{
			desc:              "generate valid response with signature operation error",
			generateError:     false,
			operationStatusOk: false,
		},
	}

	for _, test := range testCases {
		signatory := &MockLocalAuthenticatedConnectionsSignatory{
			returnError:       test.generateError,
			operationStatusOk: test.operationStatusOk,
		}
		signer := &inProcessSigner{signatory: signatory}
		signatureMessage, err := signer.Sign("http://test.com", []byte{})
		if test.generateError {
			assert.EqualError(t, err, "Test error", "incorrect error returned for test: %s", test.desc)
		} else {
			if test.operationStatusOk {
				assert.NoError(t, err, "incorrect result for test: %s", test.desc)
				assert.Equal(t, "Success", signatureMessage, "incorrect message returned for test : %s", test.desc)
			} else {
				assert.EqualError(t, err, "error signing request: SIGNATURE_OPERATION_STATUS_UNDEFINED", "incorrect error type returned for test: %s", test.desc)
			}
		}
	}
}

func TestInvalidInProcessSignerConfig(t *testing.T) {
	type aTest struct {
		desc                  string
		inProcessSignerConfig config.AdsCertInProcess
		expectedError         string
	}
	testCases := []aTest{
		{
			desc:                  "empty origin url passed to config",
			inProcessSignerConfig: config.AdsCertInProcess{Origin: "", PrivateKey: "pk", DNSCheckIntervalInSeconds: 10, DNSRenewalIntervalInSeconds: 10},
			expectedError:         "invalid url for inprocess signer",
		},
		{
			desc:                  "invaild origin url passed to config",
			inProcessSignerConfig: config.AdsCertInProcess{Origin: "test@com", PrivateKey: "pk", DNSCheckIntervalInSeconds: 10, DNSRenewalIntervalInSeconds: 10},
			expectedError:         "invalid url for inprocess signer",
		},
		{
			desc:                  "empty private key passed to config",
			inProcessSignerConfig: config.AdsCertInProcess{Origin: "http://test.com", PrivateKey: "", DNSCheckIntervalInSeconds: 10, DNSRenewalIntervalInSeconds: 10},
			expectedError:         "invalid private key for inprocess signer",
		},
		{
			desc:                  "negative dns check interval passed to config",
			inProcessSignerConfig: config.AdsCertInProcess{Origin: "http://test.com", PrivateKey: "pk", DNSCheckIntervalInSeconds: -10, DNSRenewalIntervalInSeconds: 10},
			expectedError:         "invalid dns check interval for inprocess signer",
		},
		{
			desc:                  "zero dns renewal interval passed to config",
			inProcessSignerConfig: config.AdsCertInProcess{Origin: "http://test.com", PrivateKey: "pk", DNSCheckIntervalInSeconds: 10, DNSRenewalIntervalInSeconds: 0},
			expectedError:         "invalid dns renewal interval for inprocess signer",
		},
	}

	for _, test := range testCases {
		err := validateInProcessSignerConfig(test.inProcessSignerConfig)
		assert.EqualError(t, err, test.expectedError, "error message should match for test: %s", test.desc)
	}
}

func TestValidInProcessSignerConfig(t *testing.T) {
	conf := config.AdsCertInProcess{Origin: "http://test.com", PrivateKey: "pk", DNSCheckIntervalInSeconds: 10, DNSRenewalIntervalInSeconds: 10}
	err := validateInProcessSignerConfig(conf)
	assert.NoError(t, err, "error message should not be returned")
}
