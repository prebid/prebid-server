package adscert

import (
	crypto_rand "crypto/rand"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"github.com/IABTechLab/adscert/pkg/adscert/api"
	"github.com/IABTechLab/adscert/pkg/adscert/discovery"
	"github.com/IABTechLab/adscert/pkg/adscert/signatory"
	"github.com/benbjohnson/clock"
	"github.com/prebid/prebid-server/config"
	"time"
)

const SignHeader = "X-Ads-Cert-Auth"

type Signer interface {
	Sign(destinationURL string, body []byte) (string, error)
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
	return "", fmt.Errorf("Error signing request: %s", resp.GetSignatureOperationStatus().String())

}

func newInProcessSigner(inProcessSignerConfig config.InProcess) *inProcessSigner {

	return &inProcessSigner{
		localSignatory: *signatory.NewLocalAuthenticatedConnectionsSignatory(
			inProcessSignerConfig.Origin,
			crypto_rand.Reader,
			clock.New(),
			discovery.NewDefaultDnsResolver(),
			discovery.NewDefaultDomainStore(),
			time.Duration(inProcessSignerConfig.DNSCheckIntervalInSeconds)*time.Second,
			time.Duration(inProcessSignerConfig.DNSRenewalIntervalInSeconds)*time.Second,
			[]string{inProcessSignerConfig.PrivateKey}),
	}
}

type remoteSigner struct {
	//stub
}

func (rs *remoteSigner) Sign(destinationURL string, body []byte) (string, error) {
	//stub
	return "", nil
}

func newRemoteSigner(remoteSignerConfig config.Remote) *remoteSigner {
	//stub
	return &remoteSigner{}
}

func createRequestInfo(destinationURL string, body []byte) *api.RequestInfo {
	reqInfo := &api.RequestInfo{}
	signatory.SetRequestInfo(reqInfo, destinationURL, body)
	return reqInfo
}

//decodeKey omits algorythm related data from key and returns base64 encoded result
//inputKey format example: -----BEGIN PRIVATE KEY-----
//MC4CAQAwBQYDK2VuBCIEIOBY0UbGUgGCuk09FVM9p2VeoglOj76NWJ66aJSSszpl
//-----END PRIVATE KEY-----
func decodeKey(inputKey string) string {
	block, _ := pem.Decode([]byte(inputKey))
	key := block.Bytes[len(block.Bytes)-32:]             //32 bytes key
	swEnc := base64.RawStdEncoding.EncodeToString((key)) //should be URL-safe

	///--------To check key can be decoded-----
	//decodedKeyBytes, err := base64.RawURLEncoding.DecodeString(swEnc)
	//fmt.Println("error ", err)
	//fmt.Println("publicKeyBytes", string(decodedKeyBytes))
	///--------

	return swEnc
}
