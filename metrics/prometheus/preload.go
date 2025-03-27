package prometheusmetrics

import (
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prometheus/client_golang/prometheus"
)

func preloadLabelValues(m *Metrics, syncerKeys []string, moduleStageNames map[string][]string) {
	var (
		adapterErrorValues        = enumAsString(metrics.AdapterErrors())
		adapterValues             = enumAsLowerCaseString(openrtb_ext.CoreBidderNames())
		bidTypeValues             = []string{markupDeliveryAdm, markupDeliveryNurl}
		boolValues                = boolValuesAsString()
		cacheResultValues         = enumAsString(metrics.CacheResults())
		connectionErrorValues     = []string{connectionAcceptError, connectionCloseError}
		cookieSyncStatusValues    = enumAsString(metrics.CookieSyncStatuses())
		cookieValues              = enumAsString(metrics.CookieTypes())
		overheadTypes             = enumAsString(metrics.OverheadTypes())
		requestStatusValues       = enumAsString(metrics.RequestStatuses())
		requestTypeValues         = enumAsString(metrics.RequestTypes())
		setUidStatusValues        = enumAsString(metrics.SetUidStatuses())
		sourceValues              = []string{sourceRequest}
		storedDataErrorValues     = enumAsString(metrics.StoredDataErrors())
		storedDataFetchTypeValues = enumAsString(metrics.StoredDataFetchTypes())
		syncerRequestStatusValues = enumAsString(metrics.SyncerRequestStatuses())
		syncerSetsStatusValues    = enumAsString(metrics.SyncerSetUidStatuses())
		tcfVersionValues          = enumAsString(metrics.TCFVersions())
	)

	preloadLabelValuesForCounter(m.connectionsError, map[string][]string{
		connectionErrorLabel: connectionErrorValues,
	})

	preloadLabelValuesForCounter(m.cookieSync, map[string][]string{
		statusLabel: cookieSyncStatusValues,
	})

	preloadLabelValuesForCounter(m.setUid, map[string][]string{
		statusLabel: setUidStatusValues,
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

	preloadLabelValuesForHistogram(m.storedResponsesFetchTimer, map[string][]string{
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

	preloadLabelValuesForCounter(m.storedResponsesErrors, map[string][]string{
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

	preloadLabelValuesForCounter(m.adapterErrors, map[string][]string{
		adapterLabel:      adapterValues,
		adapterErrorLabel: adapterErrorValues,
	})

	preloadLabelValuesForCounter(m.adapterPanics, map[string][]string{
		adapterLabel: adapterValues,
	})

	preloadLabelValuesForCounter(m.adapterBidResponseSecureMarkupError, map[string][]string{
		adapterLabel: adapterValues,
		successLabel: boolValues,
	})

	preloadLabelValuesForCounter(m.adapterBidResponseSecureMarkupWarn, map[string][]string{
		adapterLabel: adapterValues,
		successLabel: boolValues,
	})

	preloadLabelValuesForCounter(m.adapterBidResponseValidationSizeError, map[string][]string{
		adapterLabel: adapterValues,
		successLabel: boolValues,
	})

	preloadLabelValuesForCounter(m.adapterBidResponseValidationSizeWarn, map[string][]string{
		adapterLabel: adapterValues,
		successLabel: boolValues,
	})

	preloadLabelValuesForHistogram(m.adapterPrices, map[string][]string{
		adapterLabel: adapterValues,
	})

	preloadLabelValuesForCounter(m.adapterRequests, map[string][]string{
		adapterLabel: adapterValues,
		cookieLabel:  cookieValues,
		hasBidsLabel: boolValues,
	})

	preloadLabelValuesForCounter(m.adsCertRequests, map[string][]string{
		successLabel: boolValues,
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

	preloadLabelValuesForHistogram(m.overheadTimer, map[string][]string{
		overheadTypeLabel: overheadTypes,
	})

	preloadLabelValuesForCounter(m.syncerRequests, map[string][]string{
		syncerLabel: syncerKeys,
		statusLabel: syncerRequestStatusValues,
	})

	preloadLabelValuesForCounter(m.syncerSets, map[string][]string{
		syncerLabel: syncerKeys,
		statusLabel: syncerSetsStatusValues,
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
		versionLabel: tcfVersionValues,
	})

	if !m.metricsDisabled.AdapterBuyerUIDScrubbed {
		preloadLabelValuesForCounter(m.adapterScrubbedBuyerUIDs, map[string][]string{
			adapterLabel: adapterValues,
		})
	}

	if !m.metricsDisabled.AdapterGDPRRequestBlocked {
		preloadLabelValuesForCounter(m.adapterGDPRBlockedRequests, map[string][]string{
			adapterLabel: adapterValues,
		})
	}

	for module, stageValues := range moduleStageNames {
		preloadLabelValuesForHistogram(m.moduleDuration[module], map[string][]string{
			stageLabel: stageValues,
		})

		preloadLabelValuesForCounter(m.moduleCalls[module], map[string][]string{
			stageLabel: stageValues,
		})

		preloadLabelValuesForCounter(m.moduleFailures[module], map[string][]string{
			stageLabel: stageValues,
		})

		preloadLabelValuesForCounter(m.moduleSuccessNoops[module], map[string][]string{
			stageLabel: stageValues,
		})

		preloadLabelValuesForCounter(m.moduleSuccessUpdates[module], map[string][]string{
			stageLabel: stageValues,
		})

		preloadLabelValuesForCounter(m.moduleSuccessRejects[module], map[string][]string{
			stageLabel: stageValues,
		})

		preloadLabelValuesForCounter(m.moduleExecutionErrors[module], map[string][]string{
			stageLabel: stageValues,
		})

		preloadLabelValuesForCounter(m.moduleTimeouts[module], map[string][]string{
			stageLabel: stageValues,
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
