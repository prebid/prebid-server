package endpoints

import (
	"encoding/json"
	"net/http"

	"github.com/golang/glog"
)

type versionModel struct {
	Version  string `json:"version"`
	Revision string `json:"revision"`
}

// NewVersionEndpoint returns the latest git tag as the version and commit hash as the revision from which the binary was built
func NewVersionEndpoint(version string, revision string) http.HandlerFunc {
	if version == "" {
		version = "not-set"
	}
	if revision == "" {
		revision = "not-set"
	}

	return func(w http.ResponseWriter, _ *http.Request) {
		jsonOutput, err := json.Marshal(versionModel{
			Version:  version,
			Revision: revision,
		})
		if err != nil {
			glog.Errorf("/version Critical error when trying to marshal versionModel: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Write(jsonOutput)
	}
}
