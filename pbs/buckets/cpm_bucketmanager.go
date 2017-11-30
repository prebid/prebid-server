package buckets

import "math"
import (
	"fmt"
	"github.com/prebid/prebid-server/openrtb_ext"
	"strconv"
)

const DEFAULT_PRECISION = 2

// Parameters every price bucket needs.
type priceBucketParams struct {
	min       float64
	max       float64
	increment float64
}

// A type to hold the price bucket configurations
type priceBucketConf []priceBucketParams

var priceBucketConfigMap = map[openrtb_ext.PriceGranularity]priceBucketConf{
	openrtb_ext.PriceGranularityLow:    priceBucketLow,
	openrtb_ext.PriceGranularityMedium: priceBucketMed,
	openrtb_ext.PriceGranularityMedPBS: priceBucketMed,
	openrtb_ext.PriceGranularityHigh:   priceBucketHigh,
	openrtb_ext.PriceGranularityAuto:   priceBucketAuto,
	openrtb_ext.PriceGranularityDense:  priceBucketDense,
}
var priceBucketLow = priceBucketConf{
	{
		min:       0,
		max:       5,
		increment: 0.5,
	},
}

var priceBucketMed = priceBucketConf{
	{
		min:       0,
		max:       20,
		increment: 0.1,
	},
}

var priceBucketHigh = priceBucketConf{
	{
		min:       0,
		max:       20,
		increment: 0.01,
	},
}

var priceBucketDense = priceBucketConf{
	{
		min:       0,
		max:       3,
		increment: 0.01,
	},
	{
		min:       3,
		max:       8,
		increment: 0.05,
	},
	{
		min:       8,
		max:       20,
		increment: 0.5,
	},
}

var priceBucketAuto = priceBucketConf{
	{
		min:       0,
		max:       5,
		increment: 0.05,
	},
	{
		min:       5,
		max:       10,
		increment: 0.1,
	},
	{
		min:       10,
		max:       20,
		increment: 0.5,
	},
}

func getCpmStringValue(cpm float64, config priceBucketConf, precision int) string {
	cpmStr := ""
	bucketMax := 0.0
	increment := 0.0
	if precision == 0 {
		precision = DEFAULT_PRECISION
	}
	// calculate max of highest bucket
	for i := 0; i < len(config); i++ {
		if config[i].max > bucketMax {
			bucketMax = config[i].max
		}
	} // calculate which bucket cpm is in
	if cpm > bucketMax {
		// If we are over max, just return that
		return strconv.FormatFloat(bucketMax, 'f', precision, 64)
	}
	for i := 0; i < len(config); i++ {
		if cpm >= config[i].min && cpm <= config[i].max {
			increment = config[i].increment
		}
	}
	if increment > 0 {
		cpmStr = getCpmTarget(cpm, increment, precision)
	}
	return cpmStr
}

func getCpmTarget(cpm float64, increment float64, precision int) string {
	// Probably don't need this default check given it is in getCpmStringValue
	if precision == 0 {
		precision = DEFAULT_PRECISION
	}
	roundedCPM := math.Floor(cpm/increment) * increment
	return strconv.FormatFloat(roundedCPM, 'f', precision, 64)
}

// Externally facing function for computing CPM buckets
// We don't currently have a precision config, so enforcing the default here.
func GetPriceBucketString(cpm float64, granularity openrtb_ext.PriceGranularity) (string, error) {
	// Default to medium if no granularity is given
	if granularity == "" {
		granularity = "medium"
	}
	config, ok := priceBucketConfigMap[granularity]
	if ok {
		return getCpmStringValue(cpm, config, DEFAULT_PRECISION), nil
	}
	return "", fmt.Errorf("Price bucket granularity error: '%s' is not a recognized granularity", string(granularity))
}
