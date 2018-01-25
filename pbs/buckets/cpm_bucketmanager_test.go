package buckets

import (
	"github.com/prebid/prebid-server/openrtb_ext"
	"testing"
)

func TestGetPriceBucketString(t *testing.T) {
	price := 1.87
	getOnePriceBucket(t, openrtb_ext.PriceGranularityLow, price, "1.50")
	getOnePriceBucket(t, openrtb_ext.PriceGranularityMedium, price, "1.80")
	getOnePriceBucket(t, openrtb_ext.PriceGranularityHigh, price, "1.87")
	getOnePriceBucket(t, openrtb_ext.PriceGranularityAuto, price, "1.85")
	getOnePriceBucket(t, openrtb_ext.PriceGranularityDense, price, "1.87")

	// test a cpm above the max in low price bucket
	price = 5.72
	getOnePriceBucket(t, openrtb_ext.PriceGranularityLow, price, "5.00")
	getOnePriceBucket(t, openrtb_ext.PriceGranularityMedium, price, "5.70")
	getOnePriceBucket(t, openrtb_ext.PriceGranularityHigh, price, "5.72")
	getOnePriceBucket(t, openrtb_ext.PriceGranularityAuto, price, "5.70")
	getOnePriceBucket(t, openrtb_ext.PriceGranularityDense, price, "5.70")

}

func getOnePriceBucket(t *testing.T, granularity openrtb_ext.PriceGranularity, price float64, expected string) {
	t.Helper()
	priceBucket, err := GetPriceBucketString(price, granularity, 0)
	if err != nil {
		t.Errorf("GetPriceBucketString: %s", err.Error())
	}
	if priceBucket != expected {
		t.Errorf("Expected %s, got %s from %f", expected, priceBucket, price)
	}
}

func TestGetCpmStringValue(t *testing.T) {
	cpm := 1.55
	precision := 0
	granularityMultiplier := 108.00

	// Testing with low price bucket
	cpmStr := getCpmStringValue(cpm, priceBucketLow, precision, granularityMultiplier)

	// cpmStr should be 162 because 1.50 (low price bucket for 1.55) * 108 granularity multipler
	if cpmStr != "162.00" {
		t.Errorf("Expected %s, got %s", "162.00", cpmStr)
	}

	// Testing with medium price bucket
	cpmStr = getCpmStringValue(cpm, priceBucketMed, precision, granularityMultiplier)

	// cpmStr should be 162 because 1.50 (medium price bucket for 1.55) * 108 granularity multipler
	if cpmStr != "162.00" {
		t.Errorf("Expected %s, got %s", "162.00", cpmStr)
	}

	// Testing with high price bucket
	cpmStr = getCpmStringValue(cpm, priceBucketHigh, precision, granularityMultiplier)

	// cpmStr should be 162 because 1.55 (high price bucket for 1.55) * 108 granularity multipler
	if cpmStr != "167.40" {
		t.Errorf("Expected %s, got %s", "167.40", cpmStr)
	}
}
