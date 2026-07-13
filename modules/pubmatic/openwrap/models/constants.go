package models

import "github.com/golang/glog"

const (
	PARTNER_ID                 = "partnerId"
	ADAPTER_ID                 = "adapterId"
	PARTNER_ACCOUNT_NAME       = "partnerName"
	ADAPTER_NAME               = "adapterName"
	PREBID_PARTNER_NAME        = "prebidPartnerName"
	BidderCode                 = "bidderCode"
	BidderFilters              = "bidderFilters"
	IsAlias                    = "isAlias"
	PROTOCOL                   = "protocol"
	SERVER_SIDE_FLAG           = "serverSideEnabled"
	DisplayVersionID           = "displayVersionId"
	KEY_PUBLISHER_ID           = "publisherId"
	KEY_PROFILE_ID             = "profileId"
	KEY_SLOT_NAME              = "slotName"
	LEVEL                      = "level"
	KEY_GEN_PATTERN            = "kgp"
	TIMEOUT                    = "timeout"
	AdserverKey                = "adserver"
	MopubAdserver              = "MoPub"
	CustomAdserver             = "CUSTOM"
	PriceGranularityKey        = "priceGranularity"
	VideoAdDurationKey         = "videoAdDuration"
	VideoAdDurationMatchingKey = "videoAdDurationMatching"
	// Country Filtering specific keys
	CountryFilterModeKey            = "countryFilterMode"
	PartnerLevelThrottlingFeatureID = "1" // Bidder_Exclusion
	MaxRetryAttempts                = 3

	CountryCodesKey             = "countryCodes"
	REVSHARE                    = "rev_share"
	THROTTLE                    = "throttle"
	REFRESH_INTERVAL            = "refreshInterval"
	CreativeType                = "crtype"
	GDPR_ENABLED                = "gdpr"
	PLATFORM_KEY                = "platform"
	SendAllBidsKey              = "sendAllBids"
	VastUnwrapperEnableKey      = "enableVastUnwrapper"
	GoogleSSUFeatureEnabledKey  = "googleSSUFeature"
	VastUnwrapTrafficPercentKey = "vastUnwrapTrafficPercent"
	SSTimeoutKey                = "ssTimeout"
	PWC                         = "awc"
	MAX_SLOT_COUNT              = 5000
	SITE_CACHE_KEY              = "site"
	TAG_CACHE_KEY               = "adtag"
	GA_ID_CACHE_KEY             = "gaid"
	FLOOR_CACHE_KEY             = "floor"
	PUBMATIC                    = "PubMatic"
	PUBMATIC_TIMEOUT            = "PubmaticTimeout"
	PUBMATIC_PROTOCOL           = "/gads"
	PUBMATIC_LEVEL              = "multi"
	PUBMATIC_SS_FLAG            = "1"
	PUBMATIC_PARTNER_ID_STRING  = "1"
	PUBMATIC_ADAPTER_ID_STRING  = "1"
	VersionLevelConfigID        = -1
	ERROR_CODE                  = "ErrorCode"
	ERROR_STRING                = "Error"
	PUBMATIC_PARTNER_ID         = 1
	PUBMATIC_ADAPTER_ID         = 1
	DEFAULT_STRING              = ""
	DEFAULT_INT                 = 0
	DEFAULT_FLOAT               = 0.00
	BID_PRECISION               = 2
	Debug                       = "debug"
	WrapperLoggerDebug          = "owLoggerDebug"
	KEY_OW_SLOT_NAME            = "owSlotName"
	VENDORID                    = "vendorId"
	BidderPubMatic              = "pubmatic"
	//ADSERVER_URL used by S2S to redirect the OW bids if owredirect parameter is not found in video/json
	ADSERVER_URL = "adServerUrl"

	AdServerCurrency = "adServerCurrency"

	MarketplaceBidders = "marketplaceBidders"

	UserAgent   = "UserAgent"
	IP          = "ip"
	StoreURL    = "StoreURL"
	Consent     = "consent"
	GDPR        = "gdpr"
	PublisherID = "pubid"
	ProfileID   = "profileID"
	VersionID   = "versionID"
	Origin      = "origin"

	DEFAULT_DEALCHANNEL = "PMP"

	WLPUBID           = "pubid"
	WLJSON            = "json"
	WLGDPR            = "gdEn"
	USER_AGENT_HEADER = "User-Agent"
	IP_HEADER         = "SOURCE_IP"

	GADS_UNMAPPED_SLOT_ERROR_MSG    = "Slot not mapped"
	GADS_MISSING_CONF_ERROR_MSG     = "Missing Configuration"
	TIMEOUT_ERROR_MSG               = "Timeout Error"
	NO_BID_PREBID_MSG               = "No Bid"
	PARTNER_TIMEOUT_ERR_MSG         = "Partner Timed out"
	PREBID_DEFAULT_TIMEOUT_ERR_MSG  = "Timed out"
	INVALID_CONFIGURATION_ERR_MSG   = "Invalid Configuration"
	NO_GDPR_CONSENT_ERR_MSG         = "No Consent Present"
	API_RESPONSE_ERROR_MSG          = "API Error"
	INVALID_IMPRESSION_ERR_MSG      = "No Valid Impression Found"
	CACHE_PUT_FAILED_ERROR_MSG      = "Cache PUT Failed"
	INVALID_PARAMETERS_ERROR_MSG    = "Invalid Parameters"
	BANNER_VIDEO_DISABLED_ERROR_MSG = "Banner/Video disabled through config"
	// PrebidUnknownErrorMsg is the Error message for Unknown Error returned from prebid-server
	PrebidUnknownErrorMsg = "Unknown error received from Prebid"

	ALL_PARTNERS_THROTTLED_ERROR_MSG = "All partners throttled"
	PARTNER_THROTTLED_ERROR_MSG      = "Partner throttled"
	PriceGranularityCustom           = "custom"                       //  contains `custom` price granularity as value
	PriceGranularityCustomConfig     = "customPriceGranularityConfig" // key which holds configurations around custom price granularity

	// Below is added for Comapring error returned by Prebid Server
	PARTNER_CONTEXT_DEADLINE   = "context deadline exceeded"
	INVALID_CREATIVE_ERROR_MSG = "Invalid Creative"

	//constants for macros of logger/tracker keys
	MacroPartnerName = "${PARTNER_NAME}"
	MacroBidderCode  = "${BIDDER_CODE}"
	MacroKGPV        = "${KGPV}"
	MacroGrossECPM   = "${G_ECPM}"
	MacroNetECPM     = "${N_ECPM}"
	MacroBidID       = "${BID_ID}"
	MacroOrigBidID   = "${ORIGBID_ID}"
	MacroSlotID      = "${SLOT_ID}"
	MacroAdunit      = "${ADUNIT}"
	MacroRewarded    = "${REWARDED}"

	//constants for targetting keys in AMP
	PWT_PUBID      = "pwtpubid"
	PWT_PROFILEID  = "pwtprofid"
	PWT_VERSIONID  = "pwtverid"
	PWT_ECPM       = "pwtecp"
	PWT_BIDSTATUS  = "pwtbst"
	PWT_DEALID     = "pwtdid"
	PWT_SLOTID     = "pwtsid"
	PWT_PARTNERID  = "pwtpid"
	PWT_CACHEID    = "pwtcid"
	PWT_CACHEURL   = "pwtcurl"
	PWT_CACHE_PATH = "pwtcpath"
	PWT_PLATFORM   = "pwtplt"
	PWT_SZ         = "pwtsz"
	PWT_DURATION   = "pwtdur"
	PwtBidID       = "pwtbidid" // Represents bid.id value from oRTB response
	PwtPb          = "pwtpb"
	PwtCat         = "pwtcat"
	PwtPbCatDur    = "pwtpb_cat_dur"
	PwtDT          = "pwtdt"

	//constants for query params in AMP request
	PUBID_KEY         = "pubId"
	PROFILEID_KEY     = "profId"
	ADUNIT_KEY        = "auId"
	MULTISIZE_KEY     = "ms"
	PAGEURL_KEY       = "purl"
	WIDTH_KEY         = "w"
	HEIGHT_KEY        = "h"
	VERSION_KEY       = "pwtv"
	DEBUG_KEY         = "pwtvc"
	ResponseFormatKey = "f"
	ConsentStringKey  = "consent_string"
	GDPRAppliesKey    = "gdpr_applies"
	ConsentTypeKey    = "consent_type"
	CanonicalUrl      = "curl"
	TargetingKey      = "targeting"

	AMP_CACHE_PATH         = "/cache"
	AMP_ORIGIN             = "__amp_source_origin"
	ResponseFormatJSON     = "json"
	ResponseFormatRedirect = "redirect"
	Test                   = "test"
	PubmaticTest           = "pubmaticTest"

	// constants for query params in Video request
	OWRedirectURLKey      = "owredirect"
	CustParams            = "cust_params"
	MimeTypes             = "pwtmime"
	InventoryUnitKey      = "iu"
	InventoryUnitMacroKey = "pwtm_iu"
	Correlator            = "correlator"
	MacroPrefix           = "pwtm_"
	GDPRFlag              = "pwtgdpr"
	CCPAUSPrivacyKey      = "pwtccpa"
	ConsentString         = "pwtcnst"
	AppId                 = "pwtappid"
	AppRequest            = "pwtapp"
	DeviceLMT             = "pwtlmt"
	DeviceDNT             = "pwtdnt"
	UserID                = "pwtuid"
	ContentTransparency   = "owcontenttransparency"
	FloorValue            = "floor_val"
	FloorCurrency         = "floor_cur"

	// constants for error related query params to be added to DFP call
	ErrorKey                 = "pwterr"
	ErrorMsg                 = "pwterrmsg"
	PartnerConfigNotFoundErr = "1"
	CachePutFailedErr        = "2"
	TimeoutErr               = "3"
	ParameterValidationErr   = "4"
	SlotNotMappedErr         = "5"

	//constants for video
	VIDEO_CACHE_PATH          = "/cache"
	VideoSizeSuffix           = "v"
	PartnerURLPlaceholder     = "$PARTNER_URL_PLACEHOLDER"
	TrackerPlaceholder        = "$TRACKER_PLACEHOLDER"
	ErrorPlaceholder          = "$ERROR_PLACEHOLDER"
	ImpressionElement         = "Impression"
	ErrorElement              = "Error"
	VASTAdElement             = ".//VAST/Ad"
	AdWrapperElement          = "./Wrapper"
	AdInlineElement           = "./InLine"
	VASTAdWrapperElement      = ".//VAST/Ad/Wrapper"
	VASTAdInlineElement       = ".//VAST/Ad/InLine"
	CdataPrefix               = "<![CDATA["
	CdataSuffix               = "]]>"
	HTTPProtocol              = "http"
	HTTPSProtocol             = "https"
	VASTImpressionURLTemplate = `<Impression><![CDATA[` + TrackerPlaceholder + `]]></Impression>`
	VASTErrorURLTemplate      = `<Error><![CDATA[` + ErrorPlaceholder + `]]></Error>`
	VastWrapper               = `<VAST version="3.0"><Ad id="1"><Wrapper><AdSystem>PubMatic Wrapper</AdSystem><VASTAdTagURI><![CDATA[$PARTNER_URL_PLACEHOLDER]]></VASTAdTagURI>` + VASTImpressionURLTemplate + VASTErrorURLTemplate + `</Wrapper></Ad></VAST>`

	//constants for wrapper platforms
	PLATFORM_DISPLAY        = "display"
	PLATFORM_AMP            = "amp"
	PLATFORM_APP            = "in-app"
	PLATFORM_VIDEO          = "video"
	PlatformAppTargetingKey = "inapp"
	HB_PLATFORM_APP         = "app"

	//constants for headers
	ORIGIN             = "origin"
	KADUSERCOOKIE      = "KADUSERCOOKIE"
	COOKIE             = "Cookie"
	WrapperLoggerImpID = "wiid"
	UidCookieName      = "uids"

	//constant for gzip response
	AcceptEncodingHeader = "Accept-Encoding"
	GZIPEncoding         = "gzip"

	//bidresponse extension
	ResponseTime       = "responsetimemillis"
	ResponseExtAdPod   = "adpod"
	MatchedImpression  = "matchedimpression"
	SendAllBidsFlagKey = "sendallbids"
	LoggerKey          = "owlogger"

	//keys for reading values from Impression Extension JSON
	SKAdnetwork = "skadn"
	PrebidKey   = "prebid"
	ImpExtData  = "data"

	//Node and Pod names for K8S
	DEFAULT_NODENAME  = "Default_Node"
	DEFAULT_PODNAME   = "Default_Pod"
	ENV_VAR_NODE_NAME = "MY_NODE_NAME"
	ENV_VAR_POD_NAME  = "MY_POD_NAME"

	// PrebidTargetingKeyPrefix is Prebid's prefix for ext.Prebid.targeting keys
	PrebidTargetingKeyPrefix = "hb_"
	// OWTargetingKeyPrefix is OpenWrap's prefix for ext.Prebid.targeting keys
	OWTargetingKeyPrefix = "pwt"

	//constants for reading adunit Config JSON
	AdunitConfigDefaultKey       = "default"
	AdunitConfigSlotConfigKey    = "slotConfig"
	AdunitConfigSlotNameKey      = "slotname"
	AdunitConfigSlotBannerKey    = "banner"
	AdunitConfigSlotVideoKey     = "video"
	AdunitConfigEnabledKey       = "enabled"
	AdUnitConfigClientConfigKey  = "clientconfig"
	AdunitConfigConfigKey        = "config"
	AdunitConfigConfigPatternKey = "configPattern"
	AdunitConfigExpKey           = "exp"
	AdunitConfigExtKey           = "ext"

	AdunitConfigBidFloor    = "bidfloor"
	AdunitConfigBidFloorCur = "bidfloorcur"
	AdunitConfigFloorJSON   = "floors"
	AdunitConfigRegex       = "regex"

	OpenRTBDeviceOsIosRegexPattern     string = `(ios).*`
	OpenRTBDeviceOsAndroidRegexPattern string = `(android).*`
	IosUARegexPattern                  string = `\b(iphone|ipad|darwin)\b`
	AndroidUARegexPattern              string = `android.*`
	MobileDeviceUARegexPattern         string = `(mobi|tablet|ios).*`
	ConnectedDeviceUARegexPattern      string = `Roku|SMART-TV|SmartTV|AndroidTV|Android TV|AppleTV|Apple TV|VIZIO|PHILIPS|BRAVIA|PlayStation|Chromecast|ExoPlayerLib|MIBOX3|Xbox|ComcastAppPlatform|AFT|HiSmart|BeyondTV|D.*ATV|PlexTV|Xstream|MiTV|AI PONT`

	HbBuyIdPrefix               = "hb_buyid_"
	HbBuyIdPubmaticConstantKey  = "hb_buyid_pubmatic"
	PwtBuyIdPubmaticConstantKey = "pwtbuyid_pubmatic"
	DefaultTargetingKeyPrefix   = "hb"

	SChainDBKey           = "sChain"
	SChainObjectDBKey     = "sChainObj"
	SChainKey             = "schain"
	SChainConfigKey       = "config"
	BidderSChainObjectKey = "bidderSChainObject"
	AllBidderSChainObj    = "allBidderSChainObject"
	AllBidderSChainKey    = "allBidderSChain"

	PriceFloorURL        = "jsonUrl"
	FloorModuleEnabled   = "floorPriceModuleEnabled"
	FloorType            = "floorType"
	SoftFloorType        = "soft"
	HardFloorType        = "hard"
	FloorMin             = "floorMin"
	FloorDealEnforcement = "dealsEnforcement"
	FloorsURLTemplate    = "https://ads.pubmatic.com/AdServer/js/pwt/floors/%d/%d/floors.json"

	OwRedirectURL = "owRedirectURL"

	//include brand categories values
	IncludeNoCategory            = 0
	IncludeIABBranchCategory     = 1
	IncludeAdServerBrandCategory = 2

	//OpenWrap Primary AdServer DFP
	OWPrimaryAdServerDFP = "DFP"

	//Prebid Primary AdServers
	PrebidPrimaryAdServerFreeWheel = "freewheel"
	PrebidPrimaryAdServerDFP       = "dfp"

	//Prebid Primary AdServer ID's
	PrebidPrimaryAdServerFreeWheelID = 1
	PrebidPrimaryAdServerDFPID       = 2

	//ab test constants
	AbTestEnabled              = "abTestEnabled"
	TestGroupSize              = "testGroupSize"
	TestType                   = "testType"
	PartnerTestEnabledKey      = "testEnabled"
	TestTypeAuctionTimeout     = "Auction Timeout"
	TestTypePartners           = "Partners"
	TestTypeClientVsServerPath = "Client-side vs. Server-side Path"

	DataTypeUnknown         = 0
	DataTypeInteger         = 1
	DataTypeFloat           = 2
	DataTypeString          = 3
	DataTypeBoolean         = 4
	DataTypeArrayOfIntegers = 5
	DataTypeArrayOfFloats   = 6
	DataTypeArrayOfStrings  = 7

	Device           = "device"
	DeviceType       = "deviceType"
	DeviceExtIFAType = "ifa_type"
	DeviceExtATTS    = "atts"

	//constant for native tracker
	EventTrackers = "eventtrackers"
	ImpTrackers   = "imptrackers"
	Event         = "event"
	Methods       = "methods"
	EventValue    = "1"
	MethodValue   = "1"

	//constants for Universal Pixel
	PixelTypeUrl  = "url"
	PixelTypeJS   = "js"
	PixelPosAbove = "above"
	PixelPosBelow = "below"

	//constants for tracker impCountingMethod
	ImpCountingMethod = "imp_ct_mthd"

	DealIDNotApplicable   = "na"
	DealTierNotApplicable = "na"
	PwtDealTier           = "pwtdealtier"
	DealTierLineItemSetup = "dealTierLineItemSetup"
	DealIDLineItemSetup   = "dealIdLineItemSetup"

	//floor types
	SoftFloor = 0
	HardFloor = 1

	CustomDimensions = "cds"
	Enabled          = "1"

	// VAST Unwrap
	RequestContext          = "rctx"
	UnwrapCount             = "unwrap-count"
	UnwrapStatus            = "unwrap-status"
	Timeout                 = "Timeout"
	UnwrapSucessStatus      = "0"
	UnwrapEmptyVASTStatus   = "4"
	UnwrapInvalidVASTStatus = "6"
	UnwrapTimeout           = "unwrap-timeout"
	MediaTypeVideo          = "video"
	ProfileId               = "profileID"
	VersionId               = "versionID"
	DisplayId               = "DisplayID"
	XUserIP                 = "X-Forwarded-For"
	XUserAgent              = "X-Device-User-Agent"
	CreativeID              = "unwrap-ucrid"
	PubID                   = "pub_id"
	ImpressionID            = "imr_id"

	//Constants for new SDK reporting
	ProfileTypeKey        = "type"
	AppPlatformKey        = "appPlatform"
	IntegrationPathKey    = "integrationPath"
	SubIntegrationPathKey = "subIntegrationPath"
	// SubIntegrationPathAdMobSDKBidding is the profile subIntegrationPath value for AdMob SDK Bidding mediation.
	SubIntegrationPathAdMobSDKBidding = "AdMob - SDK Bidding"
	// AppSubIntegrationPathIDAdMobSDKBidding is the app_sub_integration_path id for SubIntegrationPathAdMobSDKBidding (must match PubMatic DB).
	AppSubIntegrationPathIDAdMobSDKBidding           = 16
	AppSubIntegrationPathIDGoogleAdManagerSDKBidding = 14

	//constants for SDK features
	CTAOVERLAY = "ctaoverlay"
)

