package adscert

import (
	"github.com/prebid/prebid-server/config"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRemoteSigner(t *testing.T) {
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
		signer := &remoteSigner{signatory: signatory}
		signatureMessage, err := signer.Sign("http://test.com", []byte{})
		if test.generateError {
			assert.EqualError(t, err, "Test error", "incorrect error returned for test: %s", test.desc)
		} else {
			if test.operationStatusOk {
				assert.NoError(t, err, "incorrect result for test: %s", test.desc)
				assert.Equal(t, "Success", signatureMessage, "incorrect message returned for test: %s", test.desc)
			} else {
				assert.EqualError(t, err, "error signing request: SIGNATURE_OPERATION_STATUS_UNDEFINED", "incorrect error type returned for test: %s", test.desc)
			}
		}
	}
}

func TestInvalidRemoteSignerConfig(t *testing.T) {
	type aTest struct {
		desc               string
		remoteSignerConfig config.AdsCertRemote
		expectedError      string
	}
	testCases := []aTest{
		{
			desc:               "empty remote url passed to config",
			remoteSignerConfig: config.AdsCertRemote{Url: "", SigningTimeoutMs: 5},
			expectedError:      "invalid url for remote signer",
		},
		{
			desc:               "invaild remote url passed to config",
			remoteSignerConfig: config.AdsCertRemote{Url: "test@com", SigningTimeoutMs: 5},
			expectedError:      "invalid url for remote signer",
		},
		{
			desc:               "empty private key passed to config",
			remoteSignerConfig: config.AdsCertRemote{Url: "http://test.com", SigningTimeoutMs: 0},
			expectedError:      "invalid signing timeout for remote signer",
		},
	}

	for _, test := range testCases {
		err := validateRemoteSignerConfig(test.remoteSignerConfig)
		assert.EqualError(t, err, test.expectedError, "error message should match for test: %s", test.desc)
	}
}

func TestValidRemoteSignerConfig(t *testing.T) {
	conf := config.AdsCertRemote{Url: "http://test.com", SigningTimeoutMs: 5}
	err := validateRemoteSignerConfig(conf)
	assert.NoError(t, err, "error message should not be returned")
}
