package tracker

import (
	"encoding/json"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models/adunitconfig"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/sdk/sdkutils"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func injectBannerTracker(rctx models.RequestCtx, tracker models.OWTracker, bid openrtb2.Bid, seat string, pixels []adunitconfig.UniversalPixel) (string, string) {
	if sdkutils.IsSdkBiddingEndpoint(rctx.Endpoint) {
		return bid.AdM, getBURL(bid.BURL, tracker)
	}

	var replacedTrackerStr, trackerFormat string
	trackerFormat = models.TrackerCallWrap
	if tracker.IsOMEnabled {
		trackerFormat = models.TrackerCallWrapOMActive
	}
	replacedTrackerStr = strings.Replace(trackerFormat, "${escapedUrl}", tracker.TrackerURL, 1)
	adm := applyTBFFeature(rctx, bid, replacedTrackerStr)
	return appendUPixelinBanner(adm, pixels), bid.BURL
}

// append universal pixels in creative based on conditions
func appendUPixelinBanner(adm string, universalPixel []adunitconfig.UniversalPixel) string {
	if universalPixel == nil {
		return adm
	}

	for _, pixelVal := range universalPixel {
		if pixelVal.Pos == models.PixelPosAbove {
			adm = pixelVal.Pixel + adm
			continue
		}
		adm = adm + pixelVal.Pixel
	}
	return adm
}

// TrackerWithOM checks for OM active condition
func trackerWithOM(rctx models.RequestCtx, prebidPartnerName string, dspID int, bidExt json.RawMessage, adformat string) bool {
	if rctx.Platform != models.PLATFORM_APP || adformat != models.Banner {
		return false
	}

	// check for OM active for inview enabled publishers and performance DSPs, handle DV360 separately
	if prebidPartnerName == string(openrtb_ext.BidderPubmatic) {
		if dspID == models.DspId_DV360 {
			return true
		}
		_, isPresent := rctx.InViewEnabledPublishers[rctx.PubID]
		_, isPerformanceDSP := rctx.PerformanceDSPs[dspID]
		return isPresent && isPerformanceDSP
	}

	// check for OM active for other bidders
	if _, isOMEnabledBidder := rctx.ImpCountingMethodEnabledBidders[prebidPartnerName]; isOMEnabledBidder {
		return true
	}

	// check for OM active for other bidders based on imp-level flag
	impCountingMethodFlag, err := jsonparser.GetInt(bidExt, models.ImpCountingMethod)
	if err != nil {
		return false
	}
	return impCountingMethodFlag == 1
}

// applyTBFFeature adds the tracker before or after the actual bid.Adm
// If TBF feature is applicable based on database-configuration for
// given pub-prof combination then injects the tracker before adm
// else injects the tracker after adm.
func applyTBFFeature(rctx models.RequestCtx, bid openrtb2.Bid, tracker string) string {
	if rctx.IsTBFFeatureEnabled {
		return tracker + bid.AdM
	}
	return bid.AdM + tracker
}
