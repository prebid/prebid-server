package madsense

import (
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

func parseImpExt(imp *openrtb2.Imp) (*openrtb_ext.ExtImpMadSense, error) {
	var bidderExt adapters.ExtImpBidder

	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Error while decoding imp.Ext, err: %v", err),
		}
	}

	ext := openrtb_ext.ExtImpMadSense{}
	err := jsonutil.Unmarshal(bidderExt.Bidder, &ext)
	if err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Error while decoding bidderExt.Bidder, err: %v", err),
		}
	}

	return &ext, nil
}

func getHeaders(request *openrtb2.BidRequest) http.Header {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json;charset=utf-8")
	headers.Set("Accept", "application/json")
	headers.Set("X-Openrtb-Version", "2.6")

	if request.Device != nil {
		if len(request.Device.UA) > 0 {
			headers.Set("User-Agent", request.Device.UA)
		}

		if len(request.Device.IP) > 0 {
			headers.Set("X-Forwarded-For", request.Device.IP)
		}

		if len(request.Device.IPv6) > 0 {
			headers.Set("X-Forwarded-For", request.Device.IPv6)
		}
	}

	if request.Site != nil {
		if request.Site.Domain != "" {
			headers.Set("Origin", request.Site.Domain)
		}
		if request.Site.Ref != "" {
			headers.Set("Referer", request.Site.Ref)
		}
	}
	return headers
}

func getTypedBidFromBid(bid *openrtb2.Bid) (*adapters.TypedBid, error) {
	bidType, err := getMediaTypeForBid(bid)
	if err != nil {
		return nil, err
	}

	var bidVideo *openrtb_ext.ExtBidPrebidVideo
	if bidType == openrtb_ext.BidTypeVideo {
		bidVideo = &openrtb_ext.ExtBidPrebidVideo{}
		if len(bid.Cat) > 0 {
			bidVideo.PrimaryCategory = bid.Cat[0]
		}
		if bid.Dur > 0 {
			bidVideo.Duration = int(bid.Dur)
		}
	}
	return &adapters.TypedBid{
		Bid:      bid,
		BidType:  bidType,
		BidVideo: bidVideo,
	}, nil
}

func getMediaTypeForBid(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("MType %v not supported", bid.MType),
		}
	}
}
