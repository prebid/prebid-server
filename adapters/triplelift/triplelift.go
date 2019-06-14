package triplelift 

import (
	//"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/mxmCherry/openrtb"
	//"github.com/prebid/prebid-server/errortypes"
	//"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/adapters"
)

type TripleliftAdapter struct {
    endpoint string
}

func (a *TripleliftAdapter)  MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {
    errs := make([]error, 0, len(request.Imp))
    reqs := make([]*adapters.RequestData, 0, 1) 
    ad := "http://localhost:8076/s2s/auction"
    reqJSON, err := json.Marshal(request)
    if err != nil {
        errs = append(errs,err)
        return nil, errs
    }
    println("hi")
    headers := http.Header{}
    headers.Add("Content-Type","application/json;charset=utf-8")
    headers.Add("Accept", "application/json")
    reqs = append(reqs, &adapters.RequestData{
        Method: "POST",
        Uri: ad,
        Body: reqJSON,
        Headers: headers})
    return reqs, errs
}

func (a *TripleliftAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
    errs := make([]error,2)
    bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)
    return bidResponse, errs
}

func NewTripleliftBidder(client *http.Client, endpoint string) *TripleliftAdapter {
    return &TripleliftAdapter{
        endpoint: endpoint}
}


