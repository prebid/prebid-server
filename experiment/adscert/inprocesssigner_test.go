package adscert

import (
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
