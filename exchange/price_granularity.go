package exchange

import (
	"math"
	"strconv"

	"github.com/prebid/prebid-server/openrtb_ext"
)

// GetPriceBucket is the externally facing function for computing CPM buckets
func GetPriceBucket(cpm float64, config openrtb_ext.PriceGranularity) string {
	cpmStr := ""
	bucketMax := 0.0
	increment := 0.0
	precision := config.Precision

	for i := 0; i < len(config.Ranges); i++ {
		if config.Ranges[i].Max > bucketMax {
			bucketMax = config.Ranges[i].Max
		}
		// find what range cpm is in
		if cpm >= config.Ranges[i].Min && cpm <= config.Ranges[i].Max {
			increment = config.Ranges[i].Increment
		}
	}

	if cpm > bucketMax {
		// We are over max, just return that
		cpmStr = strconv.FormatFloat(bucketMax, 'f', precision, 64)
	} else if increment > 0 {
		// If increment exists, get cpm string value
		cpmStr = getCpmTarget(cpm, increment, precision)
	}

	return cpmStr
}

func getCpmTarget(cpm float64, increment float64, precision int) string {
	roundedCPM := math.Floor(cpm/increment) * increment
	return strconv.FormatFloat(roundedCPM, 'f', precision, 64)
}
