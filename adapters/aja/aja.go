package aja

import (
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
)

type AJAAdapter struct {
	endpoint string
}

func (a *AJAAdapter) MakeRequests(bidReq *openrtb.BidRequest, extraInfo *adapters.ExtraRequestInfo) (adapterReqs []*adapters.RequestData, errs []error) {
	// split imps by tagid
	tagIDs := []string{}
	impsByTagID := map[string][]openrtb.Imp{}
	for _, imp := range bidReq.Imp {
		extAJA, err := parseExtAJA(imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		imp.TagID = extAJA.AdSpotID
		imp.Ext = nil
		if _, ok := impsByTagID[imp.TagID]; !ok {
			tagIDs = append(tagIDs, imp.TagID)
		}
		impsByTagID[imp.TagID] = append(impsByTagID[imp.TagID], imp)
	}

	req := *bidReq
	for _, tagID := range tagIDs {
		req.Imp = impsByTagID[tagID]
		body, err := json.Marshal(req)
		if err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: fmt.Sprintf("Failed to unmarshal bidrequest ID: %s err: %s", bidReq.ID, err),
			})
			continue
		}
		adapterReqs = append(adapterReqs, &adapters.RequestData{
			Method: "POST",
			Uri:    a.endpoint,
			Body:   body,
		})
	}

	return
}

func parseExtAJA(imp openrtb.Imp) (openrtb_ext.ExtImpAJA, error) {
	var (
		extImp adapters.ExtImpBidder
		extAJA openrtb_ext.ExtImpAJA
	)

	if err := json.Unmarshal(imp.Ext, &extImp); err != nil {
		return extAJA, &errortypes.BadInput{
			Message: fmt.Sprintf("Failed to unmarshal ext impID: %s err: %s", imp.ID, err),
		}
	}

	if err := json.Unmarshal(extImp.Bidder, &extAJA); err != nil {
		return extAJA, &errortypes.BadInput{
			Message: fmt.Sprintf("Failed to unmarshal ext.bidder impID: %s err: %s", imp.ID, err),
		}
	}

	return extAJA, nil
}

func (a *AJAAdapter) MakeBids(bidReq *openrtb.BidRequest, adapterReq *adapters.RequestData, adapterResp *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapterResp.StatusCode != http.StatusOK {
		if adapterResp.StatusCode == http.StatusNoContent {
			return nil, nil
		}
		if adapterResp.StatusCode == http.StatusBadRequest {
			return nil, []error{&errortypes.BadInput{
				Message: fmt.Sprintf("Unexpected status code: %d", adapterResp.StatusCode),
			}}
		}
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d", adapterResp.StatusCode),
		}}
	}

	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(adapterResp.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Failed to unmarshal bid response: %s", err.Error()),
		}}
	}

	bidderResp := adapters.NewBidderResponseWithBidsCapacity(len(bidReq.Imp))
	var errors []error

	for _, seatbid := range bidResp.SeatBid {
		for _, bid := range seatbid.Bid {
			for _, imp := range bidReq.Imp {
				if imp.ID == bid.ImpID {
					var bidType openrtb_ext.BidType
					if imp.Banner != nil {
						bidType = openrtb_ext.BidTypeBanner
					} else if imp.Video != nil {
						bidType = openrtb_ext.BidTypeVideo
					} else {
						errors = append(errors, &errortypes.BadServerResponse{
							Message: fmt.Sprintf("Response received for unexpected type of bid bidID: %s", bid.ID),
						})
						continue
					}
					bidderResp.Bids = append(bidderResp.Bids, &adapters.TypedBid{
						Bid:     &bid,
						BidType: bidType,
					})
					break
				}
			}
		}
	}
	return bidderResp, errors
}

func NewAJABidder(endpoint string) adapters.Bidder {
	return &AJAAdapter{
		endpoint: endpoint,
	}
}
