// Package mathutil provides utility functions for mathematical operations.
package mathutil

import "math"

// RoundTo4Decimals rounds a float64 value to 4 decimal places.
func RoundTo4Decimals(amount float64) float64 {
	return math.Round(amount*10000) / 10000
}
