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
	Parallel           bool        `json:"parallel"`
}

type placement struct {
	DivName   string `json:"divName"`
	NetworkId int    `json:"networkId"`
	SiteId    int    `json:"siteId"`
	UnitId    int    `json:"unitId"`
	UnitName  string `json:"unitName"`
	AdTypes   []int  `json:"adTypes"`
}

type user struct {
	Key string `json:"key,omitempty"`
}

type bidResponse struct {
	Decisions map[string]decision `json:"decisions"` // map by bidId
}

type decision struct {
	Pricing *pricing `json:"pricing"`
	AdID    *string  `json:"adId"`
}

type pricing struct {
	ClearPrice *float64 `json:"clearPrice"`
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
		Placements:         make([]placement, len(request.Imp)),
		Time:               a.clock.Now().Unix(),
		IncludePricingData: true,
		EnableBotFiltering: true,
		Parallel:           true,
	}

	if request.Site != nil {
		body.Referrer = request.Site.Ref
		body.Url = request.Site.Page
	}

	for i, impression := range request.Imp {

		var bidderExt adapters.ExtImpBidder
		if err := json.Unmarshal(impression.Ext, &bidderExt); err != nil {
			fmt.Printf("err: %s", err) // TODO: DELETE
			return nil, []error{&errortypes.BadInput{
				Message: err.Error(),
			}}
		}

		var consumableExt openrtb_ext.ExtImpConsumable
		if err := json.Unmarshal(bidderExt.Bidder, &consumableExt); err != nil {
			fmt.Printf("err: %s", err) // TODO: DELETE
			return nil, []error{&errortypes.BadInput{
				Message: err.Error(),
			}}
		}

		body.Placements[i] = placement{
			DivName:   impression.ID,
			NetworkId: consumableExt.NetworkId,
			SiteId:    consumableExt.SiteId,
			UnitId:    consumableExt.UnitId,
			UnitName:  consumableExt.UnitName,
			AdTypes:   getSizeCodes(impression.Banner.Format),
		}
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

	var serverResponse bidResponse // response from Consumable
	if err := json.Unmarshal(response.Body, &serverResponse); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("error while decoding response, err: %s", err),
		}}
	}

	bidderResponse := adapters.NewBidderResponse()
	var errors []error

	for impID, decision := range serverResponse.Decisions {
		println("ImpID: ", impID, " Decision: ", *decision.Pricing.ClearPrice)
		imp := getImp(impID, internalRequest.Imp)
		if imp == nil {
			errors = append(errors, &errortypes.BadServerResponse{
				Message: fmt.Sprintf("ignoring bid id=%s, request doesn't contain any impression with id=%s", "TODO: bid.ID", impID),
			})
			fmt.Printf("%s", errors[0])
			continue
		}

		if decision.Pricing != nil && decision.Pricing.ClearPrice != nil {

			bid := openrtb.Bid{}
			bid.ImpID = impID
			bid.Price = *decision.Pricing.ClearPrice
			bid.AdM = "the ad markup"
			bid.W = imp.Banner.Format[0].W // TODO: Review to check if this is correct behaviour
			bid.H = imp.Banner.Format[0].H
			bid.CrID = *decision.AdID // creative id ... to assist with quality checking
			bid.Exp = 30              // TODO: Check this is intention of TTL

			bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: getMediaTypeForImp(getImp(bid.ImpID, internalRequest.Imp)),
			})
		}
	}

	/* This is what we're working towards.
	bids = bidRequest.bidRequest;

	for (let i = 0; i < bids.length; i++) {
		bid = {};
		bidObj = bids[i];
		bidId = bidObj.bidId;


		if (decision && price) {
			bid.requestId = bidId;
			bid.cpm = price;
			bid.ad = retrieveAd(decision, bid.unitId, bid.unitName);
			bid.currency = 'USD';
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
