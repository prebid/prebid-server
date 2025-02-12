package adscert

import (
	"fmt"
	"time"

	"github.com/IABTechLab/adscert/pkg/adscert/api"
	"github.com/IABTechLab/adscert/pkg/adscert/signatory"
	"github.com/prebid/prebid-server/v3/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// remoteSigner holds the signatory to add adsCert header to requests using remote signing server
type remoteSigner struct {
	signatory signatory.AuthenticatedConnectionsSignatory
}

// Sign adds adsCert header to requests using remote signing server
func (rs *remoteSigner) Sign(destinationURL string, body []byte) (string, error) {
	signatureResponse, err := rs.signatory.SignAuthenticatedConnection(
		&api.AuthenticatedConnectionSignatureRequest{
			RequestInfo: createRequestInfo(destinationURL, []byte(body)),
		})
	if err != nil {
		return "", err
	}
	return getSignatureMessage(signatureResponse)
}

func newRemoteSigner(remoteSignerConfig config.AdsCertRemote) (*remoteSigner, error) {
	// Establish the gRPC connection that the client will use to connect to the
	// signatory server.  Secure connections are not implemented at this time.
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	conn, err := grpc.Dial(remoteSignerConfig.Url, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial remote signer: %v", err)
	}

	clientOpts := &signatory.AuthenticatedConnectionsSignatoryClientOptions{
		Timeout: time.Duration(remoteSignerConfig.SigningTimeoutMs) * time.Millisecond}
	signatoryClient := signatory.NewAuthenticatedConnectionsSignatoryClient(conn, clientOpts)
	return &remoteSigner{signatory: signatoryClient}, nil

}
