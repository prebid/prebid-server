package consumable

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/privacy/ccpa"
)

type ConsumableAdapter struct {
	clock    instant
	endpoint string
}

type bidRequest struct {
	Placements         []placement          `json:"placements"`
	Time               int64                `json:"time"`
	NetworkId          int                  `json:"networkId,omitempty"`
	SiteId             int                  `json:"siteId"`
	UnitId             int                  `json:"unitId"`
	UnitName           string               `json:"unitName,omitempty"`
	IncludePricingData bool                 `json:"includePricingData"`
	User               user                 `json:"user,omitempty"`
	Referrer           string               `json:"referrer,omitempty"`
	Ip                 string               `json:"ip,omitempty"`
	Url                string               `json:"url,omitempty"`
	EnableBotFiltering bool                 `json:"enableBotFiltering,omitempty"`
	Parallel           bool                 `json:"parallel"`
	CCPA               string               `json:"ccpa,omitempty"`
	GDPR               *bidGdpr             `json:"gdpr,omitempty"`
	Coppa              bool                 `json:"coppa,omitempty"`
	SChain             openrtb2.SupplyChain `json:"schain"`
	Content            *openrtb2.Content    `json:"content,omitempty"`
	GPP                string               `json:"gpp,omitempty"`
	GPPSID             []int8               `json:"gpp_sid,omitempty"`
}

type placement struct {
	DivName   string `json:"divName"`
	NetworkId int    `json:"networkId,omitempty"`
	SiteId    int    `json:"siteId"`
	UnitId    int    `json:"unitId"`
	UnitName  string `json:"unitName,omitempty"`
	AdTypes   []int  `json:"adTypes"`
}

type user struct {
	Key  string         `json:"key,omitempty"`
	Eids []openrtb2.EID `json:"eids,omitempty"`
}

type bidGdpr struct {
	Applies *bool  `json:"applies,omitempty"`
	Consent string `json:"consent,omitempty"`
}

type bidResponse struct {
	Decisions map[string]decision `json:"decisions"` // map by bidId
}

/**
 * See https://dev.adzerk.com/v1.0/reference/response
 */
type decision struct {
	Pricing       *pricing   `json:"pricing"`
	AdID          int64      `json:"adId"`
	BidderName    string     `json:"bidderName,omitempty"`
	CreativeID    string     `json:"creativeId,omitempty"`
	Contents      []contents `json:"contents"`
	ImpressionUrl *string    `json:"impressionUrl,omitempty"`
	Width         uint64     `json:"width,omitempty"`  // Consumable extension, not defined by Adzerk
	Height        uint64     `json:"height,omitempty"` // Consumable extension, not defined by Adzerk
	Adomain       []string   `json:"adomain,omitempty"`
	Cats          []string   `json:"cats,omitempty"`
}

type contents struct {
	Body string `json:"body"`
}

type pricing struct {
	ClearPrice *float64 `json:"clearPrice"`
}

