package adscert

import (
	"errors"
	"testing"

	"github.com/IABTechLab/adscert/pkg/adscert/api"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/stretchr/testify/assert"
)

func TestNilSigner(t *testing.T) {
	config := config.ExperimentAdsCert{Mode: "off", InProcess: config.AdsCertInProcess{Origin: ""}, Remote: config.AdsCertRemote{Url: ""}}
	signer, err := NewAdCertsSigner(config)
	assert.NoError(t, err, "error should not be returned if not inprocess nor remote signer defined, NilSigner should be returned instead")
	message, err := signer.Sign("test.com", nil)
	assert.NoError(t, err, "NilSigner should not return an error")
	assert.Equal(t, "", message, "incorrect message returned NilSigner")
}

func TestNilSignerForAdsCertDisabled(t *testing.T) {
	config := config.ExperimentAdsCert{Mode: "off", InProcess: config.AdsCertInProcess{Origin: ""}, Remote: config.AdsCertRemote{Url: ""}}
	signer, err := NewAdCertsSigner(config)
	assert.NoError(t, err, "error should not be returned if AdsCerts feature is disabled")
	message, err := signer.Sign("test.com", nil)
	assert.NoError(t, err, "NilSigner should not return an error")
	assert.Equal(t, "", message, "incorrect message returned NilSigner")
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
