package openrtb_ext

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/xeipuuv/gojsonschema"
)

const schemaDirectory = "static/bidder-params"

// BidderName refers to a core bidder id or an alias id.
type BidderName string

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
	BidderReservedPrebid  BidderName = "prebid"  // Reserved for Prebid Server configuration.
	BidderReservedSKAdN   BidderName = "skadn"   // Reserved for Apple's SKAdNetwork OpenRTB extension.
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

	if strings.EqualFold(name, string(BidderReservedSKAdN)) {
		return true
	}

	if strings.EqualFold(name, string(BidderReservedPrebid)) {
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
	BidderAcuityAds         BidderName = "acuityads"
	BidderAdagio            BidderName = "adagio"
	BidderAdf               BidderName = "adf"
	BidderAdform            BidderName = "adform"
	BidderAdgeneration      BidderName = "adgeneration"
	BidderAdhese            BidderName = "adhese"
	BidderAdkernel          BidderName = "adkernel"
	BidderAdkernelAdn       BidderName = "adkernelAdn"
	BidderAdman             BidderName = "adman"
	BidderAdmixer           BidderName = "admixer"
	BidderAdOcean           BidderName = "adocean"
	BidderAdoppler          BidderName = "adoppler"
	BidderAdot              BidderName = "adot"
	BidderAdpone            BidderName = "adpone"
	BidderAdprime           BidderName = "adprime"
	BidderAdtarget          BidderName = "adtarget"
	BidderAdtelligent       BidderName = "adtelligent"
	BidderAdvangelists      BidderName = "advangelists"
	BidderAdxcg             BidderName = "adxcg"
	BidderAdyoulike         BidderName = "adyoulike"
	BidderAJA               BidderName = "aja"
	BidderAlgorix           BidderName = "algorix"
	BidderAMX               BidderName = "amx"
	BidderApplogy           BidderName = "applogy"
	BidderAppnexus          BidderName = "appnexus"
	BidderAudienceNetwork   BidderName = "audienceNetwork"
	BidderAvocet            BidderName = "avocet"
	BidderAxonix            BidderName = "axonix"
	BidderBeachfront        BidderName = "beachfront"
	BidderBeintoo           BidderName = "beintoo"
	BidderBetween           BidderName = "between"
	BidderBidmachine        BidderName = "bidmachine"
	BidderBidmyadz          BidderName = "bidmyadz"
	BidderBidsCube          BidderName = "bidscube"
	BidderBmtm              BidderName = "bmtm"
	BidderBrightroll        BidderName = "brightroll"
	BidderColossus          BidderName = "colossus"
	BidderConnectAd         BidderName = "connectad"
	BidderConsumable        BidderName = "consumable"
	BidderConversant        BidderName = "conversant"
	BidderCpmstar           BidderName = "cpmstar"
	BidderCriteo            BidderName = "criteo"
	BidderDatablocks        BidderName = "datablocks"
	BidderDmx               BidderName = "dmx"
	BidderDecenterAds       BidderName = "decenterads"
	BidderDeepintent        BidderName = "deepintent"
	BidderEmxDigital        BidderName = "emx_digital"
	BidderEngageBDR         BidderName = "engagebdr"
	BidderEPlanning         BidderName = "eplanning"
	BidderEpom              BidderName = "epom"
	BidderEVolution         BidderName = "e_volution"
	BidderGamma             BidderName = "gamma"
	BidderGamoshi           BidderName = "gamoshi"
	BidderGrid              BidderName = "grid"
	BidderGumGum            BidderName = "gumgum"
	BidderImprovedigital    BidderName = "improvedigital"
	BidderInMobi            BidderName = "inmobi"
	BidderInteractiveoffers BidderName = "interactiveoffers"
	BidderInvibes           BidderName = "invibes"
	BidderIx                BidderName = "ix"
	BidderJixie             BidderName = "jixie"
	BidderKayzen            BidderName = "kayzen"
	BidderKidoz             BidderName = "kidoz"
	BidderKrushmedia        BidderName = "krushmedia"
	BidderKubient           BidderName = "kubient"
	BidderLockerDome        BidderName = "lockerdome"
	BidderLogicad           BidderName = "logicad"
	BidderLunaMedia         BidderName = "lunamedia"
	BidderSaLunaMedia       BidderName = "sa_lunamedia"
	BidderMadvertise        BidderName = "madvertise"
	BidderMarsmedia         BidderName = "marsmedia"
	BidderMediafuse         BidderName = "mediafuse"
	BidderMgid              BidderName = "mgid"
	BidderMobfoxpb          BidderName = "mobfoxpb"
	BidderMobileFuse        BidderName = "mobilefuse"
	BidderNanoInteractive   BidderName = "nanointeractive"
	BidderNinthDecimal      BidderName = "ninthdecimal"
	BidderNoBid             BidderName = "nobid"
	BidderOneTag            BidderName = "onetag"
	BidderOpenx             BidderName = "openx"
	BidderOperaads          BidderName = "operaads"
	BidderOrbidder          BidderName = "orbidder"
	BidderOutbrain          BidderName = "outbrain"
	BidderPangle            BidderName = "pangle"
	BidderPubmatic          BidderName = "pubmatic"
	BidderPubnative         BidderName = "pubnative"
	BidderPulsepoint        BidderName = "pulsepoint"
	BidderRevcontent        BidderName = "revcontent"
	BidderRhythmone         BidderName = "rhythmone"
	BidderRTBHouse          BidderName = "rtbhouse"
	BidderRubicon           BidderName = "rubicon"
	BidderSharethrough      BidderName = "sharethrough"
	BidderSilverMob         BidderName = "silvermob"
	BidderSmaato            BidderName = "smaato"
	BidderSmartAdserver     BidderName = "smartadserver"
	BidderSmartHub          BidderName = "smarthub"
	BidderSmartRTB          BidderName = "smartrtb"
	BidderSmartyAds         BidderName = "smartyads"
	BidderSmileWanted       BidderName = "smilewanted"
	BidderSomoaudience      BidderName = "somoaudience"
	BidderSonobi            BidderName = "sonobi"
	BidderSovrn             BidderName = "sovrn"
	BidderSynacormedia      BidderName = "synacormedia"
	BidderTappx             BidderName = "tappx"
	BidderTelaria           BidderName = "telaria"
	BidderTriplelift        BidderName = "triplelift"
	BidderTripleliftNative  BidderName = "triplelift_native"
	BidderTrustX            BidderName = "trustx"
	BidderUcfunnel          BidderName = "ucfunnel"
	BidderUnicorn           BidderName = "unicorn"
	BidderUnruly            BidderName = "unruly"
	BidderValueImpression   BidderName = "valueimpression"
	BidderVerizonMedia      BidderName = "verizonmedia"
	BidderVisx              BidderName = "visx"
	BidderViewdeos          BidderName = "viewdeos"
	BidderVrtcal            BidderName = "vrtcal"
	BidderYeahmobi          BidderName = "yeahmobi"
	BidderYieldlab          BidderName = "yieldlab"
	BidderYieldmo           BidderName = "yieldmo"
	BidderYieldone          BidderName = "yieldone"
	BidderZeroClickFraud    BidderName = "zeroclickfraud"
)

