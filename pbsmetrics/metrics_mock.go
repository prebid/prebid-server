package pbsmetrics

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
	return
}

// RecordConnectionAccept mock
func (me *MetricsEngineMock) RecordConnectionAccept(success bool) {
	me.Called(success)
	return
}

// RecordConnectionClose mock
func (me *MetricsEngineMock) RecordConnectionClose(success bool) {
	me.Called(success)
	return
}

// RecordImps mock
func (me *MetricsEngineMock) RecordImps(labels Labels, numImps int) {
	me.Called(labels, numImps)
	return
}

// RecordRequestTime mock
func (me *MetricsEngineMock) RecordRequestTime(labels Labels, length time.Duration) {
	me.Called(labels, length)
	return
}

// RecordAdapterPanic mock
func (me *MetricsEngineMock) RecordAdapterPanic(labels AdapterLabels) {
	me.Called(labels)
	return
}

// RecordAdapterRequest mock
func (me *MetricsEngineMock) RecordAdapterRequest(labels AdapterLabels) {
	me.Called(labels)
	return
}

// RecordAdapterBidReceived mock
func (me *MetricsEngineMock) RecordAdapterBidReceived(labels AdapterLabels, bidType openrtb_ext.BidType, hasAdm bool) {
	me.Called(labels, bidType, hasAdm)
	return
}

// RecordAdapterPrice mock
func (me *MetricsEngineMock) RecordAdapterPrice(labels AdapterLabels, cpm float64) {
	me.Called(labels, cpm)
	return
}

// RecordAdapterTime mock
func (me *MetricsEngineMock) RecordAdapterTime(labels AdapterLabels, length time.Duration) {
	me.Called(labels, length)
	return
}

// RecordCookieSync mock
func (me *MetricsEngineMock) RecordCookieSync(labels Labels) {
	me.Called(labels)
	return
}

// RecordAdapterCookieSync mock
func (me *MetricsEngineMock) RecordAdapterCookieSync(adapter openrtb_ext.BidderName, gdprBlocked bool) {
	me.Called(adapter, gdprBlocked)
	return
}

// RecordUserIDSet mock
func (me *MetricsEngineMock) RecordUserIDSet(userLabels UserLabels) {
	me.Called(userLabels)
	return
}

// RecordStoredReqCacheResult mock
func (me *MetricsEngineMock) RecordStoredReqCacheResult(cacheResult CacheResult, inc int) {
	me.Called(cacheResult, inc)
	return
}

// RecordStoredImpCacheResult mock
func (me *MetricsEngineMock) RecordStoredImpCacheResult(cacheResult CacheResult, inc int) {
	me.Called(cacheResult, inc)
	return
}
