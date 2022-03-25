package aspects

import (
	"net/http"
	"strconv"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/metrics"
)

func QueuedRequestTimeout(f httprouter.Handle, reqTimeoutHeaders config.RequestTimeoutHeaders, metricsEngine metrics.MetricsEngine, requestType metrics.RequestType) httprouter.Handle {

	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {

		reqTimeInQueue := r.Header.Get(reqTimeoutHeaders.RequestTimeInQueue)
		reqTimeout := r.Header.Get(reqTimeoutHeaders.RequestTimeoutInQueue)

		//If request timeout headers are not specified - process request as usual
		if reqTimeInQueue == "" || reqTimeout == "" {
			f(w, r, params)
			return
		}

		reqTimeFloat, reqTimeFloatErr := strconv.ParseFloat(reqTimeInQueue, 64)
		reqTimeoutFloat, reqTimeoutFloatErr := strconv.ParseFloat(reqTimeout, 64)

		//Return HTTP 500 if request timeout headers are incorrect (wrong format)
		if reqTimeFloatErr != nil || reqTimeoutFloatErr != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Request timeout headers are incorrect (wrong format)"))
			return
		}

		reqTimeDuration := time.Duration(reqTimeFloat * float64(time.Second))

		//Return HTTP 408 if requests stays too long in queue
		if reqTimeFloat >= reqTimeoutFloat {
			w.WriteHeader(http.StatusRequestTimeout)
			w.Write([]byte("Queued request processing time exceeded maximum"))
			metricsEngine.RecordRequestQueueTime(false, requestType, reqTimeDuration)
			return
		}

		metricsEngine.RecordRequestQueueTime(true, requestType, reqTimeDuration)
		f(w, r, params)
	}

}
