package aspects

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueuedRequestTimeoutWithTimeout(t *testing.T) {

	rw := ExecuteAspwctRequest(t, "6")

	assert.Equal(t, http.StatusRequestTimeout, rw.Code, "Http response code is incorrect, should be 408")

}

func TestQueuedRequestTimeoutNoTimeout(t *testing.T) {

	rw := ExecuteAspwctRequest(t, "0.9")

	assert.Equal(t, http.StatusOK, rw.Code, "Http response code is incorrect, should be 200")

}

func MockEndpoint() httprouter.Handle {
	return httprouter.Handle(MockFunct)
}

func MockFunct(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {}

func ExecuteAspwctRequest(t *testing.T, timeout string) *httptest.ResponseRecorder {
	rw := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/test", nil)
	if err != nil {
		assert.Fail(t, "Unable create mock http request")
	}
	req.Header.Set("X-Ngx-Request-Time", timeout)

	handler := QueuedRequestTimeout(MockEndpoint())

	r := httprouter.New()
	r.POST("/test", handler)

	r.ServeHTTP(rw, req)

	return rw
}
