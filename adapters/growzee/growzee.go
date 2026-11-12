package growzee

import (
	"encoding/json"
	"fmt"
	"net/http"
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

	// Keep the rest of imp.ext (skadn, gpid, data); only the bidder params are ours.
	delete(ext, "bidder")
	delete(ext, "prebid")
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

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)
	if response.Cur != "" {
		bidResponse.Currency = response.Cur
	}
	var errs []error
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bid := &seatBid.Bid[i]
			bidType, err := getBidType(bid, request.Imp)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{Bid: bid, BidType: bidType})
		}
	}
	return bidResponse, errs
}

// getBidType trusts mtype, then a single-format imp, then sniffs VAST on multi-format imps.
func getBidType(bid *openrtb2.Bid, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	}

	for _, imp := range imps {
		if imp.ID != bid.ImpID {
			continue
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
	return "", &errortypes.BadServerResponse{Message: fmt.Sprintf("bid %s has unknown impid %s", bid.ID, bid.ImpID)}
}

func isVAST(adm string) bool {
	return strings.Contains(adm[:min(len(adm), 512)], "<VAST")
}
