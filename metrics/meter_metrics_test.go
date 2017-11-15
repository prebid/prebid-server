package metrics

import (
	"github.com/magiconair/properties/assert"
	"github.com/prebid/prebid-server/config"
	"github.com/rcrowley/go-metrics"
	"testing"
	"time"
)

func TestAllMeterMetrics(t *testing.T) {
	acm := AllMeterMetrics{}
	cfg, _ := config.New()
	setupExchanges(*cfg)
	acm.Setup(metrics.NewPrefixedRegistry(""), exchanges)
	if len(acm.accountMetrics) <= 0 && len(acm.adapterMetrics) <= 0 {
		t.Error("Setup of AllMeterMetrics fail")
	}
}

func TestAccountMeterMetrics_IncRequest(t *testing.T) {
	acm := AccountMeterMetrics{}
	acm.RequestMeter = metrics.NewMeter()
	acm.IncRequest(value)
	assert.Equal(t, acm.RequestMeter.Count(), value, "AccountMeterMetrics: RequestMeter failed")
}

func TestAccountMeterMetrics_IncBidsReceived(t *testing.T) {
	acm := AccountMeterMetrics{}
	acm.BidsReceivedMeter = metrics.NewMeter()
	acm.IncBidsReceived(value)
	assert.Equal(t, acm.BidsReceivedMeter.Count(), value, "AccountMeterMetrics: BidsReceivedMeter failed")
}

func TestAccountMeterMetrics_UpdatePriceHistogram(t *testing.T) {
	acm := AccountMeterMetrics{}
	acm.PriceHistogram = metrics.GetOrRegisterHistogram("account.rubicon.prices", metrics.NewPrefixedRegistry(""), metrics.NewExpDecaySample(1028, 0.015))
	acm.UpdatePriceHistogram(2000)
	assert.Equal(t, acm.PriceHistogram.Count(), int64(1), "AccountMeterMetrics: PriceHistogram failed")
}

func TestAllMeterMetrics_IncAppRequest(t *testing.T) {
	acm := AllMeterMetrics{}
	acm.mAppRequestMeter = metrics.NewMeter()
	acm.IncAppRequest(value)
	assert.Equal(t, acm.mAppRequestMeter.Count(), value, "AllMeterMetrics: AppRequestMeter failed")
}

func TestAdapterMeterMetrics_IncBidsReceived(t *testing.T) {
	acm := AdapterMeterMetrics{}
	acm.BidsReceivedMeter = metrics.NewMeter()
	acm.IncBidsReceived(value)
	assert.Equal(t, acm.BidsReceivedMeter.Count(), value, "AdapterMeterMetrics: BidsReceivedMeter failed")
}

func TestAdapterMeterMetrics_IncError(t *testing.T) {
	acm := AdapterMeterMetrics{}
	acm.ErrorMeter = metrics.NewMeter()
	acm.IncError(value)
	assert.Equal(t, acm.ErrorMeter.Count(), value, "AdapterMeterMetrics: ErrorMeter failed")
}

func TestAdapterMeterMetrics_IncNoBid(t *testing.T) {
	acm := AdapterMeterMetrics{}
	acm.NoBidMeter = metrics.NewMeter()
	acm.IncNoBid(value)
	assert.Equal(t, acm.NoBidMeter.Count(), value, "AdapterMeterMetrics: NoBidMeter failed")
}

func TestAdapterMeterMetrics_IncNoCookie(t *testing.T) {
	acm := AdapterMeterMetrics{}
	acm.NoCookieMeter = metrics.NewMeter()
	acm.IncNoCookie(value)
	assert.Equal(t, acm.NoCookieMeter.Count(), value, "AdapterMeterMetrics: NoCookieMeter failed")
}

func TestAdapterMeterMetrics_IncRequest(t *testing.T) {
	acm := AdapterMeterMetrics{}
	acm.RequestMeter = metrics.NewMeter()
	acm.IncRequest(value)
	assert.Equal(t, acm.RequestMeter.Count(), value, "AdapterMeterMetrics: RequestMeter failed")
}

