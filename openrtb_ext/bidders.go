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
	BidderAdagio,
	BidderAdelement,
	BidderAdf,
	BidderAdgeneration,
	BidderAdhese,
	BidderAdkernel,
	BidderAdkernelAdn,
	BidderAdman,
	BidderAdmatic,
	BidderAdmixer,
	BidderAdnuntius,
	BidderAdOcean,
	BidderAdoppler,
	BidderAdot,
	BidderAdpone,
	BidderAdprime,
	BidderAdquery,
	BidderAdrino,
	BidderAdsInteractive,
	BidderAdsinteractive,
	BidderAdtarget,
	BidderAdtrgtme,
	BidderAdtelligent,
	BidderAdTonos,
	BidderAdUpTech,
	BidderAdvangelists,
	BidderAdverxo,
	BidderAdView,
	BidderAdxcg,
	BidderAdyoulike,
	BidderAidem,
	BidderAJA,
	BidderAkcelo,
	BidderAlgorix,
	BidderAlkimi,
	BidderAMX,
	BidderApacdex,
	BidderAppnexus,
	BidderAppush,
	BidderAso,
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
	BidderBidmatic,
	BidderBidmyadz,
	BidderBidsCube,
	BidderBidstack,
	BidderBidtheatre,
	BidderBigoAd,
	BidderBlasto,
	BidderBliink,
	BidderBlue,
	BidderBluesea,
	BidderBmtm,
	BidderBoldwin,
	BidderBrave,
	BidderBWX,
	BidderCadentApertureMX,
	BidderCcx,
	BidderCointraffic,
	BidderCoinzilla,
	BidderColossus,
	BidderCompass,
	BidderConcert,
	BidderConnatix,
	BidderConnectAd,
	BidderConsumable,
	BidderConversant,
	BidderCopper6ssp,
	BidderCpmstar,
	BidderCriteo,
	BidderCWire,
	BidderDatablocks,
	BidderDecenterAds,
	BidderDeepintent,
	BidderDefinemedia,
	BidderDianomi,
	BidderDisplayio,
	BidderEdge226,
	BidderDmx,
	BidderDXKulture,
	BidderDriftPixel,
	BidderEmtv,
	BidderEmxDigital,
	BidderEPlanning,
	BidderEpom,
	BidderEscalax,
	BidderEVolution,
	BidderExco,
	BidderFeedAd,
	BidderFlatads,
	BidderFlipp,
	BidderFreewheelSSP,
	BidderFWSSP,
	BidderFRVRAdNetwork,
	BidderGamma,
	BidderGamoshi,
	BidderGlobalsun,
	BidderGothamads,
	BidderGrid,
	BidderGumGum,
	BidderHuaweiAds,
	BidderImds,
	BidderImpactify,
	BidderImprovedigital,
	BidderInfyTV,
	BidderInMobi,
	BidderInsticator,
	BidderInteractiveoffers,
	BidderIntertech,
	BidderInvibes,
	BidderIQX,
	BidderIQZone,
	BidderIx,
	BidderJixie,
	BidderKargo,
	BidderKayzen,
	BidderKidoz,
	BidderKiviads,
	BidderLmKiviads,
	BidderKobler,
	BidderKrushmedia,
	BidderKueezRTB,
	BidderLemmadigital,
	BidderLimelightDigital,
	BidderLockerDome,
	BidderLogan,
	BidderLogicad,
	BidderLoopme,
	BidderLoyal,
	BidderLunaMedia,
	BidderMabidder,
	BidderMadSense,
	BidderMadvertise,
	BidderMarsmedia,
	BidderMediafuse,
	BidderMediaGo,
	BidderMedianet,
	BidderMediasquare,
	BidderMeloZen,
	BidderMetaX,
	BidderMgid,
	BidderMgidX,
	BidderMinuteMedia,
	BidderMissena,
	BidderMobfoxpb,
	BidderMobileFuse,
	BidderMobkoi,
	BidderMotorik,
	BidderNativo,
	BidderNextMillennium,
	BidderNexx360,
	BidderNoBid,
	BidderOgury,
	BidderOms,
	BidderOneTag,
	BidderOpenWeb,
	BidderOpenx,
	BidderOperaads,
	BidderOptidigital,
	BidderOraki,
	BidderOrbidder,
	BidderOutbrain,
	BidderOwnAdx,
	BidderPangle,
	BidderPGAMSsp,
	BidderPlaydigo,
	BidderPubmatic,
	BidderPubrise,
	BidderPubnative,
	BidderPulsepoint,
	BidderPWBid,
	BidderQT,
	BidderReadpeak,
	BidderRediads,
	BidderRelevantDigital,
	BidderResetDigital,
	BidderRevcontent,
	BidderRichaudience,
	BidderRise,
	BidderRoulax,
	BidderRTBHouse,
	BidderRubicon,
	BidderSeedingAlliance,
	BidderSeedtag,
	BidderSaLunaMedia,
	BidderScreencore,
	BidderSharethrough,
	BidderSilverMob,
	BidderSilverPush,
	BidderSmaato,
	BidderSmartAdserver,
	BidderSmartHub,
	BidderSmartRTB,
	BidderSmartx,
	BidderSmartyAds,
	BidderSmileWanted,
	BidderSmoot,
	BidderSmrtconnect,
	BidderSonobi,
	BidderSovrn,
	BidderSovrnXsp,
	BidderSparteo,
	BidderSspBC,
	BidderStartIO,
	BidderStroeerCore,
	BidderTaboola,
	BidderTappx,
	BidderTeads,
	BidderTelaria,
	BidderTheadx,
	BidderTheTradeDesk,
	BidderTpmn,
	BidderTradPlus,
	BidderTrafficGate,
	BidderTriplelift,
	BidderTripleliftNative,
	BidderTrustedstack,
	BidderUcfunnel,
	BidderUndertone,
	BidderUnicorn,
	BidderUnruly,
	BidderVidazoo,
	BidderVideoByte,
	BidderVideoHeroes,
	BidderVidoomy,
	BidderVisibleMeasures,
	BidderVisx,
	BidderVox,
	BidderVrtcal,
	BidderVungle,
	BidderXeworks,
	BidderYahooAds,
	BidderYandex,
	BidderYeahmobi,
	BidderYieldlab,
	BidderYieldmo,
	BidderYieldone,
	BidderZentotem,
	BidderZeroClickFraud,
	BidderZetaGlobalSsp,
	BidderZmaticoo,
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
	BidderReservedAE      BidderName = "ae"      // Reserved for PAAPI Auction Environment.
	BidderReservedIGS     BidderName = "igs"     // Reserved for PAAPI Interest Group Seller object.
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

	if strings.EqualFold(name, string(BidderReservedIGS)) {
		return true
	}

	return false
}

