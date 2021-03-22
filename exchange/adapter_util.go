package exchange

import (
	"fmt"
	"net/http"

	"github.com/prebid/prebid-server/metrics"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/lifestreet"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func BuildAdapters(client *http.Client, cfg *config.Configuration, infos config.BidderInfos, me metrics.MetricsEngine) (map[openrtb_ext.BidderName]adaptedBidder, []error) {
	exchangeBidders := buildExchangeBiddersLegacy(cfg.Adapters, infos)

	exchangeBiddersModern, errs := buildExchangeBidders(cfg, infos, client, me)
	if len(errs) > 0 {
		return nil, errs
	}

	// Merge legacy and modern bidders, giving priority to the modern bidders.
	for bidderName, bidder := range exchangeBiddersModern {
		exchangeBidders[bidderName] = bidder
	}

	wrapWithMiddleware(exchangeBidders)

	return exchangeBidders, nil
}

func buildExchangeBidders(cfg *config.Configuration, infos config.BidderInfos, client *http.Client, me metrics.MetricsEngine) (map[openrtb_ext.BidderName]adaptedBidder, []error) {
	bidders, errs := buildBidders(cfg.Adapters, infos, newAdapterBuilders())
	if len(errs) > 0 {
		return nil, errs
	}

	exchangeBidders := make(map[openrtb_ext.BidderName]adaptedBidder, len(bidders))
	for bidderName, bidder := range bidders {
		info, infoFound := infos[string(bidderName)]
		if !infoFound {
			errs = append(errs, fmt.Errorf("%v: bidder info not found", bidder))
			continue
		}
		exchangeBidders[bidderName] = adaptBidder(bidder, client, cfg, me, bidderName, info.Debug)
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

		// Ignore Legacy Bidders
		if bidderName == openrtb_ext.BidderLifestreet {
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

			bidderWithInfoEnforcement := adapters.BuildInfoAwareBidder(bidderInstance, info)

			bidders[bidderName] = bidderWithInfoEnforcement
		}
	}

	return bidders, errs
}

func buildExchangeBiddersLegacy(adapterConfig map[string]config.Adapter, infos config.BidderInfos) map[openrtb_ext.BidderName]adaptedBidder {
	bidders := make(map[openrtb_ext.BidderName]adaptedBidder, 2)

	// Lifestreet
	if infos[string(openrtb_ext.BidderLifestreet)].Enabled {
		adapter := lifestreet.NewLifestreetLegacyAdapter(adapters.DefaultHTTPAdapterConfig, adapterConfig[string(openrtb_ext.BidderLifestreet)].Endpoint)
		bidders[openrtb_ext.BidderLifestreet] = adaptLegacyAdapter(adapter)
	}

	return bidders
}

func wrapWithMiddleware(bidders map[openrtb_ext.BidderName]adaptedBidder) {
	for name, bidder := range bidders {
		bidders[name] = addValidatedBidderMiddleware(bidder)
	}
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
	disabledBidders := make(map[string]string)

	for name, info := range infos {
		if !info.Enabled {
			msg := fmt.Sprintf(`Bidder "%s" has been disabled on this instance of Prebid Server. Please work with the PBS host to enable this bidder again.`, name)
			disabledBidders[name] = msg
		}
	}

	return disabledBidders
}