const (
	MACRO_WIDTH         = "_W_"
	MACRO_HEIGHT        = "_H_"
	MACRO_AD_UNIT_ID    = "_AU_"
	MACRO_AD_UNIT_INDEX = "_AUI_"
	MACRO_INTEGER       = "_I_"
	MACRO_DIV           = "_DIV_"
	MACRO_SOURCE        = "_SRC_"
	MACRO_VASTTAG       = "_VASTTAG_"

	ADUNIT_SIZE_KGP           = "_AU_@_W_x_H_"
	ADUNIT_SIZE_REGEX_KGP     = "_RE_@_W_x_H_"
	REGEX_KGP                 = "_AU_@_DIV_@_W_x_H_"
	DIV_SIZE_KGP              = "_DIV_@_W_x_H_"
	ADUNIT_SOURCE_VASTTAG_KGP = "_AU_@_SRC_@_VASTTAG_"
	SIZE_KGP                  = "_W_x_H_@_W_x_H_"
)

var (
	//EmptyVASTResponse Empty VAST Response
	EmptyVASTResponse = []byte(`<VAST version="2.0"/>`)
	//EmptyString to check for empty value
	EmptyString = ""
	//HeaderOpenWrapStatus Status of OW Request
	HeaderOpenWrapStatus = "X-Ow-Status"
	//ErrorFormat parsing error format
	ErrorFormat = `{"` + ERROR_CODE + `":%v,"` + ERROR_STRING + `":"%s"}`
	//ContentType HTTP Response Header Content Type
	ContentType = `Content-Type`
	//ContentTypeApplicationJSON HTTP Header Content-Type Value
	ContentTypeApplicationJSON = `application/json`
	//ContentTypeApplicationXML HTTP Header Content-Type Value
	ContentTypeApplicationXML = `application/xml`
	//EmptyJSONResponse Empty JSON Response
	EmptyJSONResponse = []byte{}
	//VASTErrorResponse VAST Error Response
	VASTErrorResponse = `<VAST version="2.0"><Ad><InLine><Extensions><Extension><OWStatus><Error code="%v">%v</Error></OWStatus></Extension></Extensions></InLine></Ad></VAST>`
	//TrackerCallWrap
	TrackerCallWrap = `<div style="position:absolute;left:0px;top:0px;visibility:hidden;"><img src="${escapedUrl}"></div>`
	//Tracker Format for Native
	NativeTrackerMacro = `{"event":1,"method":1,"url":"${trackerUrl}"}`
	//TrackerCallWrapOMActive for Open Measurement in In-App Banner
	TrackerCallWrapOMActive = `<script id="OWPubOMVerification" data-owurl="${escapedUrl}" src="${OMScript}"></script>`
	//Universal Pixel Macro
	UniversalPixelMacroForUrl = `<div style="position:absolute;left:0px;top:0px;visibility:hidden;"><img src="${pixelUrl}"></div>`
)

