package ttx

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type TtxAdapter struct {
	endpoint string
}

type Ext struct {
	Ttx impTtxExt `json:"ttx"`
}

type impTtxExt struct {
	Prod   string `json:"prod"`
	Zoneid string `json:"zoneid,omitempty"`
}

type reqExt struct {
	Ttx *reqTtxExt `json:"ttx,omitempty"`
}

type reqTtxExt struct {
	Caller []TtxCaller `json:"caller,omitempty"`
}

type TtxCaller struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

// CALLER Info used to track Prebid Server
// as one of the hops in the request to exchange
var CALLER = TtxCaller{"Prebid-Server", "n/a"}

type bidExt struct {
	Ttx bidTtxExt `json:"ttx,omitempty"`
}

type bidTtxExt struct {
	MediaType string `json:mediaType,omitempty`
}

// MakeRequests create the object for TTX Reqeust.
func (a *TtxAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	var adapterRequests []*adapters.RequestData

	// Construct request extension common to all imps
	// NOTE: not blocking adapter requests on errors
	// since request extension is optional.
	reqExt, err := makeReqExt(request)
	if err != nil {
		errs = append(errs, err)
	}
	request.Ext = reqExt

	// Break up multi-imp request into multiple external requests since we don't
	// support SRA in our exchange server
	for i := 0; i < len(request.Imp); i++ {
		if adapterReq, err := a.makeRequest(*request, request.Imp[i]); err == nil {
			adapterRequests = append(adapterRequests, adapterReq)
		} else {
			errs = append(errs, err)
		}
	}

	return adapterRequests, errs
}

func (a *TtxAdapter) makeRequest(request openrtb2.BidRequest, imp openrtb2.Imp) (*adapters.RequestData, error) {
	impCopy, err := makeImps(imp)

	if err != nil {
		return nil, err
	}

	request.Imp = []openrtb2.Imp{*impCopy}

	// Last Step
	reqJSON, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	return &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    reqJSON,
		Headers: headers,
	}, nil
}

func makeImps(imp openrtb2.Imp) (*openrtb2.Imp, error) {
	if imp.Banner == nil && imp.Video == nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Imp ID %s must have at least one of [Banner, Video] defined", imp.ID),
		}
	}

	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	var ttxExt openrtb_ext.ExtImp33across
	if err := json.Unmarshal(bidderExt.Bidder, &ttxExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	var impExt Ext
	impExt.Ttx.Prod = ttxExt.ProductId

	impExt.Ttx.Zoneid = ttxExt.SiteId

	if len(ttxExt.ZoneId) > 0 {
		impExt.Ttx.Zoneid = ttxExt.ZoneId
	}

	impExtJSON, err := json.Marshal(impExt)
	if err != nil {
		return nil, &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	imp.Ext = impExtJSON

	// Validate Video if it exists
	if imp.Video != nil {
		videoCopy, err := validateVideoParams(imp.Video, impExt.Ttx.Prod)

		imp.Video = videoCopy

		if err != nil {
			return nil, &errortypes.BadInput{
				Message: err.Error(),
			}
		}
	}

	return &imp, nil
}

func makeReqExt(request *openrtb2.BidRequest) ([]byte, error) {
	var reqExt reqExt

	if len(request.Ext) > 0 {
		if err := json.Unmarshal(request.Ext, &reqExt); err != nil {
			return nil, err
		}
	}

	if reqExt.Ttx == nil {
		reqExt.Ttx = &reqTtxExt{}
	}

	if reqExt.Ttx.Caller == nil {
		reqExt.Ttx.Caller = make([]TtxCaller, 0)
	}

	reqExt.Ttx.Caller = append(reqExt.Ttx.Caller, CALLER)

	return json.Marshal(reqExt)
}

// MakeBids make the bids for the bid response.
func (a *TtxAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var bidResp openrtb2.BidResponse

	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			var bidExt bidExt
			var bidType openrtb_ext.BidType

			if err := json.Unmarshal(sb.Bid[i].Ext, &bidExt); err != nil {
				bidType = openrtb_ext.BidTypeBanner
			} else {
				bidType = getBidType(bidExt)
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: bidType,
			})
		}
	}
	return bidResponse, nil

}

func validateVideoParams(video *openrtb2.Video, prod string) (*openrtb2.Video, error) {
	videoCopy := *video
	if videoCopy.W == 0 ||
		videoCopy.H == 0 ||
		videoCopy.Protocols == nil ||
		videoCopy.MIMEs == nil ||
		videoCopy.PlaybackMethod == nil {

		return nil, &errortypes.BadInput{
			Message: "One or more invalid or missing video field(s) w, h, protocols, mimes, playbackmethod",
		}
	}

	if videoCopy.Placement == 0 {
		videoCopy.Placement = 2
	}

	if prod == "instream" {
		videoCopy.Placement = 1

		if videoCopy.StartDelay == nil {
			videoCopy.StartDelay = openrtb2.StartDelay.Ptr(0)
		}
	}

	return &videoCopy, nil
}

func getBidType(ext bidExt) openrtb_ext.BidType {
	if ext.Ttx.MediaType == "video" {
		return openrtb_ext.BidTypeVideo
	}

	return openrtb_ext.BidTypeBanner
}

// Builder builds a new instance of the 33Across adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &TtxAdapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}
