package buckets

import (
	"testing"
)

func TestGetPriceBucketString(t *testing.T) {
	price := 1.87
	getOnePriceBucket(t, PriceGranularityLow, price, "1.50")
	getOnePriceBucket(t, PriceGranularityMedium, price, "1.80")
	getOnePriceBucket(t, PriceGranularityHigh, price, "1.87")
	getOnePriceBucket(t, PriceGranularityAuto, price, "1.85")
	getOnePriceBucket(t, PriceGranularityDense, price, "1.87")

	// test a cpm above the max in low price bucket
	price = 5.72
	getOnePriceBucket(t, PriceGranularityLow, price, "5.00")
	getOnePriceBucket(t, PriceGranularityMedium, price, "5.70")
	getOnePriceBucket(t, PriceGranularityHigh, price, "5.72")
	getOnePriceBucket(t, PriceGranularityAuto, price, "5.70")
	getOnePriceBucket(t, PriceGranularityDense, price, "5.70")

}

func getOnePriceBucket(t *testing.T, granularity PriceGranularity, price float64, expected string) {
	t.Helper()
	priceBucket, err := GetPriceBucketString(price, granularity)
	if err != nil {
		t.Errorf("GetPriceBucketString: %s", err.Error())
	}
	if priceBucket != expected {
		t.Errorf("Expected %s, got %s from %f", expected, priceBucket, price)
	}
}
