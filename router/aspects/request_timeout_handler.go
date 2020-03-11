package aspects

import (
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/config"
	"net/http"
	"strconv"
)

func QueuedRequestTimeout(f httprouter.Handle, reqTimeoutHeaders config.RequestTimeoutHeaders) httprouter.Handle {

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

		//Return HTTP 400 if request timeout headers are incorrect (wrong format)
		if reqTimeFloatErr != nil || reqTimeoutFloatErr != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		//Return HTTP 408 if requests stays too long in queue
		if reqTimeFloat >= reqTimeoutFloat {
			w.WriteHeader(http.StatusRequestTimeout)
			return
		}

		f(w, r, params)
	}

}
