package endpoints

import (
	"encoding/json"
	"net/http"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

const versionEndpointValueNotSet = "not-set"

// NewVersionEndpoint returns the latest git tag as the version and commit hash as the revision from which the binary was built
func NewVersionEndpoint(version, revision string) http.HandlerFunc {
	response, err := prepareVersionEndpointResponse(version, revision)
	if err != nil {
		glog.Fatalf("error creating /version endpoint response: %v", err)
	}

	return func(w http.ResponseWriter, _ *http.Request) {
		w.Write(response)
	}
}

func prepareVersionEndpointResponse(version, revision string) (json.RawMessage, error) {
	if version == "" {
		version = versionEndpointValueNotSet
	}
	if revision == "" {
		revision = versionEndpointValueNotSet
	}

	return jsonutil.Marshal(struct {
		Revision string `json:"revision"`
		Version  string `json:"version"`
	}{
		Revision: revision,
		Version:  version,
	})
}
