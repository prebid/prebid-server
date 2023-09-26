package openrtb_ext

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xeipuuv/gojsonschema"
)

// BidderName refers to a core bidder id or an alias id.
type BidderName string

var aliasBidderToParent map[BidderName]BidderName = map[BidderName]BidderName{}

var coreBidderNames []BidderName = []BidderName{
	Bidder33Across,
	BidderAax,
	BidderAceex,
	BidderAcuityAds,
	BidderAdf,
	BidderAdform,
	BidderAdgeneration,
	BidderAdhese,
	BidderAdkernel,
	BidderAdkernelAdn,
	BidderAdman,
	BidderAdmixer,
	BidderAdnuntius,
	BidderAdOcean,
	BidderAdoppler,
	BidderAdot,
	BidderAdpone,
	BidderAdprime,
	BidderAdquery,
	BidderAdrino,
	BidderAdsinteractive,
	BidderAdsyield,
	BidderAdtarget,
	BidderAdtrgtme,
	BidderAdtelligent,
	BidderAdvangelists,
	BidderAdView,
	BidderAdxcg,
	BidderAdyoulike,
	BidderAidem,
	BidderAJA,
	BidderAlgorix,
	BidderAMX,
	BidderApacdex,
	BidderAppnexus,
	BidderAppstock,
	BidderAppush,
	BidderAudienceNetwork,
	BidderAutomatad,
	BidderAvocet,
	BidderAxis,
	BidderAxonix,
	BidderBeachfront,
	BidderBeintoo,
	BidderBematterfull,
	BidderBetween,
	BidderBeyondMedia,
	BidderBidmachine,
	BidderBidmyadz,
	BidderBidsCube,
	BidderBidstack,
	BidderBizzclick,
	BidderBliink,
	BidderBlue,
	BidderBluesea,
	BidderBmtm,
	BidderBoldwin,
	BidderBrave,
	BidderCadentApertureMX,
	BidderCcx,
	BidderCoinzilla,
	BidderColossus,
	BidderCompass,
	BidderConnectAd,
	BidderConsumable,
	BidderConversant,
	BidderCopper6,
	BidderCpmstar,
	BidderCriteo,
	BidderCWire,
	BidderDatablocks,
	BidderDecenterAds,
	BidderDeepintent,
	BidderDianomi,
	BidderDmx,
	BidderEmtv,
	BidderEmxDigital,
	BidderEPlanning,
	BidderEpom,
	BidderEpsilon,
	BidderEVolution,
	BidderEvtech,
	BidderFlipp,
	BidderFreewheelSSP,
	BidderFreewheelSSPOld,
	BidderFRVRAdNetwork,
	BidderGamma,
	BidderGamoshi,
	BidderGlobalsun,
	BidderGothamads,
	BidderGreedygame,
	BidderGrid,
	BidderGumGum,
	BidderHuaweiAds,
	BidderIionads,
	BidderImds,
	BidderImpactify,
	BidderImprovedigital,
	BidderInfyTV,
	BidderInMobi,
	BidderInteractiveoffers,
	BidderInvibes,
	BidderIQZone,
	BidderIx,
	BidderJANet,
	BidderJixie,
	BidderKargo,
	BidderKayzen,
	BidderKidoz,
	BidderKiviads,
	BidderLmKiviads,
	BidderKrushmedia,
	BidderLiftoff,
	BidderLimelightDigital,
	BidderLockerDome,
	BidderLogan,
	BidderLogicad,
	BidderLunaMedia,
	BidderMabidder,
	BidderMadvertise,
	BidderMarsmedia,
	BidderMediafuse,
	BidderMedianet,
	BidderMgid,
	BidderMgidX,
	BidderMobfoxpb,
	BidderMobileFuse,
	BidderMotorik,
	BidderNextMillennium,
	BidderNoBid,
	BidderOneTag,
	BidderOpenWeb,
	BidderOpenx,
	BidderOperaads,
	BidderOrbidder,
	BidderOutbrain,
	BidderOwnAdx,
	BidderPangle,
	BidderPGAM,
	BidderPGAMSsp,
	BidderPubmatic,
	BidderPubnative,
	BidderPulsepoint,
	BidderPWBid,
	BidderQuantumdex,
	BidderRevcontent,
	BidderRichaudience,
	BidderRise,
	BidderRTBHouse,
	BidderRubicon,
	BidderSeedingAlliance,
	BidderSaLunaMedia,
	BidderScreencore,
	BidderSharethrough,
	BidderSilverMob,
	BidderSilverPush,
	BidderSmaato,
	BidderSmartAdserver,
	BidderSmartHub,
	BidderSmartRTB,
	BidderSmartyAds,
	BidderSmileWanted,
	BidderSonobi,
	BidderSovrn,
	BidderSspBC,
	BidderStreamkey,
	BidderStroeerCore,
	BidderSuntContent,
	BidderSynacormedia,
	BidderTaboola,
	BidderTappx,
	BidderTelaria,
	BidderTpmn,
	BidderTrafficGate,
	BidderTriplelift,
	BidderTripleliftNative,
	BidderTrustX,
	BidderUcfunnel,
	BidderUndertone,
	BidderUnicorn,
	BidderUnruly,
	BidderValueImpression,
	BidderVideoByte,
	BidderVideoHeroes,
	BidderVidoomy,
	BidderViewdeos,
	BidderVisibleMeasures,
	BidderVisx,
	BidderVox,
	BidderVrtcal,
	BidderXeworks,
	BidderXtrmqb,
	BidderYahooAds,
	BidderYahooAdvertising,
	BidderYahooSSP,
	BidderYieldlab,
	BidderYieldmo,
	BidderYieldone,
	BidderZeroClickFraud,
	BidderZetaGlobalSsp,
}

