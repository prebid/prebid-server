package adscert

import "github.com/prebid/prebid-server/config"

type Signer interface {
	Sign()
}

func NewAdCertsSigner(experimentAdCertsConfig config.ExperimentAdCerts) Signer {
	if !experimentAdCertsConfig.Enabled {
		return nil
	}
	if len(experimentAdCertsConfig.InProcess.Origin) > 0 {
		//for initial implementation support in-process signer only
		return newInProcessSigner(experimentAdCertsConfig.InProcess)
	}
	if len(experimentAdCertsConfig.Remote.Url) > 0 {
		return newRemoteSigner(experimentAdCertsConfig.Remote)
	}

	return nil
}

type inProcessSigner struct {
	iam string
}

func (ips *inProcessSigner) Sign() {

}

func newInProcessSigner(inProcessSignerConfig config.InProcess) *inProcessSigner {
	return &inProcessSigner{iam: "in-process signer"}
}

type remoteSigner struct {
	iam string
	//stub
}

func (rs *remoteSigner) Sign() {
	//stub
}

func newRemoteSigner(remoteSignerConfig config.Remote) *remoteSigner {
	//stub
	return &remoteSigner{iam: "remote signer"}
}
