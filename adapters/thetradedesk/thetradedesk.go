package thetradedesk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/prebid/prebid-server/v2/adapters"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/errortypes"
	"github.com/prebid/prebid-server/v2/openrtb_ext"

	"github.com/prebid/openrtb/v20/openrtb2"
)

type adapter struct {
	URI        string
	bidderName string
}

type ExtImpBidderTheTradeDesk struct {
	adapters.ExtImpBidder
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	pubID, err := getPublisherId(request.Imp)

	if err != nil {
		return nil, []error{err}
	}

	for i := 0; i < len(request.Imp); i++ {
		imp := request.Imp[i]

		if imp.Banner != nil {
			if len(imp.Banner.Format) > 0 {
				firstFormat := imp.Banner.Format[0]
				imp.Banner.H = &firstFormat.H
				imp.Banner.W = &firstFormat.W

			}
		}
	}

	if request.Site != nil {
		siteCopy := *request.Site
		if siteCopy.Publisher != nil {
			publisherCopy := *siteCopy.Publisher
			publisherCopy.ID = pubID
			siteCopy.Publisher = &publisherCopy
		} else {
			siteCopy.Publisher = &openrtb2.Publisher{ID: pubID}
		}
		request.Site = &siteCopy
	} else if request.App != nil {
		appCopy := *request.App
		if appCopy.Publisher != nil {
			publisherCopy := *appCopy.Publisher
			publisherCopy.ID = pubID
			appCopy.Publisher = &publisherCopy
		} else {
			appCopy.Publisher = &openrtb2.Publisher{ID: pubID}
		}
		request.App = &appCopy
	}

	var bidderEndpoint = strings.TrimRight(a.URI, "/") + "/" + a.bidderName

	errs := make([]error, 0, len(request.Imp))
	reqJSON, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	PREBID_INTEGRATION_TYPE := "1"
	headers.Add("x-integration-type", PREBID_INTEGRATION_TYPE)
	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     bidderEndpoint,
		Body:    reqJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}}, errs
}

func getPublisherId(impressions []openrtb2.Imp) (string, error) {
	for i := 0; i < len(impressions); i++ {
		var imp = &impressions[i]

		var bidderExt ExtImpBidderTheTradeDesk
		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			return "", err
		}

		var ttdExt openrtb_ext.ExtImpTheTradeDesk
		if err := json.Unmarshal(bidderExt.Bidder, &ttdExt); err != nil {
			return "", err
		}

		if ttdExt.PublisherId != "" {
			return ttdExt.PublisherId, nil
		}
	}
	return "", nil
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		err := &errortypes.BadInput{
			Message: "Unexpected status code: 400. Run with request.debug = 1 for more info.",
		}
		return nil, []error{err}
	}

	if response.StatusCode == http.StatusNoContent {
		return adapters.NewBidderResponse(), nil
	}

	if response.StatusCode != http.StatusOK {
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info.", response.StatusCode),
		}
		return nil, []error{err}
	}

	var bidResponse openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &bidResponse); err != nil {
		return nil, []error{err}
	}

	bidderResponse := adapters.NewBidderResponse()
	bidderResponse.Currency = bidResponse.Cur

	for _, seatBid := range bidResponse.SeatBid {
		for _, bid := range seatBid.Bid {
			bid := bid

			bidType, err := getBidType(bid.MType)

			if err != nil {
				return nil, []error{err}
			}

			b := &adapters.TypedBid{
				Bid:     &bid,
				BidType: bidType,
			}
			bidderResponse.Bids = append(bidderResponse.Bids, b)
		}
	}

	return bidderResponse, nil
}

func getBidType(markupType openrtb2.MarkupType) (openrtb_ext.BidType, error) {
	switch markupType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupAudio:
		return openrtb_ext.BidTypeAudio, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	default:
		return "", fmt.Errorf("unsupported mtype: %d", markupType)
	}
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	return &adapter{
		URI:        config.Endpoint,
		bidderName: config.ExtraAdapterInfo,
	}, nil
}
