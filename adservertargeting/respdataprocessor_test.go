package adservertargeting

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/openrtb/v20/openrtb3"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestProcessRequestTargetingData(t *testing.T) {

	testCases := []struct {
		description            string
		inputAdServerTargeting adServerTargetingData
		inputTargetingData     map[string]string
		expectedTargetingData  map[string]string
	}{

		{
			description: "Add single and targeting value by imp to not empty input targeting data",
			inputAdServerTargeting: adServerTargetingData{
				RequestTargetingData: map[string]RequestTargetingData{
					"key1": {SingleVal: json.RawMessage(`val1`)},
					"key2": {
						TargetingValueByImpId: map[string][]byte{
							"impId1": []byte(`impId1val`), "impId2": []byte(`impId2val`),
						},
					},
				},
			},
			inputTargetingData: map[string]string{"inKey1": "inVal1"},
			expectedTargetingData: map[string]string{
				"inKey1": "inVal1", "key1": "val1", "key2": "impId1val",
			},
		},
		{
			description: "Add single and targeting value by imp to empty input targeting data",
			inputAdServerTargeting: adServerTargetingData{
				RequestTargetingData: map[string]RequestTargetingData{
					"key1": {SingleVal: json.RawMessage(`val1`)},
					"key2": {
						TargetingValueByImpId: map[string][]byte{
							"impId1": []byte(`impId1val`), "impId2": []byte(`impId2val`),
						},
					},
				},
			},
			inputTargetingData: map[string]string{},
			expectedTargetingData: map[string]string{
				"key1": "val1", "key2": "impId1val",
			},
		},
		{
			description: "Add single value by imp to not empty input targeting data",
			inputAdServerTargeting: adServerTargetingData{
				RequestTargetingData: map[string]RequestTargetingData{
					"key1": {SingleVal: json.RawMessage(`val1`)},
				},
			},
			inputTargetingData: map[string]string{"inKey1": "inVal1"},
			expectedTargetingData: map[string]string{
				"inKey1": "inVal1", "key1": "val1",
			},
		},
		{
			description: "Add targeting value by imp to not empty input targeting data",
			inputAdServerTargeting: adServerTargetingData{
				RequestTargetingData: map[string]RequestTargetingData{
					"key2": {
						TargetingValueByImpId: map[string][]byte{
							"impId1": []byte(`impId1val`), "impId2": []byte(`impId2val`),
						},
					},
				},
			},
			inputTargetingData: map[string]string{"inKey1": "inVal1"},
			expectedTargetingData: map[string]string{
				"inKey1": "inVal1", "key2": "impId1val",
			},
		},
	}

	bidImpId := "impId1"
	for _, test := range testCases {
		processRequestTargetingData(&test.inputAdServerTargeting, test.inputTargetingData, bidImpId)
		assert.Equal(t, test.expectedTargetingData, test.inputTargetingData, "incorrect targeting data")
	}

}

