package pbsmetrics

import (
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/rcrowley/go-metrics"
)

func TestNewMetrics(t *testing.T) {
	registry := metrics.NewRegistry()
	m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus, openrtb_ext.BidderRubicon})

	ensureContains(t, registry, "app_requests", m.AppRequestMeter)
	ensureContains(t, registry, "no_cookie_requests", m.NoCookieMeter)
	ensureContains(t, registry, "safari_requests", m.SafariRequestMeter)
	ensureContains(t, registry, "safari_no_cookie_requests", m.SafariNoCookieMeter)
	ensureContains(t, registry, "request_time", m.RequestTimer)
	ensureContains(t, registry, "amp_no_cookie_requests", m.AmpNoCookieMeter)
	ensureContainsAdapterMetrics(t, registry, "adapter.appnexus", m.AdapterMetrics["appnexus"])
	ensureContainsAdapterMetrics(t, registry, "adapter.rubicon", m.AdapterMetrics["rubicon"])
	ensureContains(t, registry, "usersync.appnexus.gdpr_prevent", m.userSyncGDPRPrevent["appnexus"])
	ensureContains(t, registry, "usersync.rubicon.gdpr_prevent", m.userSyncGDPRPrevent["rubicon"])
	ensureContains(t, registry, "usersync.unknown.gdpr_prevent", m.userSyncGDPRPrevent["unknown"])

	ensureContains(t, registry, "requests.ok.legacy", m.RequestStatuses[ReqTypeLegacy][RequestStatusOK])
	ensureContains(t, registry, "requests.badinput.legacy", m.RequestStatuses[ReqTypeLegacy][RequestStatusBadInput])
	ensureContains(t, registry, "requests.err.legacy", m.RequestStatuses[ReqTypeLegacy][RequestStatusErr])
	ensureContains(t, registry, "requests.ok.openrtb2", m.RequestStatuses[ReqTypeORTB2][RequestStatusOK])
	ensureContains(t, registry, "requests.badinput.openrtb2", m.RequestStatuses[ReqTypeORTB2][RequestStatusBadInput])
	ensureContains(t, registry, "requests.err.openrtb2", m.RequestStatuses[ReqTypeORTB2][RequestStatusErr])
	ensureContains(t, registry, "requests.ok.amp", m.RequestStatuses[ReqTypeAMP][RequestStatusOK])
	ensureContains(t, registry, "requests.badinput.amp", m.RequestStatuses[ReqTypeAMP][RequestStatusBadInput])
	ensureContains(t, registry, "requests.err.amp", m.RequestStatuses[ReqTypeAMP][RequestStatusErr])
}

func TestRecordBidType(t *testing.T) {
	registry := metrics.NewRegistry()
	m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus})

	m.RecordAdapterBidReceived(AdapterLabels{
		Adapter: openrtb_ext.BidderAppnexus,
	}, openrtb_ext.BidTypeBanner, true)
	VerifyMetrics(t, "Appnexus Banner Adm Bids", m.AdapterMetrics[openrtb_ext.BidderAppnexus].MarkupMetrics[openrtb_ext.BidTypeBanner].AdmMeter.Count(), 1)
	VerifyMetrics(t, "Appnexus Banner Nurl Bids", m.AdapterMetrics[openrtb_ext.BidderAppnexus].MarkupMetrics[openrtb_ext.BidTypeBanner].NurlMeter.Count(), 0)

	m.RecordAdapterBidReceived(AdapterLabels{
		Adapter: openrtb_ext.BidderAppnexus,
	}, openrtb_ext.BidTypeVideo, false)
	VerifyMetrics(t, "Appnexus Video Adm Bids", m.AdapterMetrics[openrtb_ext.BidderAppnexus].MarkupMetrics[openrtb_ext.BidTypeVideo].AdmMeter.Count(), 0)
	VerifyMetrics(t, "Appnexus Video Nurl Bids", m.AdapterMetrics[openrtb_ext.BidderAppnexus].MarkupMetrics[openrtb_ext.BidTypeVideo].NurlMeter.Count(), 1)
}

func TestRecordGDPRRejection(t *testing.T) {
	registry := metrics.NewRegistry()
	m := NewMetrics(registry, []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus})
	m.RecordUserIDSet(UserLabels{
		Action: RequestActionGDPR,
		Bidder: openrtb_ext.BidderAppnexus,
	})
	VerifyMetrics(t, "GDPR sync rejects", m.userSyncGDPRPrevent[openrtb_ext.BidderAppnexus].Count(), 1)
}

func ensureContains(t *testing.T, registry metrics.Registry, name string, metric interface{}) {
	t.Helper()
	if inRegistry := registry.Get(name); inRegistry == nil {
		t.Errorf("No metric in registry at %s.", name)
	} else if inRegistry != metric {
		t.Errorf("Bad value stored at metric %s.", name)
	}
}

func ensureContainsAdapterMetrics(t *testing.T, registry metrics.Registry, name string, adapterMetrics *AdapterMetrics) {
	t.Helper()
	ensureContains(t, registry, name+".no_cookie_requests", adapterMetrics.NoCookieMeter)
	ensureContains(t, registry, name+".error_requests", adapterMetrics.ErrorMeter)
	ensureContains(t, registry, name+".requests", adapterMetrics.RequestMeter)
	ensureContains(t, registry, name+".no_bid_requests", adapterMetrics.NoBidMeter)
	ensureContains(t, registry, name+".timeout_requests", adapterMetrics.TimeoutMeter)
	ensureContains(t, registry, name+".request_time", adapterMetrics.RequestTimer)
	ensureContains(t, registry, name+".prices", adapterMetrics.PriceHistogram)
	ensureContainsBidTypeMetrics(t, registry, name, adapterMetrics.MarkupMetrics)
}

func ensureContainsBidTypeMetrics(t *testing.T, registry metrics.Registry, prefix string, mdm map[openrtb_ext.BidType]*MarkupDeliveryMetrics) {
	ensureContains(t, registry, prefix+".banner.adm_bids_received", mdm[openrtb_ext.BidTypeBanner].AdmMeter)
	ensureContains(t, registry, prefix+".banner.nurl_bids_received", mdm[openrtb_ext.BidTypeBanner].NurlMeter)
	ensureContains(t, registry, prefix+".video.adm_bids_received", mdm[openrtb_ext.BidTypeVideo].AdmMeter)
	ensureContains(t, registry, prefix+".video.nurl_bids_received", mdm[openrtb_ext.BidTypeVideo].NurlMeter)
	ensureContains(t, registry, prefix+".audio.adm_bids_received", mdm[openrtb_ext.BidTypeAudio].AdmMeter)
	ensureContains(t, registry, prefix+".audio.nurl_bids_received", mdm[openrtb_ext.BidTypeAudio].NurlMeter)
	ensureContains(t, registry, prefix+".native.adm_bids_received", mdm[openrtb_ext.BidTypeNative].AdmMeter)
	ensureContains(t, registry, prefix+".native.nurl_bids_received", mdm[openrtb_ext.BidTypeNative].NurlMeter)
}
