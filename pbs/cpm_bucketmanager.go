package pbs

import "math"
import "strconv"

const DEFAULT_PRECISION = 2

func getLowPriceConfig() map[string][]map[string]float64 {
	return map[string][]map[string]float64{
		"buckets": {
			{
				"min":       0,
				"max":       5,
				"increment": 0.5,
			},
		},
	}
}

func getMedPriceConfig() map[string][]map[string]float64 {
	return map[string][]map[string]float64{
		"buckets": {
			{
				"min":       0,
				"max":       20,
				"increment": 0.1,
			},
		},
	}
}

func getHighPriceConfig() map[string][]map[string]float64 {
	return map[string][]map[string]float64{
		"buckets": {
			{
				"min":       0,
				"max":       20,
				"increment": 0.01,
			},
		},
	}
}

func getDensePriceConfig() map[string][]map[string]float64 {
	return map[string][]map[string]float64{
		"buckets": {
			{
				"min":       0,
				"max":       3,
				"increment": 0.01,
			},
			{
				"min":       3,
				"max":       8,
				"increment": 0.05,
			},
			{
				"min":       8,
				"max":       20,
				"increment": 0.5,
			},
		},
	}
}

func getAutoPriceConfig() map[string][]map[string]float64 {
	return map[string][]map[string]float64{
		"buckets": {
			{
				"min":       0,
				"max":       5,
				"increment": 0.05,
			},
			{
				"min":       5,
				"max":       10,
				"increment": 0.1,
			},
			{
				"min":       10,
				"max":       20,
				"increment": 0.5,
			},
		},
	}
}

func getCpmStringValue(cpm float64, config map[string][]map[string]float64) string {
	bucketsArr := config["buckets"]
	cpmStr := ""
	bucket := make(map[string]float64)
	bucketMax := 0.0
	// calculate max of highest bucket
	for i := 0; i < len(bucketsArr); i++ {
		if bucketsArr[i]["max"] > bucketMax {
			bucketMax = bucketsArr[i]["max"]
		}
	} // calculate which bucket cpm is in
	for i := 0; i < len(bucketsArr); i++ {
		currentBucket := bucketsArr[i]
		if cpm > bucketMax {
			var precision int = DEFAULT_PRECISION
			if currentBucket["precision"] != 0 {
				precision = int(currentBucket["precision"])
			}
			cpmStr = strconv.FormatFloat(bucketMax, 'f', precision, 64)
		} else if cpm >= currentBucket["min"] && cpm <= currentBucket["max"] {
			bucket = currentBucket
		}
	}
	if len(bucket) > 0 {
		cpmStr = getCpmTarget(cpm, bucket["increment"], int(bucket["precision"]))
	}
	return cpmStr
}

func getCpmTarget(cpm float64, increment float64, precision int) string {
	if precision == 0 {
		precision = DEFAULT_PRECISION
	}
	d := RoundUp(cpm/increment, precision)
	roundedCPM := math.Floor(d) * increment
	return strconv.FormatFloat(roundedCPM, 'f', precision, 64)
}

func RoundUp(input float64, places int) (newVal float64) {
	var round float64
	pow := math.Pow(10, float64(places))
	digit := pow * input
	round = math.Ceil(digit)
	newVal = round / pow
	return
}

func GetPriceBucketString(cpm float64) map[string]string {
	return map[string]string{
		"low":   getCpmStringValue(cpm, getLowPriceConfig()),
		"med":   getCpmStringValue(cpm, getMedPriceConfig()),
		"high":  getCpmStringValue(cpm, getHighPriceConfig()),
		"auto":  getCpmStringValue(cpm, getAutoPriceConfig()),
		"dense": getCpmStringValue(cpm, getDensePriceConfig()),
	}
}
