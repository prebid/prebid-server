package bidwave

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

const defaultCurrency = "USD"

var uuidRegex = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

type adapter struct {
	endpoint string
}

type requestExtBidwave struct {
	PID string `json:"pid"`
}

type impGroup struct {
	publisherID string
	imps        []openrtb2.Imp
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter, _ config.Server) (adapters.Bidder, error) {
	return &adapter{
		endpoint: config.Endpoint,
	}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	groups, errs := groupImpsByPublisherID(request.Imp, reqInfo)

	requests := make([]*adapters.RequestData, 0, len(groups))
	for _, group := range groups {
		outgoingRequest := *request
		outgoingRequest.Cur = []string{defaultCurrency}
		outgoingRequest.Imp = group.imps

		ext, err := setBidwaveExt(outgoingRequest.Ext, group.publisherID)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		outgoingRequest.Ext = ext

		requestJSON, err := jsonutil.Marshal(outgoingRequest)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		headers := http.Header{}
		headers.Add("Content-Type", "application/json;charset=utf-8")

		requests = append(requests, &adapters.RequestData{
			Method:  http.MethodPost,
			Uri:     a.endpoint,
			Body:    requestJSON,
			Headers: headers,
			ImpIDs:  openrtb_ext.GetImpIDs(group.imps),
		})
	}

	return requests, errs
}

func prepareImpCurrency(imp openrtb2.Imp, reqInfo *adapters.ExtraRequestInfo) (openrtb2.Imp, error) {
	if imp.BidFloor > 0 && strings.TrimSpace(imp.BidFloorCur) != "" && !strings.EqualFold(imp.BidFloorCur, defaultCurrency) {
		bidFloor, err := reqInfo.ConvertCurrency(imp.BidFloor, imp.BidFloorCur, defaultCurrency)
		if err != nil {
			return imp, &errortypes.BadInput{
				Message: fmt.Sprintf("expected currency %s for bid floor; unable to convert from %s for impression %s: %s", defaultCurrency, imp.BidFloorCur, imp.ID, err.Error()),
			}
		}

		imp.BidFloor = bidFloor
		imp.BidFloorCur = defaultCurrency
	}

	return imp, nil
}

func groupImpsByPublisherID(imps []openrtb2.Imp, reqInfo *adapters.ExtraRequestInfo) ([]impGroup, []error) {
	groupIndexes := make(map[string]int)
	groups := make([]impGroup, 0, len(imps))
	errs := make([]error, 0)

	for i := range imps {
		publisherID, err := parsePublisherID(imps[i])
		if err != nil {
			errs = append(errs, err)
			continue
		}

		imp, err := prepareImpCurrency(imps[i], reqInfo)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if groupIndex, ok := groupIndexes[publisherID]; ok {
			groups[groupIndex].imps = append(groups[groupIndex].imps, imp)
			continue
		}

		groupIndexes[publisherID] = len(groups)
		groups = append(groups, impGroup{
			publisherID: publisherID,
			imps:        []openrtb2.Imp{imp},
		})
	}

	return groups, errs
}

func parsePublisherID(imp openrtb2.Imp) (string, error) {
	var ext adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &ext); err != nil {
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf("invalid imp.ext for impression %s: %s", imp.ID, err.Error()),
		}
	}

	var params openrtb_ext.ExtImpBidwave
	if err := jsonutil.Unmarshal(ext.Bidder, &params); err != nil {
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf("invalid bidwave params for impression %s: %s", imp.ID, err.Error()),
		}
	}

	if !uuidRegex.MatchString(params.PublisherID) {
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf("invalid publisherId for impression %s", imp.ID),
		}
	}

	return params.PublisherID, nil
}

func setBidwaveExt(rawExt json.RawMessage, publisherID string) (json.RawMessage, error) {
	ext := map[string]json.RawMessage{}
	if len(rawExt) > 0 {
		if err := jsonutil.Unmarshal(rawExt, &ext); err != nil {
			return nil, err
		}
	}

	bidwaveExt, err := jsonutil.Marshal(requestExtBidwave{PID: publisherID})
	if err != nil {
		return nil, err
	}

	ext["bidwave"] = bidwaveExt
	return jsonutil.Marshal(ext)
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, _ *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(response) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{err}
	}

	var bidResponse openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResponse); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Bad Server Response",
		}}
	}

	result := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	if bidResponse.Cur != "" {
		result.Currency = bidResponse.Cur
	}

	var errs []error
	for seatBidIndex := range bidResponse.SeatBid {
		for bidIndex := range bidResponse.SeatBid[seatBidIndex].Bid {
			bid := &bidResponse.SeatBid[seatBidIndex].Bid[bidIndex]
			bidType, err := getBidType(bid)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			result.Bids = append(result.Bids, &adapters.TypedBid{
				Bid:     bid,
				BidType: bidType,
			})
		}
	}

	return result, errs
}

func getBidType(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case 0:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Bid must have non-zero MType for impression with ID: \"%s\"", bid.ImpID),
		}
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unsupported MType %d for impression with ID: \"%s\"", bid.MType, bid.ImpID),
		}
	}
}
