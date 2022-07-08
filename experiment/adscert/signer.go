package adscert

import (
	"errors"
	"fmt"
	"github.com/IABTechLab/adscert/pkg/adscert/api"
	"github.com/IABTechLab/adscert/pkg/adscert/signatory"
	"github.com/prebid/prebid-server/config"
)

const SignHeader = "X-Ads-Cert-Auth"

var (
	errBothSignersSpecified = errors.New("both inprocess and remote signers are specified. Please use just one signer")
)

//Signer represents interface to access request Ads Cert signing functionality
type Signer interface {
	Sign(destinationURL string, body []byte) (string, error)
}

type NilSigner struct {
}

func (ns *NilSigner) Sign(destinationURL string, body []byte) (string, error) {
	return "", nil
}

func NewAdCertsSigner(experimentAdCertsConfig config.ExperimentAdsCert) (Signer, error) {
	if !experimentAdCertsConfig.Enabled {
		return &NilSigner{}, nil
	}
	if len(experimentAdCertsConfig.InProcess.Origin) > 0 && len(experimentAdCertsConfig.Remote.Url) > 0 {
		return nil, errBothSignersSpecified
	}
	if len(experimentAdCertsConfig.InProcess.Origin) > 0 {
		return newInProcessSigner(experimentAdCertsConfig.InProcess)
	}
	if len(experimentAdCertsConfig.Remote.Url) > 0 {
		return newRemoteSigner(experimentAdCertsConfig.Remote)
	}
	return &NilSigner{}, nil
}

func createRequestInfo(destinationURL string, body []byte) *api.RequestInfo {
	// The RequestInfo proto contains details about the individual ad request
	// being signed.  A SetRequestInfo helper function derives a hash of the
	// destination URL and body, setting these value on the RequestInfo message.
	reqInfo := &api.RequestInfo{}
	signatory.SetRequestInfo(reqInfo, destinationURL, body)
	return reqInfo
}

func getSignatureMessage(signatureResponse *api.AuthenticatedConnectionSignatureResponse) (string, error) {
	if signatureResponse.GetSignatureOperationStatus() == api.SignatureOperationStatus_SIGNATURE_OPERATION_STATUS_OK {
		signatureMessage := signatureResponse.RequestInfo.SignatureInfo[0].SignatureMessage
		return signatureMessage, nil
	}
	return "", fmt.Errorf("error signing request: %s", signatureResponse.GetSignatureOperationStatus())
}
