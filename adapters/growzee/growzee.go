package growzee

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"

	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/macros"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

type adapter struct {
	endpoint *template.Template
}

// Builder builds a new instance of the Growzee adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	tmpl, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}
	return &adapter{endpoint: tmpl}, nil
}

// MakeRequests sends one request per imp: the Growzee endpoint accepts a single imp per auction.
func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var requests []*adapters.RequestData
	var errs []error

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("X-Openrtb-Version", "2.5")
	if d := request.Device; d != nil {
		if d.UA != "" {
			headers.Add("User-Agent", d.UA)
		}
		if d.IP != "" {
			headers.Add("X-Forwarded-For", d.IP)
		} else if d.IPv6 != "" {
			headers.Add("X-Forwarded-For", d.IPv6)
		}
	}

	reqCopy := *request
	for _, imp := range request.Imp {
		reqData, err := a.makeRequest(reqCopy, imp, reqInfo, headers)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		requests = append(requests, reqData)
	}
	return requests, errs
}

func (a *adapter) makeRequest(req openrtb2.BidRequest, imp openrtb2.Imp, reqInfo *adapters.ExtraRequestInfo, headers http.Header) (*adapters.RequestData, error) {
	var ext map[string]json.RawMessage
	if err := jsonutil.Unmarshal(imp.Ext, &ext); err != nil {
		return nil, &errortypes.BadInput{Message: fmt.Sprintf("imp %s: invalid ext: %v", imp.ID, err)}
	}
	var params openrtb_ext.ExtImpGrowzee
	if err := jsonutil.Unmarshal(ext["bidder"], &params); err != nil {
		return nil, &errortypes.BadInput{Message: fmt.Sprintf("imp %s: invalid ext.bidder: %v", imp.ID, err)}
	}

	uri, err := macros.ResolveMacros(a.endpoint, macros.EndpointTemplateParams{PublisherID: params.PublisherID})
	if err != nil {
		return nil, err
	}

	if imp.BidFloor > 0 && imp.BidFloorCur != "" && !strings.EqualFold(imp.BidFloorCur, "USD") {
		floor, err := reqInfo.ConvertCurrency(imp.BidFloor, imp.BidFloorCur, "USD")
		if err != nil {
			return nil, err
		}
		imp.BidFloor, imp.BidFloorCur = floor, "USD"
	}

	// Keep the rest of imp.ext (skadn, gpid, prebid.is_rewarded_inventory); only the bidder params are ours.
	delete(ext, "bidder")
	if len(ext) == 0 {
		imp.Ext = nil
	} else if imp.Ext, err = jsonutil.Marshal(ext); err != nil {
		return nil, err
	}

	req.Imp = []openrtb2.Imp{imp}
	body, err := jsonutil.Marshal(req)
	if err != nil {
		return nil, err
	}
	return &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     uri,
		Body:    body,
		Headers: headers,
		ImpIDs:  []string{imp.ID},
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
		return nil, []error{err}
	}

	// A stored bid response has no Uri and no ImpIDs, and its imp is gone from the request.
	stored := requestData.Uri == "" && len(requestData.ImpIDs) == 0

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)
	if response.Cur != "" {
		bidResponse.Currency = response.Cur
	}
	var errs []error
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bid := &seatBid.Bid[i]
			var imp *openrtb2.Imp
			if !stored {
				if imp = requestedImp(request, requestData.ImpIDs, bid.ImpID); imp == nil {
					errs = append(errs, &errortypes.BadServerResponse{
						Message: fmt.Sprintf("bid %s has impid %s, not an imp of its request", bid.ID, bid.ImpID),
					})
					continue
				}
			}
			bidType, err := getBidType(bid, imp)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{Bid: bid, BidType: bidType})
		}
	}
	return bidResponse, errs
}

// requestedImp returns the imp a bid names, only if it was sent in this request (ImpIDs).
func requestedImp(request *openrtb2.BidRequest, sent []string, impID string) *openrtb2.Imp {
	if len(sent) > 0 && !slices.Contains(sent, impID) {
		return nil
	}
	for i := range request.Imp {
		if request.Imp[i].ID == impID {
			return &request.Imp[i]
		}
	}
	return nil
}

// getBidType trusts mtype, then a single-format imp, then sniffs VAST on multi-format imps.
// Without an imp (stored bid response) only mtype can tell.
func getBidType(bid *openrtb2.Bid, imp *openrtb2.Imp) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	}
	if imp == nil {
		return "", &errortypes.BadServerResponse{Message: fmt.Sprintf("bid %s: stored bid response without mtype", bid.ID)}
	}

	switch {
	case imp.Video != nil && (imp.Banner == nil || isVAST(bid.AdM)):
		return openrtb_ext.BidTypeVideo, nil
	case imp.Banner != nil:
		return openrtb_ext.BidTypeBanner, nil
	case imp.Native != nil:
		return openrtb_ext.BidTypeNative, nil
	}
	return "", &errortypes.BadServerResponse{Message: fmt.Sprintf("unsupported media type for imp %s", imp.ID)}
}

func isVAST(adm string) bool {
	return strings.Contains(adm[:min(len(adm), 512)], "<VAST")
}
