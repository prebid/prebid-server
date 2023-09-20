package prometheusmetrics

import (
	"strconv"
)

func enumAsString[T ~string](values []T) []string {
	valuesAsString := make([]string, len(values))
	for i, v := range values {
		valuesAsString[i] = string(v)
	}
	return valuesAsString
}

func boolValuesAsString() []string {
	return []string{
		strconv.FormatBool(true),
		strconv.FormatBool(false),
	}
}
