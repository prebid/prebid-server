package prometheusmetrics

import (
	"github.com/prebid/prebid-server/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

func preloadLabelValues(m *Metrics) {
	var (
		actionValues              = actionsAsString()
		adapterErrorValues        = adapterErrorsAsString()
		adapterValues             = adaptersAsString()
		bidTypeValues             = []string{markupDeliveryAdm, markupDeliveryNurl}
		boolValues                = boolValuesAsString()
		cacheResultValues         = cacheResultsAsString()
		connectionErrorValues     = []string{connectionAcceptError, connectionCloseError}
		cookieValues              = cookieTypesAsString()
		requestStatusValues       = requestStatusesAsString()
		requestTypeValues         = requestTypesAsString()
		storedDataFetchTypeValues = storedDataFetchTypesAsString()
		storedDataErrorValues     = storedDataErrorsAsString()
		sourceValues              = []string{sourceRequest}
	)

	preloadLabelValuesForCounter(m.connectionsError, map[string][]string{
		connectionErrorLabel: connectionErrorValues,
	})

	preloadLabelValuesForCounter(m.impressions, map[string][]string{
		isBannerLabel: boolValues,
		isVideoLabel:  boolValues,
		isAudioLabel:  boolValues,
		isNativeLabel: boolValues,
	})

	preloadLabelValuesForHistogram(m.prebidCacheWriteTimer, map[string][]string{
		successLabel: boolValues,
	})

	preloadLabelValuesForCounter(m.requests, map[string][]string{
		requestTypeLabel:   requestTypeValues,
		requestStatusLabel: requestStatusValues,
	})

	preloadLabelValuesForHistogram(m.requestsTimer, map[string][]string{
		requestTypeLabel: requestTypeValues,
	})

	preloadLabelValuesForHistogram(m.storedAccountFetchTimer, map[string][]string{
		storedDataFetchTypeLabel: storedDataFetchTypeValues,
	})

	preloadLabelValuesForHistogram(m.storedAMPFetchTimer, map[string][]string{
		storedDataFetchTypeLabel: storedDataFetchTypeValues,
	})

	preloadLabelValuesForHistogram(m.storedCategoryFetchTimer, map[string][]string{
		storedDataFetchTypeLabel: storedDataFetchTypeValues,
	})

	preloadLabelValuesForHistogram(m.storedRequestFetchTimer, map[string][]string{
		storedDataFetchTypeLabel: storedDataFetchTypeValues,
	})

	preloadLabelValuesForHistogram(m.storedVideoFetchTimer, map[string][]string{
		storedDataFetchTypeLabel: storedDataFetchTypeValues,
	})

	preloadLabelValuesForCounter(m.storedAccountErrors, map[string][]string{
		storedDataErrorLabel: storedDataErrorValues,
	})

	preloadLabelValuesForCounter(m.storedAMPErrors, map[string][]string{
		storedDataErrorLabel: storedDataErrorValues,
	})

	preloadLabelValuesForCounter(m.storedCategoryErrors, map[string][]string{
		storedDataErrorLabel: storedDataErrorValues,
	})

	preloadLabelValuesForCounter(m.storedRequestErrors, map[string][]string{
		storedDataErrorLabel: storedDataErrorValues,
	})

	preloadLabelValuesForCounter(m.storedVideoErrors, map[string][]string{
		storedDataErrorLabel: storedDataErrorValues,
	})

	preloadLabelValuesForCounter(m.requestsWithoutCookie, map[string][]string{
		requestTypeLabel: requestTypeValues,
	})

	preloadLabelValuesForCounter(m.storedImpressionsCacheResult, map[string][]string{
		cacheResultLabel: cacheResultValues,
	})

	preloadLabelValuesForCounter(m.storedRequestCacheResult, map[string][]string{
		cacheResultLabel: cacheResultValues,
	})

	preloadLabelValuesForCounter(m.accountCacheResult, map[string][]string{
		cacheResultLabel: cacheResultValues,
	})

	preloadLabelValuesForCounter(m.adapterBids, map[string][]string{
		adapterLabel:        adapterValues,
		markupDeliveryLabel: bidTypeValues,
	})

	preloadLabelValuesForCounter(m.adapterCookieSync, map[string][]string{
		adapterLabel:        adapterValues,
		privacyBlockedLabel: boolValues,
	})

	preloadLabelValuesForCounter(m.adapterErrors, map[string][]string{
		adapterLabel:      adapterValues,
		adapterErrorLabel: adapterErrorValues,
	})

	preloadLabelValuesForCounter(m.adapterPanics, map[string][]string{
		adapterLabel: adapterValues,
	})

	preloadLabelValuesForHistogram(m.adapterPrices, map[string][]string{
		adapterLabel: adapterValues,
	})

	preloadLabelValuesForCounter(m.adapterRequests, map[string][]string{
		adapterLabel: adapterValues,
		cookieLabel:  cookieValues,
		hasBidsLabel: boolValues,
	})

	if !m.metricsDisabled.AdapterConnectionMetrics {
		preloadLabelValuesForCounter(m.adapterCreatedConnections, map[string][]string{
			adapterLabel: adapterValues,
		})

		preloadLabelValuesForCounter(m.adapterReusedConnections, map[string][]string{
			adapterLabel: adapterValues,
		})

		preloadLabelValuesForHistogram(m.adapterConnectionWaitTime, map[string][]string{
			adapterLabel: adapterValues,
		})
	}

	preloadLabelValuesForHistogram(m.adapterRequestsTimer, map[string][]string{
		adapterLabel: adapterValues,
	})

	preloadLabelValuesForCounter(m.adapterUserSync, map[string][]string{
		adapterLabel: adapterValues,
		actionLabel:  actionValues,
	})

	//to minimize memory usage, queuedTimeout metric is now supported for video endpoint only
	//boolean value represents 2 general request statuses: accepted and rejected
	preloadLabelValuesForHistogram(m.requestsQueueTimer, map[string][]string{
		requestTypeLabel:   {string(metrics.ReqTypeVideo)},
		requestStatusLabel: {requestSuccessLabel, requestRejectLabel},
	})

	preloadLabelValuesForCounter(m.privacyCCPA, map[string][]string{
		sourceLabel: sourceValues,
		optOutLabel: boolValues,
	})

	preloadLabelValuesForCounter(m.privacyCOPPA, map[string][]string{
		sourceLabel: sourceValues,
	})

	preloadLabelValuesForCounter(m.privacyLMT, map[string][]string{
		sourceLabel: sourceValues,
	})

	preloadLabelValuesForCounter(m.privacyTCF, map[string][]string{
		sourceLabel:  sourceValues,
		versionLabel: tcfVersionsAsString(),
	})

	if !m.metricsDisabled.AdapterGDPRRequestBlocked {
		preloadLabelValuesForCounter(m.adapterGDPRBlockedRequests, map[string][]string{
			adapterLabel: adapterValues,
		})
	}
}

func preloadLabelValuesForCounter(counter *prometheus.CounterVec, labelsWithValues map[string][]string) {
	registerLabelPermutations(labelsWithValues, func(labels prometheus.Labels) {
		counter.With(labels)
	})
}

func preloadLabelValuesForHistogram(histogram *prometheus.HistogramVec, labelsWithValues map[string][]string) {
	registerLabelPermutations(labelsWithValues, func(labels prometheus.Labels) {
		histogram.With(labels)
	})
}

func registerLabelPermutations(labelsWithValues map[string][]string, register func(prometheus.Labels)) {
	if len(labelsWithValues) == 0 {
		return
	}

	keys := make([]string, 0, len(labelsWithValues))
	values := make([][]string, 0, len(labelsWithValues))
	for k, v := range labelsWithValues {
		keys = append(keys, k)
		values = append(values, v)
	}

	labels := prometheus.Labels{}
	registerLabelPermutationsRecursive(0, keys, values, labels, register)
}

func registerLabelPermutationsRecursive(depth int, keys []string, values [][]string, labels prometheus.Labels, register func(prometheus.Labels)) {
	label := keys[depth]
	isLeaf := depth == len(keys)-1

	if isLeaf {
		for _, v := range values[depth] {
			labels[label] = v
			register(cloneLabels(labels))
		}
	} else {
		for _, v := range values[depth] {
			labels[label] = v
			registerLabelPermutationsRecursive(depth+1, keys, values, labels, register)
		}
	}
}

func cloneLabels(labels prometheus.Labels) prometheus.Labels {
	clone := prometheus.Labels{}
	for k, v := range labels {
		clone[k] = v
	}
	return clone
}
