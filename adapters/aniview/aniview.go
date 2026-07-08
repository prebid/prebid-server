package aniview

import (
	"bytes"
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

type adapter struct {
	endpoint string
}

type aniviewExt struct {
	PBS int `json:"pbs"`
}

// Builder builds a new instance of the Aniview adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var requests []*adapters.RequestData
	var errors []error

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	requestExt, err := buildRequestExt(request.Ext)
	if err != nil {
		return nil, []error{err}
	}

	// One outgoing request per imp and per media type, mirroring the Prebid.js adapter.
	for _, imp := range request.Imp {
		impExt, err := extractImpExt(&imp)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		imp.TagID = strings.TrimSpace(impExt.ChannelId)

		for _, singleTypeImp := range splitImpByMediaType(imp) {
			requestCopy := *request
			requestCopy.Imp = []openrtb2.Imp{singleTypeImp}
			requestCopy.Ext = requestExt

			requestJSON, err := jsonutil.Marshal(&requestCopy)
			if err != nil {
				errors = append(errors, fmt.Errorf("marshal bidRequest: %w", err))
				continue
			}

			requests = append(requests, &adapters.RequestData{
				Method:  "POST",
				Uri:     a.endpoint,
				Body:    requestJSON,
				Headers: headers,
				ImpIDs:  []string{singleTypeImp.ID},
			})
		}
	}

	return requests, errors
}

// buildRequestExt merges ext.aniview into the existing request.ext, preserving
// whatever PBS core has put there.
func buildRequestExt(requestExt []byte) ([]byte, error) {
	extMap := map[string]interface{}{}
	if len(requestExt) > 0 {
		if err := jsonutil.Unmarshal(requestExt, &extMap); err != nil {
			return nil, fmt.Errorf("unmarshal request.ext: %w", err)
		}
	}

	extMap["aniview"] = aniviewExt{PBS: 1}

	ext, err := jsonutil.Marshal(extMap)
	if err != nil {
		return nil, fmt.Errorf("marshal request.ext: %w", err)
	}
	return ext, nil
}

// splitImpByMediaType returns one imp per media type, so each outgoing request
// carries a single media type, as the Aniview endpoint expects.
func splitImpByMediaType(imp openrtb2.Imp) []openrtb2.Imp {
	if imp.Video == nil || imp.Banner == nil {
		return []openrtb2.Imp{imp}
	}

	videoImp := imp
	videoImp.Banner = nil
	bannerImp := imp
	bannerImp.Video = nil
	return []openrtb2.Imp{videoImp, bannerImp}
}

func extractImpExt(imp *openrtb2.Imp) (*openrtb_ext.ImpExtAniview, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, fmt.Errorf("unmarshal bidderExt: %w", err)
	}

	var impExt openrtb_ext.ImpExtAniview
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &impExt); err != nil {
		return nil, fmt.Errorf("unmarshal ImpExtAniview: %w", err)
	}

	if strings.TrimSpace(impExt.ChannelId) == "" {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Missing AV_CHANNELID for imp: %s", imp.ID),
		}
	}
	return &impExt, nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var errs []error

	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", responseData.StatusCode),
		}}
	}

	// The exchange may answer a no-bid as 200 with an empty/whitespace body.
	if len(bytes.TrimSpace(responseData.Body)) == 0 {
		return nil, nil
	}

	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("bad server response: %s", err),
		}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(response.SeatBid))

	if response.Cur != "" {
		bidResponse.Currency = response.Cur
	}

	for _, seatBid := range response.SeatBid {
		for i, bid := range seatBid.Bid {
			// Mirror the Prebid.js adapter: a bid without markup or a VAST url is unusable.
			if bid.AdM == "" && bid.NURL == "" {
				continue
			}

			bidType, err := getMediaTypeForBid(&bid)
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

// getMediaTypeForBid resolves the bid media type from bid.mtype, which the
// exchange sets explicitly on every bid.
func getMediaTypeForBid(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Could not define bid type for imp: %s", bid.ImpID),
		}
	}
}
