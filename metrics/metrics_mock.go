package metrics

import (
	"time"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
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

// RecordTMaxTimeout mock
func (me *MetricsEngineMock) RecordTMaxTimeout() {
	me.Called()
}

// RecordConnectionClose mock
func (me *MetricsEngineMock) RecordConnectionClose(success bool) {
	me.Called(success)
}

// RecordImps mock
func (me *MetricsEngineMock) RecordImps(labels ImpLabels) {
	me.Called(labels)
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

// RecordBidderServerResponseTime mock
func (me *MetricsEngineMock) RecordBidderServerResponseTime(bidderServerResponseTime time.Duration) {
	me.Called(bidderServerResponseTime)
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

// RecordOverheadTime mock
func (me *MetricsEngineMock) RecordOverheadTime(overhead OverheadType, length time.Duration) {
	me.Called(overhead, length)
}

// RecordCookieSync mock
func (me *MetricsEngineMock) RecordCookieSync(status CookieSyncStatus) {
	me.Called(status)
}

// RecordSyncerRequest mock
func (me *MetricsEngineMock) RecordSyncerRequest(key string, status SyncerCookieSyncStatus) {
	me.Called(key, status)
}

// RecordSetUid mock
func (me *MetricsEngineMock) RecordSetUid(status SetUidStatus) {
	me.Called(status)
}

// RecordSyncerSet mock
func (me *MetricsEngineMock) RecordSyncerSet(key string, status SyncerSetUidStatus) {
	me.Called(key, status)
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

// RecordAdapterBuyerUIDScrubbed mock
func (me *MetricsEngineMock) RecordAdapterBuyerUIDScrubbed(adapterName openrtb_ext.BidderName) {
	me.Called(adapterName)
}

// RecordAdapterGDPRRequestBlocked mock
func (me *MetricsEngineMock) RecordAdapterGDPRRequestBlocked(adapterName openrtb_ext.BidderName) {
	me.Called(adapterName)
}

// RecordDebugRequest mock
func (me *MetricsEngineMock) RecordDebugRequest(debugEnabled bool, pubId string) {
	me.Called(debugEnabled, pubId)
}

func (me *MetricsEngineMock) RecordStoredResponse(pubId string) {
	me.Called(pubId)
}

func (me *MetricsEngineMock) RecordAdsCertReq(success bool) {
	me.Called(success)
}

func (me *MetricsEngineMock) RecordAdsCertSignTime(adsCertSignTime time.Duration) {
	me.Called(adsCertSignTime)
}

func (me *MetricsEngineMock) RecordBidValidationCreativeSizeError(adapter openrtb_ext.BidderName, account string) {
	me.Called(adapter, account)
}

func (me *MetricsEngineMock) RecordBidValidationCreativeSizeWarn(adapter openrtb_ext.BidderName, account string) {
	me.Called(adapter, account)
}

func (me *MetricsEngineMock) RecordBidValidationSecureMarkupError(adapter openrtb_ext.BidderName, account string) {
	me.Called(adapter, account)
}

func (me *MetricsEngineMock) RecordBidValidationSecureMarkupWarn(adapter openrtb_ext.BidderName, account string) {
	me.Called(adapter, account)
}

func (me *MetricsEngineMock) RecordModuleCalled(labels ModuleLabels, duration time.Duration) {
	me.Called(labels, duration)
}

func (me *MetricsEngineMock) RecordModuleFailed(labels ModuleLabels) {
	me.Called(labels)
}

func (me *MetricsEngineMock) RecordModuleSuccessNooped(labels ModuleLabels) {
	me.Called(labels)
}

func (me *MetricsEngineMock) RecordModuleSuccessUpdated(labels ModuleLabels) {
	me.Called(labels)
}

func (me *MetricsEngineMock) RecordModuleSuccessRejected(labels ModuleLabels) {
	me.Called(labels)
}

func (me *MetricsEngineMock) RecordModuleExecutionError(labels ModuleLabels) {
	me.Called(labels)
}

func (me *MetricsEngineMock) RecordModuleTimeout(labels ModuleLabels) {
	me.Called(labels)
}