func GetAliasBidderToParent() map[BidderName]BidderName {
	return aliasBidderToParent
}

func SetAliasBidderName(aliasBidderName string, parentBidderName BidderName) error {
	if IsBidderNameReserved(aliasBidderName) {
		return fmt.Errorf("alias %s is a reserved bidder name and cannot be used", aliasBidderName)
	}
	aliasBidder := BidderName(aliasBidderName)
	coreBidderNames = append(coreBidderNames, aliasBidder)
	aliasBidderToParent[aliasBidder] = parentBidderName
	bidderNameLookup[strings.ToLower(aliasBidderName)] = aliasBidder
	return nil
}

func (name BidderName) MarshalJSON() ([]byte, error) {
	return []byte(name), nil
}

func (name *BidderName) String() string {
	if name == nil {
		return ""
	}
	return string(*name)
}

// Names of reserved bidders. These names may not be used by a core bidder or alias.
const (
	BidderReservedAll     BidderName = "all"     // Reserved for the /info/bidders/all endpoint.
	BidderReservedContext BidderName = "context" // Reserved for first party data.
	BidderReservedData    BidderName = "data"    // Reserved for first party data.
	BidderReservedGeneral BidderName = "general" // Reserved for non-bidder specific messages when using a map keyed on the bidder name.
	BidderReservedGPID    BidderName = "gpid"    // Reserved for Global Placement ID (GPID).
	BidderReservedPrebid  BidderName = "prebid"  // Reserved for Prebid Server configuration.
	BidderReservedSKAdN   BidderName = "skadn"   // Reserved for Apple's SKAdNetwork OpenRTB extension.
	BidderReservedTID     BidderName = "tid"     // Reserved for Per-Impression Transactions IDs for Multi-Impression Bid Requests.
	BidderReservedAE      BidderName = "ae"      // Reserved for FLEDGE Auction Environment
)

// IsBidderNameReserved returns true if the specified name is a case insensitive match for a reserved bidder name.
func IsBidderNameReserved(name string) bool {
	if strings.EqualFold(name, string(BidderReservedAll)) {
		return true
	}

	if strings.EqualFold(name, string(BidderReservedContext)) {
		return true
	}

	if strings.EqualFold(name, string(BidderReservedData)) {
		return true
	}

	if strings.EqualFold(name, string(BidderReservedGeneral)) {
		return true
	}

	if strings.EqualFold(name, string(BidderReservedGPID)) {
		return true
	}

	if strings.EqualFold(name, string(BidderReservedSKAdN)) {
		return true
	}

	if strings.EqualFold(name, string(BidderReservedPrebid)) {
		return true
	}

	if strings.EqualFold(name, string(BidderReservedTID)) {
		return true
	}

	if strings.EqualFold(name, string(BidderReservedAE)) {
		return true
	}

	return false
}

