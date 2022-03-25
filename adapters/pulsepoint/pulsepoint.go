package pulsepoint

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
)

type PulsePointAdapter struct {
	URI string
}

// Builds an instance of PulsePointAdapter
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &PulsePointAdapter{
		URI: config.Endpoint,
	}
	return bidder, nil
}

func (a *PulsePointAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	errs := make([]error, 0, len(request.Imp))

	var err error
	pubID := ""
	imps := make([]openrtb2.Imp, 0, len(request.Imp))
	for i := 0; i < len(request.Imp); i++ {
		imp := request.Imp[i]
		var bidderExt adapters.ExtImpBidder
		if err = json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}
		var pulsepointExt openrtb_ext.ExtImpPulsePoint
		if err = json.Unmarshal(bidderExt.Bidder, &pulsepointExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}
		// parse pubid and keep it for reference
		if pubID == "" && pulsepointExt.PubID > 0 {
			pubID = strconv.Itoa(pulsepointExt.PubID)
		}
		// tag id to be sent
		imp.TagID = strconv.Itoa(pulsepointExt.TagID)
		imps = append(imps, imp)
	}

	// verify there are valid impressions
	if len(imps) == 0 {
		return nil, errs
	}

	// add the publisher id from ext to the site.pub.id or app.pub.id
	if request.Site != nil {
		site := *request.Site
		if site.Publisher != nil {
			publisher := *site.Publisher
			publisher.ID = pubID
			site.Publisher = &publisher
		} else {
			site.Publisher = &openrtb2.Publisher{ID: pubID}
		}
		request.Site = &site
	} else if request.App != nil {
		app := *request.App
		if app.Publisher != nil {
			publisher := *app.Publisher
			publisher.ID = pubID
			app.Publisher = &publisher
		} else {
			app.Publisher = &openrtb2.Publisher{ID: pubID}
		}
		request.App = &app
	}

	request.Imp = imps
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
		Uri:     a.URI,
		Body:    reqJSON,
		Headers: headers,
	}}, errs
}

func (a *PulsePointAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	// passback
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	// bad requests
	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Bad user input: HTTP status %d", response.StatusCode),
		}}
	}
	// error
	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Bad server response: HTTP status %d", response.StatusCode),
		}}
	}
	// parse response
	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)
	// map imps by id
	impsByID := make(map[string]openrtb2.Imp)
	for i := 0; i < len(internalRequest.Imp); i++ {
		impsByID[internalRequest.Imp[i].ID] = internalRequest.Imp[i]
	}

	var errs []error
	for _, sb := range bidResp.SeatBid {
		for i := 0; i < len(sb.Bid); i++ {
			bid := sb.Bid[i]
			imp := impsByID[bid.ImpID]
			bidType := getBidType(imp)
			if &imp != nil && bidType != "" {
				bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
					Bid:     &bid,
					BidType: bidType,
				})
			}
		}
	}
	return bidResponse, errs
}

func getBidType(imp openrtb2.Imp) openrtb_ext.BidType {
	// derive the bidtype purely from the impression itself
	if imp.Banner != nil {
		return openrtb_ext.BidTypeBanner
	} else if imp.Video != nil {
		return openrtb_ext.BidTypeVideo
	} else if imp.Audio != nil {
		return openrtb_ext.BidTypeAudio
	} else if imp.Native != nil {
		return openrtb_ext.BidTypeNative
	}
	return ""
}
