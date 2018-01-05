package openrtb_ext

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/xeipuuv/gojsonschema"
	"io/ioutil"
	"net/http"
	"strings"
)

const schemaDirectory = "static/bidder-params"

type BidderName string

const (
	BidderAppnexus   BidderName = "appnexus"
	BidderFacebook   BidderName = "facebook"
	BidderIndex      BidderName = "index"
	BidderLifestreet BidderName = "lifestreet"
	BidderPubmatic   BidderName = "pubmatic"
	BidderPulsepoint BidderName = "pulsepoint"
	BidderRubicon    BidderName = "rubicon"
	BidderConversant BidderName = "conversant"
)

var bidderMap = map[string]BidderName{
	"appnexus":   BidderAppnexus,
	"facebook":   BidderFacebook,
	"index":      BidderIndex,
	"lifestreet": BidderLifestreet,
	"pubmatic":   BidderPubmatic,
	"pulsepoint": BidderPulsepoint,
	"rubicon":    BidderRubicon,
	"conversant": BidderConversant,
}

// GetBidderName returns the BidderName for the given string, if it exists.
// The second argument is true if the name was valid, and false otherwise.
func GetBidderName(name string) (BidderName, bool) {
	bidderName, ok := bidderMap[name]
	return bidderName, ok
}

func (name BidderName) MarshalJSON() ([]byte, error) {
	return []byte(name), nil
}

func (name *BidderName) String() string {
	if name == nil {
		return ""
	} else {
		return string(*name)
	}
}

// The BidderParamValidator is used to enforce bidrequest.imp[i].ext.{anyBidder} values.
//
// This is treated differently from the other types because we rely on JSON-schemas to validate bidder params.
type BidderParamValidator interface {
	Validate(name BidderName, ext openrtb.RawJSON) error
	// Schema returns the JSON schema used to perform validation.
	Schema(name BidderName) string
}

// NewBidderParamsValidator makes a BidderParamValidator, assuming all the necessary files exist in the filesystem.
// This will error if, for example, a Bidder gets added but no JSON schema is written for them.
func NewBidderParamsValidator(schemaDirectory string) (BidderParamValidator, error) {
	filesystem := http.Dir(schemaDirectory)
	fileInfos, err := ioutil.ReadDir(schemaDirectory)
	if err != nil {
		return nil, fmt.Errorf("Failed to read JSON schemas from directory %s. %v", schemaDirectory, err)
	}

	schemaContents := make(map[BidderName]string, 50)
	schemas := make(map[BidderName]*gojsonschema.Schema, 50)
	for _, fileInfo := range fileInfos {
		bidderName := strings.TrimSuffix(fileInfo.Name(), ".json")
		if _, isValid := GetBidderName(bidderName); !isValid {
			return nil, fmt.Errorf("File %s/%s does not match a valid BidderName.", schemaDirectory, fileInfo.Name())
		}

		schemaLoader := gojsonschema.NewReferenceLoaderFileSystem(fmt.Sprintf("file:///%s", fileInfo.Name()), filesystem)
		loadedSchema, err := gojsonschema.NewSchema(schemaLoader)
		if err != nil {
			return nil, fmt.Errorf("Failed to load json schema at %s/%s: %v", schemaDirectory, fileInfo.Name(), err)
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

func (validator *bidderParamValidator) Validate(name BidderName, ext openrtb.RawJSON) error {
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
