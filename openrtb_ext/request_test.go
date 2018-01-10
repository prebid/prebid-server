package openrtb_ext

import (
	"encoding/json"
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
		if extRequest.Prebid.Targeting.PriceGranularity != "dense" {
			t.Errorf("ext2 expected Price granularity \"dense\", found \"%s\"", extRequest.Prebid.Targeting.PriceGranularity)
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
		if extRequest.Prebid.Targeting.PriceGranularity != "medium" {
			t.Errorf("ext3 expected Price granularity \"medium\", found \"%s\"", extRequest.Prebid.Targeting.PriceGranularity)
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
			"pricegranularity": "dense",
			"lengthmax": 20
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
