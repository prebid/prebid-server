package endpoints

import (
	"encoding/json"
	"net/http"

	"github.com/golang/glog"
)

type versionModel struct {
	Revision string `json:"revision"`
}

// NewVersionEndpoint returns the latest commit sha1 from which the binary was built
func NewVersionEndpoint(version string) http.HandlerFunc {
	if version == "" {
		version = "not-set"
	}

	return func(w http.ResponseWriter, _ *http.Request) {
		jsonOutput, err := json.Marshal(versionModel{
			Revision: version,
		})
		if err != nil {
			glog.Errorf("/version Critical error when trying to marshal versionModel: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Write(jsonOutput)
	}
}