func TestProcessResponseTargetingData(t *testing.T) {

	testCases := []struct {
		description            string
		inputAdServerTargeting adServerTargetingData
		inputTargetingData     map[string]string
		inputBid               openrtb2.Bid
		inputResponse          *openrtb2.BidResponse
		inputSeatExt           json.RawMessage

		expectedTargetingData map[string]string
		expectedWarnings      []openrtb_ext.ExtBidderMessage
	}{

		{
			description: "get value from seatbid.bid",
			inputAdServerTargeting: adServerTargetingData{
				ResponseTargetingData: []ResponseTargetingData{
					{Key: "{{BIDDER}}_custom1", HasMacro: true, Path: "seatbid.bid.impid"},
				},
			},
			inputTargetingData: map[string]string{"inKey1": "inVal1"},
			inputBid:           openrtb2.Bid{ID: "testBidId", ImpID: "testBidImpId1"},
			inputResponse:      nil,
			inputSeatExt:       nil,

			expectedTargetingData: map[string]string{
				"inKey1": "inVal1", "bidderA_custom1": "testBidImpId1",
			},
			expectedWarnings: []openrtb_ext.ExtBidderMessage(nil),
		},
		{
			description: "get value from seatbid.bid.ext.prebid.foo",
			inputAdServerTargeting: adServerTargetingData{
				ResponseTargetingData: []ResponseTargetingData{
					{Key: "{{BIDDER}}_custom1", HasMacro: true, Path: "seatbid.bid.ext.prebid.foo"},
				},
			},
			inputTargetingData: map[string]string{"inKey1": "inVal1"},
			inputBid:           openrtb2.Bid{ID: "testBidId", ImpID: "testBidImpId1"},
			inputResponse:      nil,
			inputSeatExt:       nil,

			expectedTargetingData: map[string]string{
				"inKey1": "inVal1", "bidderA_custom1": "bar1",
			},
			expectedWarnings: []openrtb_ext.ExtBidderMessage(nil),
		},
		{
			description: "get value from ext.testData.foo",
			inputAdServerTargeting: adServerTargetingData{
				ResponseTargetingData: []ResponseTargetingData{
					{Key: "{{BIDDER}}_custom1", HasMacro: true, Path: "ext.testData.foo"},
				},
			},
			inputTargetingData: map[string]string{"inKey1": "inVal1"},
			inputBid:           openrtb2.Bid{ID: "testBidId", ImpID: "testBidImpId1"},
			inputResponse:      &openrtb2.BidResponse{Cur: "UAH", Ext: json.RawMessage(`{"testData": {"foo": "barExt"}}`)},
			inputSeatExt:       nil,

			expectedTargetingData: map[string]string{
				"inKey1": "inVal1", "bidderA_custom1": "barExt",
			},
			expectedWarnings: []openrtb_ext.ExtBidderMessage(nil),
		},
		{
			description: "get value from resp",
			inputAdServerTargeting: adServerTargetingData{
				ResponseTargetingData: []ResponseTargetingData{
					{Key: "{{BIDDER}}_custom1", HasMacro: true, Path: "cur"},
				},
			},
			inputTargetingData: map[string]string{"inKey1": "inVal1"},
			inputBid:           openrtb2.Bid{ID: "testBidId", ImpID: "testBidImpId1"},
			inputResponse:      &openrtb2.BidResponse{Cur: "UAH", Ext: json.RawMessage(`{"testData": {"foo": "barExt"}}`)},
			inputSeatExt:       nil,

			expectedTargetingData: map[string]string{
				"inKey1": "inVal1", "bidderA_custom1": "UAH",
			},
			expectedWarnings: []openrtb_ext.ExtBidderMessage(nil),
		},
		{
			description: "get value from resp ext",
			inputAdServerTargeting: adServerTargetingData{
				ResponseTargetingData: []ResponseTargetingData{
					{Key: "{{BIDDER}}_custom1", HasMacro: true, Path: "seatbid.ext.testData.foo"},
				},
			},
			inputTargetingData: map[string]string{"inKey1": "inVal1"},
			inputBid:           openrtb2.Bid{ID: "testBidId", ImpID: "testBidImpId1"},
			inputResponse:      nil,
			inputSeatExt:       json.RawMessage(`{"testData": {"foo": "barBidderA"}}`),

			expectedTargetingData: map[string]string{
				"inKey1": "inVal1", "bidderA_custom1": "barBidderA",
			},
			expectedWarnings: []openrtb_ext.ExtBidderMessage(nil),
		},
		{
			description: "get value from resp ext with incorrect format",
			inputAdServerTargeting: adServerTargetingData{
				ResponseTargetingData: []ResponseTargetingData{
					{Key: "{{BIDDER}}_custom1", HasMacro: true, Path: "seatbid.ext.testData"},
				},
			},
			inputTargetingData: map[string]string{"inKey1": "inVal1"},
			inputBid:           openrtb2.Bid{ID: "testBidId", ImpID: "testBidImpId1"},
			inputResponse:      &openrtb2.BidResponse{Cur: "UAH", Ext: json.RawMessage(`{"testData": {"foo": "barExt"}}`)},
			inputSeatExt:       json.RawMessage(`{"testData": {"foo": "barBidderA"}}`),

			expectedTargetingData: map[string]string{
				"inKey1": "inVal1",
			},
			expectedWarnings: []openrtb_ext.ExtBidderMessage{
				{Code: 10007, Message: "incorrect value type for path: testData, value can only be string or number for bidder: bidderA, bid id: testBidId"},
			},
		},
	}

	bidderName := "bidderA"

	inputBidsCache := bidsCache{
		bids: map[string]map[string][]byte{
			"bidderA": {"testBidId": []byte(`{"id":"testBidId","impid":"testBidImpId1","price":10,"cat":["cat11","cat12"],"ext":{"prebid":{"foo":"bar1"}}}`)},
		},
	}

	for _, test := range testCases {
		actualWarnings := processResponseTargetingData(&test.inputAdServerTargeting,
			test.inputTargetingData, bidderName, test.inputBid, inputBidsCache,
			test.inputResponse, test.inputSeatExt)
		assert.Equal(t, test.expectedWarnings, actualWarnings, "incorrect warnings returned")
		assert.Equal(t, test.expectedTargetingData, test.inputTargetingData, "incorrect targeting data")
	}
}

