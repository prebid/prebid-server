package aspects

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
	"strconv"
)

var headerName = "X-Ngx-Request-Time"
var queuedReqTimeout = 5.0 //seconds

func QueuedRequestTimeout(f httprouter.Handle) httprouter.Handle {

	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {

		reqTimeInQueue := r.Header.Get(headerName)
		reqTime, _ := strconv.ParseFloat(reqTimeInQueue, 64)

		if reqTime >= queuedReqTimeout {
			w.WriteHeader(http.StatusRequestTimeout)
			return
		}

		f(w, r, params)
	}

}
