package revantage

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

type adapter struct {
	endpoint string
}

// Builder builds a new instance of the Revantage adapter for the given bidder with the given config.
func Builder(_ openrtb_ext.BidderName, cfg config.Adapter, _ config.Server) (adapters.Bidder, error) {
	return &adapter{endpoint: cfg.Endpoint}, nil
}

// rewrittenImpExt is the shape the Revantage endpoint expects on imp.ext.
// It mirrors the public Prebid.js client adapter (revantageBidAdapter.js) so the
// upstream endpoint can be a single code path for both client- and server-side
// integrations.
type rewrittenImpExt struct {
	FeedID string             `json:"feedId"`
	Bidder rewrittenImpBidder `json:"bidder"`
}

type rewrittenImpBidder struct {
	PlacementID string `json:"placementId,omitempty"`
	PublisherID string `json:"publisherId,omitempty"`
}

// MakeRequests converts an OpenRTB bid request into one or more HTTP calls to the Revantage endpoint.
//
// Impressions are grouped by feedId. Each group becomes a separate HTTP call so that a single
// auction can serve multiple feeds without conflict — the endpoint's `?feed=` query param must
// match the feedId used in every imp it carries.
func (a *adapter) MakeRequests(request *openrtb2.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if request == nil || len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{Message: "no impressions in bid request"}}
	}

	// Preserve insertion order of feed groups for deterministic test output.
	type group struct {
		imps []openrtb2.Imp
	}
	groups := make(map[string]*group)
	feedOrder := make([]string, 0)
	var errs []error

	for i := range request.Imp {
		imp := request.Imp[i]
		feedID, err := rewriteImpExt(&imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		g, ok := groups[feedID]
		if !ok {
			g = &group{}
			groups[feedID] = g
			feedOrder = append(feedOrder, feedID)
		}
		g.imps = append(g.imps, imp)
	}

	if len(groups) == 0 {
		return nil, errs
	}

	requests := make([]*adapters.RequestData, 0, len(groups))
	for _, feedID := range feedOrder {
		imps := groups[feedID].imps

		// Shallow copy the request and replace imps with this feed's slice.
		reqCopy := *request
		reqCopy.Imp = imps

		body, err := json.Marshal(reqCopy)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to marshal request for feed %s: %w", feedID, err))
			continue
		}

		headers := http.Header{}
		headers.Add("Content-Type", "application/json;charset=utf-8")
		headers.Add("Accept", "application/json")

		requests = append(requests, &adapters.RequestData{
			Method:  http.MethodPost,
			Uri:     a.endpoint + "?feed=" + url.QueryEscape(feedID),
			Body:    body,
			Headers: headers,
			ImpIDs:  collectImpIDs(imps),
		})
	}

	return requests, errs
}

// rewriteImpExt validates the bidder params on a single imp and rewrites imp.ext into the
// shape the Revantage endpoint expects. Returns the resolved feedId.
func rewriteImpExt(imp *openrtb2.Imp) (string, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: invalid imp.ext: %s", imp.ID, err.Error()),
		}
	}

	var revantageExt openrtb_ext.ImpExtRevantage
	if err := json.Unmarshal(bidderExt.Bidder, &revantageExt); err != nil {
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: invalid imp.ext.bidder: %s", imp.ID, err.Error()),
		}
	}

	feedID := strings.TrimSpace(revantageExt.FeedID)
	if feedID == "" {
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: missing required param feedId", imp.ID),
		}
	}

	rewritten := rewrittenImpExt{
		FeedID: feedID,
		Bidder: rewrittenImpBidder{
			PlacementID: revantageExt.PlacementID,
			PublisherID: revantageExt.PublisherID,
		},
	}
	extBytes, err := json.Marshal(rewritten)
	if err != nil {
		return "", fmt.Errorf("imp %s: failed to marshal rewritten ext: %w", imp.ID, err)
	}
	imp.Ext = extBytes

	return feedID, nil
}

func collectImpIDs(imps []openrtb2.Imp) []string {
	ids := make([]string, len(imps))
	for i, imp := range imps {
		ids[i] = imp.ID
	}
	return ids
}

// MakeBids parses the upstream Revantage response into typed Prebid bids.
func (a *adapter) MakeBids(request *openrtb2.BidRequest, _ *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}
	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{err}
	}

	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "invalid bid response: " + err.Error(),
		}}
	}

	if len(bidResp.SeatBid) == 0 {
		return nil, nil
	}

	response := adapters.NewBidderResponse()
	if bidResp.Cur != "" {
		response.Currency = bidResp.Cur
	} else {
		response.Currency = "USD"
	}

	var errs []error
	for _, seat := range bidResp.SeatBid {
		for i := range seat.Bid {
			bid := &seat.Bid[i]
			mt, err := resolveMediaType(bid, request.Imp)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			response.Bids = append(response.Bids, &adapters.TypedBid{
				Bid:     bid,
				BidType: mt,
				Seat:    openrtb_ext.BidderName(seat.Seat),
			})
		}
	}
	return response, errs
}

// resolveMediaType picks banner or video based on (in priority order):
//  1. bid.mtype (oRTB 2.6)
//  2. bid.ext.mediaType
//  3. VAST shape detection on bid.adm
//  4. The single configured media type on the originating imp
func resolveMediaType(bid *openrtb2.Bid, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	}

	if len(bid.Ext) > 0 {
		var ext struct {
			MediaType string `json:"mediaType"`
		}
		if err := json.Unmarshal(bid.Ext, &ext); err == nil {
			switch strings.ToLower(ext.MediaType) {
			case "banner":
				return openrtb_ext.BidTypeBanner, nil
			case "video":
				return openrtb_ext.BidTypeVideo, nil
			}
		}
	}

	if isVastMarkup(bid.AdM) {
		return openrtb_ext.BidTypeVideo, nil
	}

	for _, imp := range imps {
		if imp.ID != bid.ImpID {
			continue
		}
		hasBanner := imp.Banner != nil
		hasVideo := imp.Video != nil
		switch {
		case hasVideo && !hasBanner:
			return openrtb_ext.BidTypeVideo, nil
		case hasBanner && !hasVideo:
			return openrtb_ext.BidTypeBanner, nil
		case hasBanner && hasVideo:
			// Multi-format with no explicit signal — default to banner. The Revantage
			// endpoint should set mtype or ext.mediaType in this case; this is a fallback.
			return openrtb_ext.BidTypeBanner, nil
		}
		break
	}

	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("could not determine media type for bid %s on imp %s", bid.ID, bid.ImpID),
	}
}

func isVastMarkup(adm string) bool {
	trimmed := strings.TrimSpace(adm)
	if trimmed == "" {
		return false
	}
	upper := strings.ToUpper(trimmed)
	return strings.HasPrefix(upper, "<VAST") || strings.HasPrefix(upper, "<?XML")
}
