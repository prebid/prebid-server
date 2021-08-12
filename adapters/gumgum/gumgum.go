package gumgum

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
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
func (g *GumGumAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var validImps []openrtb2.Imp
	var siteCopy openrtb2.Site
	if request.Site != nil {
		siteCopy = *request.Site
	}

	numRequests := len(request.Imp)
	errs := make([]error, 0, numRequests)

	for i := 0; i < numRequests; i++ {
		imp := request.Imp[i]
		gumgumExt, err := preprocess(&imp)
		if err != nil {
			errs = append(errs, err)
		} else {
			if gumgumExt.Zone != "" {
				siteCopy.ID = gumgumExt.Zone
			}

			if gumgumExt.PubID != 0 {
				if siteCopy.Publisher != nil {
					siteCopy.Publisher.ID = strconv.FormatFloat(gumgumExt.PubID, 'f', -1, 64)
				} else {
					siteCopy.Publisher = &openrtb2.Publisher{ID: strconv.FormatFloat(gumgumExt.PubID, 'f', -1, 64)}
				}
			}

			validImps = append(validImps, imp)
		}
	}

	if len(validImps) == 0 {
		return nil, errs
	}

	request.Imp = validImps

	if request.Site != nil {
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
func (g *GumGumAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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
	var bidResp openrtb2.BidResponse
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

func preprocess(imp *openrtb2.Imp) (*openrtb_ext.ExtImpGumGum, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		err = &errortypes.BadInput{
			Message: err.Error(),
		}
		return nil, err
	}

	var gumgumExt openrtb_ext.ExtImpGumGum
	if err := json.Unmarshal(bidderExt.Bidder, &gumgumExt); err != nil {
		err = &errortypes.BadInput{
			Message: err.Error(),
		}
		return nil, err
	}

	if imp.Banner != nil && imp.Banner.W == nil && imp.Banner.H == nil && len(imp.Banner.Format) > 0 {
		bannerCopy := *imp.Banner
		format := bannerCopy.Format[0]
		bannerCopy.W = &(format.W)
		bannerCopy.H = &(format.H)

		if gumgumExt.Slot != 0 {
			var err error
			bannerExt := getBiggerFormat(bannerCopy.Format, gumgumExt.Slot)
			bannerCopy.Ext, err = json.Marshal(&bannerExt)
			if err != nil {
				return nil, err
			}
		}

		imp.Banner = &bannerCopy
	}

	if imp.Video != nil {
		err := validateVideoParams(imp.Video)
		if err != nil {
			return nil, err
		}

		if gumgumExt.IrisID != "" {
			videoCopy := *imp.Video
			videoExt := openrtb_ext.ExtImpGumGumVideo{IrisID: gumgumExt.IrisID}
			videoCopy.Ext, err = json.Marshal(&videoExt)
			if err != nil {
				return nil, err
			}
			imp.Video = &videoCopy
		}
	}

	return &gumgumExt, nil
}

func getBiggerFormat(formatList []openrtb2.Format, slot float64) openrtb_ext.ExtImpGumGumBanner {
	maxw := int64(0)
	maxh := int64(0)
	greatestVal := int64(0)
	for _, size := range formatList {
		var biggerSide int64
		if size.W > size.H {
			biggerSide = size.W
		} else {
			biggerSide = size.H
		}

		if biggerSide > greatestVal || (biggerSide == greatestVal && size.W >= maxw && size.H >= maxh) {
			greatestVal = biggerSide
			maxh = size.H
			maxw = size.W
		}
	}

	bannerExt := openrtb_ext.ExtImpGumGumBanner{Si: slot, MaxW: float64(maxw), MaxH: float64(maxh)}

	return bannerExt
}

func getMediaTypeForImpID(impID string, imps []openrtb2.Imp) openrtb_ext.BidType {
	for _, imp := range imps {
		if imp.ID == impID && imp.Banner != nil {
			return openrtb_ext.BidTypeBanner
		}
	}
	return openrtb_ext.BidTypeVideo
}

func validateVideoParams(video *openrtb2.Video) (err error) {
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
