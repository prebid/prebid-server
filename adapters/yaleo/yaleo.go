package yaleo

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

type adapter struct {
	endpoint string
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	return &adapter{
		endpoint: config.Endpoint,
	}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if err := validateSupply(request); err != nil {
		return nil, []error{err}
	}

	groups, errs := groupImpsByMemberID(request.Imp)
	if len(groups) == 0 {
		return nil, errs
	}

	headers := buildHeaders()
	requests := make([]*adapters.RequestData, 0, len(groups))

	for _, groupImps := range groups {
		requestCopy := *request
		requestCopy.Imp = groupImps

		body, err := jsonutil.Marshal(&requestCopy)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		requests = append(requests, &adapters.RequestData{
			Method:  http.MethodPost,
			Uri:     a.endpoint,
			Body:    body,
			Headers: headers.Clone(),
			ImpIDs:  openrtb_ext.GetImpIDs(groupImps),
		})
	}

	return requests, errs
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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
		for i := range seatBid.Bid {
			bidType, err := getMediaTypeForBid(seatBid.Bid[i])
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

func validateSupply(request *openrtb2.BidRequest) error {
	if request.App != nil {
		if request.App.Bundle == "" && request.App.Domain == "" && request.App.ID == "" {
			return &errortypes.BadInput{
				Message: `"app" must include bundle, domain, or id.`,
			}
		}
		return nil
	}

	if request.Site == nil || request.Site.Page == "" {
		return &errortypes.BadInput{
			Message: `Either "site.page" or an "app" with bundle, domain, or id is required.`,
		}
	}

	return nil
}

func groupImpsByMemberID(imps []openrtb2.Imp) ([][]openrtb2.Imp, []error) {
	var groups [][]openrtb2.Imp
	indexByKey := make(map[string]int)
	var errs []error

	for i := range imps {
		prepared, memberID, err := prepareImp(imps[i])
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// Yaleo applies one memberId per request, so an omitted memberId
		// cannot share a request with a specified one.
		key := memberKey(memberID)
		if idx, ok := indexByKey[key]; ok {
			groups[idx] = append(groups[idx], prepared)
			continue
		}

		indexByKey[key] = len(groups)
		groups = append(groups, []openrtb2.Imp{prepared})
	}

	return groups, errs
}

func prepareImp(imp openrtb2.Imp) (openrtb2.Imp, *int, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return openrtb2.Imp{}, nil, &errortypes.BadInput{Message: err.Error()}
	}

	var params openrtb_ext.ImpExtYaleo
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &params); err != nil {
		return openrtb2.Imp{}, nil, &errortypes.BadInput{Message: err.Error()}
	}

	return imp, params.MemberID, nil
}

func memberKey(memberID *int) string {
	if memberID == nil {
		return ""
	}
	return strconv.Itoa(*memberID)
}

func buildHeaders() http.Header {
	headers := http.Header{}
	// text/plain skips the CORS preflight and is the only type express.text() parses.
	headers.Set("Content-Type", "text/plain")
	headers.Set("Accept", "application/json")
	return headers
}

func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	if bid.MType == openrtb2.MarkupBanner {
		return openrtb_ext.BidTypeBanner, nil
	}

	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("unsupported mtype %d for imp %s", bid.MType, bid.ImpID),
	}
}
