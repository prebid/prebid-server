package adscert

import (
	"crypto/rand"
	"fmt"
	"github.com/IABTechLab/adscert/pkg/adscert/api"
	"github.com/IABTechLab/adscert/pkg/adscert/discovery"
	"github.com/IABTechLab/adscert/pkg/adscert/signatory"
	"github.com/benbjohnson/clock"
	"github.com/prebid/prebid-server/config"
	"time"
)

const SignHeader = "X-Ads-Cert-Auth"

//Signer represents interface to access request Ads Cert signing functionality
type Signer interface {
	Sign(destinationURL string, body []byte) (string, error)
}

func NewAdCertsSigner(experimentAdCertsConfig config.ExperimentAdCerts) Signer {
	if !experimentAdCertsConfig.Enabled {
		return &NilSigner{}
	}
	if len(experimentAdCertsConfig.InProcess.Origin) > 0 {
		//for initial implementation support in-process signer only
		return newInProcessSigner(experimentAdCertsConfig.InProcess)
	}
	return &NilSigner{}
}

type inProcessSigner struct {
	localSignatory signatory.LocalAuthenticatedConnectionsSignatory
}

func (ips *inProcessSigner) Sign(destinationURL string, body []byte) (string, error) {
	req := &api.AuthenticatedConnectionSignatureRequest{
		RequestInfo: createRequestInfo(destinationURL, body),
	}
	resp, err := ips.localSignatory.SignAuthenticatedConnection(req)
	if err != nil {
		return "", err
	}
	if resp.GetSignatureOperationStatus() == api.SignatureOperationStatus_SIGNATURE_OPERATION_STATUS_OK {
		signatureMessage := resp.RequestInfo.SignatureInfo[0].SignatureMessage
		return signatureMessage, nil
	}
	return "", fmt.Errorf("Error signing request: %s", resp.GetSignatureOperationStatus())
}

func newInProcessSigner(inProcessSignerConfig config.InProcess) *inProcessSigner {
	return &inProcessSigner{
		localSignatory: *signatory.NewLocalAuthenticatedConnectionsSignatory(
			inProcessSignerConfig.Origin,
			rand.Reader,
			clock.New(),
			discovery.NewDefaultDnsResolver(),
			discovery.NewDefaultDomainStore(),
			time.Duration(inProcessSignerConfig.DNSCheckIntervalInSeconds)*time.Second,
			time.Duration(inProcessSignerConfig.DNSRenewalIntervalInSeconds)*time.Second,
			[]string{inProcessSignerConfig.PrivateKey}),
	}
}

func createRequestInfo(destinationURL string, body []byte) *api.RequestInfo {
	reqInfo := &api.RequestInfo{}
	signatory.SetRequestInfo(reqInfo, destinationURL, body)
	return reqInfo
}

type NilSigner struct {
}

func (ns *NilSigner) Sign(destinationURL string, body []byte) (string, error) {
	return "", nil
}
