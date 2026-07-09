// Package realry is the Prebid Server bidder adapter for the Realry
// commerce DSP (https://bid.realry.com). The adapter is intentionally
// thin — Realry's /bid/openrtb endpoint speaks standard OpenRTB 2.6
// with Native 1.2 admObject support, so MakeRequests just forwards the
// BidRequest and MakeBids walks the BidResponse seatbids.
//
// Bid type inference: Realry's endpoint does NOT set ext.prebid.type on
// returned bids (it returns plain openrtb2.Bid). The adapter infers the
// bid type from the matching imp's media types — imp.Native present →
// native, otherwise banner. Realry does not currently bid on video.
package realry

import (
	"net/http"

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

// Builder constructs the Realry adapter. Per Prebid convention the
// endpoint URL is wired from `pbs.yaml` config (`adapters.realry.endpoint`)
// — we don't pin it in code so a host can repoint at a staging Realry
// for integration testing.
func Builder(_ openrtb_ext.BidderName, config config.Adapter, _ config.Server) (adapters.Bidder, error) {
	return &adapter{endpoint: config.Endpoint}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	requestJSON, err := jsonutil.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	// X-SSP-Id is required by bid.realry.com — the Prebid Server host
	// is the SSP from Realry's perspective, identified by a stable label.
	// "prebid-server" is the default; host can override per-deployment
	// via prebid-server config if Realry assigns a unique partner id.
	headers.Add("X-SSP-Id", "prebid-server")
	// Bearer token: while bid.realry.com is in OPEN MODE the value is
	// not checked beyond non-empty. When Realry flips to closed mode
	// (OPENRTB_SSPS set on their side) the host running prebid-server
	// will need to inject the assigned token — see config.Adapter
	// ExtraAdapterInfo for the wiring point.
	headers.Add("Authorization", "Bearer prebid-server")

	return []*adapters.RequestData{{
		Method:  http.MethodPost,
		Uri:     a.endpoint,
		Body:    requestJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}}, nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, _ *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	if responseData.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: "Realry: HTTP 400 from bid.realry.com (run with request.debug = 1 for more info).",
		}}
	}
	if responseData.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Realry: unexpected status from bid.realry.com.",
		}}
	}

	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	// Build a quick impid → mediaType index so each returned bid's
	// type can be inferred from the imp it answered.
	mediaByImp := make(map[string]openrtb_ext.BidType, len(request.Imp))
	for _, imp := range request.Imp {
		switch {
		case imp.Native != nil:
			mediaByImp[imp.ID] = openrtb_ext.BidTypeNative
		case imp.Banner != nil:
			mediaByImp[imp.ID] = openrtb_ext.BidTypeBanner
		default:
			mediaByImp[imp.ID] = openrtb_ext.BidTypeBanner
		}
	}

	bidResponse := adapters.NewBidderResponse()
	bidResponse.Currency = response.Cur
	var errs []error
	for _, seat := range response.SeatBid {
		for i := range seat.Bid {
			bid := &seat.Bid[i]
			bidType, ok := mediaByImp[bid.ImpID]
			if !ok {
				errs = append(errs, &errortypes.BadServerResponse{
					Message: "Realry: bid for unknown impid " + bid.ImpID,
				})
				continue
			}
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     bid,
				BidType: bidType,
			})
		}
	}
	return bidResponse, errs
}
