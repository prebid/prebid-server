package impactify

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"

	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type adapter struct {
	endpoint string
}

type ImpactifyExtBidder struct {
	Impactify openrtb_ext.ExtImpImpactify `json:"impactify"`
}

type DefaultExtBidder struct {
	Bidder openrtb_ext.ExtImpImpactify `json:"bidder"`
}

func (a *adapter) MakeRequests(bidRequest *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var adapterRequests []*adapters.RequestData

	for i := 0; i < len(bidRequest.Imp); i++ {
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

		// Set the CUR of bid to USD after converting all floors
		bidRequest.Cur = []string{"USD"}

		var impactifyExt ImpactifyExtBidder

		var defaultExt DefaultExtBidder
		err := jsonutil.Unmarshal(bidRequest.Imp[i].Ext, &defaultExt)
		if err != nil {
			return nil, []error{&errortypes.BadInput{
				Message: fmt.Sprintf("Unable to decode the imp ext : \"%s\"", bidRequest.Imp[i].ID),
			}}
		}

		impactifyExt.Impactify = defaultExt.Bidder
		bidRequest.Imp[i].Ext, err = json.Marshal(impactifyExt)
		if err != nil {
			return nil, []error{&errortypes.BadInput{
				Message: fmt.Sprintf("Unable to decode the imp ext : \"%s\"", bidRequest.Imp[i].ID),
			}}
		}
	}

	if len(bidRequest.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{
			Message: "No valid impressions in the bid request",
		}}
	}

	reqJSON, err := json.Marshal(bidRequest)
	if err != nil {
		return nil, []error{err}
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("x-openrtb-version", "2.5")
	if bidRequest.Device != nil {
		if bidRequest.Device.UA != "" {
			headers.Add("User-Agent", bidRequest.Device.UA)
		}
		// Add IPv4 or IPv6 if available
		if bidRequest.Device.IP != "" {
			headers.Add("X-Forwarded-For", bidRequest.Device.IP)
		} else if bidRequest.Device.IPv6 != "" {
			headers.Add("X-Forwarded-For", bidRequest.Device.IPv6)
		}
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
		ImpIDs:  openrtb_ext.GetImpIDs(bidRequest.Imp),
	})

	return adapterRequests, nil
}

func (a *adapter) MakeBids(
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
			Message: fmt.Sprintf("Unexpected HTTP status %d.", response.StatusCode),
		}}
	}

	var openRtbBidResponse openrtb2.BidResponse

	if err := jsonutil.Unmarshal(response.Body, &openRtbBidResponse); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Bad server body response",
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

		impMediaType, err := getMediaTypeForImp(sb.Bid[i].ImpID, internalRequest.Imp)
		if err != nil {
			return nil, []error{err}
		}

		bid := sb.Bid[i]
		bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
			Bid:     &bid,
			BidType: impMediaType,
		})
	}
	return bidResponse, nil
}

func getMediaTypeForImp(impID string, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	for _, imp := range imps {
		if imp.ID == impID {
			if imp.Banner != nil {
				return openrtb_ext.BidTypeBanner, nil
			} else if imp.Video != nil {
				return openrtb_ext.BidTypeVideo, nil
			}
		}
	}

	return "", &errortypes.BadInput{
		Message: fmt.Sprintf("Failed to find a supported media type impression \"%s\"", impID),
	}
}

// Builder builds a new instance of the Impactify adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}
