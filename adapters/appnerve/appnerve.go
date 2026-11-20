package appnerve

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

type adapter struct{ endpoint string }

// Builder builds a new instance of the Appnerve adapter for the given bidder with the given config.
func Builder(_ openrtb_ext.BidderName, cfg config.Adapter, _ config.Server) (adapters.Bidder, error) {
	endpoint, err := url.Parse(cfg.Endpoint)
	if err != nil || endpoint.Scheme == "" || endpoint.Host == "" {
		return nil, fmt.Errorf("invalid appnerve endpoint %q", cfg.Endpoint)
	}
	return &adapter{endpoint: cfg.Endpoint}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var requests []*adapters.RequestData
	var errs []error
	for _, imp := range request.Imp {
		var ext adapters.ExtImpBidder
		var params openrtb_ext.ExtImpAppnerve
		if err := json.Unmarshal(imp.Ext, &ext); err != nil {
			errs = append(errs, &errortypes.BadInput{Message: "invalid impression extension"})
			continue
		}
		if err := json.Unmarshal(ext.Bidder, &params); err != nil {
			errs = append(errs, &errortypes.BadInput{Message: "invalid Appnerve parameters"})
			continue
		}
		id := strings.TrimSpace(params.SourceID)
		if id == "" {
			id = strings.TrimSpace(params.PlacementID)
		}
		if id == "" || imp.ID == "" {
			errs = append(errs, &errortypes.BadInput{Message: "sourceId (or placementId) and imp.id are required"})
			continue
		}
		// Preserve publisher fields, but remove the PBS adapter's private parameters.
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(imp.Ext, &fields); err != nil {
			errs = append(errs, err)
			continue
		}
		delete(fields, "bidder")
		imp.Ext, _ = json.Marshal(fields)
		imp.TagID = id
		copied := *request
		copied.Imp = []openrtb2.Imp{imp}
		body, err := json.Marshal(copied)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		requests = append(requests, &adapters.RequestData{
			Method: http.MethodPost, Uri: a.endpoint + "?ssp_id=" + url.QueryEscape(id), Body: body,
			Headers: http.Header{"Content-Type": {"application/json"}, "Accept": {"application/json"}, "X-OpenRTB-Version": {"2.6"}},
			ImpIDs:  []string{imp.ID},
		})
	}
	return requests, errs
}

func (a *adapter) MakeBids(_ *openrtb2.BidRequest, sent *adapters.RequestData, received *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(received) {
		return nil, nil
	}
	if err := adapters.CheckResponseStatusCodeForErrors(received); err != nil {
		return nil, []error{err}
	}
	var response openrtb2.BidResponse
	if err := json.Unmarshal(received.Body, &response); err != nil {
		return nil, []error{&errortypes.BadServerResponse{Message: "invalid auction response JSON"}}
	}
	var outbound openrtb2.BidRequest
	if err := json.Unmarshal(sent.Body, &outbound); err != nil || len(outbound.Imp) != 1 {
		return nil, []error{&errortypes.BadServerResponse{Message: "invalid adapter request context"}}
	}
	imp := outbound.Imp[0]
	result := adapters.NewBidderResponseWithBidsCapacity(1)
	result.Currency = response.Cur
	if result.Currency == "" {
		result.Currency = "USD"
	}
	var errs []error
	for _, seat := range response.SeatBid {
		for _, bid := range seat.Bid {
			if bid.ImpID != imp.ID || bid.ID == "" || bid.AdM == "" || bid.Price <= 0 || math.IsNaN(bid.Price) || math.IsInf(bid.Price, 0) {
				errs = append(errs, &errortypes.BadServerResponse{Message: "invalid bid identity, creative or price"})
				continue
			}
			kind, err := bidType(bid, imp)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			copied := bid
			result.Bids = append(result.Bids, &adapters.TypedBid{Bid: &copied, BidType: kind})
		}
	}
	return result, errs
}

func bidType(bid openrtb2.Bid, imp openrtb2.Imp) (openrtb_ext.BidType, error) {
	var ext struct {
		Prebid struct {
			Type string `json:"type"`
		} `json:"prebid"`
	}
	if len(bid.Ext) > 0 {
		if err := json.Unmarshal(bid.Ext, &ext); err != nil {
			return "", &errortypes.BadServerResponse{Message: "invalid bid extension"}
		}
	}
	types := map[string]bool{"banner": imp.Banner != nil, "video": imp.Video != nil, "native": imp.Native != nil, "audio": imp.Audio != nil}
	if ext.Prebid.Type != "" {
		if types[ext.Prebid.Type] {
			return openrtb_ext.BidType(ext.Prebid.Type), nil
		}
		return "", &errortypes.BadServerResponse{Message: "unrequested bid media type"}
	}
	var kind string
	count := 0
	for candidate, present := range types {
		if present {
			kind = candidate
			count++
		}
	}
	if count == 1 {
		return openrtb_ext.BidType(kind), nil
	}
	return "", &errortypes.BadServerResponse{Message: "multi-format bid requires ext.prebid.type"}
}
