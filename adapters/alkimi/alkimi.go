package alkimi

import (
	"encoding/json"
	"fmt"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/floors"
	"net/http"
	"net/url"
	"strings"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const PRICE_MACRO = "${AUCTION_PRICE}"

type AlkimiAdapter struct {
	endpoint string
}

// Builder builds a new instance of the Alkimi adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	endpointURL, err := url.Parse(config.Endpoint)
	if err != nil || len(endpointURL.String()) == 0 {
		return nil, fmt.Errorf("invalid endpoint: %v", err)
	}

	bidder := &AlkimiAdapter{
		endpoint: endpointURL.String(),
	}
	return bidder, nil
}

// MakeRequests creates Alkimi adapter requests
func (adapter *AlkimiAdapter) MakeRequests(request *openrtb2.BidRequest, req *adapters.ExtraRequestInfo) (reqsBidder []*adapters.RequestData, errs []error) {
	reqCopy := *request
	reqCopy.Imp = _updateImps(reqCopy)
	encoded, err := json.Marshal(reqCopy)
	if err != nil {
		errs = append(errs, err)
	} else {
		reqBidder := _buildBidderRequest(adapter, encoded)
		reqsBidder = append(reqsBidder, reqBidder)
	}
	return
}

func _updateImps(bidRequest openrtb2.BidRequest) []openrtb2.Imp {
	updatedImps := make([]openrtb2.Imp, 0, len(bidRequest.Imp))
	for _, imp := range bidRequest.Imp {
		var extImpAlkimi openrtb_ext.ExtImpAlkimi
		if err := json.Unmarshal(imp.Ext, &extImpAlkimi); err == nil {
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

			extJson, err := json.Marshal(extImpAlkimi)
			if err != nil {
				continue
			}
			// imp.Ext = extJson
			updatedImps = append(updatedImps, imp)
		}
	}
	return updatedImps
}

func _buildBidderRequest(adapter *AlkimiAdapter, encoded []byte) *adapters.RequestData {
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
func (adapter *AlkimiAdapter) MakeBids(request *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var errs []error

	if request == nil || response == nil || http.StatusNoContent == response.StatusCode {
		return nil, nil
	}

	var bidResp openrtb2.BidResponse
	err := json.Unmarshal(response.Body, &bidResp)
	if err != nil {
		return nil, []error{err}
	}

	bidCount := len(bidResp.SeatBid)
	if bidCount > 0 {
		bidResponse := adapters.NewBidderResponseWithBidsCapacity(bidCount)
		for _, seatBid := range bidResp.SeatBid {
			for _, bid := range seatBid.Bid {
				copyBid := bid
				_resolveMacros(&copyBid)
				impId := copyBid.ImpID
				imp := request.Imp
				bidType, _ := _getMediaTypeForImp(impId, imp)
				bidderBid := &adapters.TypedBid{
					Bid:     &copyBid,
					BidType: bidType,
					Seat:    "alkimi",
				}
				bidResponse.Bids = append(bidResponse.Bids, bidderBid)
			}
		}
		return bidResponse, errs
	}
	return nil, nil
}

func _resolveMacros(bid *openrtb2.Bid) {
	price := bid.Price
	strPrice := fmt.Sprint(price)
	bid.NURL = strings.ReplaceAll(bid.NURL, PRICE_MACRO, strPrice)
	bid.AdM = strings.ReplaceAll(bid.AdM, PRICE_MACRO, strPrice)
}

func _getMediaTypeForImp(impId string, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
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
			if imp.Native != nil {
				return openrtb_ext.BidTypeNative, nil
			}
		}
	}
	return openrtb_ext.BidTypeBanner, &errortypes.BadInput{
		Message: fmt.Sprintf("Failed to find imp \"%s\"", impId),
	}
}
