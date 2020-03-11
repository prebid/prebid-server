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
const reqTimeoutHeaderName = "X-Request-Timeout"

func TestAny(t *testing.T) {
	testCases := []struct {
		reqTimeInQueue          string
		reqTimeOut              string
		setHeaders              bool
		extectedRespCode        int
		expectedRespCodeMessage string
		expectedRespBody        string
		expectedRespBodyMessage string
	}{
		{
			//TestQueuedRequestTimeoutWithTimeout
			reqTimeInQueue:          "6",
			reqTimeOut:              "5",
			setHeaders:              true,
			extectedRespCode:        http.StatusRequestTimeout,
			expectedRespCodeMessage: "Http response code is incorrect, should be 408",
			expectedRespBody:        "",
			expectedRespBodyMessage: "Body should not be present in response",
		},
		{
			//TestQueuedRequestTimeoutNoTimeout
			reqTimeInQueue:          "0.9",
			reqTimeOut:              "5",
			setHeaders:              true,
			extectedRespCode:        http.StatusOK,
			expectedRespCodeMessage: "Http response code is incorrect, should be 200",
			expectedRespBody:        "Executed",
			expectedRespBodyMessage: "Body should be present in response",
		},
		{
			//TestQueuedRequestNoHeaders
			reqTimeInQueue:          "",
			reqTimeOut:              "",
			setHeaders:              false,
			extectedRespCode:        http.StatusOK,
			expectedRespCodeMessage: "Http response code is incorrect, should be 200",
			expectedRespBody:        "Executed",
			expectedRespBodyMessage: "Body should be present in response",
		},
		{
			//TestQueuedRequestSomeHeaders
			reqTimeInQueue:          "2",
			reqTimeOut:              "",
			setHeaders:              true,
			extectedRespCode:        http.StatusOK,
			expectedRespCodeMessage: "Http response code is incorrect, should be 200",
			expectedRespBody:        "Executed",
			expectedRespBodyMessage: "Body should be present in response",
		},
		{
			//TestQueuedRequestAllHeadersIncorrect
			reqTimeInQueue:          "test1",
			reqTimeOut:              "test2",
			setHeaders:              true,
			extectedRespCode:        http.StatusBadRequest,
			expectedRespCodeMessage: "Http response code is incorrect, should be 400",
			expectedRespBody:        "",
			expectedRespBodyMessage: "Body should not be present in response",
		},
		{
			//TestQueuedRequestSomeHeadersIncorrect
			reqTimeInQueue:          "test1",
			reqTimeOut:              "123",
			setHeaders:              true,
			extectedRespCode:        http.StatusBadRequest,
			expectedRespCodeMessage: "Http response code is incorrect, should be 400",
			expectedRespBody:        "",
			expectedRespBodyMessage: "Body should not be present in response",
		},
	}

	for _, test := range testCases {
		result := ExecuteAspectRequest(t, test.reqTimeInQueue, test.reqTimeOut, test.setHeaders)
		assert.Equal(t, test.extectedRespCode, result.Code, test.expectedRespCodeMessage)
		assert.Equal(t, test.expectedRespBody, string(result.Body.Bytes()), test.expectedRespBodyMessage)
	}
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

	customHeaders := config.RequestTimeoutHeaders{reqTimeInQueueHeaderName, reqTimeoutHeaderName}

	handler := QueuedRequestTimeout(MockEndpoint(), customHeaders)

	r := httprouter.New()
	r.POST("/test", handler)

	r.ServeHTTP(rw, req)

	return rw
}
