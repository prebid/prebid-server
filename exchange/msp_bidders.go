package exchange

import (
	"github.com/golang/glog"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"

	mspPlugin "github.com/prebid/prebid-server/msp/plugin"
)

type PluginBuilder interface {
	Build(openrtb_ext.BidderName, config.Adapter, config.Server) (adapters.Bidder, error)
}

func mspLoadBidderAdapterPlugins(cfg config.BidderInfos) map[openrtb_ext.BidderName]adapters.Builder {
	plugins := make(map[openrtb_ext.BidderName]adapters.Builder)

	for name, bidderInfo := range cfg {
		if bidderInfo.MspSoPath != "" {
			builder, err := mspPlugin.LoadBuilderFromPath[PluginBuilder](name, bidderInfo.MspSoPath)

			if err != nil {
				panic(err)
			}

			bidderName := openrtb_ext.BidderName(name)
			plugins[bidderName] = builder.Build
			glog.Infof("Loaded Bidder Adapter plugin: %s\n", name)
		}
	}

	return plugins
}

func mspAddAdaptersFromPlugins(adapters map[openrtb_ext.BidderName]adapters.Builder, cfg config.BidderInfos) map[openrtb_ext.BidderName]adapters.Builder {
	pluginAdatpers := mspLoadBidderAdapterPlugins(cfg)
	for key, val := range pluginAdatpers {
		adapters[key] = val
	}

	return adapters
}