// Names of core bidders. These names *must* match the bidder code in Prebid.js if an adapter also exists in that
// project. You may *not* use the name 'general' as that is reserved for general error messages nor 'context' as
// that is reserved for first party data.
//
// Please keep this list alphabetized to minimize merge conflicts.
const (
	Bidder33Across          BidderName = "33across"
	BidderAax               BidderName = "aax"
	BidderAceex             BidderName = "aceex"
	BidderAcuityAds         BidderName = "acuityads"
	BidderAdf               BidderName = "adf"
	BidderAdform            BidderName = "adform"
	BidderAdgeneration      BidderName = "adgeneration"
	BidderAdhese            BidderName = "adhese"
	BidderAdkernel          BidderName = "adkernel"
	BidderAdkernelAdn       BidderName = "adkernelAdn"
	BidderAdman             BidderName = "adman"
	BidderAdmixer           BidderName = "admixer"
	BidderAdnuntius         BidderName = "adnuntius"
	BidderAdOcean           BidderName = "adocean"
	BidderAdoppler          BidderName = "adoppler"
	BidderAdot              BidderName = "adot"
	BidderAdpone            BidderName = "adpone"
	BidderAdprime           BidderName = "adprime"
	BidderAdquery           BidderName = "adquery"
	BidderAdrino            BidderName = "adrino"
	BidderAdsinteractive    BidderName = "adsinteractive"
	BidderAdsyield          BidderName = "adsyield"
	BidderAdtarget          BidderName = "adtarget"
	BidderAdtrgtme          BidderName = "adtrgtme"
	BidderAdtelligent       BidderName = "adtelligent"
	BidderAdvangelists      BidderName = "advangelists"
	BidderAdView            BidderName = "adview"
	BidderAdxcg             BidderName = "adxcg"
	BidderAdyoulike         BidderName = "adyoulike"
	BidderAidem             BidderName = "aidem"
	BidderAJA               BidderName = "aja"
	BidderAlgorix           BidderName = "algorix"
	BidderAMX               BidderName = "amx"
	BidderApacdex           BidderName = "apacdex"
	BidderAppnexus          BidderName = "appnexus"
	BidderAppstock          BidderName = "appstock"
	BidderAppush            BidderName = "appush"
	BidderAudienceNetwork   BidderName = "audienceNetwork"
	BidderAutomatad         BidderName = "automatad"
	BidderAvocet            BidderName = "avocet"
	BidderAxis              BidderName = "axis"
	BidderAxonix            BidderName = "axonix"
	BidderBeachfront        BidderName = "beachfront"
	BidderBeintoo           BidderName = "beintoo"
	BidderBematterfull      BidderName = "bematterfull"
	BidderBetween           BidderName = "between"
	BidderBeyondMedia       BidderName = "beyondmedia"
	BidderBidmachine        BidderName = "bidmachine"
	BidderBidmyadz          BidderName = "bidmyadz"
	BidderBidsCube          BidderName = "bidscube"
	BidderBidstack          BidderName = "bidstack"
	BidderBizzclick         BidderName = "bizzclick"
	BidderBliink            BidderName = "bliink"
	BidderBlue              BidderName = "blue"
	BidderBluesea           BidderName = "bluesea"
	BidderBmtm              BidderName = "bmtm"
	BidderBoldwin           BidderName = "boldwin"
	BidderBrave             BidderName = "brave"
	BidderCadentApertureMX  BidderName = "cadent_aperture_mx"
	BidderCcx               BidderName = "ccx"
	BidderCoinzilla         BidderName = "coinzilla"
	BidderColossus          BidderName = "colossus"
	BidderCompass           BidderName = "compass"
	BidderConnectAd         BidderName = "connectad"
	BidderConsumable        BidderName = "consumable"
	BidderConversant        BidderName = "conversant"
	BidderCopper6           BidderName = "copper6"
	BidderCpmstar           BidderName = "cpmstar"
	BidderCriteo            BidderName = "criteo"
	BidderCWire             BidderName = "cwire"
	BidderDatablocks        BidderName = "datablocks"
	BidderDecenterAds       BidderName = "decenterads"
	BidderDeepintent        BidderName = "deepintent"
	BidderDianomi           BidderName = "dianomi"
	BidderDmx               BidderName = "dmx"
	BidderEmtv              BidderName = "emtv"
	BidderEmxDigital        BidderName = "emx_digital"
	BidderEPlanning         BidderName = "eplanning"
	BidderEpsilon           BidderName = "epsilon"
	BidderEpom              BidderName = "epom"
	BidderEVolution         BidderName = "e_volution"
	BidderEvtech            BidderName = "evtech"
	BidderFlipp             BidderName = "flipp"
	BidderFreewheelSSP      BidderName = "freewheelssp"
	BidderFreewheelSSPOld   BidderName = "freewheel-ssp"
	BidderFRVRAdNetwork     BidderName = "frvradn"
	BidderGamma             BidderName = "gamma"
	BidderGamoshi           BidderName = "gamoshi"
	BidderGlobalsun         BidderName = "globalsun"
	BidderGothamads         BidderName = "gothamads"
	BidderGreedygame        BidderName = "greedygame"
	BidderGrid              BidderName = "grid"
	BidderGumGum            BidderName = "gumgum"
	BidderHuaweiAds         BidderName = "huaweiads"
	BidderIionads           BidderName = "iionads"
	BidderImds              BidderName = "imds"
	BidderImpactify         BidderName = "impactify"
	BidderImprovedigital    BidderName = "improvedigital"
	BidderInfyTV            BidderName = "infytv"
	BidderInMobi            BidderName = "inmobi"
	BidderInteractiveoffers BidderName = "interactiveoffers"
	BidderInvibes           BidderName = "invibes"
	BidderIQZone            BidderName = "iqzone"
	BidderIx                BidderName = "ix"
	BidderJANet             BidderName = "janet"
	BidderJixie             BidderName = "jixie"
	BidderKargo             BidderName = "kargo"
	BidderKayzen            BidderName = "kayzen"
	BidderKidoz             BidderName = "kidoz"
	BidderKiviads           BidderName = "kiviads"
	BidderLmKiviads         BidderName = "lm_kiviads"
	BidderKrushmedia        BidderName = "krushmedia"
	BidderLiftoff           BidderName = "liftoff"
	BidderLimelightDigital  BidderName = "limelightDigital"
	BidderLockerDome        BidderName = "lockerdome"
	BidderLogan             BidderName = "logan"
	BidderLogicad           BidderName = "logicad"
	BidderLunaMedia         BidderName = "lunamedia"
	BidderMabidder          BidderName = "mabidder"
	BidderMadvertise        BidderName = "madvertise"
	BidderMarsmedia         BidderName = "marsmedia"
	BidderMediafuse         BidderName = "mediafuse"
	BidderMedianet          BidderName = "medianet"
	BidderMgid              BidderName = "mgid"
	BidderMgidX             BidderName = "mgidX"
	BidderMobfoxpb          BidderName = "mobfoxpb"
	BidderMobileFuse        BidderName = "mobilefuse"
	BidderMotorik           BidderName = "motorik"
	BidderNextMillennium    BidderName = "nextmillennium"
	BidderNoBid             BidderName = "nobid"
	BidderOneTag            BidderName = "onetag"
	BidderOpenWeb           BidderName = "openweb"
	BidderOpenx             BidderName = "openx"
	BidderOperaads          BidderName = "operaads"
	BidderOrbidder          BidderName = "orbidder"
	BidderOutbrain          BidderName = "outbrain"
	BidderOwnAdx            BidderName = "ownadx"
	BidderPangle            BidderName = "pangle"
	BidderPGAM              BidderName = "pgam"
	BidderPGAMSsp           BidderName = "pgamssp"
	BidderPubmatic          BidderName = "pubmatic"
	BidderPubnative         BidderName = "pubnative"
	BidderPulsepoint        BidderName = "pulsepoint"
	BidderPWBid             BidderName = "pwbid"
	BidderQuantumdex        BidderName = "quantumdex"
	BidderRevcontent        BidderName = "revcontent"
	BidderRichaudience      BidderName = "richaudience"
	BidderRise              BidderName = "rise"
	BidderRTBHouse          BidderName = "rtbhouse"
	BidderRubicon           BidderName = "rubicon"
	BidderSeedingAlliance   BidderName = "seedingAlliance"
	BidderSaLunaMedia       BidderName = "sa_lunamedia"
	BidderScreencore        BidderName = "screencore"
	BidderSharethrough      BidderName = "sharethrough"
	BidderSilverMob         BidderName = "silvermob"
	BidderSilverPush        BidderName = "silverpush"
	BidderSmaato            BidderName = "smaato"
	BidderSmartAdserver     BidderName = "smartadserver"
	BidderSmartHub          BidderName = "smarthub"
	BidderSmartRTB          BidderName = "smartrtb"
	BidderSmartyAds         BidderName = "smartyads"
	BidderSmileWanted       BidderName = "smilewanted"
	BidderSonobi            BidderName = "sonobi"
	BidderSovrn             BidderName = "sovrn"
	BidderSspBC             BidderName = "sspBC"
	BidderStreamkey         BidderName = "streamkey"
	BidderStroeerCore       BidderName = "stroeerCore"
	BidderSuntContent       BidderName = "suntContent"
	BidderSynacormedia      BidderName = "synacormedia"
	BidderTaboola           BidderName = "taboola"
	BidderTappx             BidderName = "tappx"
	BidderTelaria           BidderName = "telaria"
	BidderTpmn              BidderName = "tpmn"
	BidderTrafficGate       BidderName = "trafficgate"
	BidderTriplelift        BidderName = "triplelift"
	BidderTripleliftNative  BidderName = "triplelift_native"
	BidderTrustX            BidderName = "trustx"
	BidderUcfunnel          BidderName = "ucfunnel"
	BidderUndertone         BidderName = "undertone"
	BidderUnicorn           BidderName = "unicorn"
	BidderUnruly            BidderName = "unruly"
	BidderValueImpression   BidderName = "valueimpression"
	BidderVideoByte         BidderName = "videobyte"
	BidderVideoHeroes       BidderName = "videoheroes"
	BidderVidoomy           BidderName = "vidoomy"
	BidderViewdeos          BidderName = "viewdeos"
	BidderVisibleMeasures   BidderName = "visiblemeasures"
	BidderVisx              BidderName = "visx"
	BidderVox               BidderName = "vox"
	BidderVrtcal            BidderName = "vrtcal"
	BidderXeworks           BidderName = "xeworks"
	BidderXtrmqb            BidderName = "xtrmqb"
	BidderYahooAds          BidderName = "yahooAds"
	BidderYahooAdvertising  BidderName = "yahooAdvertising"
	BidderYahooSSP          BidderName = "yahoossp"
	BidderYieldlab          BidderName = "yieldlab"
	BidderYieldmo           BidderName = "yieldmo"
	BidderYieldone          BidderName = "yieldone"
	BidderZeroClickFraud    BidderName = "zeroclickfraud"
	BidderZetaGlobalSsp     BidderName = "zeta_global_ssp"
)

