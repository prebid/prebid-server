package pbsmetrics

import (
	"fmt"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/rcrowley/go-metrics"
)

func TestNewMetrics(t *testing.T) {
	registry := metrics.NewRegistry()
	m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus, openrtb_ext.BidderRubicon})

	ensureContains(t, registry, "requests", m.RequestMeter)
	ensureContains(t, registry, "app_requests", m.AppRequestMeter)
	ensureContains(t, registry, "no_cookie_requests", m.NoCookieMeter)
	ensureContains(t, registry, "safari_requests", m.SafariRequestMeter)
	ensureContains(t, registry, "safari_no_cookie_requests", m.SafariNoCookieMeter)
	ensureContains(t, registry, "error_requests", m.ErrorMeter)
	ensureContains(t, registry, "request_time", m.RequestTimer)
	ensureContains(t, registry, "ortb_requests", m.ORTBRequestMeter)
	ensureContainsAdapterMetrics(t, registry, "adapter.appnexus", m.AdapterMetrics["appnexus"])
	ensureContainsAdapterMetrics(t, registry, "adapter.rubicon", m.AdapterMetrics["rubicon"])
}

func ensureMissing(t *testing.T, registry metrics.Registry, name string) {
	t.Helper()
	if registry.Get(name) != nil {
		t.Errorf("Found unexpected metric in registry: %s", name)
	}
}

func ensureContains(t *testing.T, registry metrics.Registry, name string, metric interface{}) {
	t.Helper()
	if registry.Get(name) != metric {
		t.Errorf("Bad value stored at metric %s.", name)
	}
}

func ensureContainsAdapterMetrics(t *testing.T, registry metrics.Registry, name string, adapterMetrics *AdapterMetrics) {
	t.Helper()
	ensureContains(t, registry, fmt.Sprintf("%s.no_cookie_requests", name), adapterMetrics.NoCookieMeter)
	ensureContains(t, registry, fmt.Sprintf("%s.error_requests", name), adapterMetrics.ErrorMeter)
	ensureContains(t, registry, fmt.Sprintf("%s.requests", name), adapterMetrics.RequestMeter)
	ensureContains(t, registry, fmt.Sprintf("%s.no_bid_requests", name), adapterMetrics.NoBidMeter)
	ensureContains(t, registry, fmt.Sprintf("%s.timeout_requests", name), adapterMetrics.TimeoutMeter)
	ensureContains(t, registry, fmt.Sprintf("%s.request_time", name), adapterMetrics.RequestTimer)
	ensureContains(t, registry, fmt.Sprintf("%s.prices", name), adapterMetrics.PriceHistogram)
}
