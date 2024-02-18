package roulax

import (
	"encoding/json"
	"fmt"
	"github.com/prebid/prebid-server/v2/macros"
	"net/http"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/adapters"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

type adapter struct {
	endpoint string
}





// Builder builds a new instance of the Roulax adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}



// getImpAdotExt parses and return first imp ext or nil
func getImpRoulaxExt(imp *openrtb2.Imp) (ExtImpRoulax, error) {
	var extBidder adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &extBidder); err != nil {
		return ExtImpRoulax{}, err
	}

	var extImpRoulax ExtImpRoulax
	if err = json.Unmarshal(extBidder.Bidder, &extImpRoulax); err != nil {
		return ExtImpRoulax{}, err
	}

	return extImpRoulax, nil
}

// MakeRequests makes the HTTP requests which should be made to fetch bids.
func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) (res []*adapters.RequestData, errs []error) {
	reqJson, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	roulaxExt := getImpRoulaxExt(&request.Imp[0])


	a.endpoint = macros.ResolveMacros(a.endpoint, "{PUBLISHER_PATH}", roulaxExt.PublisherPath, -1)
	a.endpoint = macros.ResolveMacros(a.endpoint, "{PID}", roulaxExt.Pid, -1)


	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    reqJson,
		Headers: headers,
	}}, errs
}

// MakeBids unpacks the server's response into Bids.
// The bidder return a status code 204 when it cannot delivery an ad.
// MakeBids unpacks the server's response into Bids.
func (a *adapter) MakeBids(request *openrtb2.BidRequest, _ *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur
	var errs []error
	for _, seatBid := range response.SeatBid {
		for i, bid := range seatBid.Bid {
			bidType, err := getMediaTypeForImp(bid,request.Imp)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			})
		}
	}

	return bidResponse, errs
}

func getMediaTypeForImp(bid openrtb2.Bid, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	var mediaType openrtb_ext.BidType
	var typeCnt = 0
	for _, imp := range imps {
		if imp.ID == bid.ImpID {
			if imp.Banner != nil {
				typeCnt += 1
				mediaType = openrtb_ext.BidTypeBanner
			}
			if imp.Native != nil {
				typeCnt += 1
				mediaType = openrtb_ext.BidTypeNative
			}
			if imp.Video != nil {
				typeCnt += 1
				mediaType = openrtb_ext.BidTypeVideo
			}
		}
	}
	if typeCnt == 1 {
		return mediaType, nil
	}
	return penrtb_ext.BidTypeBanner,nil
}