func (a *ConsumableAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error

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

	// Set azk cookie to one we got via sync
	if request.User != nil {
		userID := strings.TrimSpace(request.User.BuyerUID)
		if len(userID) > 0 {
			headers.Add("Cookie", fmt.Sprintf("%s=%s", "azk", userID))
		}
	}

	if request.Site != nil && request.Site.Page != "" {
		headers.Set("Referer", request.Site.Page)

		pageUrl, err := url.Parse(request.Site.Page)
		if err != nil {
			errs = append(errs, err)
		} else {
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
		body.Referrer = request.Site.Ref // Effectively the previous page to the page where the ad will be shown
		body.Url = request.Site.Page     // where the impression will be made
	}

	gdpr := bidGdpr{}

	ccpaPolicy, err := ccpa.ReadFromRequest(request)
	if err != nil {
		errs = append(errs, err)
	} else {
		body.CCPA = ccpaPolicy.Consent
	}

	if request.Regs != nil && request.Regs.Ext != nil {
		var extRegs openrtb_ext.ExtRegs
		if err := json.Unmarshal(request.Regs.Ext, &extRegs); err != nil {
			errs = append(errs, err)
		} else {
			if extRegs.GDPR != nil {
				applies := *extRegs.GDPR != 0
				gdpr.Applies = &applies
				body.GDPR = &gdpr
			}
		}
	}

	if request.User != nil && request.User.Ext != nil {
		var extUser openrtb_ext.ExtUser
		if err := json.Unmarshal(request.User.Ext, &extUser); err != nil {
			errs = append(errs, err)
		} else {
			gdpr.Consent = extUser.Consent
			body.GDPR = &gdpr

			if hasEids(extUser.Eids) {
				body.User.Eids = extUser.Eids
			}
		}
	}

	if request.Source != nil && request.Source.Ext != nil {
		var extSChain openrtb_ext.ExtRequestPrebidSChain
		if err := json.Unmarshal(request.Source.Ext, &extSChain); err != nil {
			errs = append(errs, err)
		} else {
			body.SChain = extSChain.SChain
		}
	}

	body.Coppa = request.Regs != nil && request.Regs.COPPA > 0

	if request.Regs != nil && request.Regs.GPP != "" {
		body.GPP = request.Regs.GPP
	}

	if request.Regs != nil && request.Regs.GPPSID != nil {
		body.GPPSID = request.Regs.GPPSID
	}

	if request.Site != nil && request.Site.Content != nil {
		body.Content = request.Site.Content
	} else if request.App != nil && request.App.Content != nil {
		body.Content = request.App.Content
	}

	for i, impression := range request.Imp {

		_, consumableExt, err := extractExtensions(impression)

		if err != nil {
			return nil, err
		}

		// These get set on the first one in observed working requests
		if i == 0 {
			body.NetworkId = consumableExt.NetworkId
			body.SiteId = consumableExt.SiteId
			body.UnitId = consumableExt.UnitId
			body.UnitName = consumableExt.UnitName
		}

		body.Placements[i] = placement{
			DivName:   impression.ID,
			NetworkId: consumableExt.NetworkId,
			SiteId:    consumableExt.SiteId,
			UnitId:    consumableExt.UnitId,
			UnitName:  consumableExt.UnitName,
			AdTypes:   getSizeCodes(impression.Banner.Format), // was adTypes: bid.adTypes || getSize(bid.sizes) in prebid.js
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

	return requests, errs
}

/*
internal original request in OpenRTB, external = result of us having converted it (what comes out of MakeRequests)
*/
func (a *ConsumableAdapter) MakeBids(
	internalRequest *openrtb2.BidRequest,
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

		if decision.Pricing != nil && decision.Pricing.ClearPrice != nil {
			bid := openrtb2.Bid{}
			bid.ID = internalRequest.ID
			bid.ImpID = impID
			bid.Price = *decision.Pricing.ClearPrice
			bid.AdM = retrieveAd(decision)
			bid.W = int64(decision.Width)
			bid.H = int64(decision.Height)
			bid.CrID = strconv.FormatInt(decision.AdID, 10)
			bid.Exp = 30 // TODO: Check this is intention of TTL
			bid.ADomain = decision.Adomain
			bid.Cat = decision.Cats
			// not yet ported from prebid.js adapter
			//bid.requestId = bidId;
			//bid.currency = 'USD';
			//bid.netRevenue = true;
			//bid.referrer = utils.getTopWindowUrl();

			bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
				Bid: &bid,
				// Consumable units are always HTML, never VAST.
				// From Prebid's point of view, this means that Consumable units
				// are always "banners".
				BidType: openrtb_ext.BidTypeBanner,
			})
		}
	}
	return bidderResponse, errors
}

func extractExtensions(impression openrtb2.Imp) (*adapters.ExtImpBidder, *openrtb_ext.ExtImpConsumable, []error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(impression.Ext, &bidderExt); err != nil {
		return nil, nil, []error{&errortypes.BadInput{
			Message: err.Error(),
		}}
	}

	var consumableExt openrtb_ext.ExtImpConsumable
	if err := json.Unmarshal(bidderExt.Bidder, &consumableExt); err != nil {
		return nil, nil, []error{&errortypes.BadInput{
			Message: err.Error(),
		}}
	}

	return &bidderExt, &consumableExt, nil
}

// Builder builds a new instance of the Consumable adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &ConsumableAdapter{
		clock:    realInstant{},
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func hasEids(eids []openrtb2.EID) bool {
	for i := 0; i < len(eids); i++ {
		if len(eids[i].UIDs) > 0 && eids[i].UIDs[0].ID != "" {
			return true
		}
	}
	return false
}
