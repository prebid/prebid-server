package aspects

import (
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/config"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

const reqTimeInQueueHeaderName = "X-Ngx-Request-Time"
const reqTimeoutHeaderName = "X-Ngx-Request-Timeout"

func TestQueuedRequestTimeoutWithTimeout(t *testing.T) {

	rw := ExecuteAspectRequest(t, "6", "5", true)

	assert.Equal(t, http.StatusRequestTimeout, rw.Code, "Http response code is incorrect, should be 408")
	assert.Equal(t, "", string(rw.Body.Bytes()), "Body should not be present in response")

}

func TestQueuedRequestTimeoutNoTimeout(t *testing.T) {

	rw := ExecuteAspectRequest(t, "0.9", "5", true)

	assert.Equal(t, http.StatusOK, rw.Code, "Http response code is incorrect, should be 200")
	assert.Equal(t, "Executed", string(rw.Body.Bytes()), "Body should be present in response")

}

func TestQueuedRequestNoHeaders(t *testing.T) {

	rw := ExecuteAspectRequest(t, "", "", false)

	assert.Equal(t, http.StatusOK, rw.Code, "Http response code is incorrect, should be 200")
	assert.Equal(t, "Executed", string(rw.Body.Bytes()), "Body should be present in response")

}

func TestQueuedRequestAllHeadersIncorrect(t *testing.T) {

	rw := ExecuteAspectRequest(t, "test1", "test2", true)

	assert.Equal(t, http.StatusBadRequest, rw.Code, "Http response code is incorrect, should be 400")
	assert.Equal(t, "", string(rw.Body.Bytes()), "Body should not be present in response")

}

func TestQueuedRequestSomeHeadersIncorrect(t *testing.T) {

	rw := ExecuteAspectRequest(t, "test1", "123", true)

	assert.Equal(t, http.StatusBadRequest, rw.Code, "Http response code is incorrect, should be 400")
	assert.Equal(t, "", string(rw.Body.Bytes()), "Body should not be present in response")

}

func MockEndpoint() httprouter.Handle {
	return httprouter.Handle(MockHandler)
}

func MockHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Write([]byte("Executed"))
}

func ExecuteAspectRequest(t *testing.T, timeInQueue string, reqTimeout string, setHeaders bool) *httptest.ResponseRecorder {
	rw := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/test", nil)
	if err != nil {
		assert.Fail(t, "Unable create mock http request")
	}
	if setHeaders {
		req.Header.Set(reqTimeInQueueHeaderName, timeInQueue)
		req.Header.Set(reqTimeoutHeaderName, reqTimeout)
	}

	customHeaders := config.CustomHeaders{reqTimeInQueueHeaderName, reqTimeoutHeaderName}

	handler := QueuedRequestTimeout(MockEndpoint(), customHeaders)

	r := httprouter.New()
	r.POST("/test", handler)

	r.ServeHTTP(rw, req)

	return rw
}
