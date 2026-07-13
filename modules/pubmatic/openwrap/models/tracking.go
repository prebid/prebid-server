package models

import "github.com/prebid/prebid-server/v3/openrtb_ext"

// impression tracker url parameters
const (
	// constants for query parameter names for tracker call
	TRKPubID                  = "pubid"
	TRKPageURL                = "purl"
	TRKTimestamp              = "tst"
	TRKIID                    = "iid"
	TRKProfileID              = "pid"
	TRKVersionID              = "pdvid"
	TRKIP                     = "ip"
	TRKUserAgent              = "ua"
	TRKSlotID                 = "slot"
	TRKAdunit                 = "au"
	TRKRewardedInventory      = "rwrd"
	TRKPartnerID              = "pn"
	TRKBidderCode             = "bc"
	TRKKGPV                   = "kgpv"
	TRKGrossECPM              = "eg"
	TRKNetECPM                = "en"
	TRKBidID                  = "bidid"
	TRKOrigBidID              = "origbidid"
	TRKQMARK                  = "?"
	TRKAmpersand              = "&"
	TRKSSAI                   = "ssai"
	TRKPlatform               = "plt"
	TRKAdSize                 = "psz"
	TRKTestGroup              = "tgid"
	TRKAdvertiser             = "adv"
	TRKPubDomain              = "orig"
	TRKServerSide             = "ss"
	TRKAdformat               = "af"
	TRKAdDuration             = "dur"
	TRKAdPodExist             = "aps"
	TRKFloorType              = "ft"
	TRKFloorModelVersion      = "fmv"
	TRKFloorProvider          = "fp"
	TRKFloorSkippedFlag       = "fskp"
	TRKFloorSource            = "fsrc"
	TRKFloorValue             = "fv"
	TRKFloorRuleValue         = "frv"
	TRKServerLogger           = "sl"
	TRKDealID                 = "di"
	TRKCustomDimensions       = "cds"
	TRKATTS                   = "atts"
	TRKProfileType            = "pt"
	TRKProfileTypePlatform    = "ptp"
	TRKAppPlatform            = "ap"
	TRKAppIntegrationPath     = "aip"
	TRKAppSubIntegrationPath  = "asip"
	TRKPriceBucket            = "pb"
	TRKMultiBidMultiFloorFlag = "mbmf"
	TRKDisplayManager         = "dm"
	TRKDisplayManagerVer      = "dmv"
	TRKCountryCode            = "cc"
	TRKNetworkID              = "nwid"
	TRKVastUnwrapEnabled      = "vu"
	TRKInViewCountingFlag     = "ctm"
	// TRKBidExp carries OpenRTB bid.exp (seconds) for impression counting / bid expiration at the ad server.
	TRKBidExp = "bexp"
	// TRKBidExpEnf carries bid.ext.bidexp_enf on the impression tracker (omitted with bexp when Google SDK bidding omit rule applies).
	TRKBidExpEnf = "bexpef"
)

// video error tracker url parameters
const (
	ERROperIDValue    = "8"
	ERROperID         = "operId"
	ERROperIDParam    = ERROperID + "=" + ERROperIDValue
	ERRPubID          = "p"
	ERRProfileID      = "pid"
	ERRVersionID      = "v"
	ERRTimestamp      = "ts"
	ERRPartnerID      = "pn"
	ERRBidderCode     = "bc"
	ERRAdunit         = "au"
	ERRCreativeID     = "crId"
	ERRErrorCode      = "ier"
	ERRErrorCodeMacro = "[ERRORCODE]"
	ERRErrorCodeParam = ERRErrorCode + "=" + ERRErrorCodeMacro
	ERRSUrl           = "sURL" // key represents either domain or bundle from request
	ERRPlatform       = "pfi"
	ERRAdvertiser     = "adv"
	ERRTestGroup      = "tgid"
	ERRSSAI           = "ssai"
	ERRVastUnwrap     = "vu"
)

// EventTrackingMacros Video Event Tracker's custom macros
type EventTrackingMacros string

const (
	MacroProfileID           EventTrackingMacros = "[PROFILE_ID]"            // Pass Profile ID using this macro
	MacroProfileVersionID    EventTrackingMacros = "[PROFILE_VERSION]"       // Pass Profile's version ID using this macro
	MacroUnixTimeStamp       EventTrackingMacros = "[UNIX_TIMESTAMP]"        // Pass Current Unix Time when Event Tracking URL is generated using this macro
	MacroPlatform            EventTrackingMacros = "[PLATFORM]"              // Pass PubMatic's Platform using this macro
	MacroWrapperImpressionID EventTrackingMacros = "[WRAPPER_IMPRESSION_ID]" // Pass Wrapper Impression ID using this macro
	MacroSSAI                EventTrackingMacros = "[SSAI]"                  // Pass SSAI vendor name using this macro
)

// DspId for Pixel Based Open Measurement
const (
	DspId_DV360 = 80
)

var FloorSourceMap = map[string]int{
	openrtb_ext.NoDataLocation:  0,
	openrtb_ext.RequestLocation: 1,
	openrtb_ext.FetchLocation:   2,
}

// FetchStatusMap maps floor fetch status with integer codes
var FetchStatusMap = map[string]int{
	openrtb_ext.FetchNone:       0,
	openrtb_ext.FetchSuccess:    1,
	openrtb_ext.FetchError:      2,
	openrtb_ext.FetchInprogress: 3,
	openrtb_ext.FetchTimeout:    4,
}

const (
	NotSet       = -1
	DealIDAbsent = "-1"
)
