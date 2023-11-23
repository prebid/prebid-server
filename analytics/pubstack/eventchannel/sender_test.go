package eventchannel

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildEndpointSender(t *testing.T) {
	requestBody := make([]byte, 10)
	server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()
		requestBody, _ = io.ReadAll(req.Body)
		res.WriteHeader(200)
	}))

	defer server.Close()

	sender := BuildEndpointSender(server.Client(), server.URL, "module")
	err := sender([]byte("message"))

	assert.Equal(t, []byte("message"), requestBody)
	assert.NoError(t, err)
}

func TestBuildEndpointSender_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(400)
	}))

	defer server.Close()

	sender := BuildEndpointSender(server.Client(), server.URL, "module")
	err := sender([]byte("message"))

	assert.Error(t, err)
}
