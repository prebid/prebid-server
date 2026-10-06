package goadserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"

	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

// GoAdserver is a self-hosted, multi-tenant ad server. All requests go to one
// fixed GoAdserver gateway, which routes each to the publisher's deployment by
// the publisher token. Impressions are grouped by token and each group is sent
// with its token in site.publisher.id.
type adapter struct {
	endpoint string
}

type impGroup struct {
	token string
	imps  []openrtb2.Imp
}

// Builder builds a new instance of the GoAdserver adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	return &adapter{endpoint: config.Endpoint}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if request.Site == nil {
		return nil, []error{&errortypes.BadInput{Message: "goadserver supports site requests only"}}
	}

	var errs []error
	var groups []*impGroup
	byKey := make(map[string]*impGroup)

	for _, imp := range request.Imp {
		params, err := parseImpExt(&imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		applyParams(&imp, params)

		group, ok := byKey[params.Token]
		if !ok {
			group = &impGroup{token: params.Token}
			byKey[params.Token] = group
			groups = append(groups, group)
		}
		group.imps = append(group.imps, imp)
	}

	requests := make([]*adapters.RequestData, 0, len(groups))
	for _, group := range groups {
		requestData, err := a.buildRequest(request, group)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		requests = append(requests, requestData)
	}
	return requests, errs
}

// parseImpExt reads and validates the bidder params of one impression.
func parseImpExt(imp *openrtb2.Imp) (*openrtb_ext.ExtImpGoadserver, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{Message: fmt.Sprintf("imp %s: invalid ext: %v", imp.ID, err)}
	}
	var params openrtb_ext.ExtImpGoadserver
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &params); err != nil {
		return nil, &errortypes.BadInput{Message: fmt.Sprintf("imp %s: invalid ext.bidder: %v", imp.ID, err)}
	}

	params.Token = strings.TrimSpace(params.Token)
	if params.Token == "" {
		return nil, &errortypes.BadInput{Message: fmt.Sprintf("imp %s: missing token", imp.ID)}
	}
	return &params, nil
}

// applyParams applies the optional floor and subid params and replaces the
// bidder ext with the shape the GoAdserver endpoint reads.
func applyParams(imp *openrtb2.Imp, params *openrtb_ext.ExtImpGoadserver) {
	if params.Floor > 0 && imp.BidFloor == 0 {
		imp.BidFloor = params.Floor
		imp.BidFloorCur = "USD"
	}

	imp.Ext = nil
	if subID := subIDString(params.SubID); subID != "" {
		imp.Ext, _ = jsonutil.Marshal(map[string]any{
			"goadserver": map[string]string{"subid": subID},
		})
	}
}

// subIDString accepts the subid param as a JSON string or number.
func subIDString(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if err := jsonutil.Unmarshal(raw, &s); err == nil {
		return s
	}
	return string(raw)
}

func (a *adapter) buildRequest(request *openrtb2.BidRequest, group *impGroup) (*adapters.RequestData, error) {
	requestCopy := *request
	requestCopy.Imp = group.imps

	site := *request.Site
	var publisher openrtb2.Publisher
	if site.Publisher != nil {
		publisher = *site.Publisher
	}
	publisher.ID = group.token
	site.Publisher = &publisher
	requestCopy.Site = &site

	body, err := jsonutil.Marshal(requestCopy)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("X-Openrtb-Version", "2.5")

	return &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     a.endpoint,
		Body:    body,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(requestCopy.Imp),
	}, nil
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
		return nil, []error{&errortypes.BadServerResponse{Message: fmt.Sprintf("invalid response body: %v", err)}}
	}
	if len(response.SeatBid) == 0 {
		return nil, nil
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	if response.Cur != "" {
		bidResponse.Currency = response.Cur
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
			bid.Ext = keepDSA(bid.Ext)
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     bid,
				BidType: bidType,
			})
		}
	}
	return bidResponse, errs
}

// getBidType reads the bid's media type from bid.mtype, falling back to
// bid.ext.prebid.type, which GoAdserver sets on every bid.
func getBidType(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	}

	if len(bid.Ext) > 0 {
		var ext openrtb_ext.ExtBid
		if err := jsonutil.Unmarshal(bid.Ext, &ext); err == nil && ext.Prebid != nil {
			switch ext.Prebid.Type {
			case openrtb_ext.BidTypeBanner, openrtb_ext.BidTypeVideo, openrtb_ext.BidTypeNative:
				return ext.Prebid.Type, nil
			}
		}
	}

	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("unsupported media type for bid %s on imp %s", bid.ID, bid.ImpID),
	}
}

// keepDSA drops the Prebid targeting GoAdserver adds for its direct
// Prebid.js integration and keeps only the DSA transparency object, which
// Prebid Server validates and relays.
func keepDSA(ext json.RawMessage) json.RawMessage {
	if len(ext) == 0 {
		return nil
	}
	var parsed struct {
		DSA json.RawMessage `json:"dsa"`
	}
	if err := jsonutil.Unmarshal(ext, &parsed); err != nil || len(parsed.DSA) == 0 || string(parsed.DSA) == "null" {
		return nil
	}
	out, err := jsonutil.Marshal(map[string]json.RawMessage{"dsa": parsed.DSA})
	if err != nil {
		return nil
	}
	return out
}
