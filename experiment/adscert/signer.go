package adscert

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/IABTechLab/adscert/pkg/adscert/api"
	"github.com/IABTechLab/adscert/pkg/adscert/discovery"
	"github.com/IABTechLab/adscert/pkg/adscert/signatory"
	"github.com/benbjohnson/clock"
	"github.com/prebid/prebid-server/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"time"
)

const SignHeader = "X-Ads-Cert-Auth"

//Signer represents interface to access request Ads Cert signing functionality
type Signer interface {
	Sign(destinationURL string, body []byte) (string, error)
}

func NewAdCertsSigner(experimentAdCertsConfig config.ExperimentAdCerts) (Signer, error) {
	if !experimentAdCertsConfig.Enabled {
		return &NilSigner{}, nil
	}
	if len(experimentAdCertsConfig.InProcess.Origin) > 0 && len(experimentAdCertsConfig.Remote.Url) > 0 {
		return nil, errors.New("both in-process and remote signers are specified. Please use just one signer")
	}
	if len(experimentAdCertsConfig.InProcess.Origin) > 0 {
		//for initial implementation support in-process signer only
		return newInProcessSigner(experimentAdCertsConfig.InProcess), nil
	}
	if len(experimentAdCertsConfig.Remote.Url) > 0 {
		return newRemoteSigner(experimentAdCertsConfig.Remote)
	}
	return &NilSigner{}, nil
}

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

func newInProcessSigner(inProcessSignerConfig config.InProcess) *inProcessSigner {
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

type remoteSigner struct {
	signatory signatory.AuthenticatedConnectionsSignatory
}

func (rs *remoteSigner) Sign(destinationURL string, body []byte) (string, error) {
	// The RequestInfo proto contains details about the individual ad request
	// being signed.  A SetRequestInfo helper function derives a hash of the
	// destination URL and body, setting these value on the RequestInfo message.
	reqInfo := &api.RequestInfo{}
	signatory.SetRequestInfo(reqInfo, destinationURL, []byte(body))

	// Request the signature.
	signatureResponse, err := rs.signatory.SignAuthenticatedConnection(
		&api.AuthenticatedConnectionSignatureRequest{
			RequestInfo: reqInfo,
		})
	if err != nil {
		return "", err
	}
	if signatureResponse != nil && signatureResponse.SignatureOperationStatus == api.SignatureOperationStatus_SIGNATURE_OPERATION_STATUS_OK {
		signatureMessage := signatureResponse.RequestInfo.SignatureInfo[0].SignatureMessage
		return signatureMessage, err
	}
	return "", nil
}

func newRemoteSigner(remoteSignerConfig config.Remote) (*remoteSigner, error) {
	// Establish the gRPC connection that the client will use to connect to the
	// signatory server.  This basic example uses unauthenticated connections
	// which should not be used in a production environment.
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	conn, err := grpc.Dial(remoteSignerConfig.Url, opts...)
	if err != nil {
		fmt.Errorf("failed to dial remote signer: %v", err)
	}
	//defer conn.Close() -- where this should be?

	// Optional: performs a health check against the server before actually
	// trying to invoke the signatory service.
	err = performOptionalHealthCheckRPC(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to perform a health check of remote signer: %v", err)
	}

	clientOpts := &signatory.AuthenticatedConnectionsSignatoryClientOptions{
		Timeout: time.Duration(remoteSignerConfig.SigningTimeoutMs) * time.Millisecond}
	signatoryClient := signatory.NewAuthenticatedConnectionsSignatoryClient(conn, clientOpts)
	return &remoteSigner{signatory: signatoryClient}, nil

}

type NilSigner struct {
}

func (ns *NilSigner) Sign(destinationURL string, body []byte) (string, error) {
	return "", nil
}

func performOptionalHealthCheckRPC(conn *grpc.ClientConn) error {
	hctx, hcancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer hcancel()
	healthClient := grpc_health_v1.NewHealthClient(conn)
	healthCheckResponse, err := healthClient.Check(hctx, &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		return fmt.Errorf("failed to pass heath check of remote signer: %v", err)
	}
	if healthCheckResponse.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		return fmt.Errorf("failed to pass heath status of remote signer: %v", healthCheckResponse.Status)
	}
	return nil
}
