package aspects

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/metrics"

	"github.com/stretchr/testify/assert"
)

const reqTimeInQueueHeaderName = "X-Ngx-Request-Time"
const reqTimeoutHeaderName = "X-Request-Timeout"

func TestAny(t *testing.T) {
	testCases := []struct {
		reqTimeInQueue          string
		reqTimeOut              string
		setHeaders              bool
		expectedRespCode        int
		expectedRespCodeMessage string
		expectedRespBody        string
		expectedRespBodyMessage string
		requestStatusMetrics    bool
	}{
		{
			//TestQueuedRequestTimeoutWithTimeout
			reqTimeInQueue:          "6",
			reqTimeOut:              "5",
			setHeaders:              true,
			expectedRespCode:        http.StatusRequestTimeout,
			expectedRespCodeMessage: "Http response code is incorrect, should be 408",
			expectedRespBody:        "Queued request processing time exceeded maximum",
			expectedRespBodyMessage: "Body should have error message",
			requestStatusMetrics:    false,
		},
		{
			//TestQueuedRequestTimeoutNoTimeout
			reqTimeInQueue:          "0.9",
			reqTimeOut:              "5",
			setHeaders:              true,
			expectedRespCode:        http.StatusOK,
			expectedRespCodeMessage: "Http response code is incorrect, should be 200",
			expectedRespBody:        "Executed",
			expectedRespBodyMessage: "Body should be present in response",
			requestStatusMetrics:    true,
		},
		{
			//TestQueuedRequestNoHeaders
			reqTimeInQueue:          "",
			reqTimeOut:              "",
			setHeaders:              false,
			expectedRespCode:        http.StatusOK,
			expectedRespCodeMessage: "Http response code is incorrect, should be 200",
			expectedRespBody:        "Executed",
			expectedRespBodyMessage: "Body should be present in response",
			requestStatusMetrics:    true,
		},
		{
			//TestQueuedRequestSomeHeaders
			reqTimeInQueue:          "2",
			reqTimeOut:              "",
			setHeaders:              true,
			expectedRespCode:        http.StatusOK,
			expectedRespCodeMessage: "Http response code is incorrect, should be 200",
			expectedRespBody:        "Executed",
			expectedRespBodyMessage: "Body should be present in response",
			requestStatusMetrics:    true,
		},
	}

	for _, test := range testCases {
		reqTimeFloat, _ := strconv.ParseFloat(test.reqTimeInQueue, 64)
		result := ExecuteAspectRequest(t, test.reqTimeInQueue, test.reqTimeOut, test.setHeaders, metrics.ReqTypeVideo, test.requestStatusMetrics, reqTimeFloat)
		assert.Equal(t, test.expectedRespCode, result.Code, test.expectedRespCodeMessage)
		assert.Equal(t, test.expectedRespBody, result.Body.String(), test.expectedRespBodyMessage)
	}
}

func MockEndpoint() httprouter.Handle {
	return httprouter.Handle(MockHandler)
}

func MockHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Write([]byte("Executed"))
}

func ExecuteAspectRequest(t *testing.T, timeInQueue string, reqTimeout string, setHeaders bool, requestType metrics.RequestType, status bool, requestDuration float64) *httptest.ResponseRecorder {
	rw := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/test", nil)
	if err != nil {
		assert.Fail(t, "Unable create mock http request")
	}
	if setHeaders {
		req.Header.Set(reqTimeInQueueHeaderName, timeInQueue)
		req.Header.Set(reqTimeoutHeaderName, reqTimeout)
	}

	customHeaders := config.RequestTimeoutHeaders{RequestTimeInQueue: reqTimeInQueueHeaderName, RequestTimeoutInQueue: reqTimeoutHeaderName}

	metrics := &metrics.MetricsEngineMock{}

	metrics.On("RecordRequestQueueTime", status, requestType, time.Duration(requestDuration*float64(time.Second))).Once()

	handler := QueuedRequestTimeout(MockEndpoint(), customHeaders, metrics, requestType)

	r := httprouter.New()
	r.POST("/test", handler)

	r.ServeHTTP(rw, req)

	return rw
}
