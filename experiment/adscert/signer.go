package adscert

import (
	"errors"
	"github.com/prebid/prebid-server/config"
)

const SignHeader = "X-Ads-Cert-Auth"

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
		return nil, errors.New("both inprocess and remote signers are specified. Please use just one signer")
	}
	if len(experimentAdCertsConfig.InProcess.Origin) > 0 {
		return newInProcessSigner(experimentAdCertsConfig.InProcess), nil
	}
	if len(experimentAdCertsConfig.Remote.Url) > 0 {
		return newRemoteSigner(experimentAdCertsConfig.Remote)
	}
	return &NilSigner{}, nil
}
