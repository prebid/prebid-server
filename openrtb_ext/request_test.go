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

func TestCacheDefaults(t *testing.T) {
	var bids ExtRequestPrebidCacheBids
	if err := json.Unmarshal([]byte("{}"), &bids); err != nil {
		t.Fatalf("Error unmarshaling empty object: %v", err)
	}

	if !bids.Winners {
		t.Errorf("bids.winners should default to true. Got false.")
	}
	if bids.Deals {
		t.Errorf("bids.deals should default to false. Got true.")
	}
}

func TestCacheOverrides(t *testing.T) {
	var bids ExtRequestPrebidCacheBids
	if err := json.Unmarshal([]byte(`{"winners":false,"deals":true}`), &bids); err != nil {
		t.Fatalf("Error unmarshaling empty object: %v", err)
	}

	if bids.Winners {
		t.Errorf("bids.winners should set to false. Got true.")
	}
	if !bids.Deals {
		t.Errorf("bids.deals should set to true. Got false.")
	}
}
