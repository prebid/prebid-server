package odeeo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

const (
	headerSP       = "x-odeeo-sp"
	headerTK       = "x-odeeo-tk"
	currencyUSD    = "USD"
	openRTBVersion = "2.6"
)

type adapter struct {
	endpoint string
}

// Builder builds a new instance of the Odeeo adapter for the given bidder with the given config.
func Builder(_ openrtb_ext.BidderName, config config.Adapter, _ config.Server) (adapters.Bidder, error) {
	return &adapter{endpoint: config.Endpoint}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var (
		errs          []error
		sp, tk        string
		processedImps = make([]openrtb2.Imp, 0, len(request.Imp))
	)

	for _, imp := range request.Imp {
		params, impExt, err := parseImpExt(imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if params.SP == "" || params.TK == "" {
			// An absent or empty bidder object unmarshals cleanly, and params are not re-checked after
			// auction hooks run - without this the request would go out with empty auth headers.
			errs = append(
				errs, &errortypes.BadInput{
					Message: fmt.Sprintf("imp %q: sp and tk are required", imp.ID),
				},
			)
			continue
		}

		if len(processedImps) > 0 && (params.SP != sp || params.TK != tk) {
			// The pair is already fixed for this request, so an imp naming a different partner cannot be sent.
			errs = append(
				errs, &errortypes.BadInput{
					Message: fmt.Sprintf("imp %q: all imps must have the same sp and tk", imp.ID),
				},
			)
			continue
		}

		if err := convertBidFloorToUSD(&imp, requestInfo); err != nil {
			errs = append(errs, err)
			continue
		}

		// One pair authenticates the whole request; mixed-partner requests are not supported.
		// Tied to the slice so the pair always belongs to an imp that is actually sent.
		if len(processedImps) == 0 {
			sp, tk = params.SP, params.TK
		}

		imp.Ext = impExt
		processedImps = append(processedImps, imp)
	}

	if len(processedImps) == 0 {
		if len(errs) == 0 {
			errs = append(errs, &errortypes.BadInput{Message: "no impressions in request"})
		}
		return nil, errs
	}

	// The request goes out as a single call; splitting multi-imp and multi-format is the endpoint's job.
	requestCopy := *request
	requestCopy.Imp = processedImps
	// The endpoint accepts USD only, which is why the floors above were converted.
	requestCopy.Cur = []string{currencyUSD}

	body, err := jsonutil.Marshal(requestCopy)
	if err != nil {
		errs = append(errs, &errortypes.FailedToRequestBids{Message: fmt.Sprintf("unable to marshal request: %s", err.Error())})
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("x-openrtb-version", openRTBVersion)
	// sp/tk identify the supply partner and are read from headers by the endpoint; they are not sent in the body.
	headers.Add(headerSP, sp)
	headers.Add(headerTK, tk)

	requestData := &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     a.endpoint,
		Body:    body,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(processedImps),
	}

	return []*adapters.RequestData{requestData}, errs
}

func parseImpExt(imp openrtb2.Imp) (openrtb_ext.ImpExtOdeeo, json.RawMessage, error) {
	var params openrtb_ext.ImpExtOdeeo

	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return params, nil, &errortypes.BadInput{
			Message: fmt.Sprintf("imp %q: invalid imp.ext: %s", imp.ID, err.Error()),
		}
	}

	if err := jsonutil.Unmarshal(bidderExt.Bidder, &params); err != nil {
		return params, nil, &errortypes.BadInput{
			Message: fmt.Sprintf("imp %q: invalid imp.ext.bidder: %s", imp.ID, err.Error()),
		}
	}

	// sp/tk travel in headers and nothing else in this object is declared in our schema.
	// Splicing the key out leaves the rest of imp.ext exactly as the publisher sent it.
	strippedExt := jsonparser.Delete(imp.Ext, "bidder")
	if !hasAnyKey(strippedExt) {
		return params, nil, nil
	}

	return params, strippedExt, nil
}

// hasAnyKey reports whether anything is left in imp.ext once the bidder params are removed.
func hasAnyKey(ext []byte) bool {
	found := false
	_ = jsonparser.ObjectEach(ext, func([]byte, []byte, jsonparser.ValueType, int) error {
		found = true
		return nil
	})

	return found
}

func convertBidFloorToUSD(imp *openrtb2.Imp, requestInfo *adapters.ExtraRequestInfo) error {
	if imp.BidFloor > 0 && imp.BidFloorCur != "" && !strings.EqualFold(imp.BidFloorCur, currencyUSD) {
		converted, err := requestInfo.ConvertCurrency(imp.BidFloor, imp.BidFloorCur, currencyUSD)
		if err != nil {
			return &errortypes.BadInput{
				Message: fmt.Sprintf(
					"imp %q: unable to convert bidfloor from %s to %s: %s",
					imp.ID, imp.BidFloorCur, currencyUSD, err.Error(),
				),
			}
		}

		imp.BidFloor = converted
		imp.BidFloorCur = currencyUSD
	}

	return nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, _ *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("unable to parse bid response: %s", err.Error()),
		}}
	}

	bidderResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))

	if response.Cur != "" {
		bidderResponse.Currency = response.Cur
	}

	var errs []error
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bid := &seatBid.Bid[i]

			bidType, err := getBidType(bid)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
				Bid:      bid,
				BidType:  bidType,
				BidMeta:  buildBidMeta(bid, seatBid.Seat, bidType),
				BidVideo: buildBidVideo(bid, bidType),
			})
		}
	}

	return bidderResponse, errs
}

func getBidType(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupAudio:
		return openrtb_ext.BidTypeAudio, nil
	// mtype is mandatory in the 2.6 response contract; a missing value is a server error, not something to guess.
	case 0:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("bid %q (imp %q): missing mtype", bid.ID, bid.ImpID),
		}
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("bid %q (imp %q): unsupported mtype %d", bid.ID, bid.ImpID, bid.MType),
		}
	}
}

func buildBidMeta(bid *openrtb2.Bid, seat string, bidType openrtb_ext.BidType) *openrtb_ext.ExtBidPrebidMeta {
	meta := &openrtb_ext.ExtBidPrebidMeta{
		MediaType: string(bidType),
		Seat:      seat,
	}

	if len(bid.ADomain) > 0 {
		meta.AdvertiserDomains = bid.ADomain
	}

	if len(bid.Cat) > 0 {
		meta.PrimaryCategoryID = bid.Cat[0]
		if len(bid.Cat) > 1 {
			meta.SecondaryCategoryIDs = bid.Cat[1:]
		}
	}

	return meta
}

func buildBidVideo(bid *openrtb2.Bid, bidType openrtb_ext.BidType) *openrtb_ext.ExtBidPrebidVideo {
	if bidType != openrtb_ext.BidTypeVideo {
		return nil
	}

	var primaryCategory string
	if len(bid.Cat) > 0 {
		primaryCategory = bid.Cat[0]
	}

	return &openrtb_ext.ExtBidPrebidVideo{
		Duration:        int(bid.Dur),
		PrimaryCategory: primaryCategory,
	}
}
