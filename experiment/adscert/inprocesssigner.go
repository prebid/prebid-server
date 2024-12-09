package adscert

import (
	"crypto/rand"
	"time"

	"github.com/IABTechLab/adscert/pkg/adscert/api"
	"github.com/IABTechLab/adscert/pkg/adscert/discovery"
	"github.com/IABTechLab/adscert/pkg/adscert/signatory"
	"github.com/benbjohnson/clock"
	"github.com/prebid/prebid-server/v3/config"
)

// inProcessSigner holds the signatory to add adsCert header to requests using in process go library
type inProcessSigner struct {
	signatory signatory.AuthenticatedConnectionsSignatory
}

// Sign adds adsCert header to requests using in process go library
func (ips *inProcessSigner) Sign(destinationURL string, body []byte) (string, error) {
	req := &api.AuthenticatedConnectionSignatureRequest{
		RequestInfo: createRequestInfo(destinationURL, body),
	}
	signatureResponse, err := ips.signatory.SignAuthenticatedConnection(req)
	if err != nil {
		return "", err
	}
	return getSignatureMessage(signatureResponse)
}

func newInProcessSigner(inProcessSignerConfig config.AdsCertInProcess) (*inProcessSigner, error) {
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
	}, nil
}
