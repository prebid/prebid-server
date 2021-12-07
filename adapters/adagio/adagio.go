package adagio

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
)

// Builder builds a new instance of the Adagio adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

type adapter struct {
	endpoint string
}

type extBid struct {
	Prebid *openrtb_ext.ExtBidPrebid
}

// MakeRequests prepares the HTTP requests which should be made to fetch bids.
func (a *adapter) MakeRequests(request *openrtb2.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	json, err := json.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	if request.Device != nil {
		if len(request.Device.IPv6) > 0 {
			headers.Add("X-Forwarded-For", request.Device.IPv6)
		}
		if len(request.Device.IP) > 0 {
			headers.Add("X-Forwarded-For", request.Device.IP)
		}
	}

	if request.Test == 0 {
		// Gzip the body
		// Note: Gzipping could be handled natively later: https://github.com/prebid/prebid-server/issues/1812
		var bodyBuf bytes.Buffer
		gz := gzip.NewWriter(&bodyBuf)
		if _, err = gz.Write(json); err == nil {
			if err = gz.Close(); err == nil {
				json = bodyBuf.Bytes()
				headers.Add("Content-Encoding", "gzip")
				// /!\ Go already sets the `Accept-Encoding: gzip` header. Never add it manually, or Go won't decompress the response.
				//headers.Add("Accept-Encoding", "gzip")
			}
		}
	}

	requestToBidder := &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    json,
		Headers: headers,
	}

	return []*adapters.RequestData{requestToBidder}, nil
}

const unexpectedStatusCodeFormat = "Unexpected status code: %d. Run with request.debug = 1 for more info"

// MakeBids unpacks the server's response into Bids.
func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, _ *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	switch response.StatusCode {
	case http.StatusOK:
		break
	case http.StatusNoContent:
		return nil, nil
	case http.StatusServiceUnavailable:
		fallthrough
	case http.StatusBadRequest:
		fallthrough
	case http.StatusUnauthorized:
		fallthrough
	case http.StatusForbidden:
		err := &errortypes.BadInput{
			Message: fmt.Sprintf(unexpectedStatusCodeFormat, response.StatusCode),
		}
		return nil, []error{err}
	default:
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf(unexpectedStatusCodeFormat, response.StatusCode),
		}
		return nil, []error{err}
	}

	var openRTBBidderResponse openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &openRTBBidderResponse); err != nil {
		return nil, []error{err}
	}

	bidsCapacity := len(internalRequest.Imp)
	errs := make([]error, 0, bidsCapacity)
	bidderResponse := adapters.NewBidderResponseWithBidsCapacity(bidsCapacity)
	var typedBid *adapters.TypedBid
	for _, seatBid := range openRTBBidderResponse.SeatBid {
		for _, bid := range seatBid.Bid {
			activeBid := bid

			activeExt := &extBid{}
			if err := json.Unmarshal(activeBid.Ext, activeExt); err != nil {
				errs = append(errs, err)
			}

			var bidType openrtb_ext.BidType
			if activeExt.Prebid != nil && activeExt.Prebid.Type != "" {
				bidType = activeExt.Prebid.Type
			} else {
				err := &errortypes.BadServerResponse{
					Message: fmt.Sprintf("Failed to find native/banner/video mediaType \"%s\" ", activeBid.ImpID),
				}
				errs = append(errs, err)
				continue
			}

			typedBid = &adapters.TypedBid{Bid: &activeBid, BidType: bidType}
			bidderResponse.Bids = append(bidderResponse.Bids, typedBid)
		}
	}

	return bidderResponse, nil
}