// LogOnlyWinBidArr is an array containing Partners who only want winning bids to be logged
var LogOnlyWinBidArr = []string{"facebook"}

// contextKey will be used to pass the object through request.Context
type contextKey string

const (
	ContextOWLoggerKey contextKey = "owlogger"
)

const (
	AnanlyticsThrottlingLoggerType  = "wl"
	AnanlyticsThrottlingTrackerType = "wt"
)

const (
	Pipe           = "|"
	BidIdSeparator = "::"
)

const (
	EndpointV25            = "v25"
	EndpointAMP            = "amp"
	EndpintInappVideo      = "inappvideo"
	EndpointVideo          = "video"
	EndpointJson           = "json"
	EndpointORTB           = "ortb"
	EndpointVAST           = "vast"
	EndpointWebS2S         = "webs2s"
	EndPointCTV            = "ctv"
	EndpointHybrid         = "hybrid"
	EndpointAppLovinMax    = "applovinmax"
	EndpointGoogleSDK      = "googlesdk"
	EndpointUnityLevelPlay = "ulevelplay"
	EndpointAPS            = "aps"
	EndpointGeo            = "geo"
	EndpointInvalid        = "invalid"
	Openwrap               = "openwrap"
	ImpTypeBanner          = "banner"
	ImpTypeVideo           = "video"
	ImpTypeNative          = "native"
	ContentTypeSite        = "site"
	ContentTypeApp         = "app"
)

