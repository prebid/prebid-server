package metrics

import (
	"github.com/magiconair/properties/assert"
	"github.com/prebid/prebid-server/config"
	"github.com/rcrowley/go-metrics"
	"testing"
	"time"
)

const value int64 = 2

func TestAllCounterMetrics(t *testing.T) {
	acm := AllCounterMetrics{}
	cfg, _ := config.New()
	setupExchanges(*cfg)
	acm.Setup(metrics.NewPrefixedRegistry(""), exchanges)
	if len(acm.accountMetrics) <= 0 && len(acm.adapterMetrics) <= 0 {
		t.Error("Setup of AllCounterMetrics fail")
	}
}

func TestAccountCounterMetrics_IncRequest(t *testing.T) {
	acm := AccountCounterMetrics{}
	acm.RequestCounter = metrics.NewCounter()
	acm.IncRequest(value)
	assert.Equal(t, acm.RequestCounter.Count(), value, "AccountCounterMetrics: RequestCounter failed")
}

func TestAccountCounterMetrics_IncBidsReceived(t *testing.T) {
	acm := AccountCounterMetrics{}
	acm.BidsReceivedCounter = metrics.NewCounter()
	acm.IncBidsReceived(value)
	assert.Equal(t, acm.BidsReceivedCounter.Count(), value, "AccountCounterMetrics: BidsReceivedCounter failed")
}

func TestAccountCounterMetrics_UpdatePriceHistogram(t *testing.T) {
	acm := AccountCounterMetrics{}
	acm.PriceHistogram = metrics.GetOrRegisterHistogram("account.rubicon.prices", metrics.NewPrefixedRegistry(""), metrics.NewExpDecaySample(1028, 0.015))
	acm.UpdatePriceHistogram(2000)
	assert.Equal(t, acm.PriceHistogram.Count(), int64(1), "AccountCounterMetrics: PriceHistogram failed")
}

func TestAllCounterMetrics_IncAppRequest(t *testing.T) {
	acm := AllCounterMetrics{}
	acm.mAppRequestCounter = metrics.NewCounter()
	acm.IncAppRequest(value)
	assert.Equal(t, acm.mAppRequestCounter.Count(), value, "AllCounterMetrics: AppRequestCounter failed")
}

func TestAdapterCounterMetrics_IncBidsReceived(t *testing.T) {
	acm := AdapterCounterMetrics{}
	acm.BidsReceivedCounter = metrics.NewCounter()
	acm.IncBidsReceived(value)
	assert.Equal(t, acm.BidsReceivedCounter.Count(), value, "AdapterCounterMetrics: BidsReceivedCounter failed")
}

func TestAdapterCounterMetrics_IncError(t *testing.T) {
	acm := AdapterCounterMetrics{}
	acm.ErrorCounter = metrics.NewCounter()
	acm.IncError(value)
	assert.Equal(t, acm.ErrorCounter.Count(), value, "AdapterCounterMetrics: ErrorCounter failed")
}

func TestAdapterCounterMetrics_IncNoBid(t *testing.T) {
	acm := AdapterCounterMetrics{}
	acm.NoBidCounter = metrics.NewCounter()
	acm.IncNoBid(value)
	assert.Equal(t, acm.NoBidCounter.Count(), value, "AdapterCounterMetrics: NoBidCounter failed")
}

func TestAdapterCounterMetrics_IncNoCookie(t *testing.T) {
	acm := AdapterCounterMetrics{}
	acm.NoCookieCounter = metrics.NewCounter()
	acm.IncNoCookie(value)
	assert.Equal(t, acm.NoCookieCounter.Count(), value, "AdapterCounterMetrics: NoCookieCounter failed")
}

func TestAdapterCounterMetrics_IncRequest(t *testing.T) {
	acm := AdapterCounterMetrics{}
	acm.RequestCounter = metrics.NewCounter()
	acm.IncRequest(value)
	assert.Equal(t, acm.RequestCounter.Count(), value, "AdapterCounterMetrics: RequestCounter failed")
}

