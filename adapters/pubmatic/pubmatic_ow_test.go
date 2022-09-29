package pubmatic

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAdServerTargetingForEmptyExt(t *testing.T) {
	ext := json.RawMessage(`{}`)
	targets := getTargetingKeys(ext, "pubmatic")
	// banner is the default bid type when no bidType key is present in the bid.ext
	if targets != nil && targets["hb_buyid_pubmatic"] != "" {
		t.Errorf("It should not contained AdserverTageting")
	}
}

func TestGetAdServerTargetingForValidExt(t *testing.T) {
	ext := json.RawMessage("{\"buyid\":\"testBuyId\"}")
	targets := getTargetingKeys(ext, "pubmatic")
	// banner is the default bid type when no bidType key is present in the bid.ext
	if targets == nil {
		t.Error("It should have targets")
		t.FailNow()
	}
	if targets != nil && targets["hb_buyid_pubmatic"] != "testBuyId" {
		t.Error("It should have testBuyId as targeting")
		t.FailNow()
	}
}

func TestGetAdServerTargetingForPubmaticAlias(t *testing.T) {
	ext := json.RawMessage("{\"buyid\":\"testBuyId-alias\"}")
	targets := getTargetingKeys(ext, "dummy-alias")
	// banner is the default bid type when no bidType key is present in the bid.ext
	if targets == nil {
		t.Error("It should have targets")
		t.FailNow()
	}
	if targets != nil && targets["hb_buyid_dummy-alias"] != "testBuyId-alias" {
		t.Error("It should have testBuyId as targeting")
		t.FailNow()
	}
}

func TestGetMapFromJSON(t *testing.T) {
	ext := json.RawMessage("{\"buyid\":\"testBuyId\"}")
	extMap := getMapFromJSON(ext)
	if extMap == nil {
		t.Errorf("it should be converted in extMap")
	}
}

func TestGetMapFromJSONWithInvalidJSON(t *testing.T) {
	ext := json.RawMessage("{\"buyid\":\"testBuyId\"}}}}")
	extMap := getMapFromJSON(ext)
	if extMap != nil {
		t.Errorf("it should be converted in extMap")
	}
}

func TestCopySBExtToBidExtWithBidExt(t *testing.T) {
	sbext := json.RawMessage("{\"buyid\":\"testBuyId\"}")
	bidext := json.RawMessage("{\"dspId\":\"9\"}")
	// expectedbid := json.RawMessage("{\"dspId\":\"9\",\"buyid\":\"testBuyId\"}")
	bidextnew := copySBExtToBidExt(sbext, bidext)
	if bidextnew == nil {
		t.Errorf("it should not be nil")
	}
}

func TestCopySBExtToBidExtWithNoBidExt(t *testing.T) {
	sbext := json.RawMessage("{\"buyid\":\"testBuyId\"}")
	bidext := json.RawMessage("{\"dspId\":\"9\"}")
	// expectedbid := json.RawMessage("{\"dspId\":\"9\",\"buyid\":\"testBuyId\"}")
	bidextnew := copySBExtToBidExt(sbext, bidext)
	if bidextnew == nil {
		t.Errorf("it should not be nil")
	}
}

func TestCopySBExtToBidExtWithNoSeatExt(t *testing.T) {
	bidext := json.RawMessage("{\"dspId\":\"9\"}")
	// expectedbid := json.RawMessage("{\"dspId\":\"9\",\"buyid\":\"testBuyId\"}")
	bidextnew := copySBExtToBidExt(nil, bidext)
	if bidextnew == nil {
		t.Errorf("it should not be nil")
	}
}

