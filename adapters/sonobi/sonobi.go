package sonobi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/adapters"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/errortypes"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

// SonobiAdapter - Sonobi SonobiAdapter definition
type SonobiAdapter struct {
	URI string
}

// Builder builds a new instance of the Sonobi adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &SonobiAdapter{
		URI: config.Endpoint,
	}
	return bidder, nil
}

// MakeRequests Makes the OpenRTB request payload
func (a *SonobiAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	var sonobiExt openrtb_ext.ExtImpSonobi
	var err error

	var adapterRequests []*adapters.RequestData

	// Sonobi currently only supports 1 imp per request to sonobi.
	// Loop over the imps from the initial bid request to form many adapter requests to sonobi with only 1 imp.
	for _, imp := range request.Imp {
		// Make a copy as we don't want to change the original request
		reqCopy := *request
		reqCopy.Imp = append(make([]openrtb2.Imp, 0, 1), imp)

		var bidderExt adapters.ExtImpBidder
		if err = json.Unmarshal(reqCopy.Imp[0].Ext, &bidderExt); err != nil {
			errs = append(errs, err)
			continue
		}

		if err = json.Unmarshal(bidderExt.Bidder, &sonobiExt); err != nil {
			errs = append(errs, err)
			continue
		}

		reqCopy.Imp[0].TagID = sonobiExt.TagID

		// If the bid floor currency is not USD, do the conversion to USD
		if reqCopy.Imp[0].BidFloor > 0 && reqCopy.Imp[0].BidFloorCur != "" && strings.ToUpper(reqCopy.Imp[0].BidFloorCur) != "USD" {

			// Convert to US dollars
			convertedValue, err := reqInfo.ConvertCurrency(reqCopy.Imp[0].BidFloor, reqCopy.Imp[0].BidFloorCur, "USD")
			if err != nil {
				return nil, []error{err}
			}

			// Update after conversion. All imp elements inside request.Imp are shallow copies
			// therefore, their non-pointer values are not shared memory and are safe to modify.
			reqCopy.Imp[0].BidFloorCur = "USD"
			reqCopy.Imp[0].BidFloor = convertedValue
		}

		// Sonobi only bids in USD
		reqCopy.Cur = append(make([]string, 0, 1), "USD")

		adapterReq, errors := a.makeRequest(&reqCopy)
		if adapterReq != nil {
			adapterRequests = append(adapterRequests, adapterReq)
		}
		errs = append(errs, errors...)
	}

	return adapterRequests, errs

}

// makeRequest helper method to crete the http request data
func (a *SonobiAdapter) makeRequest(request *openrtb2.BidRequest) (*adapters.RequestData, []error) {

	var errs []error

	reqJSON, err := json.Marshal(request)

	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	return &adapters.RequestData{
		Method:  "POST",
		Uri:     a.URI,
		Body:    reqJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}, errs
}

// MakeBids makes the bids
func (a *SonobiAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var errs []error

	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var bidResp openrtb2.BidResponse

	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)
	bidResponse.Currency = "USD" // Sonobi only bids in USD

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bid := sb.Bid[i]
			bidType, err := getMediaTypeForImp(bid.ImpID, internalRequest.Imp)
			//bidType, err := getBidType(bid)
			if err != nil {
				return nil, []error{err}
			}
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: bidType,
			})
		}
	}
	return bidResponse, errs
}

func getMediaTypeForImp(impID string, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	mediaType := openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == impID {
			if imp.Banner == nil && imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			}
			if imp.Banner == nil && imp.Video == nil && imp.Native != nil {
				mediaType = openrtb_ext.BidTypeNative
			}
			return mediaType, nil
		}
	}

	// This shouldnt happen. Lets handle it just incase by returning an error.
	return "", &errortypes.BadInput{
		Message: fmt.Sprintf("Failed to find impression \"%s\" ", impID),
	}
}
func getBidType(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	// determinate media type by bid response field mtype
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupAudio:
		return openrtb_ext.BidTypeAudio, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	}

	return "", &errortypes.BadInput{
		Message: fmt.Sprintf("Could not define media type for impression: %s", bid.ImpID),
	}
}
