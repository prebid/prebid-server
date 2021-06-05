package adyoulike

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"

	"github.com/buger/jsonparser"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
)

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	return &adapter{
		endpoint: config.Endpoint,
	}, nil
}

type adapter struct {
	endpoint string
}

func (a *adapter) MakeRequests(
	openRTBRequest *openrtb2.BidRequest,
	reqInfo *adapters.ExtraRequestInfo,
) (
	requestsToBidder []*adapters.RequestData,
	errs []error,
) {
	var err error
	var tagID string

	reqCopy := *openRTBRequest
	reqCopy.Imp = []openrtb2.Imp{}
	for ind, imp := range openRTBRequest.Imp {
		reqCopy.Imp = append(reqCopy.Imp, imp)

		tagID, err = jsonparser.GetString(reqCopy.Imp[ind].Ext, "bidder", "placement")
		if err != nil {
			errs = append(errs, err)
			continue
		}

		reqCopy.Imp[ind].TagID = tagID
	}

	openRTBRequestJSON, err := json.Marshal(reqCopy)
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("x-openrtb-version", "2.5")

	requestToBidder := &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    openRTBRequestJSON,
		Headers: headers,
	}
	requestsToBidder = append(requestsToBidder, requestToBidder)

	return requestsToBidder, errs
}

const unexpectedStatusCodeFormat = "" +
	"Unexpected status code: %d. Run with request.debug = 1 for more info"

func (a *adapter) MakeBids(
	openRTBRequest *openrtb2.BidRequest,
	requestToBidder *adapters.RequestData,
	bidderRawResponse *adapters.ResponseData,
) (
	bidderResponse *adapters.BidderResponse,
	errs []error,
) {
	switch bidderRawResponse.StatusCode {
	case http.StatusOK:
		break
	case http.StatusNoContent:
		return nil, []error{errors.New("MakeBids error: No Content")}
	case http.StatusBadRequest:
		err := &errortypes.BadInput{
			Message: fmt.Sprintf(unexpectedStatusCodeFormat, bidderRawResponse.StatusCode),
		}
		return nil, []error{err}
	default:
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf(unexpectedStatusCodeFormat, bidderRawResponse.StatusCode),
		}
		return nil, []error{err}
	}

	var openRTBBidderResponse openrtb2.BidResponse
	if err := json.Unmarshal(bidderRawResponse.Body, &openRTBBidderResponse); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(openRTBRequest.Imp))
	bidResponse.Currency = openRTBBidderResponse.Cur
	for _, seatBid := range openRTBBidderResponse.SeatBid {
		for idx := range seatBid.Bid {
			b := &adapters.TypedBid{
				Bid:     &seatBid.Bid[idx],
				BidType: getMediaTypeForImp(seatBid.Bid[idx].ImpID, openRTBRequest.Imp),
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}
	return bidResponse, nil
}

// getMediaTypeForBid determines which type of bid.
func getMediaTypeForImp(impID string, imps []openrtb2.Imp) openrtb_ext.BidType {
	mediaType := openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == impID {
			if imp.Banner == nil && imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			} else if imp.Banner == nil && imp.Native != nil {
				mediaType = openrtb_ext.BidTypeNative
			}
		}
	}

	return mediaType
}
