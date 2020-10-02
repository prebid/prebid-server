package exchange

import (
	"fmt"
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

	// Limit the number of decimal significant figures. Very large values lead to Panics
	if precision > 4 {
		return cpmStr, fmt.Errorf("Limit the number of precision figures to 4. Parsed value: %d", precision)
	}

	for i := 0; i < len(config.Ranges); i++ {
		// calculate max of highest bucket
		if bucketMax < config.Ranges[i].Max {
			bucketMax = config.Ranges[i].Max
		}
		// find range cpm is in
		if cpm >= config.Ranges[i].Min && cpm <= config.Ranges[i].Max {
			increment = config.Ranges[i].Increment
		}
	}
	// If we are over max, just return that
	if cpm > bucketMax {
		return strconv.FormatFloat(bucketMax, 'f', precision, 64), nil
	}
	// If increment exists, get cpm string value
	if increment > 0 {
		cpmStr = getCpmTarget(cpm, increment, precision)
	}
	return cpmStr, nil
}

func getCpmTarget(cpm float64, increment float64, precision int) string {
	roundedCPM := math.Floor(cpm/increment) * increment
	return strconv.FormatFloat(roundedCPM, 'f', precision, 64)
}
