package prometheusmetrics

import (
	"strconv"

	"github.com/prebid/prebid-server/metrics"
	"github.com/prebid/prebid-server/openrtb_ext"
)

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

func cacheResultsAsString() []string {
	values := metrics.CacheResults()
	valuesAsString := make([]string, len(values))
	for i, v := range values {
		valuesAsString[i] = string(v)
	}
	return valuesAsString
}

func cookieTypesAsString() []string {
	values := metrics.CookieTypes()
	valuesAsString := make([]string, len(values))
	for i, v := range values {
		valuesAsString[i] = string(v)
	}
	return valuesAsString
}

func cookieSyncStatusesAsString() []string {
	values := metrics.CookieSyncStatuses()
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

func syncerRequestStatusesAsString() []string {
	values := metrics.SyncerRequestStatuses()
	valuesAsString := make([]string, len(values))
	for i, v := range values {
		valuesAsString[i] = string(v)
	}
	return valuesAsString
}

func syncerSetStatusesAsString() []string {
	values := metrics.SyncerSetUidStatuses()
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

func setUidStatusesAsString() []string {
	values := metrics.SetUidStatuses()
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

func modulesAsString(moduleStageNames map[string][]string) ([]string, []string) {
	mods := make([]string, len(moduleStageNames))
	stages := make([]string, 0)

	i := 0
	for m, s := range moduleStageNames {
		mods[i] = m
		stages = append(stages, s...)
		i++
	}

	return mods, stages
}
