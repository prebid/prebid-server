package endpoints

import (
	"encoding/json"
	"net/http"

	"github.com/golang/glog"
)

type versionModel struct {
	Version string `json:"version"`
}

type revisionModel struct {
	Revision string `json:"revision"`
}

// NewVersionEndpoint returns the version derived from the latest git tag from which the binary was built
func NewVersionEndpoint(version string) http.HandlerFunc {
	if version == "" {
		version = "not-set"
	}

	return func(w http.ResponseWriter, _ *http.Request) {
		jsonOutput, err := json.Marshal(versionModel{
			Version: version,
		})
		if err != nil {
			glog.Errorf("/version Critical error when trying to marshal versionModel: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Write(jsonOutput)
	}
}

// NewRevisionEndpoint returns the latest commit sha1 from which the binary was built
func NewRevisionEndpoint(revision string) http.HandlerFunc {
	if revision == "" {
		revision = "not-set"
	}

	return func(w http.ResponseWriter, _ *http.Request) {
		jsonOutput, err := json.Marshal(revisionModel{
			Revision: revision,
		})
		if err != nil {
			glog.Errorf("/revision Critical error when trying to marshal revisionModel: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Write(jsonOutput)
	}
}
