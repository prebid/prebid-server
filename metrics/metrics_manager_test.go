package metrics

import (
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/appnexus"
	"github.com/prebid/prebid-server/adapters/facebook"
	"github.com/prebid/prebid-server/adapters/index"
	"github.com/prebid/prebid-server/adapters/lifestreet"
	"github.com/prebid/prebid-server/adapters/pubmatic"
	"github.com/prebid/prebid-server/adapters/pulsepoint"
	"github.com/prebid/prebid-server/adapters/rubicon"
	"github.com/prebid/prebid-server/config"
	"testing"
)

var exchanges map[string]adapters.Adapter

func setupExchanges(cfg config.Configuration) {
	exchanges = map[string]adapters.Adapter{
		"appnexus":      appnexus.NewAppNexusAdapter(adapters.DefaultHTTPAdapterConfig, cfg.ExternalURL),
		"districtm":     appnexus.NewAppNexusAdapter(adapters.DefaultHTTPAdapterConfig, cfg.ExternalURL),
		"indexExchange": index.NewIndexAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters["indexexchange"].Endpoint, cfg.Adapters["indexexchange"].UserSyncURL),
		"pubmatic":      pubmatic.NewPubmaticAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters["pubmatic"].Endpoint, cfg.ExternalURL),
		"pulsepoint":    pulsepoint.NewPulsePointAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters["pulsepoint"].Endpoint, cfg.ExternalURL),
		"rubicon": rubicon.NewRubiconAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters["rubicon"].Endpoint,
			cfg.Adapters["rubicon"].XAPI.Username, cfg.Adapters["rubicon"].XAPI.Password, cfg.Adapters["rubicon"].XAPI.Tracker, cfg.Adapters["rubicon"].UserSyncURL),
		"audienceNetwork": facebook.NewFacebookAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters["facebook"].PlatformID, cfg.Adapters["facebook"].UserSyncURL),
		"lifestreet":      lifestreet.NewLifestreetAdapter(adapters.DefaultHTTPAdapterConfig, cfg.ExternalURL),
	}
}

func TestSetupMetrics(t *testing.T) {
	cfg, _ := config.New()
	setupExchanges(*cfg)
	m, _ := SetupMetrics(cfg.Metrics, exchanges)
	if m.GetMetrics() == nil {
		t.Error("Setup of PBSMetrics failed")
	}
}
