package tracker

import (
	"errors"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/parser"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/sdk/sdkutils"
)

// Inject Trackers in Video Creative
func injectVideoCreativeTrackers(rctx models.RequestCtx, bid openrtb2.Bid, videoParams []models.OWTracker) (string, string, error) {
	if bid.AdM == "" || len(videoParams) == 0 {
		return "", bid.BURL, errors.New("bid is nil or tracker data is missing")
	}

	skipTracker := false
	if sdkutils.IsSdkBiddingEndpoint(rctx.Endpoint) {
		skipTracker = true
	}

	creative := bid.AdM
	if strings.HasPrefix(creative, models.HTTPProtocol) {
		creative = strings.Replace(models.VastWrapper, models.PartnerURLPlaceholder, creative, -1)
		if skipTracker {
			creative = strings.Replace(creative, models.VASTImpressionURLTemplate, "", -1)
		} else {
			creative = strings.Replace(creative, models.TrackerPlaceholder, videoParams[0].TrackerURL, -1)
		}
		creative = strings.Replace(creative, models.ErrorPlaceholder, videoParams[0].ErrorURL, -1)
		bid.AdM = creative
	} else {
		creative = strings.TrimSpace(creative)
		ti := parser.GetTrackerInjector()
		if err := ti.Parse(creative); err != nil {
			//parsing failed
			return bid.AdM, bid.BURL, errors.New("invalid creative format")
		}
		creative, err := ti.Inject(videoParams, skipTracker)
		if err != nil {
			//injection failure
			return bid.AdM, bid.BURL, errors.New("invalid creative format")
		}

		bid.AdM = creative
	}

	if skipTracker && len(videoParams) > 0 {
		bid.BURL = getBURL(bid.BURL, videoParams[0])
	}

	return bid.AdM, bid.BURL, nil
}
