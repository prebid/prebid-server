package pbsmetrics

import (
	"testing"
	"github.com/rcrowley/go-metrics"
	"fmt"
)

func TestNewMetrics(t *testing.T) {
	m := NewMetrics([]string{"appnexus", "rubicon"})
	registry := m.metricsRegistry

	ensureContains(t, registry, "requests", m.RequestMeter)
	ensureContains(t, registry, "app_requests", m.AppRequestMeter)
	ensureContains(t, registry, "no_cookie_requests", m.NoCookieMeter)
	ensureContains(t, registry, "safari_requests", m.SafariRequestMeter)
	ensureContains(t, registry, "safari_no_cookie_requests", m.SafariNoCookieMeter)
	ensureContains(t, registry, "error_requests", m.ErrorMeter)
	ensureContains(t, registry, "invalid_requests", m.InvalidMeter)
	ensureContains(t, registry, "request_time", m.RequestTimer)
	ensureContains(t, registry, "cookie_sync_requests", m.CookieSyncMeter)
	ensureContains(t, registry, "usersync.bad_requests", m.UserSyncMetrics.BadRequestMeter)
	ensureContains(t, registry, "usersync.opt_outs", m.UserSyncMetrics.OptOutMeter)
	ensureContainsAdapterMetrics(t, registry, "adapter.appnexus", m.AdapterMetrics["appnexus"])
	ensureContainsAdapterMetrics(t, registry, "adapter.rubicon", m.AdapterMetrics["rubicon"])
}

func TestLazyLoadUsersyncMetrics(t *testing.T) {
	m := NewMetrics([]string{"appnexus", "rubicon"})
	registry := m.metricsRegistry
	m.UserSyncMetrics.SuccessMeter("appnexus") // Call this twice to make sure we get the same meter
	ensureContains(t, registry, "usersync.appnexus.sets", m.UserSyncMetrics.SuccessMeter("appnexus"))
	ensureMissing(t, registry, "usersync.rubicon.sets")
}

func TestLazyLoadAccountMetrics(t *testing.T) {
	m := NewMetrics([]string{"appnexus", "rubicon"})
	registry := m.metricsRegistry
	m.GetAccountMetrics("foo") // Call this twice to make sure we get the same meter
	ensureContainsAccountMetrics(t, registry, "account.foo", m.GetAccountMetrics("foo"))
	ensureContainsAdapterMetrics(t, registry, "account.foo.appnexus", m.GetAccountMetrics("foo").AdapterMetrics["appnexus"])
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

func ensureContainsAccountMetrics(t *testing.T, registry metrics.Registry, name string, accountMetrics *AccountMetrics) {
	t.Helper()
	ensureContains(t, registry, fmt.Sprintf("%s.requests", name), accountMetrics.RequestMeter)
	ensureContains(t, registry, fmt.Sprintf("%s.bids_received", name), accountMetrics.BidsReceivedMeter)
	ensureContains(t, registry, fmt.Sprintf("%s.prices", name), accountMetrics.PriceHistogram)
}