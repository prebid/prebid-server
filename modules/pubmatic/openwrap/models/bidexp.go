package models

// OmitTrackerBidExp is true for Google Ad Manager SDK bidding (14) or AdMob SDK bidding (16) when bid.ext.bidexp_enf
// is not 1. Missing JSON key unmarshals as 0, same as legacy explicit 0. Only value 1 opts into mirroring bid.exp /
// bidexp_enf on the impression tracker for those sub-integrations.
func OmitTrackerBidExp(rctx RequestCtx, bidExpEnf int) bool {
	if bidExpEnf == 1 {
		return false
	}
	if rctx.AppSubIntegrationPath == nil || *rctx.AppSubIntegrationPath < 0 {
		return false
	}
	return *rctx.AppSubIntegrationPath == AppSubIntegrationPathIDAdMobSDKBidding ||
		*rctx.AppSubIntegrationPath == AppSubIntegrationPathIDGoogleAdManagerSDKBidding
}
