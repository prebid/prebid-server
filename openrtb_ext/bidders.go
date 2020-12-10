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

// BidderNameGeneral is reserved for non-bidder specific messages when using a map keyed on the bidder name.
const BidderNameGeneral = BidderName("general")

// BidderNameContext is reserved for first party data.
const BidderNameContext = BidderName("context")

func (name BidderName) MarshalJSON() ([]byte, error) {
	return []byte(name), nil
}

func (name *BidderName) String() string {
	if name == nil {
		return ""
	}
	return string(*name)
}

// Names of core bidders. These names *must* match the bidder code in Prebid.js if an adapter also exists in that
// project. You may *not* use the name 'general' as that is reserved for general error messages nor 'context' as
// that is reserved for first party data.
//
// Please keep this list alphabetized to minimize merge conflicts.
const (
	Bidder33Across         BidderName = "33across"
	BidderAcuityAds        BidderName = "acuityads"
	BidderAdform           BidderName = "adform"
	BidderAdgeneration     BidderName = "adgeneration"
	BidderAdhese           BidderName = "adhese"
	BidderAdkernel         BidderName = "adkernel"
	BidderAdkernelAdn      BidderName = "adkernelAdn"
	BidderAdman            BidderName = "adman"
	BidderAdmixer          BidderName = "admixer"
	BidderAdOcean          BidderName = "adocean"
	BidderAdoppler         BidderName = "adoppler"
	BidderAdpone           BidderName = "adpone"
	BidderAdprime          BidderName = "adprime"
	BidderAdtarget         BidderName = "adtarget"
	BidderAdtelligent      BidderName = "adtelligent"
	BidderAdvangelists     BidderName = "advangelists"
	BidderAJA              BidderName = "aja"
	BidderAMX              BidderName = "amx"
	BidderApplogy          BidderName = "applogy"
	BidderAppnexus         BidderName = "appnexus"
	BidderAudienceNetwork  BidderName = "audienceNetwork"
	BidderAvocet           BidderName = "avocet"
	BidderBeachfront       BidderName = "beachfront"
	BidderBeintoo          BidderName = "beintoo"
	BidderBetween          BidderName = "between"
	BidderBrightroll       BidderName = "brightroll"
	BidderColossus         BidderName = "colossus"
	BidderConnectAd        BidderName = "connectad"
	BidderConsumable       BidderName = "consumable"
	BidderConversant       BidderName = "conversant"
	BidderCpmstar          BidderName = "cpmstar"
	BidderDatablocks       BidderName = "datablocks"
	BidderDmx              BidderName = "dmx"
	BidderDeepintent       BidderName = "deepintent"
	BidderEmxDigital       BidderName = "emx_digital"
	BidderEngageBDR        BidderName = "engagebdr"
	BidderEPlanning        BidderName = "eplanning"
	BidderGamma            BidderName = "gamma"
	BidderGamoshi          BidderName = "gamoshi"
	BidderGrid             BidderName = "grid"
	BidderGumGum           BidderName = "gumgum"
	BidderImprovedigital   BidderName = "improvedigital"
	BidderInMobi           BidderName = "inmobi"
	BidderInvibes          BidderName = "invibes"
	BidderIx               BidderName = "ix"
	BidderKidoz            BidderName = "kidoz"
	BidderKrushmedia       BidderName = "krushmedia"
	BidderKubient          BidderName = "kubient"
	BidderLifestreet       BidderName = "lifestreet"
	BidderLockerDome       BidderName = "lockerdome"
	BidderLogicad          BidderName = "logicad"
	BidderLunaMedia        BidderName = "lunamedia"
	BidderMarsmedia        BidderName = "marsmedia"
	BidderMgid             BidderName = "mgid"
	BidderMobileFuse       BidderName = "mobilefuse"
	BidderNanoInteractive  BidderName = "nanointeractive"
	BidderNinthDecimal     BidderName = "ninthdecimal"
	BidderNoBid            BidderName = "nobid"
	BidderOpenx            BidderName = "openx"
	BidderOrbidder         BidderName = "orbidder"
	BidderPubmatic         BidderName = "pubmatic"
	BidderPubnative        BidderName = "pubnative"
	BidderPulsepoint       BidderName = "pulsepoint"
	BidderRhythmone        BidderName = "rhythmone"
	BidderRTBHouse         BidderName = "rtbhouse"
	BidderRubicon          BidderName = "rubicon"
	BidderSharethrough     BidderName = "sharethrough"
	BidderSilverMob        BidderName = "silvermob"
	BidderSmaato           BidderName = "smaato"
	BidderSmartAdserver    BidderName = "smartadserver"
	BidderSmartRTB         BidderName = "smartrtb"
	BidderSmartyAds        BidderName = "smartyads"
	BidderSomoaudience     BidderName = "somoaudience"
	BidderSonobi           BidderName = "sonobi"
	BidderSovrn            BidderName = "sovrn"
	BidderSynacormedia     BidderName = "synacormedia"
	BidderTappx            BidderName = "tappx"
	BidderTelaria          BidderName = "telaria"
	BidderTriplelift       BidderName = "triplelift"
	BidderTripleliftNative BidderName = "triplelift_native"
	BidderUcfunnel         BidderName = "ucfunnel"
	BidderUnruly           BidderName = "unruly"
	BidderValueImpression  BidderName = "valueimpression"
	BidderVerizonMedia     BidderName = "verizonmedia"
	BidderVisx             BidderName = "visx"
	BidderVrtcal           BidderName = "vrtcal"
	BidderYeahmobi         BidderName = "yeahmobi"
	BidderYieldlab         BidderName = "yieldlab"
	BidderYieldmo          BidderName = "yieldmo"
	BidderYieldone         BidderName = "yieldone"
	BidderZeroClickFraud   BidderName = "zeroclickfraud"
)

