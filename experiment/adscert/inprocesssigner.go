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

type inProcessSigner struct {
	signatory signatory.AuthenticatedConnectionsSignatory
}

func (ips *inProcessSigner) Sign(destinationURL string, body []byte) (string, error) {
	req := &api.AuthenticatedConnectionSignatureRequest{
		RequestInfo: createRequestInfo(destinationURL, body),
	}
	resp, err := ips.signatory.SignAuthenticatedConnection(req)
	if err != nil {
		return "", err
	}
	if resp.GetSignatureOperationStatus() == api.SignatureOperationStatus_SIGNATURE_OPERATION_STATUS_OK {
		signatureMessage := resp.RequestInfo.SignatureInfo[0].SignatureMessage
		return signatureMessage, nil
	}
	return "", fmt.Errorf("error signing request: %s", resp.GetSignatureOperationStatus())
}

func newInProcessSigner(inProcessSignerConfig config.AdsCertInProcess) *inProcessSigner {
	return &inProcessSigner{
		signatory: signatory.NewLocalAuthenticatedConnectionsSignatory(
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
