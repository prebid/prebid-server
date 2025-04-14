package exchange

import (
	"fmt"
	"net/http"

	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func BuildAdapters(client *http.Client, cfg *config.Configuration, infos config.BidderInfos, me metrics.MetricsEngine) (map[openrtb_ext.BidderName]AdaptedBidder, map[openrtb_ext.BidderName]struct{}, []error) {
	server := config.Server{ExternalUrl: cfg.ExternalURL, GvlID: cfg.GDPR.HostVendorID, DataCenter: cfg.DataCenter}
	bidders, singleFormatBidders, errs := buildBidders(infos, newAdapterBuilders(), server)

	if len(errs) > 0 {
		return nil, nil, errs
	}

	exchangeBidders := make(map[openrtb_ext.BidderName]AdaptedBidder, len(bidders))
	for bidderName, bidder := range bidders {
		info := infos[string(bidderName)]
		exchangeBidder := AdaptBidder(bidder, client, cfg, me, bidderName, info.Debug, info.EndpointCompression)
		exchangeBidder = addValidatedBidderMiddleware(exchangeBidder)
		exchangeBidders[bidderName] = exchangeBidder
	}
	return exchangeBidders, singleFormatBidders, nil
}

func buildBidders(infos config.BidderInfos, builders map[openrtb_ext.BidderName]adapters.Builder, server config.Server) (map[openrtb_ext.BidderName]adapters.Bidder, map[openrtb_ext.BidderName]struct{}, []error) {
	bidders := make(map[openrtb_ext.BidderName]adapters.Bidder)
	singleFormatBidders := make(map[openrtb_ext.BidderName]struct{})
	var errs []error

	for bidder, info := range infos {
		bidderName, bidderNameFound := openrtb_ext.NormalizeBidderName(bidder)
		if !bidderNameFound {
			errs = append(errs, fmt.Errorf("%v: unknown bidder", bidder))
			continue
		}

		if len(info.AliasOf) > 0 {
			if err := setAliasBuilder(info, builders, bidderName); err != nil {
				errs = append(errs, fmt.Errorf("%v: failed to set alias builder: %v", bidder, err))
				continue
			}
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
			if !adapters.IsMultiFormatSupported(info) {
				singleFormatBidders[bidderName] = struct{}{}
			}
		}
	}
	return bidders, singleFormatBidders, errs
}

func setAliasBuilder(info config.BidderInfo, builders map[openrtb_ext.BidderName]adapters.Builder, bidderName openrtb_ext.BidderName) error {
	parentBidderName, parentBidderFound := openrtb_ext.NormalizeBidderName(info.AliasOf)
	if !parentBidderFound {
		return fmt.Errorf("unknown parent bidder: %v for alias: %v", info.AliasOf, bidderName)
	}

	builder, builderFound := builders[parentBidderName]
	if !builderFound {
		return fmt.Errorf("%v: parent builder not registered", parentBidderName)
	}
	builders[bidderName] = builder
	return nil
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

func GetDisabledBidderWarningMessages(infos config.BidderInfos) map[string]string {
	removed := map[string]string{
		"lifestreet":      `Bidder "lifestreet" is no longer available in Prebid Server. Please update your configuration.`,
		"somoaudience":    `Bidder "somoaudience" is no longer available in Prebid Server. Please update your configuration.`,
		"yssp":            `Bidder "yssp" is no longer available in Prebid Server. If you're looking to use the Yahoo SSP adapter, please rename it to "yahooAds" in your configuration.`,
		"andbeyondmedia":  `Bidder "andbeyondmedia" is no longer available in Prebid Server. If you're looking to use the AndBeyond.Media SSP adapter, please rename it to "beyondmedia" in your configuration.`,
		"oftmedia":        `Bidder "oftmedia" is no longer available in Prebid Server. Please update your configuration.`,
		"groupm":          `Bidder "groupm" is no longer available in Prebid Server. Please update your configuration.`,
		"verizonmedia":    `Bidder "verizonmedia" is no longer available in Prebid Server. Please update your configuration.`,
		"brightroll":      `Bidder "brightroll" is no longer available in Prebid Server. Please update your configuration.`,
		"engagebdr":       `Bidder "engagebdr" is no longer available in Prebid Server. Please update your configuration.`,
		"ninthdecimal":    `Bidder "ninthdecimal" is no longer available in Prebid Server. Please update your configuration.`,
		"kubient":         `Bidder "kubient" is no longer available in Prebid Server. Please update your configuration.`,
		"applogy":         `Bidder "applogy" is no longer available in Prebid Server. Please update your configuration.`,
		"rhythmone":       `Bidder "rhythmone" is no longer available in Prebid Server. Please update your configuration.`,
		"nanointeractive": `Bidder "nanointeractive" is no longer available in Prebid Server. Please update your configuration.`,
		"bizzclick":       `Bidder "bizzclick" is no longer available in Prebid Server. Please update your configuration. "bizzclick" has been renamed to "blasto".`,
		"liftoff":         `Bidder "liftoff" is no longer available in Prebid Server. If you're looking to use the Vungle Exchange adapter, please rename it to "vungle" in your configuration.`,
	}

	return mergeRemovedAndDisabledBidderWarningMessages(removed, infos)
}

func mergeRemovedAndDisabledBidderWarningMessages(removed map[string]string, infos config.BidderInfos) map[string]string {
	disabledBidders := removed

	for name, info := range infos {
		if info.Disabled {
			msg := fmt.Sprintf(`Bidder "%s" has been disabled on this instance of Prebid Server. Please work with the PBS host to enable this bidder again.`, name)
			disabledBidders[name] = msg
		}
	}

	return disabledBidders
}
