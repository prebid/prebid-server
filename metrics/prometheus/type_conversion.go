package prometheusmetrics

import (
	"strconv"

	"github.com/prebid/prebid-server/metrics"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func actionsAsString() []string {
	values := metrics.RequestActions()
	valuesAsString := make([]string, len(values))
	for i, v := range values {
		valuesAsString[i] = string(v)
	}
	return valuesAsString
}

func adaptersAsString() []string {
	values := openrtb_ext.CoreBidderNames()
	valuesAsString := make([]string, len(values))
	for i, v := range values {
		valuesAsString[i] = string(v)
	}
	return valuesAsString
}

func adapterErrorsAsString() []string {
	values := metrics.AdapterErrors()
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

func cookieTypesAsString() []string {
	values := metrics.CookieTypes()
	valuesAsString := make([]string, len(values))
	for i, v := range values {
		valuesAsString[i] = string(v)
	}
	return valuesAsString
}

func cacheResultsAsString() []string {
	values := metrics.CacheResults()
	valuesAsString := make([]string, len(values))
	for i, v := range values {
		valuesAsString[i] = string(v)
	}
	return valuesAsString
}

func requestStatusesAsString() []string {
	values := metrics.RequestStatuses()
	valuesAsString := make([]string, len(values))
	for i, v := range values {
		valuesAsString[i] = string(v)
	}
	return valuesAsString
}

func requestTypesAsString() []string {
	values := metrics.RequestTypes()
	valuesAsString := make([]string, len(values))
	for i, v := range values {
		valuesAsString[i] = string(v)
	}
	return valuesAsString
}

func storedDataTypesAsString() []string {
	values := metrics.StoredDataTypes()
	valuesAsString := make([]string, len(values))
	for i, v := range values {
		valuesAsString[i] = string(v)
	}
	return valuesAsString
}

func storedDataFetchTypesAsString() []string {
	values := metrics.StoredDataFetchTypes()
	valuesAsString := make([]string, len(values))
	for i, v := range values {
		valuesAsString[i] = string(v)
	}
	return valuesAsString
}

func storedDataErrorsAsString() []string {
	values := metrics.StoredDataErrors()
	valuesAsString := make([]string, len(values))
	for i, v := range values {
		valuesAsString[i] = string(v)
	}
	return valuesAsString
}

func tcfVersionsAsString() []string {
	values := metrics.TCFVersions()
	valuesAsString := make([]string, len(values))
	for i, v := range values {
		valuesAsString[i] = string(v)
	}
	return valuesAsString
}