const (
	PartnerErrNoBid              = "no_bid"
	PartnerErrTimeout            = "timeout"
	PartnerErrUnknownPrebidError = "unknown"
)

// enum to report the error at partner-config level
const (
	PartnerErrSlotNotMapped = iota // 0
	PartnerErrMisConfig            //1
)

const (
	ArraySeparator = ","
)

const (
	OWExactVideoAdDurationMatching   = `exact`
	OWRoundupVideoAdDurationMatching = `roundup`
)

const (
	// MaximizeForDuration algorithm tends towards Ad Pod Maximum Duration, Ad Slot Maximum Duration
	// and Maximum number of Ads. Accordingly it computes the number of impressions
	MaximizeForDuration = iota
	// MinMaxAlgorithm algorithm ensures all possible impression breaks are plotted by considering
	// minimum as well as maxmimum durations and ads received in the ad pod request.
	// It computes number of impressions with following steps
	//  1. Passes input configuration as it is (Equivalent of MaximizeForDuration algorithm)
	//	2. Ad Pod Duration = Ad Pod Max Duration, Number of Ads = max ads
	//	3. Ad Pod Duration = Ad Pod Max Duration, Number of Ads = min ads
	//	4. Ad Pod Duration = Ad Pod Min Duration, Number of Ads = max ads
	//	5. Ad Pod Duration = Ad Pod Min Duration, Number of Ads = min ads
	MinMaxAlgorithm
	// ByDurationRanges algorithm plots the impression objects based on expected video duration
	// ranges reveived in the input prebid-request. Based on duration matching policy
	// it will generate the impression objects. in case 'exact' duration matching impression
	// min duration = max duration. In case 'round up' this algorithm will not be executed.Instead
	ByDurationRanges
)