func TestPopulateFirstPartyDataImpAttributes(t *testing.T) {
	type args struct {
		data      json.RawMessage
		impExtMap map[string]interface{}
	}
	tests := []struct {
		name           string
		args           args
		expectedImpExt map[string]interface{}
	}{
		{
			name: "Only Targeting present in imp.ext.data",
			args: args{
				data:      json.RawMessage(`{"sport":["rugby","cricket"]}`),
				impExtMap: map[string]interface{}{},
			},
			expectedImpExt: map[string]interface{}{
				"key_val": "sport=rugby,cricket",
			},
		},
		{
			name: "Targeting present in imp.ext.data and adserver object",
			args: args{
				data:      json.RawMessage(`{"adserver": {"name": "gam","adslot": "/1111/home"},"pbadslot": "/2222/home","sport":["rugby","cricket"]}`),
				impExtMap: map[string]interface{}{},
			},
			expectedImpExt: map[string]interface{}{
				"dfp_ad_unit_code": "/1111/home",
				"key_val":          "sport=rugby,cricket",
			},
		},
		{
			name: "Targeting present in imp.ext.data and pbadslot object",
			args: args{
				data:      json.RawMessage(`{"pbadslot": "/2222/home","sport":["rugby","cricket"]}`),
				impExtMap: map[string]interface{}{},
			},
			expectedImpExt: map[string]interface{}{
				"dfp_ad_unit_code": "/2222/home",
				"key_val":          "sport=rugby,cricket",
			},
		},
		{
			name: "Targeting present in imp.ext.data and Invalid Adserver object",
			args: args{
				data:      json.RawMessage(`{"adserver": "invalid","sport":["rugby","cricket"]}`),
				impExtMap: map[string]interface{}{},
			},
			expectedImpExt: map[string]interface{}{
				"key_val": "sport=rugby,cricket",
			},
		},
		{
			name: "key_val already present in imp.ext.data",
			args: args{
				data: json.RawMessage(`{"sport":["rugby","cricket"]}`),
				impExtMap: map[string]interface{}{
					"key_val": "k1=v1|k2=v2",
				},
			},
			expectedImpExt: map[string]interface{}{
				"key_val": "k1=v1|k2=v2|sport=rugby,cricket",
			},
		},
		{
			name: "int data present in imp.ext.data",
			args: args{
				data:      json.RawMessage(`{"age": 25}`),
				impExtMap: map[string]interface{}{},
			},
			expectedImpExt: map[string]interface{}{
				"key_val": "age=25",
			},
		},
		{
			name: "float data present in imp.ext.data",
			args: args{
				data:      json.RawMessage(`{"floor": 0.15}`),
				impExtMap: map[string]interface{}{},
			},
			expectedImpExt: map[string]interface{}{
				"key_val": "floor=0.15",
			},
		},
		{
			name: "bool data present in imp.ext.data",
			args: args{
				data:      json.RawMessage(`{"k1": true}`),
				impExtMap: map[string]interface{}{},
			},
			expectedImpExt: map[string]interface{}{
				"key_val": "k1=true",
			},
		},
		{
			name: "imp.ext.data is not present",
			args: args{
				data:      nil,
				impExtMap: map[string]interface{}{},
			},
			expectedImpExt: map[string]interface{}{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			populateFirstPartyDataImpAttributes(tt.args.data, tt.args.impExtMap)
			assert.Equal(t, tt.expectedImpExt, tt.args.impExtMap)
		})
	}
}

func TestPopulateFirstPartyDataImpAttributesForMultipleAttributes(t *testing.T) {
	impExtMap := map[string]interface{}{
		"key_val": "k1=v1|k2=v2",
	}
	data := json.RawMessage(`{"sport":["rugby","cricket"],"pageType":"article","age":30,"floor":1.25}`)
	expectedKeyValArr := []string{"k1=v1", "k2=v2", "sport=rugby,cricket", "pageType=article", "age=30", "floor=1.25"}

	populateFirstPartyDataImpAttributes(data, impExtMap)

	//read dctr value and split on "|" for comparison
	actualKeyValArr := strings.Split(impExtMap[dctrKeyName].(string), "|")
	assert.Equal(t, expectedKeyValArr, actualKeyValArr)

}
func TestGetString(t *testing.T) {
	tests := []struct {
		name   string
		input  interface{}
		output string
	}{
		{
			name:   "Valid String",
			input:  "hello",
			output: "hello",
		},
		{
			name:   "Invalid String",
			input:  1,
			output: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getString(tt.input)
			assert.Equal(t, tt.output, got)
		})
	}
}

func TestGetStringArray(t *testing.T) {
	tests := []struct {
		name   string
		input  interface{}
		output []string
	}{
		{
			name:   "Valid String Array",
			input:  append(make([]interface{}, 0), "hello", "world"),
			output: []string{"hello", "world"},
		},
		{
			name:   "Invalid String Array",
			input:  "hello",
			output: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getStringArray(tt.input)
			assert.Equal(t, tt.output, got)
		})
	}
}

func TestIsStringArray(t *testing.T) {
	tests := []struct {
		name   string
		input  []interface{}
		output bool
	}{
		{
			name:   "Valid String Array",
			input:  append(make([]interface{}, 0), "hello", "world"),
			output: true,
		},
		{
			name:   "Invalid String Array",
			input:  append(make([]interface{}, 0), 1, 2),
			output: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isStringArray(tt.input)
			assert.Equal(t, tt.output, got)
		})
	}
}
