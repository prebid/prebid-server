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

// BidderName may refer to a bidder ID, or an Alias which is defined in the request.
type BidderName string

// These names _must_ coincide with the bidder code in Prebid.js, if an adapter also exists in that project.
// Please keep these (and the BidderMap) alphabetized to minimize merge conflicts among adapter submissions.
const (
	Bidder33Across         BidderName = "33across"
	BidderAdform           BidderName = "adform"
	BidderAdkernel         BidderName = "adkernel"
	BidderAdkernelAdn      BidderName = "adkernelAdn"
	BidderAdpone           BidderName = "adpone"
	BidderAdtelligent      BidderName = "adtelligent"
	BidderAdvangelists     BidderName = "advangelists"
	BidderApplogy          BidderName = "applogy"
	BidderAppnexus         BidderName = "appnexus"
	BidderBeachfront       BidderName = "beachfront"
	BidderBrightroll       BidderName = "brightroll"
	BidderConsumable       BidderName = "consumable"
	BidderConversant       BidderName = "conversant"
	BidderDatablocks       BidderName = "datablocks"
	BidderEmxDigital       BidderName = "emx_digital"
	BidderEngageBDR        BidderName = "engagebdr"
	BidderEPlanning        BidderName = "eplanning"
	BidderFacebook         BidderName = "audienceNetwork"
	BidderGamma            BidderName = "gamma"
	BidderGamoshi          BidderName = "gamoshi"
	BidderGrid             BidderName = "grid"
	BidderGumGum           BidderName = "gumgum"
	BidderImprovedigital   BidderName = "improvedigital"
	BidderIx               BidderName = "ix"
	BidderKubient          BidderName = "kubient"
	BidderLifestreet       BidderName = "lifestreet"
	BidderLockerDome       BidderName = "lockerdome"
	BidderMarsmedia        BidderName = "marsmedia"
	BidderMgid             BidderName = "mgid"
	BidderOpenx            BidderName = "openx"
	BidderPubmatic         BidderName = "pubmatic"
	BidderPubnative        BidderName = "pubnative"
	BidderPulsepoint       BidderName = "pulsepoint"
	BidderRhythmone        BidderName = "rhythmone"
	BidderRTBHouse         BidderName = "rtbhouse"
	BidderRubicon          BidderName = "rubicon"
	BidderSharethrough     BidderName = "sharethrough"
	BidderSomoaudience     BidderName = "somoaudience"
	BidderSonobi           BidderName = "sonobi"
	BidderSovrn            BidderName = "sovrn"
	BidderSpotX            BidderName = "spotx"
	BidderSynacormedia     BidderName = "synacormedia"
	BidderTappx            BidderName = "tappx"
	BidderTelaria          BidderName = "telaria"
	BidderTriplelift       BidderName = "triplelift"
	BidderTripleliftNative BidderName = "triplelift_native"
	BidderUnruly           BidderName = "unruly"
	BidderVerizonMedia     BidderName = "verizonmedia"
	BidderVisx             BidderName = "visx"
	BidderVrtcal           BidderName = "vrtcal"
	BidderYieldmo          BidderName = "yieldmo"
)

// BidderMap stores all the valid OpenRTB 2.x Bidders in the project. This map *must not* be mutated.
var BidderMap = map[string]BidderName{
	"33across":          Bidder33Across,
	"adform":            BidderAdform,
	"adkernel":          BidderAdkernel,
	"adkernelAdn":       BidderAdkernelAdn,
	"adpone":            BidderAdpone,
	"adtelligent":       BidderAdtelligent,
	"advangelists":      BidderAdvangelists,
	"applogy":           BidderApplogy,
	"appnexus":          BidderAppnexus,
	"beachfront":        BidderBeachfront,
	"brightroll":        BidderBrightroll,
	"consumable":        BidderConsumable,
	"conversant":        BidderConversant,
	"datablocks":        BidderDatablocks,
	"emx_digital":       BidderEmxDigital,
	"engagebdr":         BidderEngageBDR,
	"eplanning":         BidderEPlanning,
	"audienceNetwork":   BidderFacebook,
	"gamma":             BidderGamma,
	"gamoshi":           BidderGamoshi,
	"grid":              BidderGrid,
	"gumgum":            BidderGumGum,
	"improvedigital":    BidderImprovedigital,
	"ix":                BidderIx,
	"kubient":           BidderKubient,
	"lifestreet":        BidderLifestreet,
	"lockerdome":        BidderLockerDome,
	"marsmedia":         BidderMarsmedia,
	"mgid":              BidderMgid,
	"openx":             BidderOpenx,
	"pubmatic":          BidderPubmatic,
	"pubnative":         BidderPubnative,
	"pulsepoint":        BidderPulsepoint,
	"rhythmone":         BidderRhythmone,
	"rtbhouse":          BidderRTBHouse,
	"rubicon":           BidderRubicon,
	"sharethrough":      BidderSharethrough,
	"somoaudience":      BidderSomoaudience,
	"sonobi":            BidderSonobi,
	"sovrn":             BidderSovrn,
	"spotx":             BidderSpotX,
	"synacormedia":      BidderSynacormedia,
	"tappx":             BidderTappx,
	"telaria":           BidderTelaria,
	"triplelift":        BidderTriplelift,
	"triplelift_native": BidderTripleliftNative,
	"unruly":            BidderUnruly,
	"verizonmedia":      BidderVerizonMedia,
	"visx":              BidderVisx,
	"vrtcal":            BidderVrtcal,
	"yieldmo":           BidderYieldmo,
}

// BidderList returns the values of the BidderMap
func BidderList() []BidderName {
	bidders := make([]BidderName, 0, len(BidderMap))
	for _, value := range BidderMap {
		bidders = append(bidders, value)
	}
	return bidders
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

	schemaContents := make(map[BidderName]string, 50)
	schemas := make(map[BidderName]*gojsonschema.Schema, 50)
	for _, fileInfo := range fileInfos {
		bidderName := strings.TrimSuffix(fileInfo.Name(), ".json")
		if _, isValid := BidderMap[bidderName]; !isValid {
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
