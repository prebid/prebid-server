package adscert

import (
	"fmt"

	"github.com/IABTechLab/adscert/pkg/adscert/api"
	"github.com/IABTechLab/adscert/pkg/adscert/logger"
	"github.com/IABTechLab/adscert/pkg/adscert/signatory"
	"github.com/prebid/prebid-server/v3/config"
)

const SignHeader = "X-Ads-Cert-Auth"

// Signer represents interface to access request Ads Cert signing functionality
type Signer interface {
	Sign(destinationURL string, body []byte) (string, error)
}

type NilSigner struct {
}

func (ns *NilSigner) Sign(destinationURL string, body []byte) (string, error) {
	return "", nil
}

func NewAdCertsSigner(experimentAdCertsConfig config.ExperimentAdsCert) (Signer, error) {
	logger.SetLoggerImpl(&SignerLogger{})
	if experimentAdCertsConfig.Mode == config.AdCertsSignerModeInprocess {
		return newInProcessSigner(experimentAdCertsConfig.InProcess)
	}
	if experimentAdCertsConfig.Mode == config.AdCertsSignerModeRemote {
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
