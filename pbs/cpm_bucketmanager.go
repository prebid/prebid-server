package pbs

import "math"
import "strconv"

const DEFAULT_PRECISION = 2

func getLowPriceConfig() map[string][]map[string]float64 {
	return map[string][]map[string]float64{
		"buckets": []map[string]float64{
			map[string]float64{
				"min":       0,
				"max":       5,
				"increment": 0.5,
			},
		},
	}
}

func getMedPriceConfig() map[string][]map[string]float64 {
	return map[string][]map[string]float64{
		"buckets": []map[string]float64{
			map[string]float64{
				"min":       0,
				"max":       20,
				"increment": 0.1,
			},
		},
	}
}

func getHighPriceConfig() map[string][]map[string]float64 {
	return map[string][]map[string]float64{
		"buckets": []map[string]float64{
			map[string]float64{
				"min":       0,
				"max":       20,
				"increment": 0.01,
			},
		},
	}
}

func getDensePriceConfig() map[string][]map[string]float64 {
	return map[string][]map[string]float64{
		"buckets": []map[string]float64{
			map[string]float64{
				"min":       0,
				"max":       3,
				"increment": 0.01,
			},
			map[string]float64{
				"min":       3,
				"max":       8,
				"increment": 0.05,
			},
			map[string]float64{
				"min":       8,
				"max":       20,
				"increment": 0.5,
			},
		},
	}
}

func getAutoPriceConfig() map[string][]map[string]float64 {
	return map[string][]map[string]float64{
		"buckets": []map[string]float64{
			map[string]float64{
				"min":       0,
				"max":       5,
				"increment": 0.05,
			},
			map[string]float64{
				"min":       5,
				"max":       10,
				"increment": 0.1,
			},
			map[string]float64{
				"min":       10,
				"max":       20,
				"increment": 0.5,
			},
		},
	}
}

func getCpmStringValue(cpm float64, config map[string][]map[string]float64) string {
	bucketsArr := config["buckets"]
	for i := 0; i < len(bucketsArr); i++ {
		currentBucket := bucketsArr[i]
		if cpm >= currentBucket["min"] && cpm < currentBucket["max"] {
			d := RoundUp(cpm/currentBucket["increment"], DEFAULT_PRECISION)
			roundedCPM := math.Floor(d) * currentBucket["increment"]
			return strconv.FormatFloat(roundedCPM, 'f', DEFAULT_PRECISION, 64)
		}
	}
	return ""
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
