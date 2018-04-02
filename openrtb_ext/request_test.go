package openrtb_ext

import (
	"encoding/json"
	"reflect"
	"testing"
)

// Test the unmashalling of the prebid extensions and setting default Price Granularity
func TestExtRequestTargeting(t *testing.T) {
	extRequest := &ExtRequest{}
	err := json.Unmarshal([]byte(ext1), extRequest)
	if err != nil {
		t.Errorf("ext1 Unmashall falure: %s", err.Error())
	}
	if extRequest.Prebid.Targeting != nil {
		t.Error("ext1 Targeting is not nil")
	}

	extRequest = &ExtRequest{}
	err = json.Unmarshal([]byte(ext2), extRequest)
	if err != nil {
		t.Errorf("ext2 Unmashall falure: %s", err.Error())
	}
	if extRequest.Prebid.Targeting == nil {
		t.Error("ext2 Targeting is nil")
	} else {
		pgDense := PriceGranularityFromString("dense")
		if !reflect.DeepEqual(extRequest.Prebid.Targeting.PriceGranularity, pgDense) {
			t.Errorf("ext2 expected Price granularity \"dense\" (%v), found \"%v\"", pgDense, extRequest.Prebid.Targeting.PriceGranularity)
		}
	}

	extRequest = &ExtRequest{}
	err = json.Unmarshal([]byte(ext3), extRequest)
	if err != nil {
		t.Errorf("ext3 Unmashall falure: %s", err.Error())
	}
	if extRequest.Prebid.Targeting == nil {
		t.Error("ext3 Targeting is nil")
	} else {
		pgMed := PriceGranularityFromString("medium")
		if !reflect.DeepEqual(extRequest.Prebid.Targeting.PriceGranularity, pgMed) {
			t.Errorf("ext3 expected Price granularity \"medium\", found \"%v\"", extRequest.Prebid.Targeting.PriceGranularity)
		}
	}
}

const ext1 = `{
	"prebid": {
		"non_target": "some junk"
	}
}
`

const ext2 = `{
	"prebid": {
		"targeting": {
			"pricegranularity": "dense"
		}
	}
}`

const ext3 = `{
	"prebid": {
		"targeting": { }
	}
}`

func TestCacheIllegal(t *testing.T) {
	var bids ExtRequestPrebidCache
	if err := json.Unmarshal([]byte(`{}`), &bids); err == nil {
		t.Error("Unmarshal should fail when cache.bids is undefined.")
	}
	if err := json.Unmarshal([]byte(`{"bids":null}`), &bids); err == nil {
		t.Error("Unmarshal should fail when cache.bids is null.")
	}
	if err := json.Unmarshal([]byte(`{"bids":true}`), &bids); err == nil {
		t.Error("Unmarshal should fail when cache.bids is not an object.")
	}
}

func TestCacheLegal(t *testing.T) {
	var bids ExtRequestPrebidCache
	if err := json.Unmarshal([]byte(`{"bids":{}}`), &bids); err != nil {
		t.Error("Unmarshal should succeed when cache.bids is defined.")
	}
	if bids.Bids == nil {
		t.Error("bids.Bids should not be nil.")
	}
}

type granularityTestData struct {
	json   []byte
	target PriceGranularity
}

func TestGranularityUnmarshal(t *testing.T) {
	for _, test := range validGranularityTests {
		var resolved PriceGranularity
		err := json.Unmarshal(test.json, &resolved)
		if err != nil {
			t.Errorf("Failed to Unmarshall granularity: %s", err.Error())
		}
		if !reflect.DeepEqual(test.target, resolved) {
			t.Errorf("Granularity unmarshal failed, the unmarshalled JSON did not match the target\nExpected: %v\nActual  : %v", test.target, resolved)
		}
	}
}

var validGranularityTests []granularityTestData = []granularityTestData{
	{
		json: []byte(`[{ "precision": 4, "min": 0, "max": 5, "increment": 0.1}, {"precision": 4, "min": 5, "max":10, "increment":0.5}, {"precision":4, "min":10, "max":20, "increment":1}]`),
		target: PriceGranularity{GranularityRange{Precision: 4, Min: 0.0, Max: 5.0, Increment: 0.1},
			GranularityRange{Precision: 4, Min: 5.0, Max: 10.0, Increment: 0.5},
			GranularityRange{Precision: 4, Min: 10.0, Max: 20.0, Increment: 1.0}},
	},
	{
		json: []byte(`[{ "max":5, "increment": 0.05}, {"max": 10, "increment": 0.25}, {"max": 20, "increment": 0.5}]`),
		target: PriceGranularity{GranularityRange{Precision: 2, Min: 0.0, Max: 5.0, Increment: 0.05},
			GranularityRange{Precision: 2, Min: 5.0, Max: 10.0, Increment: 0.25},
			GranularityRange{Precision: 2, Min: 10.0, Max: 20.0, Increment: 0.5}},
	},
	{
		json:   []byte(`"medium"`),
		target: priceGranularityMed,
	},
	{
		json:   []byte(`[{ "precision": 3, "max":20, "increment":0.005}]`),
		target: PriceGranularity{GranularityRange{Precision: 3, Min: 0.0, Max: 20.0, Increment: 0.005}},
	},
	{
		json: []byte(`[{"precision": 0, "max":5, "increment": 1}, {"precision": 0, "max": 10, "increment": 2}, {"precision":0, "max": 20, "increment": 5}]`),
		target: PriceGranularity{GranularityRange{Precision: 0, Min: 0.0, Max: 5.0, Increment: 1.0},
			GranularityRange{Precision: 0, Min: 5.0, Max: 10.0, Increment: 2.0},
			GranularityRange{Precision: 0, Min: 10.0, Max: 20.0, Increment: 5.0}},
	},
}

func TestGranularityUnmarshalBad(t *testing.T) {
	tests := [][]byte{
		[]byte(`{}`),
		[]byte(`[]`),
		[]byte(`[{"precision": -1, "min":0, "max":20, "increment":0.5}]`),
		[]byte(`[{"min": 5, "max":1, "increment": 0.1}]`),
		[]byte(`[{"min":0, "max":20, "increment": -1}]`),
		[]byte(`[{"min":"0", "max":"20", "increment": "0.1"}]`),
		[]byte(`[{"min":0, "max":20, "increment":0.1}, {"min":15, "max":30, "increment":1.0}]`),
		[]byte(`[{"precision": 2, "min":0, "max":10, "increment":0.1}, {"precision": 1, "min":10, "max":50, "increment":1}]`),
		[]byte(`[{"max":20, "increment":0.1}. {"max":10, "increment":0.02}]`),
		[]byte(`[{"max":20, "min":10, "increment": 0.1}, {"max":10, "min":0, "increment":0.05}]`),
	}
	var resolved PriceGranularity
	for _, b := range tests {
		resolved = PriceGranularity{}
		err := json.Unmarshal(b, &resolved)
		if err == nil {
			t.Errorf("Invalid granularity unmarshalled without error.\nJSON was: %s\n Resolved to: %v", string(b), resolved)
		}
	}
}
