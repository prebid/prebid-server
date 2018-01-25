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

func TestCurrency(t *testing.T) {
	extRequest := &ExtRequest{}
	err := json.Unmarshal([]byte(ext4), extRequest)
	if err != nil {
		t.Errorf("ext4 Unmashall falure: %s", err.Error())
	}
	if extRequest.Currency.Rates == nil {
		t.Error("ext4 Rates is nil")
	} else {
		if extRequest.Currency.Rates == nil {
			t.Errorf("ext4 expected rates \"USD: JPY:110.21\", found nil")
		}
	}

	extRequest = &ExtRequest{}
	err = json.Unmarshal([]byte(ext5), extRequest)
	if err != nil {
		t.Errorf("ext5 Unmashall falure: %s", err.Error())
	}
	if extRequest.Currency.Rates != nil {
		t.Error("ext5 Rates is not nil")
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

const ext4 = `{
	"currency": {
		"rates": {
			"USD": {
				"JPY": 110.21
			}
		}
	}
}`

const ext5 = `{
	"currency": {
		"non_currency": "some junk"
	}
}`