// constants for query_type label in stats
const (
	PartnerConfigQuery                 = "GetPartnerConfig"
	WrapperSlotMappingsQuery           = "GetWrapperSlotMappingsQuery"
	WrapperLiveVersionSlotMappings     = "GetWrapperLiveVersionSlotMappings"
	AdunitConfigQuery                  = "GetAdunitConfigQuery"
	AdunitConfigForLiveVersion         = "GetAdunitConfigForLiveVersion"
	SlotNameHash                       = "GetSlotNameHash"
	PublisherVASTTagsQuery             = "GetPublisherVASTTagsQuery"
	AdUnitFailUnmarshal                = "GetAdUnitUnmarshal"
	PublisherFeatureMapQuery           = "GetPublisherFeatureMapQuery"
	AnalyticsThrottlingPercentageQuery = "GetAnalyticsThrottlingPercentage"
	GetAdpodConfig                     = "GetAdpodConfig"
	// DisplayVersionInnerQuery           = "DisplayVersionInnerQuery"
	LiveVersionInnerQuery = "LiveVersionInnerQuery"
	//PMSlotToMappings               = "GetPMSlotToMappings"
	TestQuery                        = "TestQuery"
	ProfileTypePlatformMapQuery      = "GetProfileTypePlatformMapQuery"
	AppIntegrationPathMapQuery       = "GetAppIntegrationPathMapQuery"
	AppSubIntegrationPathMapQuery    = "GetAppSubIntegrationPathMapQuery"
	GDPRCountryCodesQuery            = "GetGDPRCountryCodes"
	ProfileAdUnitMultiFloorsQuery    = "GetProfileAdUnitMultiFloors"
	CountryPartnerFilteringDataQuery = "GetCountryPartnerFilteringData"
	PerformanceDSPsQuery             = "GetPerformanceDSPs"
	InViewEnabledPublishersQuery     = "GetInViewEnabledPublishers"
	AllDspFscAndActPcntQuery         = "GetAllDspFscAndActPcntQuery"
)

