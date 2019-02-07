package exchange

import (
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestGetPriceBucketString(t *testing.T) {
	low := openrtb_ext.PriceGranularityFromString("low")
	medium := openrtb_ext.PriceGranularityFromString("medium")
	high := openrtb_ext.PriceGranularityFromString("high")
	auto := openrtb_ext.PriceGranularityFromString("auto")
	dense := openrtb_ext.PriceGranularityFromString("dense")
	custom1 := openrtb_ext.PriceGranularity{
		Precision: 2,
		Ranges: []openrtb_ext.GranularityRange{
			{
				Min:       0.0,
				Max:       5.0,
				Increment: 0.03,
			},
			{
				Min:       5.0,
				Max:       10.0,
				Increment: 0.1,
			},
		},
	}

	price := 1.87
	getOnePriceBucket(t, "low", low, price, "1.50")
	getOnePriceBucket(t, "medium", medium, price, "1.80")
	getOnePriceBucket(t, "high", high, price, "1.87")
	getOnePriceBucket(t, "auto", auto, price, "1.85")
	getOnePriceBucket(t, "dense", dense, price, "1.87")
	getOnePriceBucket(t, "custom1", custom1, price, "1.86")

	// test a cpm above the max in low price bucket
	price = 5.72
	getOnePriceBucket(t, "low", low, price, "5.00")
	getOnePriceBucket(t, "medium", medium, price, "5.70")
	getOnePriceBucket(t, "high", high, price, "5.72")
	getOnePriceBucket(t, "auto", auto, price, "5.70")
	getOnePriceBucket(t, "dense", dense, price, "5.70")
	getOnePriceBucket(t, "custom1", custom1, price, "5.70")

}

func getOnePriceBucket(t *testing.T, name string, granularity openrtb_ext.PriceGranularity, price float64, expected string) {
	t.Helper()
	priceBucket, err := GetCpmStringValue(price, granularity)
	if err != nil {
		t.Errorf("Granularity: %s :: GetPriceBucketString: %s", name, err.Error())
	}
	if priceBucket != expected {
		t.Errorf("Granularity: %s :: Expected %s, got %s from %f", name, expected, priceBucket, price)
	}
}
