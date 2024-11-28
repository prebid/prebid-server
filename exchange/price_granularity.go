package exchange

import (
	"math"
	"strconv"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// GetPriceBucket is the externally facing function for computing CPM buckets
func GetPriceBucket(bid openrtb2.Bid, targetingData targetData) string {
	cpmStr := ""
	bucketMax := 0.0
	bucketMin := 0.0
	increment := 0.0

	config := targetingData.priceGranularity //assign default price granularity

	if bidType, err := getMediaTypeForBid(bid); err == nil {
		if bidType == openrtb_ext.BidTypeBanner && targetingData.mediaTypePriceGranularity.Banner != nil {
			config = *targetingData.mediaTypePriceGranularity.Banner
		} else if bidType == openrtb_ext.BidTypeVideo && targetingData.mediaTypePriceGranularity.Video != nil {
			config = *targetingData.mediaTypePriceGranularity.Video
		} else if bidType == openrtb_ext.BidTypeNative && targetingData.mediaTypePriceGranularity.Native != nil {
			config = *targetingData.mediaTypePriceGranularity.Native
		}
	}

	precision := *config.Precision

	cpm := bid.Price
	for i := 0; i < len(config.Ranges); i++ {
		if config.Ranges[i].Max > bucketMax {
			bucketMax = config.Ranges[i].Max
		}
		// find what range cpm is in
		if cpm >= config.Ranges[i].Min && cpm <= config.Ranges[i].Max {
			increment = config.Ranges[i].Increment
			bucketMin = config.Ranges[i].Min
		}
	}

	if cpm > bucketMax {
		// We are over max, just return that
		cpmStr = strconv.FormatFloat(bucketMax, 'f', precision, 64)
	} else if increment > 0 {
		// If increment exists, get cpm string value
		cpmStr = getCpmTarget(cpm, bucketMin, increment, precision)
	}

	return cpmStr
}

func getCpmTarget(cpm float64, bucketMin float64, increment float64, precision int) string {
	roundedCPM := math.Floor((cpm-bucketMin)/increment)*increment + bucketMin
	return strconv.FormatFloat(roundedCPM, 'f', precision, 64)
}
