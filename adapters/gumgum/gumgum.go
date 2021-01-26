package gumgum

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// GumGumAdapter implements Bidder interface.
type GumGumAdapter struct {
	URI string
}

// MakeRequests makes the HTTP requests which should be made to fetch bids.
func (g *GumGumAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var validImps []openrtb.Imp
	var trackingId string

	numRequests := len(request.Imp)
	errs := make([]error, 0, numRequests)

	for i := 0; i < numRequests; i++ {
		imp := request.Imp[i]
		zone, err := preprocess(&imp)
		if err != nil {
			errs = append(errs, err)
		} else if request.Imp[i].Banner != nil {
			bannerCopy := *request.Imp[i].Banner
			if bannerCopy.W == nil && bannerCopy.H == nil && len(bannerCopy.Format) > 0 {
				format := bannerCopy.Format[0]
				bannerCopy.W = &(format.W)
				bannerCopy.H = &(format.H)
			}
			request.Imp[i].Banner = &bannerCopy
			validImps = append(validImps, request.Imp[i])
			trackingId = zone
		} else if request.Imp[i].Video != nil {
			err := validateVideoParams(request.Imp[i].Video)
			if err != nil {
				errs = append(errs, err)
			} else {
				validImps = append(validImps, request.Imp[i])
				trackingId = zone
			}
		}
	}

	if len(validImps) == 0 {
		return nil, errs
	}

	request.Imp = validImps

	if request.Site != nil {
		siteCopy := *request.Site
		siteCopy.ID = trackingId
		request.Site = &siteCopy
	}

	reqJSON, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     g.URI,
		Body:    reqJSON,
		Headers: headers,
	}}, errs
}

// MakeBids unpacks the server's response into Bids.
func (g *GumGumAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Bad user input: HTTP status %d", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Bad server response: HTTP status %d", response.StatusCode),
		}}
	}
	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Bad server response: %d. ", err),
		}}
	}

	var errs []error
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			mediaType := getMediaTypeForImpID(sb.Bid[i].ImpID, internalRequest.Imp)
			if mediaType == openrtb_ext.BidTypeVideo {
				price := strconv.FormatFloat(sb.Bid[i].Price, 'f', -1, 64)
				sb.Bid[i].AdM = strings.Replace(sb.Bid[i].AdM, "${AUCTION_PRICE}", price, -1)
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: mediaType,
			})
		}
	}

	return bidResponse, errs
}

func preprocess(imp *openrtb.Imp) (string, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		err = &errortypes.BadInput{
			Message: err.Error(),
		}
		return "", err
	}

	var gumgumExt openrtb_ext.ExtImpGumGum
	if err := json.Unmarshal(bidderExt.Bidder, &gumgumExt); err != nil {
		err = &errortypes.BadInput{
			Message: err.Error(),
		}
		return "", err
	}

	zone := gumgumExt.Zone
	return zone, nil
}

func getMediaTypeForImpID(impID string, imps []openrtb.Imp) openrtb_ext.BidType {
	for _, imp := range imps {
		if imp.ID == impID && imp.Banner != nil {
			return openrtb_ext.BidTypeBanner
		}
	}
	return openrtb_ext.BidTypeVideo
}

func validateVideoParams(video *openrtb.Video) (err error) {
	if video.W == 0 || video.H == 0 || video.MinDuration == 0 || video.MaxDuration == 0 || video.Placement == 0 || video.Linearity == 0 {
		return &errortypes.BadInput{
			Message: "Invalid or missing video field(s)",
		}
	}
	return nil
}

// Builder builds a new instance of the GumGum adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &GumGumAdapter{
		URI: config.Endpoint,
	}
	return bidder, nil
}