// CoreBidderNames returns a slice of all core bidders.
func CoreBidderNames() []BidderName {
	return []BidderName{
		Bidder33Across,
		BidderAcuityAds,
		BidderAdform,
		BidderAdgeneration,
		BidderAdhese,
		BidderAdkernel,
		BidderAdkernelAdn,
		BidderAdman,
		BidderAdmixer,
		BidderAdOcean,
		BidderAdoppler,
		BidderAdpone,
		BidderAdprime,
		BidderAdtarget,
		BidderAdtelligent,
		BidderAdvangelists,
		BidderAJA,
		BidderAMX,
		BidderApplogy,
		BidderAppnexus,
		BidderAudienceNetwork,
		BidderAvocet,
		BidderBeachfront,
		BidderBeintoo,
		BidderBetween,
		BidderBrightroll,
		BidderColossus,
		BidderConnectAd,
		BidderConsumable,
		BidderConversant,
		BidderCpmstar,
		BidderDatablocks,
		BidderDeepintent,
		BidderDmx,
		BidderEmxDigital,
		BidderEngageBDR,
		BidderEPlanning,
		BidderGamma,
		BidderGamoshi,
		BidderGrid,
		BidderGumGum,
		BidderImprovedigital,
		BidderInMobi,
		BidderInvibes,
		BidderIx,
		BidderKidoz,
		BidderKrushmedia,
		BidderKubient,
		BidderLifestreet,
		BidderLockerDome,
		BidderLogicad,
		BidderLunaMedia,
		BidderMarsmedia,
		BidderMgid,
		BidderMobileFuse,
		BidderNanoInteractive,
		BidderNinthDecimal,
		BidderNoBid,
		BidderOpenx,
		BidderOrbidder,
		BidderPubmatic,
		BidderPubnative,
		BidderPulsepoint,
		BidderRhythmone,
		BidderRTBHouse,
		BidderRubicon,
		BidderSharethrough,
		BidderSilverMob,
		BidderSmaato,
		BidderSmartAdserver,
		BidderSmartRTB,
		BidderSmartyAds,
		BidderSomoaudience,
		BidderSonobi,
		BidderSovrn,
		BidderSynacormedia,
		BidderTappx,
		BidderTelaria,
		BidderTriplelift,
		BidderTripleliftNative,
		BidderUcfunnel,
		BidderUnruly,
		BidderValueImpression,
		BidderVerizonMedia,
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