// CoreBidderNames returns a slice of all core bidders.
func CoreBidderNames() []BidderName {
	return coreBidderNames
}

// BuildBidderMap builds a map of string to BidderName, to remain compatbile with the
// prebioud BidderMap variable.
func BuildBidderMap() map[string]BidderName {
	lookup := make(map[string]BidderName)
	for _, name := range CoreBidderNames() {
		lookup[string(name)] = name
	}
	return lookup
}

// BuildBidderStringSlice builds a slioce of strings for each BidderName.
func BuildBidderStringSlice() []string {
	coreBidders := CoreBidderNames()
	slice := make([]string, len(coreBidders))
	for i, name := range CoreBidderNames() {
		slice[i] = string(name)
	}
	return slice
}

func BuildBidderNameHashSet() map[string]struct{} {
	hashSet := make(map[string]struct{})
	for _, name := range CoreBidderNames() {
		hashSet[string(name)] = struct{}{}
	}
	return hashSet
}

// bidderNameLookup is a map of the lower case version of the bidder name to the precise BidderName value.
var bidderNameLookup = func() map[string]BidderName {
	lookup := make(map[string]BidderName)
	for _, name := range CoreBidderNames() {
		bidderNameLower := strings.ToLower(string(name))
		lookup[bidderNameLower] = name
	}
	return lookup
}()

