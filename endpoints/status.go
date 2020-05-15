package endpoints

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// NewStatusEndpoint returns a handler which writes the given response when the app is ready to serve requests.
func NewStatusEndpoint(response string) httprouter.Handle {
	// Today, the app always considers itself ready to serve requests.
	if response == "" {
		return func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
			w.WriteHeader(http.StatusNoContent)
		}
	}

	responseBytes := []byte(response)
	return func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		w.Write(responseBytes)
	}
}
