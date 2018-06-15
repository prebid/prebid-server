package metricsdef

import "github.com/prebid/prebid-server/openrtb_ext"

// Labels defines the labels that can be attached to the metrics.
type Labels struct {
	Source        DemandSource
	RType         RequestType
	PubID         string // exchange specific ID, so we cannot compile in values
	Browser       Browser
	CookieFlag    CookieFlag
	RequestStatus RequestStatus
}

// AdapterLabels defines the labels that can be attached to the adapter metrics.
type AdapterLabels struct {
	Source        DemandSource
	RType         RequestType
	Adapter       openrtb_ext.BidderName
	PubID         string // exchange specific ID, so we cannot compile in values
	Browser       Browser
	CookieFlag    CookieFlag
	AdapterStatus AdapterStatus
}

// Label typecasting. Se below the type definitions for possible values

// DemandSource : Demand source enumeration
type DemandSource string

// RequestType : Request type enumeration
type RequestType string

// Browser type enumeration
type Browser string

// CookieFlag : User ID cookie exists flag
type CookieFlag string

// RequestStatus : The request return status
type RequestStatus string

// AdapterStatus : The radapter execution status
type AdapterStatus string

// The demand sources
const (
	DemandWeb     DemandSource = "web"
	DemandApp     DemandSource = "app"
	DemandUnknown DemandSource = "unknown"
)

// The request types (endpoints)
const (
	ReqTypeLegacy RequestType = "legacy"
	ReqTypeORTB2  RequestType = "openrtb2"
	ReqTypeAMP    RequestType = "amp"
)

func RequestTypes() []RequestType {
	return []RequestType{
		ReqTypeLegacy,
		ReqTypeORTB2,
		ReqTypeAMP,
	}
}

// Browser flag; at this point we only care about identifying Safari
const (
	BrowserSafari Browser = "safari"
	BrowserOther  Browser = "other"
)

// Cookie flag
const (
	CookieFlagYes     CookieFlag = "exists"
	CookieFlagNo      CookieFlag = "no"
	CookieFlagUnknown CookieFlag = "unknown"
)

// Request/return status
const (
	RequestStatusOK       RequestStatus = "ok"
	RequestStatusBadInput RequestStatus = "badinput"
	RequestStatusErr      RequestStatus = "err"
)

func RequestStatuses() []RequestStatus {
	return []RequestStatus{
		RequestStatusOK,
		RequestStatusBadInput,
		RequestStatusErr,
	}
}

// Adapter execution status
const (
	AdapterStatusOK      AdapterStatus = "ok"
	AdapterStatusErr     AdapterStatus = "err"
	AdapterStatusNoBid   AdapterStatus = "nobid"
	AdapterStatusTimeout AdapterStatus = "timeout"
)

// UserLabels : Labels for /setuid endpoint
type UserLabels struct {
	Action RequestAction
	Bidder openrtb_ext.BidderName
}

// RequestAction : The setuid request result
type RequestAction string

// /setuid action labels
const (
	RequestActionSet    RequestAction = "set"
	RequestActionOptOut RequestAction = "opt_out"
	RequestActionGDPR   RequestAction = "gdpr"
	RequestActionErr    RequestAction = "err"
)
