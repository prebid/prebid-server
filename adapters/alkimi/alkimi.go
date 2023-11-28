package alkimi

import (
	"encoding/json"
	"fmt"
	"github.com/prebid/prebid-server/v2/errortypes"
	"github.com/prebid/prebid-server/v2/floors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/v2/adapters"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

const price_macro = "${AUCTION_PRICE}"

type adapter struct {
	endpoint string
}

type extObj struct {
	AlkimiBidderExt openrtb_ext.ExtImpAlkimi `json:"bidder"`
}

// Builder builds a new instance of the Alkimi adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	endpointURL, err := url.Parse(config.Endpoint)
	if err != nil || len(endpointURL.String()) == 0 {
		return nil, fmt.Errorf("invalid endpoint: %v", err)
	}

	bidder := &adapter{
		endpoint: endpointURL.String(),
	}
	return bidder, nil
}

// MakeRequests creates Alkimi adapter requests
func (adapter *adapter) MakeRequests(request *openrtb2.BidRequest, req *adapters.ExtraRequestInfo) (reqsBidder []*adapters.RequestData, errs []error) {
	reqCopy := *request

	updated, errs := updateImps(reqCopy)
	if len(errs) > 0 || len(reqCopy.Imp) != len(updated) {
		return nil, errs
	}

	reqCopy.Imp = updated
	encoded, err := json.Marshal(reqCopy)
	if err != nil {
		errs = append(errs, err)
	} else {
		reqBidder := buildBidderRequest(adapter, encoded)
		reqsBidder = append(reqsBidder, reqBidder)
	}
	return
}

func updateImps(bidRequest openrtb2.BidRequest) ([]openrtb2.Imp, []error) {
	var errs []error

	updatedImps := make([]openrtb2.Imp, 0, len(bidRequest.Imp))
	for _, imp := range bidRequest.Imp {

		var bidderExt adapters.ExtImpBidder
		var extImpAlkimi openrtb_ext.ExtImpAlkimi

		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errs = append(errs, err)
			continue
		}

		if err := json.Unmarshal(bidderExt.Bidder, &extImpAlkimi); err != nil {
			errs = append(errs, err)
			continue
		}

		var bidFloorPrice floors.Price
		bidFloorPrice.FloorMinCur = imp.BidFloorCur
		bidFloorPrice.FloorMin = imp.BidFloor

		if len(bidFloorPrice.FloorMinCur) > 0 && bidFloorPrice.FloorMin > 0 {
			imp.BidFloor = bidFloorPrice.FloorMin
		} else {
			imp.BidFloor = extImpAlkimi.BidFloor
		}
		imp.Instl = extImpAlkimi.Instl
		imp.Exp = extImpAlkimi.Exp

		temp := extObj{AlkimiBidderExt: extImpAlkimi}
		temp.AlkimiBidderExt.AdUnitCode = imp.ID

		extJson, err := json.Marshal(temp)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		imp.Ext = extJson
		updatedImps = append(updatedImps, imp)
	}

	return updatedImps, errs
}

func buildBidderRequest(adapter *adapter, encoded []byte) *adapters.RequestData {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	reqBidder := &adapters.RequestData{
		Method:  "POST",
		Uri:     adapter.endpoint,
		Body:    encoded,
		Headers: headers,
	}
	return reqBidder
}

// MakeBids will parse the bids from the Alkimi server
func (adapter *adapter) MakeBids(request *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var errs []error

	if adapters.IsResponseStatusCodeNoContent(response) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{err}
	}

	var bidResp openrtb2.BidResponse
	err := json.Unmarshal(response.Body, &bidResp)
	if err != nil {
		return nil, []error{err}
	}

	seatBidCount := len(bidResp.SeatBid)
	if seatBidCount == 0 {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Empty SeatBid array",
		}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	for _, seatBid := range bidResp.SeatBid {
		for _, bid := range seatBid.Bid {
			copyBid := bid
			resolveMacros(&copyBid)
			impId := copyBid.ImpID
			imp := request.Imp
			bidType, err := getMediaTypeForImp(impId, imp)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			bidderBid := &adapters.TypedBid{
				Bid:     &copyBid,
				BidType: bidType,
			}
			bidResponse.Bids = append(bidResponse.Bids, bidderBid)
		}
	}
	return bidResponse, errs
}

func resolveMacros(bid *openrtb2.Bid) {
	strPrice := strconv.FormatFloat(bid.Price, 'f', -1, 64)
	bid.NURL = strings.Replace(bid.NURL, price_macro, strPrice, -1)
	bid.AdM = strings.Replace(bid.AdM, price_macro, strPrice, -1)
}

func getMediaTypeForImp(impId string, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Banner != nil {
				return openrtb_ext.BidTypeBanner, nil
			}
			if imp.Video != nil {
				return openrtb_ext.BidTypeVideo, nil
			}
			if imp.Audio != nil {
				return openrtb_ext.BidTypeAudio, nil
			}
		}
	}
	return "", &errortypes.BadInput{
		Message: fmt.Sprintf("Failed to find imp \"%s\"", impId),
	}
}
