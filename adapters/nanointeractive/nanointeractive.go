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

type NanoInteractiveAdapter struct {
	endpoint string
}

func (a *NanoInteractiveAdapter) Name() string {
	return "Nano"
}

func (a *NanoInteractiveAdapter) SkipNoCookies() bool {
	return false
}

func (a *NanoInteractiveAdapter) MakeRequests(bidRequest *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {

	var errs []error
	var validImps []openrtb.Imp

	var adapterRequests []*adapters.RequestData
	var referer string = ""

	for i := 0; i < len(bidRequest.Imp); i++ {

		ref, err := checkImp(&bidRequest.Imp[i])

		// If the parsing is failed, remove imp and add the error.
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if referer == "" && ref != "" {
			referer = ref
		}
		validImps = append(validImps, bidRequest.Imp[i])
	}

	if len(validImps) == 0 {
		errs = append(errs, fmt.Errorf("no impressions in the bid request"))
		return nil, errs
	}

	// set referer origin
	if referer != "" {
		if bidRequest.Site == nil {
			bidRequest.Site = &openrtb.Site{}
		}
		bidRequest.Site.Ref = referer
	}

	bidRequest.Imp = validImps

	reqJSON, err := json.Marshal(bidRequest)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("x-openrtb-version", "2.5")
	if bidRequest.Device != nil {
		headers.Add("User-Agent", bidRequest.Device.UA)
		headers.Add("X-Forwarded-For", bidRequest.Device.IP)
	}
	if bidRequest.Site != nil {
		headers.Add("Referer", bidRequest.Site.Page)
	}

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

func (a *NanoInteractiveAdapter) MakeBids(
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

func checkImp(imp *openrtb.Imp) (string, error) {
	// We support only banner impression
	if imp.Banner == nil {
		return "", fmt.Errorf("invalid MediaType. NanoInteractive only supports Banner type. ImpID=%s", imp.ID)
	}

	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return "", fmt.Errorf("ext not provided; ImpID=%s", imp.ID)
	}

	var nanoExt openrtb_ext.ExtImpNanoInteractive
	if err := json.Unmarshal(bidderExt.Bidder, &nanoExt); err != nil {
		return "", fmt.Errorf("ext.bidder not provided; ImpID=%s", imp.ID)
	}
	if nanoExt.Pid == "" {
		return "", fmt.Errorf("pid is empty; ImpID=%s", imp.ID)
	}

	if nanoExt.Ref != "" {
		return string(nanoExt.Ref), nil
	}

	return "", nil
}

func NewNanoIneractiveBidder(endpoint string) *NanoInteractiveAdapter {
	return &NanoInteractiveAdapter{
		endpoint: endpoint,
	}
}

func NewNanoInteractiveAdapter(uri string) *NanoInteractiveAdapter {
	return &NanoInteractiveAdapter{
		endpoint: uri,
	}
}
