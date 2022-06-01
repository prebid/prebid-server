package adscert

import (
	"errors"
	"github.com/IABTechLab/adscert/pkg/adscert/api"
	config2 "github.com/prebid/prebid-server/config"
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
		signatureMessage, err := signer.Sign("test.com", []byte{})
		if test.generateError {
			assert.EqualError(t, err, "Test error", "incorrect error")
		} else {
			if test.operationStatusOk {
				assert.NoError(t, err, "error should not be returned")
				assert.Equal(t, "Success", signatureMessage, "incorrect message returned")
			} else {
				assert.EqualError(t, err, "error signing request: SIGNATURE_OPERATION_STATUS_UNDEFINED", "incorrect error")
			}
		}
	}
}

func TestNilSigner(t *testing.T) {
	config := config2.ExperimentAdCerts{Enabled: true, InProcess: config2.InProcess{Origin: ""}, Remote: config2.Remote{Url: ""}}
	signer, err := NewAdCertsSigner(config)
	assert.NoError(t, err, "error should not be returned")
	message, err := signer.Sign("test.com", nil)
	assert.NoError(t, err, "error should not be returned")
	assert.Equal(t, "", message, "message should be empty")
}

func TestNilSignerForAdsCertDisabled(t *testing.T) {
	config := config2.ExperimentAdCerts{Enabled: false, InProcess: config2.InProcess{Origin: ""}, Remote: config2.Remote{Url: ""}}
	signer, err := NewAdCertsSigner(config)
	assert.NoError(t, err, "error should not be returned")
	message, err := signer.Sign("test.com", nil)
	assert.NoError(t, err, "error should not be returned")
	assert.Equal(t, "", message, "message should be empty")
}

func TestInPrecessAndRemoteSignersDefined(t *testing.T) {
	config := config2.ExperimentAdCerts{Enabled: true, InProcess: config2.InProcess{Origin: "test.com"}, Remote: config2.Remote{Url: "test.com"}}
	signer, err := NewAdCertsSigner(config)
	assert.Nil(t, signer, "no signer should be returned")
	assert.Error(t, err, "error should be returned")

}

type MockLocalAuthenticatedConnectionsSignatory struct {
	returnError       bool
	operationStatusOk bool
}

func (ips *MockLocalAuthenticatedConnectionsSignatory) SignAuthenticatedConnection(request *api.AuthenticatedConnectionSignatureRequest) (*api.AuthenticatedConnectionSignatureResponse, error) {
	if ips.returnError {
		return nil, errors.New("Test error")
	}
	response := &api.AuthenticatedConnectionSignatureResponse{
		RequestInfo: &api.RequestInfo{
			SignatureInfo: []*api.SignatureInfo{
				{SignatureMessage: "Success"},
			},
		},
	}
	if ips.operationStatusOk {
		response.SignatureOperationStatus = api.SignatureOperationStatus_SIGNATURE_OPERATION_STATUS_OK
	} else {
		response.SignatureOperationStatus = api.SignatureOperationStatus_SIGNATURE_OPERATION_STATUS_UNDEFINED
	}
	return response, nil
}
func (ips *MockLocalAuthenticatedConnectionsSignatory) VerifyAuthenticatedConnection(request *api.AuthenticatedConnectionVerificationRequest) (*api.AuthenticatedConnectionVerificationResponse, error) {
	return nil, nil
}
