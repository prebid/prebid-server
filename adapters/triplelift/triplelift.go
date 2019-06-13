package adapters

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {
    errs := make([]error, 0, len(request.Imp))
    reqs = make([]*RequestData, 0, 1) 
    ad = "http://localhost:8076/s2s/auction"
    reqJSON, err = json.Marshal(request)
    if err != nil {
        errs = append(errs,err)
        return nil, errs
    }
    headers := http.Header{}
    headers.add("Content-Type","application/json;charset=utf-8")
    headers.add("Accept", "application/json")
    reqs = append(reqs, &adapters.RequestData{
        Method: "POST",
        Uri: ad,
        Body: reqJSON,
        Headers: headers
    })
    return reqs, errs
}

func MakeBids(internalRequest *openrtb.BidRequest, externalRequest *RequestData, response *ResponseData) (*BidderResponse, []error) {
    bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)
    return bidResponse
}