func TestBuildBidExt(t *testing.T) {

	testCases := []struct {
		description             string
		inputTargetingData      map[string]string
		inputBid                openrtb2.Bid
		inputWarnings           []openrtb_ext.ExtBidderMessage
		truncateTargetAttribute int
		expectedExt             json.RawMessage
		expectedWarnings        []openrtb_ext.ExtBidderMessage
	}{

		{
			description:        "build valid bid ext with existing warnings",
			inputTargetingData: map[string]string{"inKey1": "inVal1"},
			inputBid:           openrtb2.Bid{ID: "testBidId", ImpID: "testBidImpId1", Ext: json.RawMessage(`{"prebid": {"test": 1}}`)},
			inputWarnings: []openrtb_ext.ExtBidderMessage{
				{Code: 10007, Message: "incorrect value type for path: testData, value can only be string or number for bidder: bidderA, bid id: testBidId"},
			},
			truncateTargetAttribute: 20,
			expectedExt:             json.RawMessage(`{"prebid":{"targeting":{"inKey1":"inVal1"},"test":1}}`),
			expectedWarnings: []openrtb_ext.ExtBidderMessage{
				{Code: 10007, Message: "incorrect value type for path: testData, value can only be string or number for bidder: bidderA, bid id: testBidId"},
			},
		},
		{
			description:             "build valid bid ext without existing warnings",
			inputTargetingData:      map[string]string{"inKey1": "inVal1"},
			inputBid:                openrtb2.Bid{ID: "testBidId", ImpID: "testBidImpId1", Ext: json.RawMessage(`{"prebid": {"test": 1}}`)},
			inputWarnings:           []openrtb_ext.ExtBidderMessage(nil),
			truncateTargetAttribute: 20,
			expectedExt:             json.RawMessage(`{"prebid":{"targeting":{"inKey1":"inVal1"},"test":1}}`),
			expectedWarnings:        []openrtb_ext.ExtBidderMessage(nil),
		},
	}

	for _, test := range testCases {
		actualExt := buildBidExt(test.inputTargetingData, test.inputBid, test.inputWarnings, &test.truncateTargetAttribute)
		assert.Equal(t, test.expectedWarnings, test.inputWarnings, "incorrect warnings returned")
		assert.JSONEq(t, string(test.expectedExt), string(actualExt), "incorrect result extension")
	}
}

func TestResolveKey(t *testing.T) {

	testCases := []struct {
		description                string
		inputResponseTargetingData ResponseTargetingData
		inputBidderName            string
		expectedKey                string
	}{

		{
			description:                "resolve key with macro",
			inputResponseTargetingData: ResponseTargetingData{Key: "{{BIDDER}}_custom1", HasMacro: true, Path: "ext"},
			inputBidderName:            "bidderA",
			expectedKey:                "bidderA_custom1",
		},
		{
			description:                "resolve key without macro",
			inputResponseTargetingData: ResponseTargetingData{Key: "key_custom1", HasMacro: true, Path: "ext"},
			inputBidderName:            "bidderA",
			expectedKey:                "key_custom1",
		},
		{
			description:                "resolve key with macro only",
			inputResponseTargetingData: ResponseTargetingData{Key: "{{BIDDER}}", HasMacro: true, Path: "ext"},
			inputBidderName:            "bidderA",
			expectedKey:                "bidderA",
		},
		{
			description:                "resolve key with macro and empty bidder name",
			inputResponseTargetingData: ResponseTargetingData{Key: "{{BIDDER}}_custom1", HasMacro: true, Path: "ext"},
			inputBidderName:            "",
			expectedKey:                "_custom1",
		},
	}

	for _, test := range testCases {
		actualKey := resolveKey(test.inputResponseTargetingData, test.inputBidderName)
		assert.Equal(t, test.expectedKey, actualKey, "incorrect resolved key")
	}
}

