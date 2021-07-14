package metrics

import (
	"time"

	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/mock"
)

// MetricsEngineMock is mock for the MetricsEngine interface
type MetricsEngineMock struct {
	mock.Mock
}

// RecordRequest mock
func (me *MetricsEngineMock) RecordRequest(labels Labels) {
	me.Called(labels)
}

// RecordConnectionAccept mock
func (me *MetricsEngineMock) RecordConnectionAccept(success bool) {
	me.Called(success)
}

// RecordConnectionClose mock
func (me *MetricsEngineMock) RecordConnectionClose(success bool) {
	me.Called(success)
}

// RecordImps mock
func (me *MetricsEngineMock) RecordImps(labels ImpLabels) {
	me.Called(labels)
}

// RecordLegacyImps mock
func (me *MetricsEngineMock) RecordLegacyImps(labels Labels, numImps int) {
	me.Called(labels, numImps)
}

// RecordRequestTime mock
func (me *MetricsEngineMock) RecordRequestTime(labels Labels, length time.Duration) {
	me.Called(labels, length)
}

// RecordStoredDataFetchTime mock
func (me *MetricsEngineMock) RecordStoredDataFetchTime(labels StoredDataLabels, length time.Duration) {
	me.Called(labels, length)
}

// RecordStoredDataError mock
func (me *MetricsEngineMock) RecordStoredDataError(labels StoredDataLabels) {
	me.Called(labels)
}

// RecordAdapterPanic mock
func (me *MetricsEngineMock) RecordAdapterPanic(labels AdapterLabels) {
	me.Called(labels)
}

// RecordAdapterRequest mock
func (me *MetricsEngineMock) RecordAdapterRequest(labels AdapterLabels) {
	me.Called(labels)
}

// RecordAdapterConnections mock
func (me *MetricsEngineMock) RecordAdapterConnections(bidderName openrtb_ext.BidderName, connWasReused bool, connWaitTime time.Duration) {
	me.Called(bidderName, connWasReused, connWaitTime)
}

// RecordDNSTime mock
func (me *MetricsEngineMock) RecordDNSTime(dnsLookupTime time.Duration) {
	me.Called(dnsLookupTime)
}

func (me *MetricsEngineMock) RecordTLSHandshakeTime(tlsHandshakeTime time.Duration) {
	me.Called(tlsHandshakeTime)
}

// RecordAdapterBidReceived mock
func (me *MetricsEngineMock) RecordAdapterBidReceived(labels AdapterLabels, bidType openrtb_ext.BidType, hasAdm bool) {
	me.Called(labels, bidType, hasAdm)
}

// RecordAdapterPrice mock
func (me *MetricsEngineMock) RecordAdapterPrice(labels AdapterLabels, cpm float64) {
	me.Called(labels, cpm)
}

// RecordAdapterTime mock
func (me *MetricsEngineMock) RecordAdapterTime(labels AdapterLabels, length time.Duration) {
	me.Called(labels, length)
}

// RecordCookieSync mock
func (me *MetricsEngineMock) RecordCookieSync() {
	me.Called()
}

// RecordAdapterCookieSync mock
func (me *MetricsEngineMock) RecordAdapterCookieSync(adapter openrtb_ext.BidderName, gdprBlocked bool) {
	me.Called(adapter, gdprBlocked)
}

// RecordUserIDSet mock
func (me *MetricsEngineMock) RecordUserIDSet(userLabels UserLabels) {
	me.Called(userLabels)
}

// RecordStoredReqCacheResult mock
func (me *MetricsEngineMock) RecordStoredReqCacheResult(cacheResult CacheResult, inc int) {
	me.Called(cacheResult, inc)
}

// RecordStoredImpCacheResult mock
func (me *MetricsEngineMock) RecordStoredImpCacheResult(cacheResult CacheResult, inc int) {
	me.Called(cacheResult, inc)
}

// RecordAccountCacheResult mock
func (me *MetricsEngineMock) RecordAccountCacheResult(cacheResult CacheResult, inc int) {
	me.Called(cacheResult, inc)
}

// RecordPrebidCacheRequestTime mock
func (me *MetricsEngineMock) RecordPrebidCacheRequestTime(success bool, length time.Duration) {
	me.Called(success, length)
}

// RecordRequestQueueTime mock
func (me *MetricsEngineMock) RecordRequestQueueTime(success bool, requestType RequestType, length time.Duration) {
	me.Called(success, requestType, length)
}

// RecordTimeoutNotice mock
func (me *MetricsEngineMock) RecordTimeoutNotice(success bool) {
	me.Called(success)
}

// RecordRequestPrivacy mock
func (me *MetricsEngineMock) RecordRequestPrivacy(privacy PrivacyLabels) {
	me.Called(privacy)
}

// RecordAdapterGDPRRequestBlocked mock
func (me *MetricsEngineMock) RecordAdapterGDPRRequestBlocked(adapterName openrtb_ext.BidderName) {
	me.Called(adapterName)
}
