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
	priceBucket, err := GetPriceBucketString(price, granularity)
	if err != nil {
		t.Errorf("GetPriceBucketString: %s", err.Error())
	}
	if priceBucket != expected {
		t.Errorf("Expected %s, got %s from %f", expected, priceBucket, price)
	}
}