func NormalizeBidderName(name string) (BidderName, bool) {
	nameLower := strings.ToLower(name)
	bidderName, exists := bidderNameLookup[nameLower]
	return bidderName, exists
}

// The BidderParamValidator is used to enforce bidrequest.imp[i].ext.prebid.bidder.{anyBidder} values.
//
// This is treated differently from the other types because we rely on JSON-schemas to validate bidder params.
type BidderParamValidator interface {
	Validate(name BidderName, ext json.RawMessage) error
	// Schema returns the JSON schema used to perform validation.
	Schema(name BidderName) string
}

type bidderParamsFileSystem interface {
	readDir(name string) ([]os.DirEntry, error)
	readFile(name string) ([]byte, error)
	newReferenceLoader(source string) gojsonschema.JSONLoader
	newSchema(l gojsonschema.JSONLoader) (*gojsonschema.Schema, error)
	abs(path string) (string, error)
}

type standardBidderParamsFileSystem struct{}

func (standardBidderParamsFileSystem) readDir(name string) ([]os.DirEntry, error) {
	return os.ReadDir(name)
}

func (standardBidderParamsFileSystem) readFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

func (standardBidderParamsFileSystem) newReferenceLoader(source string) gojsonschema.JSONLoader {
	return gojsonschema.NewReferenceLoader(source)
}

