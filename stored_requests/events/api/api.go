package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/stored_requests/events"
)

type eventsAPI struct {
	updates       chan events.Update
	invalidations chan events.Invalidation
}

// NewEventsAPI creates an EventProducer that generates cache events from HTTP requests.
// The returned httprouter.Handle must be registered on both POST (update) and DELETE (invalidate)
// methods and provided an `:id` param via the URL, e.g.:
//
// apiEvents, apiEventsHandler, err := NewEventsApi()
// router.POST("/stored_requests/:id", apiEventsHandler)
// router.DELETE("/stored_requests/:id", apiEventsHandler)
// listener := events.Listen(cache, apiEvents)
//
// The returned HTTP endpoint should not be exposed on a public network without authentication
// as it allows direct writing to the cache via Update.
func NewEventsAPI() (events.EventProducer, httprouter.Handle) {
	api := &eventsAPI{
		invalidations: make(chan events.Invalidation),
		updates:       make(chan events.Update),
	}
	return api, httprouter.Handle(api.HandleEvent)
}

func (api *eventsAPI) HandleEvent(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	//id := ps.ByName("id")

	if r.Method == "POST" {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Missing config data.\n"))
			return
		}

		// check if valid JSON
		var config json.RawMessage
		if err := json.Unmarshal(body, &config); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid config data.\n"))
			return
		}

		//api.updates <- map[string]json.RawMessage{id: config}
	} else if r.Method == "DELETE" {
		//api.invalidations <- []string{id}
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (api *eventsAPI) Invalidations() <-chan events.Invalidation {
	return api.invalidations
}

func (api *eventsAPI) Updates() <-chan events.Update {
	return api.updates
}
