package exchange

import (
	"fmt"
	"net/http"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/metrics"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func BuildAdapters(client *http.Client, cfg *config.Configuration, infos config.BidderInfos, me metrics.MetricsEngine) (map[openrtb_ext.BidderName]adaptedBidder, []error) {
	bidders, errs := buildBidders(cfg.Adapters, infos, newAdapterBuilders())
	if len(errs) > 0 {
		return nil, errs
	}

	exchangeBidders := make(map[openrtb_ext.BidderName]adaptedBidder, len(bidders))
	for bidderName, bidder := range bidders {
		info := infos[string(bidderName)]
		exchangeBidder := adaptBidder(bidder, client, cfg, me, bidderName, info.Debug)
		exchangeBidder = addValidatedBidderMiddleware(exchangeBidder)
		exchangeBidders[bidderName] = exchangeBidder
	}
	return exchangeBidders, nil
}

func buildBidders(adapterConfig map[string]config.Adapter, infos config.BidderInfos, builders map[openrtb_ext.BidderName]adapters.Builder) (map[openrtb_ext.BidderName]adapters.Bidder, []error) {
	bidders := make(map[openrtb_ext.BidderName]adapters.Bidder)
	var errs []error

	for bidder, cfg := range adapterConfig {
		bidderName, bidderNameFound := openrtb_ext.NormalizeBidderName(bidder)
		if !bidderNameFound {
			errs = append(errs, fmt.Errorf("%v: unknown bidder", bidder))
			continue
		}

		info, infoFound := infos[string(bidderName)]
		if !infoFound {
			errs = append(errs, fmt.Errorf("%v: bidder info not found", bidder))
			continue
		}

		builder, builderFound := builders[bidderName]
		if !builderFound {
			errs = append(errs, fmt.Errorf("%v: builder not registered", bidder))
			continue
		}

		if info.Enabled {
			bidderInstance, builderErr := builder(bidderName, cfg)
			if builderErr != nil {
				errs = append(errs, fmt.Errorf("%v: %v", bidder, builderErr))
				continue
			}
			bidders[bidderName] = adapters.BuildInfoAwareBidder(bidderInstance, info)
		}
	}

	return bidders, errs
}

// GetActiveBidders returns a map of all active bidder names.
func GetActiveBidders(infos config.BidderInfos) map[string]openrtb_ext.BidderName {
	activeBidders := make(map[string]openrtb_ext.BidderName)

	for name, info := range infos {
		if info.Enabled {
			activeBidders[name] = openrtb_ext.BidderName(name)
		}
	}

	return activeBidders
}

// GetDisabledBiddersErrorMessages returns a map of error messages for disabled bidders.
func GetDisabledBiddersErrorMessages(infos config.BidderInfos) map[string]string {
	disabledBidders := map[string]string{
		"lifestreet": `Bidder "lifestreet" is no longer available in Prebid Server. Please update your configuration.`,
	}

	for name, info := range infos {
		if !info.Enabled {
			msg := fmt.Sprintf(`Bidder "%s" has been disabled on this instance of Prebid Server. Please work with the PBS host to enable this bidder again.`, name)
			disabledBidders[name] = msg
		}
	}

	return disabledBidders
}
