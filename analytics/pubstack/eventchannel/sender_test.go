package eventchannel

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewHttpSender(t *testing.T) {
	requestBody := make([]byte, 10)
	server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()
		requestBody, _ = ioutil.ReadAll(req.Body)
		res.WriteHeader(200)
	}))

	defer server.Close()

	sender := NewHttpSender(server.Client(), server.URL)
	err := sender([]byte("message"))

	assert.Equal(t, requestBody, []byte("message"))
	assert.Nil(t, err)
}

func TestNewHttpSender_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(400)
	}))

	defer server.Close()

	sender := NewHttpSender(server.Client(), server.URL)
	err := sender([]byte("message"))

	assert.NotNil(t, err)
}
