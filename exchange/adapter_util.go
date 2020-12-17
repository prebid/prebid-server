package exchange

import (
	"fmt"
	"net/http"

	"github.com/prebid/prebid-server/metrics"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/ix"
	"github.com/prebid/prebid-server/adapters/lifestreet"
	"github.com/prebid/prebid-server/adapters/pulsepoint"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func BuildAdapters(client *http.Client, cfg *config.Configuration, infos adapters.BidderInfos, me metrics.MetricsEngine) (map[openrtb_ext.BidderName]adaptedBidder, []error) {
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

func buildExchangeBidders(cfg *config.Configuration, infos adapters.BidderInfos, client *http.Client, me metrics.MetricsEngine) (map[openrtb_ext.BidderName]adaptedBidder, []error) {
	bidders, errs := buildBidders(cfg.Adapters, infos, newAdapterBuilders())
	if len(errs) > 0 {
		return nil, errs
	}

	exchangeBidders := make(map[openrtb_ext.BidderName]adaptedBidder, len(bidders))
	for bidderName, bidder := range bidders {
		exchangeBidders[bidderName] = adaptBidder(bidder, client, cfg, me, bidderName)
	}

	return exchangeBidders, nil

}

func buildBidders(adapterConfig map[string]config.Adapter, infos adapters.BidderInfos, builders map[openrtb_ext.BidderName]adapters.Builder) (map[openrtb_ext.BidderName]adapters.Bidder, []error) {
	bidders := make(map[openrtb_ext.BidderName]adapters.Bidder)
	var errs []error

	for bidder, cfg := range adapterConfig {
		bidderName, bidderNameFound := openrtb_ext.NormalizeBidderName(bidder)
		if !bidderNameFound {
			errs = append(errs, fmt.Errorf("%v: unknown bidder", bidder))
			continue
		}

		// Ignore Legacy Bidders
		if bidderName == openrtb_ext.BidderIx || bidderName == openrtb_ext.BidderLifestreet || bidderName == openrtb_ext.BidderPulsepoint {
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

		if info.Status == adapters.StatusActive {
			bidderInstance, builderErr := builder(bidderName, cfg)
			if builderErr != nil {
				errs = append(errs, fmt.Errorf("%v: %v", bidder, builderErr))
				continue
			}

			bidderWithInfoEnforcement := adapters.EnforceBidderInfo(bidderInstance, info)

			bidders[bidderName] = bidderWithInfoEnforcement
		}
	}

	return bidders, errs
}

func buildExchangeBiddersLegacy(adapterConfig map[string]config.Adapter, infos adapters.BidderInfos) map[openrtb_ext.BidderName]adaptedBidder {
	bidders := make(map[openrtb_ext.BidderName]adaptedBidder, 3)

	// Index
	if infos[string(openrtb_ext.BidderIx)].Status == adapters.StatusActive {
		adapter := ix.NewIxLegacyAdapter(adapters.DefaultHTTPAdapterConfig, adapterConfig[string(openrtb_ext.BidderIx)].Endpoint)
		bidders[openrtb_ext.BidderIx] = adaptLegacyAdapter(adapter)
	}

	// Lifestreet
	if infos[string(openrtb_ext.BidderLifestreet)].Status == adapters.StatusActive {
		adapter := lifestreet.NewLifestreetLegacyAdapter(adapters.DefaultHTTPAdapterConfig, adapterConfig[string(openrtb_ext.BidderLifestreet)].Endpoint)
		bidders[openrtb_ext.BidderLifestreet] = adaptLegacyAdapter(adapter)
	}

	// Pulsepoint
	if infos[string(openrtb_ext.BidderPulsepoint)].Status == adapters.StatusActive {
		adapter := pulsepoint.NewPulsePointLegacyAdapter(adapters.DefaultHTTPAdapterConfig, adapterConfig[string(openrtb_ext.BidderPulsepoint)].Endpoint)
		bidders[openrtb_ext.BidderPulsepoint] = adaptLegacyAdapter(adapter)
	}

	return bidders
}

func wrapWithMiddleware(bidders map[openrtb_ext.BidderName]adaptedBidder) {
	for name, bidder := range bidders {
		bidders[name] = addValidatedBidderMiddleware(bidder)
	}
}

// GetActiveBidders returns a map of all active bidder names.
func GetActiveBidders(infos adapters.BidderInfos) map[string]openrtb_ext.BidderName {
	activeBidders := make(map[string]openrtb_ext.BidderName)

	for name, info := range infos {
		if info.Status != adapters.StatusDisabled {
			activeBidders[name] = openrtb_ext.BidderName(name)
		}
	}

	return activeBidders
}

// GetDisabledBiddersErrorMessages returns a map of error messages for disabled bidders.
func GetDisabledBiddersErrorMessages(infos adapters.BidderInfos) map[string]string {
	disabledBidders := make(map[string]string)

	for bidderName, bidderInfo := range infos {
		if bidderInfo.Status == adapters.StatusDisabled {
			msg := fmt.Sprintf(`Bidder "%s" has been disabled on this instance of Prebid Server. Please work with the PBS host to enable this bidder again.`, bidderName)
			disabledBidders[bidderName] = msg
		}
	}

	return disabledBidders
}