// constants for owlogger Integration Type
const (
	TypeTag    = "tag"
	TypeInline = "inline"
	TypeAmp    = "amp"
	TypeSDK    = "sdk"
	TypeS2S    = "s2s"
	TypeWebS2S = "webs2s"
)

// constants to accept request-test value
type testValue = int8

const (
	TestValueTwo testValue = 2
)

const (
	Success = "success"
	Failure = "failure"
)

const (
	AdPodEnabled = 1
)

// constants for imp.Ext.Data fields
const (
	Pbadslot    = "pbadslot"
	GamAdServer = "gam"
)

// constants for adpod type
const (
	AdPodTypeDynamic    = "dynamic"
	AdPodTypeStructured = "structured"
	AdPodTypeHybrid     = "hybrid"
)

// constants for feature id
const (
	FeatureFSC                 = 1
	FeatureTBF                 = 2
	FeatureAMPMultiFormat      = 3
	FeatureAnalyticsThrottle   = 4
	FeatureMaxFloors           = 5
	FeatureBidRecovery         = 6
	FeatureApplovinMultiFloors = 7
	FeatureImpCountingMethod   = 8
	FeatureMBMFCountry         = 9
	FeatureMBMFPublisher       = 10
	FeatureMBMFInstlFloors     = 11
	FeatureMBMFRwddFloors      = 12
	FeatureMBMFBannerFloors    = 13
	FeatureDynamicFloor        = 15
	FeatureACT                 = 17
)

