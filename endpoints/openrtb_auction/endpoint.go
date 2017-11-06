package openrtb_auction

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
	"github.com/mxmCherry/openrtb"
	"encoding/json"
	"github.com/prebid/prebid-server/exchange"
	"fmt"
	"context"
)

type EndpointDeps struct {
	Exchange exchange.Exchange
}

func (deps *EndpointDeps) Auction(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	req, err := deps.parseRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Invalid request format: %s", err.Error())))
	}
	response := deps.Exchange.HoldAuction(context.Background(), req) // TODO: Fix the context timeout.
	responseBytes, err := json.Marshal(response)
	if err == nil {
		w.WriteHeader(200)
		w.Write(responseBytes)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error transforming response into JSON."))
	}
}

func (deps *EndpointDeps) parseRequest(httpRequest *http.Request) (*openrtb.BidRequest, error) {
	var ortbRequest openrtb.BidRequest
	err := json.NewDecoder(httpRequest.Body).Decode(&ortbRequest)
	if err != nil {
		return nil, err
	}
	return &ortbRequest, nil
}
