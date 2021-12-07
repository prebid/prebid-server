package metrics

import (
	"time"

	"github.com/prebid/prebid-server/openrtb_ext"
)

// Labels defines the labels that can be attached to the metrics.
type Labels struct {
	Source        DemandSource
	RType         RequestType
	PubID         string // exchange specific ID, so we cannot compile in values
	CookieFlag    CookieFlag
	RequestStatus RequestStatus
}

// AdapterLabels defines the labels that can be attached to the adapter metrics.
type AdapterLabels struct {
	Source        DemandSource
	RType         RequestType
	Adapter       openrtb_ext.BidderName
	PubID         string // exchange specific ID, so we cannot compile in values
	CookieFlag    CookieFlag
	AdapterBids   AdapterBid
	AdapterErrors map[AdapterError]struct{}
}

// ImpLabels defines metric labels describing the impression type.
type ImpLabels struct {
	BannerImps bool
	VideoImps  bool
	AudioImps  bool
	NativeImps bool
}

// RequestLabels defines metric labels describing the result of a network request.
type RequestLabels struct {
	RequestStatus RequestStatus
}

// PrivacyLabels defines metrics describing the result of privacy enforcement.
type PrivacyLabels struct {
	CCPAEnforced   bool
	CCPAProvided   bool
	COPPAEnforced  bool
	GDPREnforced   bool
	GDPRTCFVersion TCFVersionValue
	LMTEnforced    bool
}

type StoredDataType string

const (
	AccountDataType  StoredDataType = "account"
	AMPDataType      StoredDataType = "amp"
	CategoryDataType StoredDataType = "category"
	RequestDataType  StoredDataType = "request"
	VideoDataType    StoredDataType = "video"
)

func StoredDataTypes() []StoredDataType {
	return []StoredDataType{
		AccountDataType,
		AMPDataType,
		CategoryDataType,
		RequestDataType,
		VideoDataType,
	}
}

type StoredDataFetchType string

const (
	FetchAll   StoredDataFetchType = "all"
	FetchDelta StoredDataFetchType = "delta"
)

func StoredDataFetchTypes() []StoredDataFetchType {
	return []StoredDataFetchType{
		FetchAll,
		FetchDelta,
	}
}

type StoredDataLabels struct {
	DataType      StoredDataType
	DataFetchType StoredDataFetchType
	Error         StoredDataError
}

type StoredDataError string

const (
	StoredDataErrorNetwork   StoredDataError = "network"
	StoredDataErrorUndefined StoredDataError = "undefined"
)

func StoredDataErrors() []StoredDataError {
	return []StoredDataError{
		StoredDataErrorNetwork,
		StoredDataErrorUndefined,
	}
}

// Label typecasting. Se below the type definitions for possible values

// DemandSource : Demand source enumeration
type DemandSource string

// ImpMediaType : Media type described in the "imp" JSON object  TODO is this still needed?
type ImpMediaType string

// RequestType : Request type enumeration
type RequestType string

// CookieFlag : User ID cookie exists flag
type CookieFlag string

// RequestStatus : The request return status
type RequestStatus string

// AdapterBid : Whether or not the adapter returned bids
type AdapterBid string

// AdapterError : Errors which may have occurred during the adapter's execution
type AdapterError string

// CacheResult : Cache hit/miss
type CacheResult string

// PublisherUnknown : Default value for Labels.PubID
const PublisherUnknown = "unknown"

// The demand sources
const (
	DemandWeb     DemandSource = "web"
	DemandApp     DemandSource = "app"
	DemandUnknown DemandSource = "unknown"
)

func DemandTypes() []DemandSource {
	return []DemandSource{
		DemandWeb,
		DemandApp,
		DemandUnknown,
	}
}

// The request types (endpoints)
const (
	ReqTypeLegacy   RequestType = "legacy"
	ReqTypeORTB2Web RequestType = "openrtb2-web"
	ReqTypeORTB2App RequestType = "openrtb2-app"
	ReqTypeAMP      RequestType = "amp"
	ReqTypeVideo    RequestType = "video"
)

// The media types described in the "imp" json objects
const (
	ImpTypeBanner ImpMediaType = "banner"
	ImpTypeVideo  ImpMediaType = "video"
	ImpTypeAudio  ImpMediaType = "audio"
	ImpTypeNative ImpMediaType = "native"
)

func RequestTypes() []RequestType {
	return []RequestType{
		ReqTypeLegacy,
		ReqTypeORTB2Web,
		ReqTypeORTB2App,
		ReqTypeAMP,
		ReqTypeVideo,
	}
}