func TestTruncateTargetingKeys(t *testing.T) {

	testCases := []struct {
		description             string
		inputTargetingData      map[string]string
		truncateTargetAttribute int
		expectedTargetingData   map[string]string
	}{

		{
			description:             "truncate targeting keys",
			inputTargetingData:      map[string]string{"inKey1": "inVal1"},
			truncateTargetAttribute: 5,
			expectedTargetingData:   map[string]string{"inKey": "inVal1"},
		},
		{
			description:             "do not truncate targeting keys",
			inputTargetingData:      map[string]string{"inKey1": "inVal1"},
			truncateTargetAttribute: 10,
			expectedTargetingData:   map[string]string{"inKey1": "inVal1"},
		},
		{
			description:             "exceed truncate limit",
			inputTargetingData:      map[string]string{"inKey1": "inVal1"},
			truncateTargetAttribute: 100,
			expectedTargetingData:   map[string]string{"inKey1": "inVal1"},
		},
		{
			description:             "limit long key to default length",
			inputTargetingData:      map[string]string{"very_long_targeting_key_should_be_truncated_to_default": "inVal1"},
			truncateTargetAttribute: 0,
			expectedTargetingData:   map[string]string{"very_long_targeting_": "inVal1"},
		},
	}

	for _, test := range testCases {
		actualTargetingData := truncateTargetingKeys(test.inputTargetingData, &test.truncateTargetAttribute)
		assert.Equal(t, test.expectedTargetingData, actualTargetingData, "incorrect targeting data")
	}
}

func TestGetValueFromSeatBidBid(t *testing.T) {

	testCases := []struct {
		description   string
		inputPath     string
		expectedValue string
		expectError   bool
	}{

		{
			description:   "get existing valid value from bid",
			inputPath:     "seatbid.bid.price",
			expectedValue: "10",
			expectError:   false,
		},
		{
			description:   "get existing invalid value from bid",
			inputPath:     "seatbid.bid.cat",
			expectedValue: "",
			expectError:   true,
		},
		{
			description:   "get non-existing value from bid",
			inputPath:     "seatbid.bid.test",
			expectedValue: "",
			expectError:   true,
		},
	}

	inputBidsCache := bidsCache{
		bids: map[string]map[string][]byte{
			"bidderA": {"testBidId": []byte(`{"id":"testBidId","impid":"testBidImpId1","price":10,"cat":["cat11","cat12"],"ext":{"prebid":{"foo":"bar1"}}}`)},
		},
	}

	for _, test := range testCases {
		actualValue, actualErr := getValueFromSeatBidBid(test.inputPath, inputBidsCache, "bidderA", openrtb2.Bid{ID: "testBidId"})
		assert.Equal(t, test.expectedValue, actualValue, "incorrect value returned")
		if test.expectError {
			assert.Error(t, actualErr, "unexpected error returned")
		} else {
			assert.NoError(t, actualErr, "expected error not returned")
		}
	}
}

func TestGetValueFromExt(t *testing.T) {

	testCases := []struct {
		description    string
		inputPath      string
		inputSeparator string
		expectedValue  string
		expectError    bool
	}{

		{
			description:    "get existing valid value from bid.ext",
			inputPath:      "seatbid.ext.prebid.foo",
			inputSeparator: "seatbid.ext.",
			expectedValue:  "bar1",
			expectError:    false,
		},
		{
			description:    "get non-existing valid value from bid.ext",
			inputPath:      "seatbid.ext.prebid.foo1",
			inputSeparator: "seatbid.ext.",
			expectedValue:  "",
			expectError:    true,
		},
		{
			description:    "get existing valid with incorrect type value from bid.ext",
			inputPath:      "seatbid.ext.prebid",
			inputSeparator: "seatbid.ext.",
			expectedValue:  "",
			expectError:    true,
		},
		{
			description:    "get existing valid with unexpected separator from bid.ext",
			inputPath:      "seatbid.ext.prebid",
			inputSeparator: ".ext.",
			expectedValue:  "",
			expectError:    true,
		},
	}
	inputExt := json.RawMessage(`{"prebid":{"foo":"bar1"}}`)

	for _, test := range testCases {
		actualValue, actualErr := getValueFromExt(test.inputPath, test.inputSeparator, inputExt)
		assert.Equal(t, test.expectedValue, actualValue, "incorrect value returned")
		if test.expectError {
			assert.Error(t, actualErr, "unexpected error returned")
		} else {
			assert.NoError(t, actualErr, "expected error not returned")
		}
	}
}

