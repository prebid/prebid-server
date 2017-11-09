package buckets

import (
	"testing"
)

func TestGetPriceBucketString(t *testing.T) {
	price := 1.87
	priceBucket, err := GetPriceBucketString(price, PriceGranularityLow)
	if err != nil {
		t.Errorf("GetPriceBucketString: %s", err.Error())
	}
	if priceBucket != "1.50" {
		t.Error("Expected 1.50")
	}

	priceBucket, err = GetPriceBucketString(price, PriceGranularityMedium)
	if err != nil {
		t.Errorf("GetPriceBucketString: %s", err.Error())
	}
	if priceBucket != "1.80" {
		t.Error("Expected 1.80")
	}

	priceBucket, err = GetPriceBucketString(price, PriceGranularityHigh)
	if err != nil {
		t.Errorf("GetPriceBucketString: %s", err.Error())
	}
	if priceBucket != "1.87" {
		t.Error("Expected 1.87")
	}

	priceBucket, err = GetPriceBucketString(price, PriceGranularityAuto)
	if err != nil {
		t.Errorf("GetPriceBucketString: %s", err.Error())
	}
	if priceBucket != "1.85" {
		t.Error("Expected 1.85")
	}

	priceBucket, err = GetPriceBucketString(price, PriceGranularityDense)
	if err != nil {
		t.Errorf("GetPriceBucketString: %s", err.Error())
	}
	if priceBucket != "1.87" {
		t.Error("Expected 1.87")
	}

	// test a cpm above the max in low price bucket
	price = 5.72

	priceBucket, err = GetPriceBucketString(price, PriceGranularityLow)
	if err != nil {
		t.Errorf("GetPriceBucketString: %s", err.Error())
	}
	if priceBucket != "5.00" {
		t.Error("Expected 5.00")
	}

	priceBucket, err = GetPriceBucketString(price, PriceGranularityMedium)
	if err != nil {
		t.Errorf("GetPriceBucketString: %s", err.Error())
	}
	if priceBucket != "5.70" {
		t.Error("Expected 5.70")
	}

	priceBucket, err = GetPriceBucketString(price, PriceGranularityHigh)
	if err != nil {
		t.Errorf("GetPriceBucketString: %s", err.Error())
	}
	if priceBucket != "5.72" {
		t.Error("Expected 5.72")
	}

	priceBucket, err = GetPriceBucketString(price, PriceGranularityAuto)
	if err != nil {
		t.Errorf("GetPriceBucketString: %s", err.Error())
	}
	if priceBucket != "5.70" {
		t.Error("Expected 5.70")
	}

	priceBucket, err = GetPriceBucketString(price, PriceGranularityDense)
	if err != nil {
		t.Errorf("GetPriceBucketString: %s", err.Error())
	}
	if priceBucket != "5.70" {
		t.Error("Expected 5.70")
	}
}
