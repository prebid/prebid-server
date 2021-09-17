package impactify

import (
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
	"strings"
)

type ImpactifyAdapter struct {
	endpoint string
}

type ImpactifyExtBidder struct {
	Impactify openrtb_ext.ExtImpImpactify `json:"impactify"`
}

type DefaultExtBidder struct {
	Bidder openrtb_ext.ExtImpImpactify `json:"bidder"`
}

func (a *ImpactifyAdapter) MakeRequests(bidRequest *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	var validImps []openrtb2.Imp

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
		// Check if imp comes with bid floor amount defined in a foreign currency
		if bidRequest.Imp[i].BidFloor > 0 && bidRequest.Imp[i].BidFloorCur != "" && strings.ToUpper(bidRequest.Imp[i].BidFloorCur) != "USD" {
			// Convert to US dollars
			convertedValue, err := reqInfo.ConvertCurrency(bidRequest.Imp[i].BidFloor, bidRequest.Imp[i].BidFloorCur, "USD")
			if err != nil {
				return nil, []error{err}
			}
			bidRequest.Imp[i].BidFloorCur = "USD"
			bidRequest.Imp[i].BidFloor = convertedValue
		}

		var impactifyExt ImpactifyExtBidder
		var defaultExt DefaultExtBidder
		json.Unmarshal(bidRequest.Imp[i].Ext, &defaultExt)
		impactifyExt.Impactify = defaultExt.Bidder
		bidRequest.Imp[i].Ext, _ = json.Marshal(impactifyExt)

		validImps = append(validImps, bidRequest.Imp[i])
	}

	if len(validImps) == 0 {
		if errs != nil {
			return nil, errs
		} else {
			return nil, []error{&errortypes.BadInput{
				Message: "No impressions in the bid request",
			}}
		}
	}

	// Set referer origin
	if referer != "" {
		if bidRequest.Site == nil {
			bidRequest.Site = &openrtb2.Site{}
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
		headers.Add("Cookie", "uids="+bidRequest.User.BuyerUID)
	}

	adapterRequests = append(adapterRequests, &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    reqJSON,
		Headers: headers,
	})

	return adapterRequests, errs
}

func (a *ImpactifyAdapter) MakeBids(
	internalRequest *openrtb2.BidRequest,
	externalRequest *adapters.RequestData,
	response *adapters.ResponseData) (*adapters.BidderResponse, []error) {

	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	} else if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: "Invalid request.",
		}}
	} else if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("unexpected HTTP status %d.", response.StatusCode),
		}}
	}

	var openRtbBidResponse openrtb2.BidResponse

	if err := json.Unmarshal(response.Body, &openRtbBidResponse); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "bad server body response",
		}}
	}

	if len(openRtbBidResponse.SeatBid) == 0 {
		return nil, nil
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
			BidType: getMediaTypeForImp(bid.ImpID, internalRequest.Imp),
		})
	}
	return bidResponse, nil
}

func getMediaTypeForImp(impID string, imps []openrtb2.Imp) openrtb_ext.BidType {
	for _, imp := range imps {
		if imp.ID == impID {
			if imp.Banner != nil {
				return openrtb_ext.BidTypeBanner
			} else if imp.Video != nil {
				return openrtb_ext.BidTypeVideo
			} else if imp.Native != nil {
				return openrtb_ext.BidTypeNative
			}
		}
	}
	return openrtb_ext.BidTypeBanner
}

func checkImp(imp *openrtb2.Imp) (string, error) {
	// We support only video or banner impression
	if imp.Video == nil && imp.Banner == nil {
		return "", fmt.Errorf("No valid type imp")
	}

	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return "", fmt.Errorf("Invalid impactify ext")
	}

	var impactifyExt openrtb_ext.ExtImpImpactify
	if err := json.Unmarshal(bidderExt.Bidder, &impactifyExt); err != nil {
		return "", fmt.Errorf("ext.bidder not provided")
	}
	if impactifyExt.AppID == "" {
		return "", fmt.Errorf("appId parameter is empty")
	}
	if impactifyExt.Style == "" {
		return "", fmt.Errorf("style parameter is empty")
	}
	if impactifyExt.Format == "" {
		return "", fmt.Errorf("format parameter is empty")
	}

	return "", nil
}

// Builder builds a new instance of the Impactify adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &ImpactifyAdapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}