func TestAdapterMeterMetrics_IncTimeOut(t *testing.T) {
	acm := AdapterMeterMetrics{}
	acm.TimeoutMeter = metrics.NewMeter()
	acm.IncTimeOut(value)
	assert.Equal(t, acm.TimeoutMeter.Count(), value, "AdapterMeterMetrics: TimeoutMeter failed")
}

func TestAdapterMeterMetrics_UpdatePriceHistogram(t *testing.T) {
	acm := AdapterMeterMetrics{}
	acm.PriceHistogram = metrics.NewHistogram(metrics.NewExpDecaySample(1028, 0.015))
	acm.UpdatePriceHistogram(2000)
	assert.Equal(t, acm.PriceHistogram.Count(), int64(1), "AdapterMeterMetrics: Price Histogram failed")
}

func TestAdapterMeterMetrics_UpdateRequestTimerSince(t *testing.T) {
	acm := AdapterMeterMetrics{}
	acm.RequestTimer = metrics.GetOrRegisterTimer("x.y.requesttime", metrics.NewPrefixedRegistry(""))
	acm.UpdateRequestTimerSince(time.Now())
	assert.Equal(t, acm.RequestTimer.Count(), int64(1), "AdapterMeterMetrics: RequestTimer failed")
}

func TestAllMeterMetrics_IncCookieSync(t *testing.T) {
	acm := AllMeterMetrics{}
	acm.mCookieSyncMeter = metrics.NewMeter()
	acm.IncCookieSync(value)
	assert.Equal(t, acm.mCookieSyncMeter.Count(), value, "AllMeterMetrics: CookieSyncMeter failed")
}

func TestAllMeterMetrics_IncError(t *testing.T) {
	acm := AllMeterMetrics{}
	acm.mErrorMeter = metrics.NewMeter()
	acm.IncError(value)
	assert.Equal(t, acm.mErrorMeter.Count(), value, "AllMeterMetrics: ErrorMeter failed")
}

func TestAllMeterMetrics_IncNoCookie(t *testing.T) {
	acm := AllMeterMetrics{}
	acm.mNoCookieMeter = metrics.NewMeter()
	acm.IncNoCookie(value)
	assert.Equal(t, acm.mNoCookieMeter.Count(), value, "AllMeterMetrics: NoCookieMeter failed")
}

func TestAllMeterMetrics_IncRequest(t *testing.T) {
	acm := AllMeterMetrics{}
	acm.mRequestMeter = metrics.NewMeter()
	acm.IncRequest(value)
	assert.Equal(t, acm.mRequestMeter.Count(), value, "AllMeterMetrics: RequestMeter failed")
}

func TestAllMeterMetrics_IncSafariNoCookie(t *testing.T) {
	acm := AllMeterMetrics{}
	acm.mSafariNoCookieMeter = metrics.NewMeter()
	acm.IncSafariNoCookie(value)
	assert.Equal(t, acm.mSafariNoCookieMeter.Count(), value, "AllMeterMetrics: SafariNoCookieMeter failed")
}

func TestAllMeterMetrics_IncSafariRequest(t *testing.T) {
	acm := AllMeterMetrics{}
	acm.mSafariRequestMeter = metrics.NewMeter()
	acm.IncSafariRequest(value)
	assert.Equal(t, acm.mSafariRequestMeter.Count(), value, "AllMeterMetrics: SafariRequest failed")
}

func TestAllMeterMetrics_UpdateRequestTimerSince(t *testing.T) {
	acm := AllMeterMetrics{}
	acm.mRequestTimer = metrics.NewTimer()
	acm.UpdateRequestTimerSince(time.Now())
	assert.Equal(t, acm.mRequestTimer.Count(), int64(1), "AllMeterMetrics: RequestTimer failed")
}
