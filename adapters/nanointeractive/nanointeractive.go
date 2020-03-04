package nanointeractive

import (
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
)

type NanointeractiveAdapter struct {
	endpoint string
}

// used for cookies and such
func (a *NanointeractiveAdapter) Name() string {
	return "Nano"
}

func (a *NanointeractiveAdapter) SkipNoCookies() bool {
	return false
}

func (a *NanointeractiveAdapter) MakeRequests(bidRequest *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {

	var errs []error
	var validImps []openrtb.Imp

	var adapterRequests []*adapters.RequestData

	for i := 0; i < len(bidRequest.Imp); i++ {
		err := checkImp(&bidRequest.Imp[i])
		// If the parsing is failed, remove imp and add the error.
		if err != nil {
			errs = append(errs, err)
			continue
		}
		validImps = append(validImps, bidRequest.Imp[i])
	}

	if len(validImps) == 0 {
		errs = append(errs, fmt.Errorf("no impressions in the bid request"))
		return nil, errs
	}

	// set referrer origin
	if refO := getRefererOrigin(&bidRequest.Imp[0]); refO != "" {
		if bidRequest.Site == nil {
			bidRequest.Site = &openrtb.Site{}
		}
		bidRequest.Site.Ref = refO
	}

	bidRequest.Imp = validImps

	reqJSON, err := json.Marshal(bidRequest)
	if err != nil {
		errs = append(errs, err)
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("x-openrtb-version", "2.5")
	headers.Add("User-Agent", getSafeUa(bidRequest.Device))
	headers.Add("X-Forwarded-For", getSafeIp(bidRequest.Device))
	headers.Add("Referer", getSafeReferrer(bidRequest.Site))

	// set user's cookie
	if bidRequest.User != nil && bidRequest.User.BuyerUID != "" {
		headers.Add("Cookie", "Nano="+bidRequest.User.BuyerUID)
	}

	adapterRequests = append(adapterRequests, &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    reqJSON,
		Headers: headers,
	})

	return adapterRequests, errs
}

func (a *NanointeractiveAdapter) MakeBids(
	internalRequest *openrtb.BidRequest,
	externalRequest *adapters.RequestData,
	response *adapters.ResponseData) (*adapters.BidderResponse, []error) {

	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	} else if response.StatusCode == http.StatusBadRequest {
		return nil, []error{adapters.BadInput("Invalid request.")}
	} else if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("unexpected HTTP status %d.", response.StatusCode),
		}}
	}

	var openRtbBidResponse openrtb.BidResponse

	if err := json.Unmarshal(response.Body, &openRtbBidResponse); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("bad server body response"),
		}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(openRtbBidResponse.SeatBid[0].Bid))
	bidResponse.Currency = openRtbBidResponse.Cur

	sb := openRtbBidResponse.SeatBid[0]
	for i := 0; i < len(sb.Bid); i++ {
		if !(sb.Bid[i].Price > 0) {
			continue
		}
		bid := sb.Bid[i]
		bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
			Bid:     &bid,
			BidType: openrtb_ext.BidTypeBanner,
		})
	}
	return bidResponse, nil
}

func checkImp(imp *openrtb.Imp) error {
	// We support only banner impression
	if imp.Banner == nil {
		return fmt.Errorf("invalid MediaType. NanoInteractive only supports Banner type. ImpID=%s", imp.ID)
	}

	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return fmt.Errorf("ext not provided; ImpID=%s", imp.ID)
	}

	var nanoExt openrtb_ext.ExtImpNanoInteractive
	if err := json.Unmarshal(bidderExt.Bidder, &nanoExt); err != nil {
		return fmt.Errorf("ext.bidder not provided; ImpID=%s", imp.ID)
	}

	if nanoExt.Pid == "" {
		return fmt.Errorf("pid is empty; ImpID=%s", imp.ID)
	}

	return nil
}

func NewNanoIneractiveBidder(endpoint string) *NanointeractiveAdapter {
	return &NanointeractiveAdapter{
		endpoint: endpoint,
	}
}

func getSafeIp(device *openrtb.Device) string {
	if device == nil {
		return ""
	}
	return device.IP
}

func getSafeUa(device *openrtb.Device) string {
	if device == nil {
		return ""
	}
	return device.UA
}

func getSafeReferrer(site *openrtb.Site) string {
	if site == nil {
		return ""
	}
	return site.Page
}

func NewNanoInteractiveAdapter(uri string) *NanointeractiveAdapter {
	return &NanointeractiveAdapter{
		endpoint: uri,
	}
}

func getRefererOrigin(imp *openrtb.Imp) string {

	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return ""
	}

	var nanoExt openrtb_ext.ExtImpNanoInteractive
	if err := json.Unmarshal(bidderExt.Bidder, &nanoExt); err != nil {
		return ""
	}

	if nanoExt.Ref != "" {
		return string(nanoExt.Ref)
	}

	return ""
}