func ImpTypes() []ImpMediaType {
	return []ImpMediaType{
		ImpTypeBanner,
		ImpTypeVideo,
		ImpTypeAudio,
		ImpTypeNative,
	}
}

// Cookie flag
const (
	CookieFlagYes     CookieFlag = "exists"
	CookieFlagNo      CookieFlag = "no"
	CookieFlagUnknown CookieFlag = "unknown"
)

func CookieTypes() []CookieFlag {
	return []CookieFlag{
		CookieFlagYes,
		CookieFlagNo,
		CookieFlagUnknown,
	}
}

// Request/return status
const (
	RequestStatusOK           RequestStatus = "ok"
	RequestStatusBadInput     RequestStatus = "badinput"
	RequestStatusErr          RequestStatus = "err"
	RequestStatusNetworkErr   RequestStatus = "networkerr"
	RequestStatusBlacklisted  RequestStatus = "blacklistedacctorapp"
	RequestStatusQueueTimeout RequestStatus = "queuetimeout"
)

func RequestStatuses() []RequestStatus {
	return []RequestStatus{
		RequestStatusOK,
		RequestStatusBadInput,
		RequestStatusErr,
		RequestStatusNetworkErr,
		RequestStatusBlacklisted,
		RequestStatusQueueTimeout,
	}
}

// Adapter bid response status.
const (
	AdapterBidPresent AdapterBid = "bid"
	AdapterBidNone    AdapterBid = "nobid"
)

func AdapterBids() []AdapterBid {
	return []AdapterBid{
		AdapterBidPresent,
		AdapterBidNone,
	}
}

// Adapter execution status
const (
	AdapterErrorBadInput            AdapterError = "badinput"
	AdapterErrorBadServerResponse   AdapterError = "badserverresponse"
	AdapterErrorTimeout             AdapterError = "timeout"
	AdapterErrorFailedToRequestBids AdapterError = "failedtorequestbid"
	AdapterErrorUnknown             AdapterError = "unknown_error"
)

func AdapterErrors() []AdapterError {
	return []AdapterError{
		AdapterErrorBadInput,
		AdapterErrorBadServerResponse,
		AdapterErrorTimeout,
		AdapterErrorFailedToRequestBids,
		AdapterErrorUnknown,
	}
}

const (
	// CacheHit represents a cache hit i.e the key was found in cache
	CacheHit CacheResult = "hit"
	// CacheMiss represents a cache miss i.e that key wasn't found in cache
	// and had to be fetched from the backend
	CacheMiss CacheResult = "miss"
)

// CacheResults returns possible cache results i.e. cache hit or miss
func CacheResults() []CacheResult {
	return []CacheResult{
		CacheHit,
		CacheMiss,
	}
}

// TCFVersionValue : The possible values for TCF versions
type TCFVersionValue string

const (
	TCFVersionErr TCFVersionValue = "err"
	TCFVersionV2  TCFVersionValue = "v2"
)

// TCFVersions returns the possible values for the TCF version
func TCFVersions() []TCFVersionValue {
	return []TCFVersionValue{
		TCFVersionErr,
		TCFVersionV2,
	}
}

// TCFVersionToValue takes an integer TCF version and returns the corresponding TCFVersionValue
func TCFVersionToValue(version int) TCFVersionValue {
	switch {
	case version == 2:
		return TCFVersionV2
	}
	return TCFVersionErr
}

// CookieSyncStatus is a status code resulting from a call to the /cookie_sync endpoint.
type CookieSyncStatus string

const (
	CookieSyncOK                    CookieSyncStatus = "ok"
	CookieSyncBadRequest            CookieSyncStatus = "bad_request"
	CookieSyncOptOut                CookieSyncStatus = "opt_out"
	CookieSyncGDPRHostCookieBlocked CookieSyncStatus = "gdpr_blocked_host_cookie"
)

// CookieSyncStatuses returns possible cookie sync statuses.
func CookieSyncStatuses() []CookieSyncStatus {
	return []CookieSyncStatus{
		CookieSyncOK,
		CookieSyncBadRequest,
		CookieSyncOptOut,
		CookieSyncGDPRHostCookieBlocked,
	}
}

// SyncerCookieSyncStatus is a status code from an invocation of a syncer resulting from a call to the /cookie_sync endpoint.
type SyncerCookieSyncStatus string

const (
	SyncerCookieSyncOK               SyncerCookieSyncStatus = "ok"
	SyncerCookieSyncPrivacyBlocked   SyncerCookieSyncStatus = "privacy_blocked"
	SyncerCookieSyncAlreadySynced    SyncerCookieSyncStatus = "already_synced"
	SyncerCookieSyncTypeNotSupported SyncerCookieSyncStatus = "type_not_supported"
)

