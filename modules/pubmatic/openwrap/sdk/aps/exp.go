package aps

import (
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
)

const (
	apsBannerExpSeconds = 600  // 10 minutes — APS display / banner-only
	apsVideoExpSeconds  = 3600 // 1 hour — APS video-only
)

// apsFormatDefaultExp returns the APS default imp/bid expiry for banner-only or video-only impressions.
func apsFormatDefaultExp(imp *openrtb2.Imp) (int64, bool) {
	if imp == nil {
		return 0, false
	}
	hasBanner := imp.Banner != nil
	hasVideo := imp.Video != nil
	switch {
	case hasBanner && !hasVideo:
		return apsBannerExpSeconds, true
	case hasVideo && !hasBanner:
		return apsVideoExpSeconds, true
	default:
		return 0, false
	}
}

// setAPSImpExpIfMissing sets imp.exp from APS format defaults when the S2S request omitted it.
// Call before OWSDK signal merge so format detection uses the original S2S impression object.
func setAPSImpExpIfMissing(imp *openrtb2.Imp) {
	if imp == nil {
		return
	}

	formatDefault, hasFormatDefault := apsFormatDefaultExp(imp)
	if !hasFormatDefault {
		return
	}

	imp.Exp = formatDefault
}

// applyAPSBidExpIfMissing sets bid.exp from outbound imp.exp on the request when the partner omits it.
func applyAPSBidExpIfMissing(rctx models.RequestCtx, bidResponse *openrtb2.BidResponse) {
	if bidResponse == nil || len(rctx.ImpBidCtx) == 0 {
		return
	}

	for i := range bidResponse.SeatBid {
		for j := range bidResponse.SeatBid[i].Bid {
			bid := &bidResponse.SeatBid[i].Bid[j]
			if bid.Exp > 0 {
				continue
			}
			impID, _ := models.GetImpressionID(bid.ImpID)
			if impCtx, ok := rctx.ImpBidCtx[impID]; ok && impCtx.Exp > 0 {
				bid.Exp = impCtx.Exp
			}
		}
	}
}