func (standardBidderParamsFileSystem) newSchema(l gojsonschema.JSONLoader) (*gojsonschema.Schema, error) {
	return gojsonschema.NewSchema(l)
}

func (standardBidderParamsFileSystem) abs(path string) (string, error) {
	return filepath.Abs(path)
}

var paramsValidator bidderParamsFileSystem = standardBidderParamsFileSystem{}

// NewBidderParamsValidator makes a BidderParamValidator, assuming all the necessary files exist in the filesystem.
// This will error if, for example, a Bidder gets added but no JSON schema is written for them.
func NewBidderParamsValidator(schemaDirectory string) (BidderParamValidator, error) {
	fileInfos, err := paramsValidator.readDir(schemaDirectory)
	if err != nil {
		return nil, fmt.Errorf("Failed to read JSON schemas from directory %s. %v", schemaDirectory, err)
	}

	bidderMap := BuildBidderMap()

	schemaContents := make(map[BidderName]string, 50)
	schemas := make(map[BidderName]*gojsonschema.Schema, 50)
	for _, fileInfo := range fileInfos {
		bidderName := strings.TrimSuffix(fileInfo.Name(), ".json")
		if _, ok := bidderMap[bidderName]; !ok {
			return nil, fmt.Errorf("File %s/%s does not match a valid BidderName.", schemaDirectory, fileInfo.Name())
		}
		toOpen, err := paramsValidator.abs(filepath.Join(schemaDirectory, fileInfo.Name()))
		if err != nil {
			return nil, fmt.Errorf("Failed to get an absolute representation of the path: %s, %v", toOpen, err)
		}
		schemaLoader := paramsValidator.newReferenceLoader("file:///" + filepath.ToSlash(toOpen))
		loadedSchema, err := paramsValidator.newSchema(schemaLoader)
		if err != nil {
			return nil, fmt.Errorf("Failed to load json schema at %s: %v", toOpen, err)
		}

		fileBytes, err := paramsValidator.readFile(fmt.Sprintf("%s/%s", schemaDirectory, fileInfo.Name()))
		if err != nil {
			return nil, fmt.Errorf("Failed to read file %s/%s: %v", schemaDirectory, fileInfo.Name(), err)
		}

		schemas[BidderName(bidderName)] = loadedSchema
		schemaContents[BidderName(bidderName)] = string(fileBytes)
	}

	//set alias bidder params schema to its parent
	for alias, parent := range aliasBidderToParent {
		parentSchema := schemas[parent]
		schemas[alias] = parentSchema

		parentSchemaContents := schemaContents[parent]
		schemaContents[alias] = parentSchemaContents
	}

	return &bidderParamValidator{
		schemaContents: schemaContents,
		parsedSchemas:  schemas,
	}, nil
}

type bidderParamValidator struct {
	schemaContents map[BidderName]string
	parsedSchemas  map[BidderName]*gojsonschema.Schema
}

func (validator *bidderParamValidator) Validate(name BidderName, ext json.RawMessage) error {
	result, err := validator.parsedSchemas[name].Validate(gojsonschema.NewBytesLoader(ext))
	if err != nil {
		return err
	}
	if !result.Valid() {
		errBuilder := bytes.NewBuffer(make([]byte, 0, 300))
		for _, err := range result.Errors() {
			errBuilder.WriteString(err.String())
		}
		return errors.New(errBuilder.String())
	}
	return nil
}

func (validator *bidderParamValidator) Schema(name BidderName) string {
	return validator.schemaContents[name]
}
