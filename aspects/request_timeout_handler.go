package aspects

import (
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/config"
	"net/http"
	"strconv"
)

func QueuedRequestTimeout(f httprouter.Handle, custHeaders config.CustomHeaders) httprouter.Handle {

	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {

		reqTimeInQueue := r.Header.Get(custHeaders.RequestTimeInQueue)
		reqTimeFloat, _ := strconv.ParseFloat(reqTimeInQueue, 64)

		reqTimeout := r.Header.Get(custHeaders.RequestTimeoutInQueue)
		reqTimeoutFloat, _ := strconv.ParseFloat(reqTimeout, 64)

		if reqTimeFloat >= reqTimeoutFloat {
			w.WriteHeader(http.StatusRequestTimeout)
			return
		}

		f(w, r, params)
	}

}
