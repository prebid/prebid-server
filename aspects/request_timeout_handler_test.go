package aspects

import (
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/config"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

var reqTimeInQueueHeaderName = "X-Ngx-Request-Time"
var reqTimeoutHeaderName = "X-Ngx-Request-Timeout"

func TestQueuedRequestTimeoutWithTimeout(t *testing.T) {

	rw := ExecuteAspectRequest(t, "6")

	assert.Equal(t, http.StatusRequestTimeout, rw.Code, "Http response code is incorrect, should be 408")

}

func TestQueuedRequestTimeoutNoTimeout(t *testing.T) {

	rw := ExecuteAspectRequest(t, "0.9")

	assert.Equal(t, http.StatusOK, rw.Code, "Http response code is incorrect, should be 200")

}

func MockEndpoint() httprouter.Handle {
	return httprouter.Handle(MockHandler)
}

func MockHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {}

func ExecuteAspectRequest(t *testing.T, timeout string) *httptest.ResponseRecorder {
	rw := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/test", nil)
	if err != nil {
		assert.Fail(t, "Unable create mock http request")
	}
	req.Header.Set(reqTimeInQueueHeaderName, timeout)
	req.Header.Set(reqTimeoutHeaderName, "5")

	customHeaders := config.CustomHeaders{reqTimeInQueueHeaderName, reqTimeoutHeaderName}

	handler := QueuedRequestTimeout(MockEndpoint(), customHeaders)

	r := httprouter.New()
	r.POST("/test", handler)

	r.ServeHTTP(rw, req)

	return rw
}