// SyncerRequestStatuses returns possible syncer statuses.
func SyncerRequestStatuses() []SyncerCookieSyncStatus {
	return []SyncerCookieSyncStatus{
		SyncerCookieSyncOK,
		SyncerCookieSyncPrivacyBlocked,
		SyncerCookieSyncAlreadySynced,
		SyncerCookieSyncTypeNotSupported,
	}
}

// SetUidStatus is a status code resulting from a call to the /setuid endpoint.
type SetUidStatus string

// /setuid action labels
const (
	SetUidOK                    SetUidStatus = "ok"
	SetUidBadRequest            SetUidStatus = "bad_request"
	SetUidOptOut                SetUidStatus = "opt_out"
	SetUidGDPRHostCookieBlocked SetUidStatus = "gdpr_blocked_host_cookie"
	SetUidSyncerUnknown         SetUidStatus = "syncer_unknown"
)

// SetUidStatuses returns possible setuid statuses.
func SetUidStatuses() []SetUidStatus {
	return []SetUidStatus{
		SetUidOK,
		SetUidBadRequest,
		SetUidOptOut,
		SetUidGDPRHostCookieBlocked,
		SetUidSyncerUnknown,
	}
}

// SyncerSetUidStatus is a status code from an invocation of a syncer resulting from a call to the /setuid endpoint.
type SyncerSetUidStatus string

const (
	SyncerSetUidOK      SyncerSetUidStatus = "ok"
	SyncerSetUidCleared SyncerSetUidStatus = "cleared"
)

// SyncerSetUidStatuses returns possible syncer set statuses.
func SyncerSetUidStatuses() []SyncerSetUidStatus {
	return []SyncerSetUidStatus{
		SyncerSetUidOK,
		SyncerSetUidCleared,
	}
}

// MetricsEngine is a generic interface to record PBS metrics into the desired backend
// The first three metrics function fire off once per incoming request, so total metrics
// will equal the total number of incoming requests. The remaining 5 fire off per outgoing
// request to a bidder adapter, so will record a number of hits per incoming request. The
// two groups should be consistent within themselves, but comparing numbers between groups
// is generally not useful.
type MetricsEngine interface {
	RecordConnectionAccept(success bool)
	RecordConnectionClose(success bool)
	RecordRequest(labels Labels)                           // ignores adapter. only statusOk and statusErr fom status
	RecordImps(labels ImpLabels)                           // RecordImps across openRTB2 engines that support the 'Native' Imp Type
	RecordRequestTime(labels Labels, length time.Duration) // ignores adapter. only statusOk and statusErr fom status
	RecordAdapterRequest(labels AdapterLabels)
	RecordAdapterConnections(adapterName openrtb_ext.BidderName, connWasReused bool, connWaitTime time.Duration)
	RecordDNSTime(dnsLookupTime time.Duration)
	RecordTLSHandshakeTime(tlsHandshakeTime time.Duration)
	RecordAdapterPanic(labels AdapterLabels)
	// This records whether or not a bid of a particular type uses `adm` or `nurl`.
	// Since the legacy endpoints don't have a bid type, it can only count bids from OpenRTB and AMP.
	RecordAdapterBidReceived(labels AdapterLabels, bidType openrtb_ext.BidType, hasAdm bool)
	RecordAdapterPrice(labels AdapterLabels, cpm float64)
	RecordAdapterTime(labels AdapterLabels, length time.Duration)
	RecordCookieSync(status CookieSyncStatus)
	RecordSyncerRequest(key string, status SyncerCookieSyncStatus)
	RecordSetUid(status SetUidStatus)
	RecordSyncerSet(key string, status SyncerSetUidStatus)
	RecordStoredReqCacheResult(cacheResult CacheResult, inc int)
	RecordStoredImpCacheResult(cacheResult CacheResult, inc int)
	RecordAccountCacheResult(cacheResult CacheResult, inc int)
	RecordStoredDataFetchTime(labels StoredDataLabels, length time.Duration)
	RecordStoredDataError(labels StoredDataLabels)
	RecordPrebidCacheRequestTime(success bool, length time.Duration)
	RecordRequestQueueTime(success bool, requestType RequestType, length time.Duration)
	RecordTimeoutNotice(sucess bool)
	RecordRequestPrivacy(privacy PrivacyLabels)
	RecordAdapterGDPRRequestBlocked(adapterName openrtb_ext.BidderName)
}
