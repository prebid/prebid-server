package exchange

import (
	"math"
	"strconv"

	"github.com/prebid/prebid-server/openrtb_ext"
)

// DEFAULT_PRECISION should be taken care of in openrtb_ext/request.go, but throwing an additional safety check here.

// GetCpmStringValue is the externally facing function for computing CPM buckets
func GetCpmStringValue(cpm float64, config openrtb_ext.PriceGranularity) (string, error) {
	cpmStr := ""
	bucketMax := 0.0
	increment := 0.0
	precision := config.Precision
	// calculate max of highest bucket
	for i := 0; i < len(config.Ranges); i++ {
		if config.Ranges[i].Max > bucketMax {
			bucketMax = config.Ranges[i].Max
		}
	} // calculate which bucket cpm is in
	if cpm > bucketMax {
		// If we are over max, just return that
		return strconv.FormatFloat(bucketMax, 'f', precision, 64), nil
	}
	for i := 0; i < len(config.Ranges); i++ {
		if cpm >= config.Ranges[i].Min && cpm <= config.Ranges[i].Max {
			increment = config.Ranges[i].Increment
		}
	}
	if increment > 0 {
		cpmStr = getCpmTarget(cpm, increment, precision)
	}
	return cpmStr, nil
}

func getCpmTarget(cpm float64, increment float64, precision int) string {
	roundedCPM := math.Floor(cpm/increment) * increment
	return strconv.FormatFloat(roundedCPM, 'f', precision, 64)
}
