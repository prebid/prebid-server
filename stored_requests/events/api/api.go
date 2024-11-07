package api

import (
	"io"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/v3/stored_requests/events"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type eventsAPI struct {
	saves         chan events.Save
	invalidations chan events.Invalidation
}

// NewEventsAPI creates an EventProducer that generates cache events from HTTP requests.
// The returned httprouter.Handle must be registered on both POST (update) and DELETE (invalidate)
// methods and provided an `:id` param via the URL, e.g.:
//
// apiEvents, apiEventsHandler, err := NewEventsApi()
// router.POST("/stored_requests", apiEventsHandler)
// router.DELETE("/stored_requests", apiEventsHandler)
// listener := events.Listen(cache, apiEvents)
//
// The returned HTTP endpoint should not be exposed on a public network without authentication
// as it allows direct writing to the cache via Update.
func NewEventsAPI() (events.EventProducer, httprouter.Handle) {
	api := &eventsAPI{
		invalidations: make(chan events.Invalidation),
		saves:         make(chan events.Save),
	}
	return api, httprouter.Handle(api.HandleEvent)
}

func (api *eventsAPI) HandleEvent(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if r.Method == "POST" {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Missing update data.\n"))
			return
		}

		var save events.Save
		if err := jsonutil.UnmarshalValid(body, &save); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid update.\n"))
			return
		}

		api.saves <- save
	} else if r.Method == "DELETE" {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Missing invalidation data.\n"))
			return
		}

		var invalidation events.Invalidation
		if err := jsonutil.UnmarshalValid(body, &invalidation); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid invalidation.\n"))
			return
		}

		api.invalidations <- invalidation
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (api *eventsAPI) Invalidations() <-chan events.Invalidation {
	return api.invalidations
}

func (api *eventsAPI) Saves() <-chan events.Save {
	return api.saves
}