// IsPotentialBidder returns true if the name is not reserved within the imp[].ext context
func IsPotentialBidder(name string) bool {
	switch BidderName(name) {
	case BidderReservedContext:
		return false
	case BidderReservedData:
		return false
	case BidderReservedGPID:
		return false
	case BidderReservedPrebid:
		return false
	case BidderReservedSKAdN:
		return false
	case BidderReservedTID:
		return false
	case BidderReservedAE:
		return false
	case BidderReservedIGS:
		return false
	default:
		return true
	}
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
	BidderAdagio            BidderName = "adagio"
	BidderAdelement         BidderName = "adelement"
	BidderAdf               BidderName = "adf"
	BidderAdgeneration      BidderName = "adgeneration"
	BidderAdhese            BidderName = "adhese"
	BidderAdkernel          BidderName = "adkernel"
	BidderAdkernelAdn       BidderName = "adkernelAdn"
	BidderAdman             BidderName = "adman"
	BidderAdmatic           BidderName = "admatic"
	BidderAdmixer           BidderName = "admixer"
	BidderAdnuntius         BidderName = "adnuntius"
	BidderAdOcean           BidderName = "adocean"
	BidderAdoppler          BidderName = "adoppler"
	BidderAdot              BidderName = "adot"
	BidderAdpone            BidderName = "adpone"
	BidderAdprime           BidderName = "adprime"
	BidderAdquery           BidderName = "adquery"
	BidderAdrino            BidderName = "adrino"
	BidderAdsInteractive    BidderName = "ads_interactive"
	BidderAdsinteractive    BidderName = "adsinteractive"
	BidderAdtarget          BidderName = "adtarget"
	BidderAdtrgtme          BidderName = "adtrgtme"
	BidderAdTonos           BidderName = "adtonos"
	BidderAdtelligent       BidderName = "adtelligent"
	BidderAdUpTech          BidderName = "aduptech"
	BidderAdvangelists      BidderName = "advangelists"
	BidderAdverxo           BidderName = "adverxo"
	BidderAdView            BidderName = "adview"
	BidderAdxcg             BidderName = "adxcg"
	BidderAdyoulike         BidderName = "adyoulike"
	BidderAidem             BidderName = "aidem"
	BidderAJA               BidderName = "aja"
	BidderAkcelo            BidderName = "akcelo"
	BidderAlgorix           BidderName = "algorix"
	BidderAlkimi            BidderName = "alkimi"
	BidderAMX               BidderName = "amx"
	BidderApacdex           BidderName = "apacdex"
	BidderAppnexus          BidderName = "appnexus"
	BidderAppush            BidderName = "appush"
	BidderAso               BidderName = "aso"
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
	BidderBidmatic          BidderName = "bidmatic"
	BidderBidmyadz          BidderName = "bidmyadz"
	BidderBidsCube          BidderName = "bidscube"
	BidderBidstack          BidderName = "bidstack"
	BidderBidtheatre        BidderName = "bidtheatre"
	BidderBigoAd            BidderName = "bigoad"
	BidderBlasto            BidderName = "blasto"
	BidderBliink            BidderName = "bliink"
	BidderBlue              BidderName = "blue"
	BidderBluesea           BidderName = "bluesea"
	BidderBmtm              BidderName = "bmtm"
	BidderBoldwin           BidderName = "boldwin"
	BidderBrave             BidderName = "brave"
	BidderBWX               BidderName = "bwx"
	BidderCadentApertureMX  BidderName = "cadent_aperture_mx"
	BidderCcx               BidderName = "ccx"
	BidderCointraffic       BidderName = "cointraffic"
	BidderCoinzilla         BidderName = "coinzilla"
	BidderColossus          BidderName = "colossus"
	BidderCompass           BidderName = "compass"
	BidderConcert           BidderName = "concert"
	BidderConnatix          BidderName = "connatix"
	BidderConnectAd         BidderName = "connectad"
	BidderConsumable        BidderName = "consumable"
	BidderConversant        BidderName = "conversant"
	BidderCopper6ssp        BidderName = "copper6ssp"
	BidderCpmstar           BidderName = "cpmstar"
	BidderCriteo            BidderName = "criteo"
	BidderCWire             BidderName = "cwire"
	BidderDatablocks        BidderName = "datablocks"
	BidderDecenterAds       BidderName = "decenterads"
	BidderDeepintent        BidderName = "deepintent"
	BidderDefinemedia       BidderName = "definemedia"
	BidderDianomi           BidderName = "dianomi"
	BidderDisplayio         BidderName = "displayio"
	BidderEdge226           BidderName = "edge226"
	BidderDmx               BidderName = "dmx"
	BidderDXKulture         BidderName = "dxkulture"
	BidderDriftPixel        BidderName = "driftpixel"
	BidderEmtv              BidderName = "emtv"
	BidderEmxDigital        BidderName = "emx_digital"
	BidderEPlanning         BidderName = "eplanning"
	BidderEpom              BidderName = "epom"
	BidderEscalax           BidderName = "escalax"
	BidderExco              BidderName = "exco"
	BidderEVolution         BidderName = "e_volution"
	BidderFeedAd            BidderName = "feedad"
	BidderFlatads           BidderName = "flatads"
	BidderFlipp             BidderName = "flipp"
	BidderFreewheelSSP      BidderName = "freewheelssp"
	BidderFWSSP             BidderName = "fwssp"
	BidderFRVRAdNetwork     BidderName = "frvradn"
	BidderGamma             BidderName = "gamma"
	BidderGamoshi           BidderName = "gamoshi"
	BidderGlobalsun         BidderName = "globalsun"
	BidderGothamads         BidderName = "gothamads"
	BidderGrid              BidderName = "grid"
	BidderGumGum            BidderName = "gumgum"
	BidderHuaweiAds         BidderName = "huaweiads"
	BidderImds              BidderName = "imds"
	BidderImpactify         BidderName = "impactify"
	BidderImprovedigital    BidderName = "improvedigital"
	BidderInfyTV            BidderName = "infytv"
	BidderInMobi            BidderName = "inmobi"
	BidderInsticator        BidderName = "insticator"
	BidderInteractiveoffers BidderName = "interactiveoffers"
	BidderIntertech         BidderName = "intertech"
	BidderInvibes           BidderName = "invibes"
	BidderIQX               BidderName = "iqx"
	BidderIQZone            BidderName = "iqzone"
	BidderIx                BidderName = "ix"
	BidderJixie             BidderName = "jixie"
	BidderKargo             BidderName = "kargo"
	BidderKayzen            BidderName = "kayzen"
	BidderKidoz             BidderName = "kidoz"
	BidderKiviads           BidderName = "kiviads"
	BidderLmKiviads         BidderName = "lm_kiviads"
	BidderKobler            BidderName = "kobler"
	BidderKrushmedia        BidderName = "krushmedia"
	BidderKueezRTB          BidderName = "kueezrtb"
	BidderLemmadigital      BidderName = "lemmadigital"
	BidderLimelightDigital  BidderName = "limelightDigital"
	BidderLockerDome        BidderName = "lockerdome"
	BidderLogan             BidderName = "logan"
	BidderLogicad           BidderName = "logicad"
	BidderLoopme            BidderName = "loopme"
	BidderLoyal             BidderName = "loyal"
	BidderLunaMedia         BidderName = "lunamedia"
	BidderMabidder          BidderName = "mabidder"
	BidderMadSense          BidderName = "madsense"
	BidderMadvertise        BidderName = "madvertise"
	BidderMarsmedia         BidderName = "marsmedia"
	BidderMediafuse         BidderName = "mediafuse"
	BidderMediaGo           BidderName = "mediago"
	BidderMedianet          BidderName = "medianet"
	BidderMediasquare       BidderName = "mediasquare"
	BidderMeloZen           BidderName = "melozen"
	BidderMetaX             BidderName = "metax"
	BidderMgid              BidderName = "mgid"
	BidderMgidX             BidderName = "mgidX"
	BidderMinuteMedia       BidderName = "minutemedia"
	BidderMissena           BidderName = "missena"
	BidderMobfoxpb          BidderName = "mobfoxpb"
	BidderMobileFuse        BidderName = "mobilefuse"
	BidderMobkoi            BidderName = "mobkoi"
	BidderMotorik           BidderName = "motorik"
	BidderNativo            BidderName = "nativo"
	BidderNextMillennium    BidderName = "nextmillennium"
	BidderNexx360           BidderName = "nexx360"
	BidderNoBid             BidderName = "nobid"
	BidderOgury             BidderName = "ogury"
	BidderOms               BidderName = "oms"
	BidderOneTag            BidderName = "onetag"
	BidderOpenWeb           BidderName = "openweb"
	BidderOpenx             BidderName = "openx"
	BidderOperaads          BidderName = "operaads"
	BidderOptidigital       BidderName = "optidigital"
	BidderOraki             BidderName = "oraki"
	BidderOrbidder          BidderName = "orbidder"
	BidderOutbrain          BidderName = "outbrain"
	BidderOwnAdx            BidderName = "ownadx"
	BidderPangle            BidderName = "pangle"
	BidderPGAMSsp           BidderName = "pgamssp"
	BidderPlaydigo          BidderName = "playdigo"
	BidderPubmatic          BidderName = "pubmatic"
	BidderPubrise           BidderName = "pubrise"
	BidderPubnative         BidderName = "pubnative"
	BidderPulsepoint        BidderName = "pulsepoint"
	BidderPWBid             BidderName = "pwbid"
	BidderQT                BidderName = "qt"
	BidderReadpeak          BidderName = "readpeak"
	BidderRediads           BidderName = "rediads"
	BidderRelevantDigital   BidderName = "relevantdigital"
	BidderResetDigital      BidderName = "resetdigital"
	BidderRevcontent        BidderName = "revcontent"
	BidderRichaudience      BidderName = "richaudience"
	BidderRise              BidderName = "rise"
	BidderRoulax            BidderName = "roulax"
	BidderRTBHouse          BidderName = "rtbhouse"
	BidderRubicon           BidderName = "rubicon"
	BidderSeedingAlliance   BidderName = "seedingAlliance"
	BidderSeedtag           BidderName = "seedtag"
	BidderSaLunaMedia       BidderName = "sa_lunamedia"
	BidderScreencore        BidderName = "screencore"
	BidderSharethrough      BidderName = "sharethrough"
	BidderSilverMob         BidderName = "silvermob"
	BidderSilverPush        BidderName = "silverpush"
	BidderSmaato            BidderName = "smaato"
	BidderSmartAdserver     BidderName = "smartadserver"
	BidderSmartHub          BidderName = "smarthub"
	BidderSmartRTB          BidderName = "smartrtb"
	BidderSmartx            BidderName = "smartx"
	BidderSmartyAds         BidderName = "smartyads"
	BidderSmileWanted       BidderName = "smilewanted"
	BidderSmoot             BidderName = "smoot"
	BidderSmrtconnect       BidderName = "smrtconnect"
	BidderSonobi            BidderName = "sonobi"
	BidderSovrn             BidderName = "sovrn"
	BidderSovrnXsp          BidderName = "sovrnXsp"
	BidderSparteo           BidderName = "sparteo"
	BidderSspBC             BidderName = "sspBC"
	BidderStartIO           BidderName = "startio"
	BidderStroeerCore       BidderName = "stroeerCore"
	BidderTaboola           BidderName = "taboola"
	BidderTappx             BidderName = "tappx"
	BidderTeads             BidderName = "teads"
	BidderTelaria           BidderName = "telaria"
	BidderTheadx            BidderName = "theadx"
	BidderTheTradeDesk      BidderName = "thetradedesk"
	BidderTpmn              BidderName = "tpmn"
	BidderTradPlus          BidderName = "tradplus"
	BidderTrafficGate       BidderName = "trafficgate"
	BidderTriplelift        BidderName = "triplelift"
	BidderTripleliftNative  BidderName = "triplelift_native"
	BidderTrustedstack      BidderName = "trustedstack"
	BidderUcfunnel          BidderName = "ucfunnel"
	BidderUndertone         BidderName = "undertone"
	BidderUnicorn           BidderName = "unicorn"
	BidderUnruly            BidderName = "unruly"
	BidderVidazoo           BidderName = "vidazoo"
	BidderVideoByte         BidderName = "videobyte"
	BidderVideoHeroes       BidderName = "videoheroes"
	BidderVidoomy           BidderName = "vidoomy"
	BidderVisibleMeasures   BidderName = "visiblemeasures"
	BidderVisx              BidderName = "visx"
	BidderVox               BidderName = "vox"
	BidderVrtcal            BidderName = "vrtcal"
	BidderVungle            BidderName = "vungle"
	BidderXeworks           BidderName = "xeworks"
	BidderYahooAds          BidderName = "yahooAds"
	BidderYandex            BidderName = "yandex"
	BidderYeahmobi          BidderName = "yeahmobi"
	BidderYieldlab          BidderName = "yieldlab"
	BidderYieldmo           BidderName = "yieldmo"
	BidderYieldone          BidderName = "yieldone"
	BidderZentotem          BidderName = "zentotem"
	BidderZeroClickFraud    BidderName = "zeroclickfraud"
	BidderZetaGlobalSsp     BidderName = "zeta_global_ssp"
	BidderZmaticoo          BidderName = "zmaticoo"
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

type BidderNameNormalizer func(name string) (BidderName, bool)

func NormalizeBidderName(name string) (BidderName, bool) {
	nameLower := strings.ToLower(name)
	bidderName, exists := bidderNameLookup[nameLower]
	return bidderName, exists
}

// NormalizeBidderNameOrUnchanged returns the normalized name of known bidders, otherwise returns
// the name exactly as provided.
func NormalizeBidderNameOrUnchanged(name string) BidderName {
	if normalized, exists := NormalizeBidderName(name); exists {
		return normalized
	}
	return BidderName(name)
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

	// set alias bidder params schema to its parent
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