// constants for sdk integrations
const (
	Agent                     = "agent"
	AppLovinMaxAgent          = "max"
	GoogleSDKAgent            = "googlesdk"
	UnityLevelPlayAgent       = "ulevelplay"
	ApsAgent                  = "aps"
	TypeRewarded              = "rewarded"
	SignalData                = "signaldata"
	OwSspBurl                 = "owsspburl"
	MissingSignal             = "missing"
	InvalidSignal             = "invalid"
	AppStoreUrl               = "appStoreUrl"
	SendBurl                  = "sendburl"
	MultiBidMultiFloorValue   = "mbmfv"
	BidExtBidExpEnf           = "bidexp_enf"
	ProcessingTime            = "processing_time_ms"
	AdUnitFormatInstl         = "instl"
	AdUnitFormatBanner        = "banner"
	AdUnitFormatRwddVideo     = "rwddvideo"
	DefaultAdUnitFormatFloors = 0
)

// constants for log level
const (
	LogLevelDebug glog.Level = 3
)

const (
	// ErrDBQueryFailed reponse error
	ErrDBQueryFailed       = `[DBError] query:[%s] pubid:[%v] profileid:[%v] error:[%s]`
	ErrDBRowScanFailed     = `[DBRowsError] query:[%s] pubid:[%v] profileid:[%v] err:[%s]`
	EmptyPartnerConfig     = `[EmptyPartnerConfig] pubid:[%v] profileid:[%v] version:[%v]`
	ErrMBMFFloorsUnmarshal = `[ErrMBMFFloorsUnmarshal] pubid:[%v] profileid:[%v] error:[%s]`
)

// constants for MBMF error codes for metrics
const (
	MBMFSuccess              = 0
	MBMFCountryDisabled      = 1
	MBMFPubDisabled          = 2
	MBMFAdUnitFormatDisabled = 3
	MBMFAdUnitDisabled       = 4
	MBMFAdUnitFormatNotFound = 5
	MBMFNoEntryFound         = 6
	MBMFInvalidAdFormat      = 7
)

// constants for video protocol
const (
	CreativeVAST43Wrapper = 16
)
