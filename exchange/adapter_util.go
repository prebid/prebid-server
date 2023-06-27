package exchange

import (
	"fmt"
	"net/http"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/metrics"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func BuildAdapters(client *http.Client, cfg *config.Configuration, infos config.BidderInfos, me metrics.MetricsEngine) (map[openrtb_ext.BidderName]AdaptedBidder, []error) {
	server := config.Server{ExternalUrl: cfg.ExternalURL, GvlID: cfg.GDPR.HostVendorID, DataCenter: cfg.DataCenter}
	bidders, errs := buildBidders(infos, newAdapterBuilders(), server)

	if len(errs) > 0 {
		return nil, errs
	}

	exchangeBidders := make(map[openrtb_ext.BidderName]AdaptedBidder, len(bidders))
	for bidderName, bidder := range bidders {
		info := infos[string(bidderName)]
		exchangeBidder := AdaptBidder(bidder, client, cfg, me, bidderName, info.Debug, info.EndpointCompression)
		exchangeBidder = addValidatedBidderMiddleware(exchangeBidder)
		exchangeBidders[bidderName] = exchangeBidder
	}
	return exchangeBidders, nil
}

func buildBidders(infos config.BidderInfos, builders map[openrtb_ext.BidderName]adapters.Builder, server config.Server) (map[openrtb_ext.BidderName]adapters.Bidder, []error) {
	bidders := make(map[openrtb_ext.BidderName]adapters.Bidder)
	var errs []error

	for bidder, info := range infos {
		bidderName, bidderNameFound := openrtb_ext.NormalizeBidderName(bidder)
		if !bidderNameFound {
			errs = append(errs, fmt.Errorf("%v: unknown bidder", bidder))
			continue
		}

		builder, builderFound := builders[bidderName]
		if !builderFound {
			errs = append(errs, fmt.Errorf("%v: builder not registered", bidder))
			continue
		}

		if info.IsEnabled() {
			adapterInfo := buildAdapterInfo(info)
			bidderInstance, builderErr := builder(bidderName, adapterInfo, server)

			if builderErr != nil {
				errs = append(errs, fmt.Errorf("%v: %v", bidder, builderErr))
				continue
			}
			bidders[bidderName] = adapters.BuildInfoAwareBidder(bidderInstance, info)
		}
	}
	return bidders, errs
}

func buildAdapterInfo(bidderInfo config.BidderInfo) config.Adapter {
	adapter := config.Adapter{}
	adapter.Endpoint = bidderInfo.Endpoint
	adapter.ExtraAdapterInfo = bidderInfo.ExtraAdapterInfo
	adapter.PlatformID = bidderInfo.PlatformID
	adapter.AppSecret = bidderInfo.AppSecret
	adapter.XAPI = bidderInfo.XAPI
	return adapter
}

// GetActiveBidders returns a map of all active bidder names.
func GetActiveBidders(infos config.BidderInfos) map[string]openrtb_ext.BidderName {
	activeBidders := make(map[string]openrtb_ext.BidderName)

	for name, info := range infos {
		if info.IsEnabled() {
			activeBidders[name] = openrtb_ext.BidderName(name)
		}
	}

	return activeBidders
}

// GetDisabledBiddersErrorMessages returns a map of error messages for disabled bidders.
func GetDisabledBiddersErrorMessages(infos config.BidderInfos) map[string]string {
	disabledBidders := map[string]string{
		"lifestreet":     `Bidder "lifestreet" is no longer available in Prebid Server. Please update your configuration.`,
		"adagio":         `Bidder "adagio" is no longer available in Prebid Server. Please update your configuration.`,
		"somoaudience":   `Bidder "somoaudience" is no longer available in Prebid Server. Please update your configuration.`,
		"yssp":           `Bidder "yssp" is no longer available in Prebid Server. If you're looking to use the Yahoo SSP adapter, please rename it to "yahooAds" in your configuration.`,
		"andbeyondmedia": `Bidder "andbeyondmedia" is no longer available in Prebid Server. If you're looking to use the AndBeyond.Media SSP adapter, please rename it to "beyondmedia" in your configuration.`,
		"oftmedia":       `Bidder "oftmedia" is no longer available in Prebid Server. Please update your configuration.`,
		"groupm":         `Bidder "groupm" is no longer available in Prebid Server. Please update your configuration.`,
		"verizonmedia":   `Bidder "verizonmedia" is no longer available in Prebid Server. Please update your configuration.`,
	}

	for name, info := range infos {
		if info.Disabled {
			msg := fmt.Sprintf(`Bidder "%s" has been disabled on this instance of Prebid Server. Please work with the PBS host to enable this bidder again.`, name)
			disabledBidders[name] = msg
		}
	}

	return disabledBidders
}
