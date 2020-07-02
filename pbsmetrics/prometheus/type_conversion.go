package prometheusmetrics

import (
	"strconv"

	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbsmetrics"
)

func actionsAsString() []string {
	values := pbsmetrics.RequestActions()
	valuesAsString := make([]string, len(values))
	for i, v := range values {
		valuesAsString[i] = string(v)
	}
	return valuesAsString
}

func adaptersAsString() []string {
	values := openrtb_ext.BidderList()
	valuesAsString := make([]string, len(values))
	for i, v := range values {
		valuesAsString[i] = string(v)
	}
	return valuesAsString
}

func adapterErrorsAsString() []string {
	values := pbsmetrics.AdapterErrors()
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
	values := pbsmetrics.CookieTypes()
	valuesAsString := make([]string, len(values))
	for i, v := range values {
		valuesAsString[i] = string(v)
	}
	return valuesAsString
}

func cacheResultsAsString() []string {
	values := pbsmetrics.CacheResults()
	valuesAsString := make([]string, len(values))
	for i, v := range values {
		valuesAsString[i] = string(v)
	}
	return valuesAsString
}

func requestStatusesAsString() []string {
	values := pbsmetrics.RequestStatuses()
	valuesAsString := make([]string, len(values))
	for i, v := range values {
		valuesAsString[i] = string(v)
	}
	return valuesAsString
}

func requestTypesAsString() []string {
	values := pbsmetrics.RequestTypes()
	valuesAsString := make([]string, len(values))
	for i, v := range values {
		valuesAsString[i] = string(v)
	}
	return valuesAsString
}

func tcfVersionsAsString() []string {
	values := pbsmetrics.TCFVersions()
	valuesAsString := make([]string, len(values))
	for i, v := range values {
		valuesAsString[i] = string(v)
	}
	return valuesAsString
}