func TestAdapterCounterMetrics_IncTimeOut(t *testing.T) {
	acm := AdapterCounterMetrics{}
	acm.TimeoutCounter = metrics.NewCounter()
	acm.IncTimeOut(value)
	assert.Equal(t, acm.TimeoutCounter.Count(), value, "AdapterCounterMetrics: TimeoutCounter failed")
}

func TestAdapterCounterMetrics_UpdatePriceHistogram(t *testing.T) {
	acm := AdapterCounterMetrics{}
	acm.PriceHistogram = metrics.NewHistogram(metrics.NewExpDecaySample(1028, 0.015))
	acm.UpdatePriceHistogram(2000)
	assert.Equal(t, acm.PriceHistogram.Count(), int64(1), "AdapterCounterMetrics: Price Histogram failed")
}

func TestAdapterCounterMetrics_UpdateRequestTimerSince(t *testing.T) {
	acm := AdapterCounterMetrics{}
	acm.RequestTimer = metrics.GetOrRegisterTimer("x.y.requesttime", metrics.NewPrefixedRegistry(""))
	acm.UpdateRequestTimerSince(time.Now())
	assert.Equal(t, acm.RequestTimer.Count(), int64(1), "AdapterCounterMetrics: RequestTimer failed")
}

func TestAllCounterMetrics_IncCookieSync(t *testing.T) {
	acm := AllCounterMetrics{}
	acm.mCookieSyncCounter = metrics.NewCounter()
	acm.IncCookieSync(value)
	assert.Equal(t, acm.mCookieSyncCounter.Count(), value, "AllCounterMetrics: CookieSyncCounter failed")
}

func TestAllCounterMetrics_IncError(t *testing.T) {
	acm := AllCounterMetrics{}
	acm.mErrorCounter = metrics.NewCounter()
	acm.IncError(value)
	assert.Equal(t, acm.mErrorCounter.Count(), value, "AllCounterMetrics: ErrorCounter failed")
}

func TestAllCounterMetrics_IncNoCookie(t *testing.T) {
	acm := AllCounterMetrics{}
	acm.mNoCookieCounter = metrics.NewCounter()
	acm.IncNoCookie(value)
	assert.Equal(t, acm.mNoCookieCounter.Count(), value, "AllCounterMetrics: NoCookieCounter failed")
}

func TestAllCounterMetrics_IncRequest(t *testing.T) {
	acm := AllCounterMetrics{}
	acm.mRequestCounter = metrics.NewCounter()
	acm.IncRequest(value)
	assert.Equal(t, acm.mRequestCounter.Count(), value, "AllCounterMetrics: RequestCounter failed")
}

func TestAllCounterMetrics_IncSafariNoCookie(t *testing.T) {
	acm := AllCounterMetrics{}
	acm.mSafariNoCookieCounter = metrics.NewCounter()
	acm.IncSafariNoCookie(value)
	assert.Equal(t, acm.mSafariNoCookieCounter.Count(), value, "AllCounterMetrics: SafariNoCookieCounter failed")
}

func TestAllCounterMetrics_IncSafariRequest(t *testing.T) {
	acm := AllCounterMetrics{}
	acm.mSafariRequestCounter = metrics.NewCounter()
	acm.IncSafariRequest(value)
	assert.Equal(t, acm.mSafariRequestCounter.Count(), value, "AllCounterMetrics: SafariRequest failed")
}

func TestAllCounterMetrics_UpdateRequestTimerSince(t *testing.T) {
	acm := AllCounterMetrics{}
	acm.mRequestTimer = metrics.NewTimer()
	acm.UpdateRequestTimerSince(time.Now())
	assert.Equal(t, acm.mRequestTimer.Count(), int64(1), "AllCounterMetrics: RequestTimer failed")
}
