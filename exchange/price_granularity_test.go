package exchange

import (
	"github.com/prebid/prebid-server/openrtb_ext"
	"testing"
)

func TestGetPriceBucketString(t *testing.T) {
	price := 1.87
	getOnePriceBucket(t, "low", price, "1.50")
	getOnePriceBucket(t, "medium", price, "1.80")
	getOnePriceBucket(t, "high", price, "1.87")
	getOnePriceBucket(t, "auto", price, "1.85")
	getOnePriceBucket(t, "dense", price, "1.87")

	// test a cpm above the max in low price bucket
	price = 5.72
	getOnePriceBucket(t, "low", price, "5.00")
	getOnePriceBucket(t, "medium", price, "5.70")
	getOnePriceBucket(t, "high", price, "5.72")
	getOnePriceBucket(t, "auto", price, "5.70")
	getOnePriceBucket(t, "dense", price, "5.70")

}

func getOnePriceBucket(t *testing.T, granularity string, price float64, expected string) {
	t.Helper()
	priceBucket, err := GetCpmStringValue(price, openrtb_ext.PriceGranularityFromString(granularity))
	if err != nil {
		t.Errorf("GetPriceBucketString: %s", err.Error())
	}
	if priceBucket != expected {
		t.Errorf("Expected %s, got %s from %f", expected, priceBucket, price)
	}
}
