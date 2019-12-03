package prometheusmetrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

func preloadLabelValues(m *Metrics) {
	var (
		actionValues          = actionsAsString()
		adapterValues         = adaptersAsString()
		adapterErrorValues    = adapterErrorsAsString()
		bidTypeValues         = []string{markupDeliveryAdm, markupDeliveryNurl}
		boolValues            = boolValuesAsString()
		cacheResultValues     = cacheResultsAsString()
		cookieValues          = cookieTypesAsString()
		connectionErrorValues = []string{connectionAcceptError, connectionCloseError}
		requestStatusValues   = requestStatusesAsString()
		requestTypeValues     = requestTypesAsString()
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

	preloadLabelValuesForCounter(m.requestsWithoutCookie, map[string][]string{
		requestTypeLabel: requestTypeValues,
	})

	preloadLabelValuesForCounter(m.storedImpressionsCacheResult, map[string][]string{
		cacheResultLabel: cacheResultValues,
	})

	preloadLabelValuesForCounter(m.storedRequestCacheResult, map[string][]string{
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

	preloadLabelValuesForHistogram(m.adapterRequestsTimer, map[string][]string{
		adapterLabel: adapterValues,
	})

	preloadLabelValuesForCounter(m.adapterUserSync, map[string][]string{
		adapterLabel: adapterValues,
		actionLabel:  actionValues,
	})
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
