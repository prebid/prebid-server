package openweb

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
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

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, _ *adapters.ExtraRequestInfo) (requestsToBidder []*adapters.RequestData, errs []error) {
	org, err := checkExtAndExtractOrg(request)
	if err != nil {
		errs = append(errs, fmt.Errorf("checkExtAndExtractOrg: %w", err))
		return nil, errs
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, fmt.Errorf("marshal bidRequest: %w", err))
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	return append(requestsToBidder, &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     a.endpoint + "?publisher_id=" + org,
		Body:    requestJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}), nil
}

// checkExtAndExtractOrg checks the presence of required parameters and extracts the Org ID string.
func checkExtAndExtractOrg(request *openrtb2.BidRequest) (string, error) {
	var err error
	for _, imp := range request.Imp {
		var bidderExt adapters.ExtImpBidder
		if err = jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
			return "", fmt.Errorf("unmarshal bidderExt: %w", err)
		}

		var impExt openrtb_ext.ExtImpOpenWeb
		if err = jsonutil.Unmarshal(bidderExt.Bidder, &impExt); err != nil {
			return "", fmt.Errorf("unmarshal ExtImpOpenWeb: %w", err)
		}

		if impExt.PlacementID == "" {
			return "", errors.New("no placement id supplied")
		}

		if impExt.Org != "" {
			return strings.TrimSpace(impExt.Org), nil
		}

		if impExt.Aid != 0 {
			return strconv.Itoa(impExt.Aid), nil
		}
	}

	return "", errors.New("no org or aid supplied")
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
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	if response.Cur != "" {
		bidResponse.Currency = response.Cur
	}

	var errs []error

	for _, seatBid := range response.SeatBid {
		for i, bid := range seatBid.Bid {
			bidType, err := getMediaTypeForBid(bid)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			})
		}
	}

	return bidResponse, errs
}

func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("unsupported MType %d", bid.MType),
		}
	}
}

// Builder builds a new instance of the OpenWeb adapter for the given bidder with the given config.
func Builder(_ openrtb_ext.BidderName, config config.Adapter, _ config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}
