package consumable

import (
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
	"net/url"
	"time"
)

type ConsumableAdapter struct {
	clock instant
}

type bidRequest struct {
	Placements         []placement `json:"placements"`
	Time               int64       `json:"time"`
	IncludePricingData bool        `json:"includePricingData"`
	User               user        `json:"user,omitempty"`
	Referrer           string      `json:"referrer,omitempty"`
	Ip                 string      `json:"ip,omitempty"`
	Url                string      `json:"url,omitempty"`
	EnableBotFiltering bool        `json:"enableBotFiltering,omitempty"`
}

type placement struct {
	DivName   string `json:"divName"`
	NetworkId int    `json:"networkId"`
	SiteId    int    `json:"siteId"`
	AdTypes   []int  `json:"adTypes"`
	ZoneIds   []int  `json:"zoneIds,omitempty"`
}

type user struct {
	Key string `json:"key,omitempty"`
}

func (a *ConsumableAdapter) MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {
	headers := http.Header{
		"Content-Type": {"application/json"},
		"Accept":       {"application/json"},
	}

	if request.Device != nil {
		if request.Device.UA != "" {
			headers.Set("User-Agent", request.Device.UA)
		}

		if request.Device.IP != "" {
			headers.Set("Forwarded", "for="+request.Device.IP)
			headers.Set("X-Forwarded-For", request.Device.IP)
		}
	}

	if request.Site != nil && request.Site.Page != "" {
		headers.Set("Referer", request.Site.Page)

		pageUrl, err := url.Parse(request.Site.Page)
		if err == nil {
			origin := url.URL{
				Scheme: pageUrl.Scheme,
				Opaque: pageUrl.Opaque,
				Host:   pageUrl.Host,
			}

			headers.Set("Origin", origin.String())
		}
	}

	body := bidRequest{
		Time:               time.Now().Unix(),
		IncludePricingData: true,
		EnableBotFiltering: true,
	}

	if request.Site != nil {
		body.Referrer = request.Site.Ref
		body.Url = request.Site.Page
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, []error{err}
	}

	requests := []*adapters.RequestData{
		{
			Method:  "POST",
			Uri:     "https://e.serverbid.com/api/v2",
			Body:    bodyBytes,
			Headers: headers,
		},
	}

	return requests, nil
}

/*
internal original request in OpenRTB, external = result of us having converted it (what comes out of MakeRequests)
*/
func (a *ConsumableAdapter) MakeBids(
	internalRequest *openrtb.BidRequest,
	externalRequest *adapters.RequestData,
	response *adapters.ResponseData,
) (*adapters.BidderResponse, []error) {

	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var serverResponse openrtb.BidResponse // response from Consumable
	if err := json.Unmarshal(response.Body, &serverResponse); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("error while decoding response, err: %s", err),
		}}
	}

	bidderResponse := adapters.NewBidderResponse()
	var errors []error

	for _, sb := range serverResponse.SeatBid {
		for _, bid := range sb.Bid {

			imp := getImp(bid.ImpID, internalRequest.Imp)
			if imp == nil {
				errors = append(errors, &errortypes.BadServerResponse{
					Message: fmt.Sprintf("ignoring bid id=%s, request doesn't contain any impression with id=%s", bid.ID, bid.ImpID),
				})
				continue
			}

			if bid.Price != 0 {
				bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
					Bid:     &bid,
					BidType: getMediaTypeForImp(getImp(bid.ImpID, internalRequest.Imp)),
				})
			}
		}
	}

	/* This is what we're working towards.
	bids = bidRequest.bidRequest;

	for (let i = 0; i < bids.length; i++) {
		bid = {};
		bidObj = bids[i];
		bidId = bidObj.bidId;

		const decision = serverResponse.decisions && serverResponse.decisions[bidId];
		const price = decision && decision.pricing && decision.pricing.clearPrice;

		if (decision && price) {
			bid.requestId = bidId;
			bid.cpm = price;
			bid.width = decision.width;
			bid.height = decision.height;
			bid.unitId = bidObj.params.unitId;  // not used when sending to consumable end (but will get from
			bid.unitName = bidObj.params.unitName;
			bid.ad = retrieveAd(decision, bid.unitId, bid.unitName);
			bid.currency = 'USD';
			bid.creativeId = decision.adId;
			bid.ttl = 30;
			bid.netRevenue = true;
			bid.referrer = utils.getTopWindowUrl();

			bidResponses.push(bid);
		}
	}
	*/
	return bidderResponse, errors
}

func getImp(impId string, imps []openrtb.Imp) *openrtb.Imp {
	for _, imp := range imps {
		if imp.ID == impId {
			return &imp
		}
	}
	return nil
}

func getMediaTypeForImp(imp *openrtb.Imp) openrtb_ext.BidType {
	// TODO: Whatever logic we need here possibly as follows - may always be Video when we bid
	if imp.Banner != nil {
		return openrtb_ext.BidTypeBanner
	} else if imp.Video != nil {
		return openrtb_ext.BidTypeVideo
	}
	return openrtb_ext.BidTypeVideo
}

func testConsumableBidder(testClock instant) *ConsumableAdapter {
	return &ConsumableAdapter{testClock}
}

func NewConsumableBidder() *ConsumableAdapter {
	return &ConsumableAdapter{realInstant{}}
}
