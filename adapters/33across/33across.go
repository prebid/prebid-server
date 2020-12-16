package ttx

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type TtxAdapter struct {
	endpoint string
}

type Ext struct {
	Ttx ext `json:"ttx"`
}

type ext struct {
	Prod   string `json:"prod"`
	Zoneid string `json:"zoneid,omitempty"`
}

type bidExt struct {
	Ttx bidTtxExt `json:"ttx,omitempty"`
}

type bidTtxExt struct {
	MediaType string `json:mediaType,omitempty`
}

// MakeRequests create the object for TTX Reqeust.
func (a *TtxAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	var adapterRequests []*adapters.RequestData

	adapterReq, errors := a.makeRequest(request)
	if adapterReq != nil {
		adapterRequests = append(adapterRequests, adapterReq)
	}
	errs = append(errs, errors...)

	return adapterRequests, errors
}

// Update the request object to include custom value
// site.id
func (a *TtxAdapter) makeRequest(request *openrtb.BidRequest) (*adapters.RequestData, []error) {
	var errs []error

	// Make a copy as we don't want to change the original request
	reqCopy := *request
	if err := preprocess(&reqCopy); err != nil {
		errs = append(errs, err)
	}

	if reqCopy.Imp[0].Banner == nil && reqCopy.Imp[0].Video == nil {
		errs = append(errs, &errortypes.BadInput{
			Message: "At least one of [banner, video] formats must be defined in Imp. None found",
		})

		return nil, errs
	}

	// Last Step
	reqJSON, err := json.Marshal(reqCopy)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	return &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    reqJSON,
		Headers: headers,
	}, errs
}

// Mutate the request to get it ready to send to ttx.
func preprocess(request *openrtb.BidRequest) error {
	var imp = &request.Imp[0]
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	var ttxExt openrtb_ext.ExtImp33across
	if err := json.Unmarshal(bidderExt.Bidder, &ttxExt); err != nil {
		return &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	var impExt Ext
	impExt.Ttx.Prod = ttxExt.ProductId

	// Add zoneid if it's defined
	if len(ttxExt.ZoneId) > 0 {
		impExt.Ttx.Zoneid = ttxExt.ZoneId
	}

	impExtJSON, err := json.Marshal(impExt)
	if err != nil {
		return &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	imp.Ext = impExtJSON

	if request.Site != nil {
		siteCopy := *request.Site
		siteCopy.ID = ttxExt.SiteId
		request.Site = &siteCopy
	}

	// Validate Video if it exists
	if imp.Video != nil {
		videoCopy, err := validateVideoParams(imp.Video, impExt.Ttx.Prod)

		imp.Video = videoCopy

		if err != nil {
			return &errortypes.BadInput{
				Message: err.Error(),
			}
		}
	}

	return nil
}

// MakeBids make the bids for the bid response.
func (a *TtxAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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

	var bidResp openrtb.BidResponse

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

func validateVideoParams(video *openrtb.Video, prod string) (*openrtb.Video, error) {
	videoCopy := video
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
			videoCopy.StartDelay = openrtb.StartDelay.Ptr(0)
		}
	}

	return videoCopy, nil
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