// CoreBidderNames returns a slice of all core bidders.
func CoreBidderNames() []BidderName {
	return []BidderName{
		Bidder33Across,
		BidderAcuityAds,
		BidderAdagio,
		BidderAdf,
		BidderAdform,
		BidderAdgeneration,
		BidderAdhese,
		BidderAdkernel,
		BidderAdkernelAdn,
		BidderAdman,
		BidderAdmixer,
		BidderAdOcean,
		BidderAdoppler,
		BidderAdot,
		BidderAdpone,
		BidderAdprime,
		BidderAdtarget,
		BidderAdtelligent,
		BidderAdvangelists,
		BidderAdxcg,
		BidderAdyoulike,
		BidderAJA,
		BidderAlgorix,
		BidderAMX,
		BidderApplogy,
		BidderAppnexus,
		BidderAudienceNetwork,
		BidderAvocet,
		BidderAxonix,
		BidderBeachfront,
		BidderBeintoo,
		BidderBetween,
		BidderBidmachine,
		BidderBidmyadz,
		BidderBidsCube,
		BidderBmtm,
		BidderBrightroll,
		BidderColossus,
		BidderConnectAd,
		BidderConsumable,
		BidderConversant,
		BidderCpmstar,
		BidderCriteo,
		BidderDatablocks,
		BidderDecenterAds,
		BidderDeepintent,
		BidderDmx,
		BidderEmxDigital,
		BidderEngageBDR,
		BidderEPlanning,
		BidderEpom,
		BidderEVolution,
		BidderGamma,
		BidderGamoshi,
		BidderGrid,
		BidderGumGum,
		BidderImprovedigital,
		BidderInMobi,
		BidderInteractiveoffers,
		BidderInvibes,
		BidderIx,
		BidderJixie,
		BidderKayzen,
		BidderKidoz,
		BidderKrushmedia,
		BidderKubient,
		BidderLockerDome,
		BidderLogicad,
		BidderLunaMedia,
		BidderSaLunaMedia,
		BidderMadvertise,
		BidderMarsmedia,
		BidderMediafuse,
		BidderMgid,
		BidderMobfoxpb,
		BidderMobileFuse,
		BidderNanoInteractive,
		BidderNinthDecimal,
		BidderNoBid,
		BidderOneTag,
		BidderOpenx,
		BidderOperaads,
		BidderOrbidder,
		BidderOutbrain,
		BidderPangle,
		BidderPubmatic,
		BidderPubnative,
		BidderPulsepoint,
		BidderRevcontent,
		BidderRhythmone,
		BidderRTBHouse,
		BidderRubicon,
		BidderSharethrough,
		BidderSilverMob,
		BidderSmaato,
		BidderSmartAdserver,
		BidderSmartHub,
		BidderSmartRTB,
		BidderSmartyAds,
		BidderSmileWanted,
		BidderSomoaudience,
		BidderSonobi,
		BidderSovrn,
		BidderSynacormedia,
		BidderTappx,
		BidderTelaria,
		BidderTriplelift,
		BidderTripleliftNative,
		BidderTrustX,
		BidderUcfunnel,
		BidderUnicorn,
		BidderUnruly,
		BidderValueImpression,
		BidderVerizonMedia,
		BidderViewdeos,
		BidderVisx,
		BidderVrtcal,
		BidderYeahmobi,
		BidderYieldlab,
		BidderYieldmo,
		BidderYieldone,
		BidderZeroClickFraud,
	}
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

// The BidderParamValidator is used to enforce bidrequest.imp[i].ext.{anyBidder} values.
//
// This is treated differently from the other types because we rely on JSON-schemas to validate bidder params.
type BidderParamValidator interface {
	Validate(name BidderName, ext json.RawMessage) error
	// Schema returns the JSON schema used to perform validation.
	Schema(name BidderName) string
}

// NewBidderParamsValidator makes a BidderParamValidator, assuming all the necessary files exist in the filesystem.
// This will error if, for example, a Bidder gets added but no JSON schema is written for them.
func NewBidderParamsValidator(schemaDirectory string) (BidderParamValidator, error) {
	fileInfos, err := ioutil.ReadDir(schemaDirectory)
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
		toOpen, err := filepath.Abs(filepath.Join(schemaDirectory, fileInfo.Name()))
		if err != nil {
			return nil, fmt.Errorf("Failed to get an absolute representation of the path: %s, %v", toOpen, err)
		}
		schemaLoader := gojsonschema.NewReferenceLoader("file:///" + filepath.ToSlash(toOpen))
		loadedSchema, err := gojsonschema.NewSchema(schemaLoader)
		if err != nil {
			return nil, fmt.Errorf("Failed to load json schema at %s: %v", toOpen, err)
		}

		fileBytes, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", schemaDirectory, fileInfo.Name()))
		if err != nil {
			return nil, fmt.Errorf("Failed to read file %s/%s: %v", schemaDirectory, fileInfo.Name(), err)
		}

		schemas[BidderName(bidderName)] = loadedSchema
		schemaContents[BidderName(bidderName)] = string(fileBytes)
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