func TestGetValueFromResp(t *testing.T) {

	testCases := []struct {
		description   string
		inputPath     string
		inputResponse *openrtb2.BidResponse
		expectedValue string
		expectError   bool
	}{

		{
			description:   "get existing valid value from response",
			inputPath:     "cur",
			inputResponse: &openrtb2.BidResponse{Cur: "UAH"},
			expectedValue: "UAH",
			expectError:   false,
		},
		{
			description:   "get empty valid value from response",
			inputPath:     "id",
			inputResponse: &openrtb2.BidResponse{},
			expectedValue: "",
			expectError:   false,
		},
		{
			description:   "get non-existing valid value from response",
			inputPath:     "test",
			inputResponse: &openrtb2.BidResponse{},
			expectedValue: "",
			expectError:   true,
		},
	}

	for _, test := range testCases {
		actualValue, actualErr := getValueFromResp(test.inputPath, test.inputResponse)
		assert.Equal(t, test.expectedValue, actualValue, "incorrect value returned")
		if test.expectError {
			assert.Error(t, actualErr, "unexpected error returned")
		} else {
			assert.NoError(t, actualErr, "expected error not returned")
		}
	}
}

func TestGetRespData(t *testing.T) {

	nbr := openrtb3.NoBidProxy
	testCases := []struct {
		description   string
		inputField    string
		inputResponse *openrtb2.BidResponse
		expectedValue string
		expectError   bool
	}{
		{
			description:   "get id from response",
			inputField:    "id",
			inputResponse: &openrtb2.BidResponse{ID: "testId"},
			expectedValue: "testId",
			expectError:   false,
		},
		{
			description:   "get bidid from response",
			inputField:    "bidid",
			inputResponse: &openrtb2.BidResponse{BidID: "testBidId"},
			expectedValue: "testBidId",
			expectError:   false,
		},
		{
			description:   "get cur from response",
			inputField:    "cur",
			inputResponse: &openrtb2.BidResponse{Cur: "UAH"},
			expectedValue: "UAH",
			expectError:   false,
		},
		{
			description:   "get customdata from response",
			inputField:    "customdata",
			inputResponse: &openrtb2.BidResponse{CustomData: "testCustomdata"},
			expectedValue: "testCustomdata",
			expectError:   false,
		},
		{
			description:   "get nbr from response",
			inputField:    "nbr",
			inputResponse: &openrtb2.BidResponse{NBR: &nbr},
			expectedValue: "5",
			expectError:   false,
		},
		{
			description:   "get non-existing value from response",
			inputField:    "test",
			inputResponse: &openrtb2.BidResponse{ID: "testId"},
			expectedValue: "",
			expectError:   true,
		},
	}

	for _, test := range testCases {
		actualValue, actualErr := getRespData(test.inputResponse, test.inputField)
		assert.Equal(t, test.expectedValue, actualValue, "incorrect value returned")
		if test.expectError {
			assert.Error(t, actualErr, "unexpected error returned")
		} else {
			assert.NoError(t, actualErr, "expected error not returned")
		}
	}

}

func TestResponseObjectStructure(t *testing.T) {
	// in case BidResponse format will change in next versions this test will show the error
	// current implementation is up to date with OpenRTB 2.5 and OpenRTB 2.6 formats
	fieldsToCheck := map[string]reflect.Kind{
		"id":         reflect.String,
		"bidid":      reflect.String,
		"cur":        reflect.String,
		"customdata": reflect.String,
		"nbr":        reflect.Pointer,
	}

	tt := reflect.TypeOf(openrtb2.BidResponse{})
	fields := reflect.VisibleFields(tt)

	for fieldName, fieldType := range fieldsToCheck {
		fieldFound := false
		for _, field := range fields {
			if fieldName == strings.ToLower(field.Name) {
				fieldFound = true
				assert.Equal(t, fieldType, field.Type.Kind(), "incorrect type for field: %s", fieldName)
				break
			}
		}
		assert.True(t, fieldFound, "field %s is not found in bidResponse object", fieldName)
	}
}
