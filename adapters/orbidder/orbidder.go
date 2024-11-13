package orbidder

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

type OrbidderAdapter struct {
	endpoint string
}

// MakeRequests makes the HTTP requests which should be made to fetch bids from orbidder.
func (rcv *OrbidderAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	validImps, errs := getValidImpressions(request, reqInfo)
	if len(validImps) == 0 {
		return nil, errs
	}

	request.Imp = validImps

	requestBodyJSON, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	return []*adapters.RequestData{{
		Method:  http.MethodPost,
		Uri:     rcv.endpoint,
		Body:    requestBodyJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}}, errs
}

// getValidImpressions validate imps and check for bid floor currency. Convert to EUR if necessary
func getValidImpressions(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]openrtb2.Imp, []error) {
	var errs []error
	var validImps []openrtb2.Imp

	for _, imp := range request.Imp {
		if err := preprocessBidFloorCurrency(&imp, reqInfo); err != nil {
			errs = append(errs, err)
			continue
		}

		if err := preprocessExtensions(&imp); err != nil {
			errs = append(errs, err)
			continue
		}
		validImps = append(validImps, imp)
	}
	return validImps, errs
}

func preprocessExtensions(imp *openrtb2.Imp) error {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	var orbidderExt openrtb_ext.ExtImpOrbidder
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &orbidderExt); err != nil {
		return &errortypes.BadInput{
			Message: "Wrong orbidder bidder ext: " + err.Error(),
		}
	}

	return nil
}

func preprocessBidFloorCurrency(imp *openrtb2.Imp, reqInfo *adapters.ExtraRequestInfo) error {
	// we expect every currency related data to be EUR
	if imp.BidFloor > 0 && strings.ToUpper(imp.BidFloorCur) != "EUR" && imp.BidFloorCur != "" {
		if convertedValue, err := reqInfo.ConvertCurrency(imp.BidFloor, imp.BidFloorCur, "EUR"); err != nil {
			return err
		} else {
			imp.BidFloor = convertedValue
		}
	}
	imp.BidFloorCur = "EUR"
	return nil
}

// MakeBids unpacks server response into Bids.
func (rcv OrbidderAdapter) MakeBids(_ *openrtb2.BidRequest, _ *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode >= http.StatusInternalServerError {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Dsp server internal error.", response.StatusCode),
		}}
	}

	if response.StatusCode >= http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Bad request to dsp.", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Bad response from dsp.", response.StatusCode),
		}}
	}

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	var bidErrs []error
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)
	for _, seatBid := range bidResp.SeatBid {
		for i := range seatBid.Bid {
			// later we have to add the bid as a pointer,
			// because of this we need a variable that only exists at this loop iteration.
			// otherwise there will be issues with multibid and pointer behavior.
			bid := seatBid.Bid[i]
			bidType, err := getBidType(bid)
			if err != nil {
				// could not determinate media type, append an error and continue with the next bid.
				bidErrs = append(bidErrs, err)
				continue
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: bidType,
			})
		}
	}
	if bidResp.Cur != "" {
		bidResponse.Currency = bidResp.Cur
	}

	return bidResponse, bidErrs
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

// Builder builds a new instance of the Orbidder adapter for the given bidder with the given config.
func Builder(_ openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &OrbidderAdapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}
