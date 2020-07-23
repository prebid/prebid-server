package prometheusmetrics

import (
	"fmt"
	"testing"
	"time"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbsmetrics"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

func createMetricsForTesting() *Metrics {
	return NewMetrics(config.PrometheusMetrics{
		Port:      8080,
		Namespace: "prebid",
		Subsystem: "server",
	}, config.DisabledMetrics{})
}

func TestMetricCountGatekeeping(t *testing.T) {
	m := createMetricsForTesting()

	// Gather All Metrics
	metricFamilies, err := m.Registry.Gather()
	assert.NoError(t, err, "gather metics")

	// Summarize By Adapter Cardinality
	// - This requires metrics to be preloaded. We don't preload account metrics, so we can't test those.
	generalCardinalityCount := 0
	adapterCardinalityCount := 0
	for _, metricFamily := range metricFamilies {
		for _, metric := range metricFamily.GetMetric() {
			isPerAdapter := false
			for _, label := range metric.GetLabel() {
				if label.GetName() == adapterLabel {
					isPerAdapter = true
				}
			}

			if isPerAdapter {
				adapterCardinalityCount++
			} else {
				generalCardinalityCount++
			}
		}
	}

	// Calculate Per-Adapter Cardinality
	adapterCount := len(openrtb_ext.BidderList())
	perAdapterCardinalityCount := adapterCardinalityCount / adapterCount

	// Verify General Cardinality
	// - This assertion provides a warning for newly added high-cardinality non-adapter specific metrics. The hardcoded limit
	//   is an arbitrary soft ceiling. Thought should be given as to the value of the new metrics if you find yourself
	//   needing to increase this number.
	assert.True(t, generalCardinalityCount <= 500, "General Cardinality")

	// Verify Per-Adapter Cardinality
	// - This assertion provides a warning for newly added adapter metrics. Threre are 40+ adapters which makes the
	//   cost of new per-adapter metrics rather expensive. Thought should be given when adding new per-adapter metrics.
	assert.True(t, perAdapterCardinalityCount <= 27, "Per-Adapter Cardinality count equals %d \n", perAdapterCardinalityCount)
}

func TestConnectionMetrics(t *testing.T) {
	testCases := []struct {
		description              string
		testCase                 func(m *Metrics)
		expectedOpenedCount      float64
		expectedOpenedErrorCount float64
		expectedClosedCount      float64
		expectedClosedErrorCount float64
	}{
		{
			description: "Open Success",
			testCase: func(m *Metrics) {
				m.RecordConnectionAccept(true)
			},
			expectedOpenedCount:      1,
			expectedOpenedErrorCount: 0,
			expectedClosedCount:      0,
			expectedClosedErrorCount: 0,
		},
		{
			description: "Open Error",
			testCase: func(m *Metrics) {
				m.RecordConnectionAccept(false)
			},
			expectedOpenedCount:      0,
			expectedOpenedErrorCount: 1,
			expectedClosedCount:      0,
			expectedClosedErrorCount: 0,
		},
		{
			description: "Closed Success",
			testCase: func(m *Metrics) {
				m.RecordConnectionClose(true)
			},
			expectedOpenedCount:      0,
			expectedOpenedErrorCount: 0,
			expectedClosedCount:      1,
			expectedClosedErrorCount: 0,
		},
		{
			description: "Closed Error",
			testCase: func(m *Metrics) {
				m.RecordConnectionClose(false)
			},
			expectedOpenedCount:      0,
			expectedOpenedErrorCount: 0,
			expectedClosedCount:      0,
			expectedClosedErrorCount: 1,
		},
	}

	for _, test := range testCases {
		m := createMetricsForTesting()

		test.testCase(m)

		assertCounterValue(t, test.description, "connectionsClosed", m.connectionsClosed,
			test.expectedClosedCount)
		assertCounterValue(t, test.description, "connectionsOpened", m.connectionsOpened,
			test.expectedOpenedCount)
		assertCounterVecValue(t, test.description, "connectionsError[type=accept]", m.connectionsError,
			test.expectedOpenedErrorCount, prometheus.Labels{
				connectionErrorLabel: connectionAcceptError,
			})
		assertCounterVecValue(t, test.description, "connectionsError[type=close]", m.connectionsError,
			test.expectedClosedErrorCount, prometheus.Labels{
				connectionErrorLabel: connectionCloseError,
			})
	}
}

func TestRequestMetric(t *testing.T) {
	m := createMetricsForTesting()
	requestType := pbsmetrics.ReqTypeORTB2Web
	requestStatus := pbsmetrics.RequestStatusBlacklisted

	m.RecordRequest(pbsmetrics.Labels{
		RType:         requestType,
		RequestStatus: requestStatus,
	})

	expectedCount := float64(1)
	assertCounterVecValue(t, "", "requests", m.requests,
		expectedCount,
		prometheus.Labels{
			requestTypeLabel:   string(requestType),
			requestStatusLabel: string(requestStatus),
		})
}

func TestRequestMetricWithoutCookie(t *testing.T) {
	requestType := pbsmetrics.ReqTypeORTB2Web
	performTest := func(m *Metrics, cookieFlag pbsmetrics.CookieFlag) {
		m.RecordRequest(pbsmetrics.Labels{
			RType:         requestType,
			RequestStatus: pbsmetrics.RequestStatusBlacklisted,
			CookieFlag:    cookieFlag,
		})
	}

	testCases := []struct {
		description   string
		testCase      func(m *Metrics)
		cookieFlag    pbsmetrics.CookieFlag
		expectedCount float64
	}{
		{
			description: "Yes",
			testCase: func(m *Metrics) {
				performTest(m, pbsmetrics.CookieFlagYes)
			},
			expectedCount: 0,
		},
		{
			description: "No",
			testCase: func(m *Metrics) {
				performTest(m, pbsmetrics.CookieFlagNo)
			},
			expectedCount: 1,
		},
		{
			description: "Unknown",
			testCase: func(m *Metrics) {
				performTest(m, pbsmetrics.CookieFlagUnknown)
			},
			expectedCount: 0,
		},
	}

	for _, test := range testCases {
		m := createMetricsForTesting()

		test.testCase(m)

		assertCounterVecValue(t, test.description, "requestsWithoutCookie", m.requestsWithoutCookie,
			test.expectedCount,
			prometheus.Labels{
				requestTypeLabel: string(requestType),
			})
	}
}

func TestAccountMetric(t *testing.T) {
	knownPubID := "knownPublisher"
	performTest := func(m *Metrics, pubID string) {
		m.RecordRequest(pbsmetrics.Labels{
			RType:         pbsmetrics.ReqTypeORTB2Web,
			RequestStatus: pbsmetrics.RequestStatusBlacklisted,
			PubID:         pubID,
		})
	}

	testCases := []struct {
		description   string
		testCase      func(m *Metrics)
		expectedCount float64
	}{
		{
			description: "Known",
			testCase: func(m *Metrics) {
				performTest(m, knownPubID)
			},
			expectedCount: 1,
		},
		{
			description: "Unknown",
			testCase: func(m *Metrics) {
				performTest(m, pbsmetrics.PublisherUnknown)
			},
			expectedCount: 0,
		},
	}

	for _, test := range testCases {
		m := createMetricsForTesting()

		test.testCase(m)

		assertCounterVecValue(t, test.description, "accountRequests", m.accountRequests,
			test.expectedCount,
			prometheus.Labels{
				accountLabel: knownPubID,
			})
	}
}

func TestImpressionsMetric(t *testing.T) {
	performTest := func(m *Metrics, isBanner, isVideo, isAudio, isNative bool) {
		m.RecordImps(pbsmetrics.ImpLabels{
			BannerImps: isBanner,
			VideoImps:  isVideo,
			AudioImps:  isAudio,
			NativeImps: isNative,
		})
	}

	testCases := []struct {
		description         string
		testCase            func(m *Metrics)
		expectedBannerCount float64
		expectedVideoCount  float64
		expectedAudioCount  float64
		expectedNativeCount float64
	}{
		{
			description: "Banner Only",
			testCase: func(m *Metrics) {
				performTest(m, true, false, false, false)
			},
			expectedBannerCount: 1,
			expectedVideoCount:  0,
			expectedAudioCount:  0,
			expectedNativeCount: 0,
		},
		{
			description: "Video Only",
			testCase: func(m *Metrics) {
				performTest(m, false, true, false, false)
			},
			expectedBannerCount: 0,
			expectedVideoCount:  1,
			expectedAudioCount:  0,
			expectedNativeCount: 0,
		},
		{
			description: "Audio Only",
			testCase: func(m *Metrics) {
				performTest(m, false, false, true, false)
			},
			expectedBannerCount: 0,
			expectedVideoCount:  0,
			expectedAudioCount:  1,
			expectedNativeCount: 0,
		},
		{
			description: "Native Only",
			testCase: func(m *Metrics) {
				performTest(m, false, false, false, true)
			},
			expectedBannerCount: 0,
			expectedVideoCount:  0,
			expectedAudioCount:  0,
			expectedNativeCount: 1,
		},
		{
			description: "Multiple Types",
			testCase: func(m *Metrics) {
				performTest(m, true, false, false, true)
			},
			expectedBannerCount: 1,
			expectedVideoCount:  0,
			expectedAudioCount:  0,
			expectedNativeCount: 1,
		},
	}

	for _, test := range testCases {
		m := createMetricsForTesting()

		test.testCase(m)

		var bannerCount float64
		var videoCount float64
		var audioCount float64
		var nativeCount float64
		processMetrics(m.impressions, func(m dto.Metric) {
			value := m.GetCounter().GetValue()
			for _, label := range m.GetLabel() {
				if label.GetValue() == "true" {
					switch label.GetName() {
					case isBannerLabel:
						bannerCount += value
					case isVideoLabel:
						videoCount += value
					case isAudioLabel:
						audioCount += value
					case isNativeLabel:
						nativeCount += value
					}
				}
			}
		})
		assert.Equal(t, test.expectedBannerCount, bannerCount, test.description+":banner")
		assert.Equal(t, test.expectedVideoCount, videoCount, test.description+":video")
		assert.Equal(t, test.expectedAudioCount, audioCount, test.description+":audio")
		assert.Equal(t, test.expectedNativeCount, nativeCount, test.description+":native")
	}
}

func TestLegacyImpressionsMetric(t *testing.T) {
	m := createMetricsForTesting()

	m.RecordLegacyImps(pbsmetrics.Labels{}, 42)

	expectedCount := float64(42)
	assertCounterValue(t, "", "impressionsLegacy", m.impressionsLegacy,
		expectedCount)
}

func TestRequestTimeMetric(t *testing.T) {
	requestType := pbsmetrics.ReqTypeORTB2Web
	performTest := func(m *Metrics, requestStatus pbsmetrics.RequestStatus, timeInMs float64) {
		m.RecordRequestTime(pbsmetrics.Labels{
			RType:         requestType,
			RequestStatus: requestStatus,
		}, time.Duration(timeInMs)*time.Millisecond)
	}

	testCases := []struct {
		description   string
		testCase      func(m *Metrics)
		expectedCount uint64
		expectedSum   float64
	}{
		{
			description: "Success",
			testCase: func(m *Metrics) {
				performTest(m, pbsmetrics.RequestStatusOK, 500)
			},
			expectedCount: 1,
			expectedSum:   0.5,
		},
		{
			description: "Error",
			testCase: func(m *Metrics) {
				performTest(m, pbsmetrics.RequestStatusErr, 500)
			},
			expectedCount: 0,
			expectedSum:   0,
		},
	}

	for _, test := range testCases {
		m := createMetricsForTesting()

		test.testCase(m)

		result := getHistogramFromHistogramVec(m.requestsTimer, requestTypeLabel, string(requestType))
		assertHistogram(t, test.description, result, test.expectedCount, test.expectedSum)
	}
}

func TestAdapterBidReceivedMetric(t *testing.T) {
	adapterName := "anyName"
	performTest := func(m *Metrics, hasAdm bool) {
		labels := pbsmetrics.AdapterLabels{
			Adapter: openrtb_ext.BidderName(adapterName),
		}
		bidType := openrtb_ext.BidTypeBanner
		m.RecordAdapterBidReceived(labels, bidType, hasAdm)
	}

	testCases := []struct {
		description       string
		testCase          func(m *Metrics)
		expectedAdmCount  float64
		expectedNurlCount float64
	}{
		{
			description: "AdM",
			testCase: func(m *Metrics) {
				performTest(m, true)
			},
			expectedAdmCount:  1,
			expectedNurlCount: 0,
		},
		{
			description: "Nurl",
			testCase: func(m *Metrics) {
				performTest(m, false)
			},
			expectedAdmCount:  0,
			expectedNurlCount: 1,
		},
	}

	for _, test := range testCases {
		m := createMetricsForTesting()

		test.testCase(m)

		assertCounterVecValue(t, test.description, "adapterBids[adm]", m.adapterBids,
			test.expectedAdmCount,
			prometheus.Labels{
				adapterLabel:        adapterName,
				markupDeliveryLabel: markupDeliveryAdm,
			})
		assertCounterVecValue(t, test.description, "adapterBids[nurl]", m.adapterBids,
			test.expectedNurlCount,
			prometheus.Labels{
				adapterLabel:        adapterName,
				markupDeliveryLabel: markupDeliveryNurl,
			})
	}
}

func TestRecordAdapterPriceMetric(t *testing.T) {
	m := createMetricsForTesting()
	adapterName := "anyName"
	cpm := float64(42)

	m.RecordAdapterPrice(pbsmetrics.AdapterLabels{
		Adapter: openrtb_ext.BidderName(adapterName),
	}, cpm)

	expectedCount := uint64(1)
	expectedSum := cpm
	result := getHistogramFromHistogramVec(m.adapterPrices, adapterLabel, adapterName)
	assertHistogram(t, "adapterPrices", result, expectedCount, expectedSum)
}

func TestAdapterRequestMetrics(t *testing.T) {
	adapterName := "anyName"
	performTest := func(m *Metrics, cookieFlag pbsmetrics.CookieFlag, adapterBids pbsmetrics.AdapterBid) {
		labels := pbsmetrics.AdapterLabels{
			Adapter:     openrtb_ext.BidderName(adapterName),
			CookieFlag:  cookieFlag,
			AdapterBids: adapterBids,
		}
		m.RecordAdapterRequest(labels)
	}

	testCases := []struct {
		description                string
		testCase                   func(m *Metrics)
		expectedCount              float64
		expectedCookieNoCount      float64
		expectedCookieYesCount     float64
		expectedCookieUnknownCount float64
		expectedHasBidsCount       float64
	}{
		{
			description: "No Cookie & No Bids",
			testCase: func(m *Metrics) {
				performTest(m, pbsmetrics.CookieFlagNo, pbsmetrics.AdapterBidNone)
			},
			expectedCount:              1,
			expectedCookieNoCount:      1,
			expectedCookieYesCount:     0,
			expectedCookieUnknownCount: 0,
			expectedHasBidsCount:       0,
		},
		{
			description: "Unknown Cookie & No Bids",
			testCase: func(m *Metrics) {
				performTest(m, pbsmetrics.CookieFlagUnknown, pbsmetrics.AdapterBidNone)
			},
			expectedCount:              1,
			expectedCookieNoCount:      0,
			expectedCookieYesCount:     0,
			expectedCookieUnknownCount: 1,
			expectedHasBidsCount:       0,
		},
		{
			description: "Has Cookie & No Bids",
			testCase: func(m *Metrics) {
				performTest(m, pbsmetrics.CookieFlagYes, pbsmetrics.AdapterBidNone)
			},
			expectedCount:              1,
			expectedCookieNoCount:      0,
			expectedCookieYesCount:     1,
			expectedCookieUnknownCount: 0,
			expectedHasBidsCount:       0,
		},
		{
			description: "No Cookie & Bids Present",
			testCase: func(m *Metrics) {
				performTest(m, pbsmetrics.CookieFlagNo, pbsmetrics.AdapterBidPresent)
			},
			expectedCount:              1,
			expectedCookieNoCount:      1,
			expectedCookieYesCount:     0,
			expectedCookieUnknownCount: 0,
			expectedHasBidsCount:       1,
		},
		{
			description: "Unknown Cookie & Bids Present",
			testCase: func(m *Metrics) {
				performTest(m, pbsmetrics.CookieFlagUnknown, pbsmetrics.AdapterBidPresent)
			},
			expectedCount:              1,
			expectedCookieNoCount:      0,
			expectedCookieYesCount:     0,
			expectedCookieUnknownCount: 1,
			expectedHasBidsCount:       1,
		},
		{
			description: "Has Cookie & Bids Present",
			testCase: func(m *Metrics) {
				performTest(m, pbsmetrics.CookieFlagYes, pbsmetrics.AdapterBidPresent)
			},
			expectedCount:              1,
			expectedCookieNoCount:      0,
			expectedCookieYesCount:     1,
			expectedCookieUnknownCount: 0,
			expectedHasBidsCount:       1,
		},
	}

	for _, test := range testCases {
		m := createMetricsForTesting()

		test.testCase(m)

		var totalCount float64
		var totalCookieNoCount float64
		var totalCookieYesCount float64
		var totalCookieUnknownCount float64
		var totalHasBidsCount float64
		processMetrics(m.adapterRequests, func(m dto.Metric) {
			isMetricForAdapter := false
			for _, label := range m.GetLabel() {
				if label.GetName() == adapterLabel && label.GetValue() == adapterName {
					isMetricForAdapter = true
				}
			}

			if isMetricForAdapter {
				value := m.GetCounter().GetValue()
				totalCount += value
				for _, label := range m.GetLabel() {

					if label.GetName() == hasBidsLabel && label.GetValue() == "true" {
						totalHasBidsCount += value
					}

					if label.GetName() == cookieLabel {
						switch label.GetValue() {
						case string(pbsmetrics.CookieFlagNo):
							totalCookieNoCount += value
						case string(pbsmetrics.CookieFlagYes):
							totalCookieYesCount += value
						case string(pbsmetrics.CookieFlagUnknown):
							totalCookieUnknownCount += value
						}
					}
				}
			}
		})
		assert.Equal(t, test.expectedCount, totalCount, test.description+":total")
		assert.Equal(t, test.expectedCookieNoCount, totalCookieNoCount, test.description+":cookie=no")
		assert.Equal(t, test.expectedCookieYesCount, totalCookieYesCount, test.description+":cookie=yes")
		assert.Equal(t, test.expectedCookieUnknownCount, totalCookieUnknownCount, test.description+":cookie=unknown")
		assert.Equal(t, test.expectedHasBidsCount, totalHasBidsCount, test.description+":hasBids")
	}
}

func TestAdapterRequestErrorMetrics(t *testing.T) {
	adapterName := "anyName"
	performTest := func(m *Metrics, adapterErrors map[pbsmetrics.AdapterError]struct{}) {
		labels := pbsmetrics.AdapterLabels{
			Adapter:       openrtb_ext.BidderName(adapterName),
			AdapterErrors: adapterErrors,
			CookieFlag:    pbsmetrics.CookieFlagUnknown,
			AdapterBids:   pbsmetrics.AdapterBidPresent,
		}
		m.RecordAdapterRequest(labels)
	}

	testCases := []struct {
		description                 string
		testCase                    func(m *Metrics)
		expectedErrorsCount         float64
		expectedBadInputErrorsCount float64
	}{
		{
			description: "No Errors",
			testCase: func(m *Metrics) {
				performTest(m, nil)
			},
			expectedErrorsCount:         0,
			expectedBadInputErrorsCount: 0,
		},
		{
			description: "Bad Input Error",
			testCase: func(m *Metrics) {
				performTest(m, map[pbsmetrics.AdapterError]struct{}{
					pbsmetrics.AdapterErrorBadInput: {},
				})
			},
			expectedErrorsCount:         1,
			expectedBadInputErrorsCount: 1,
		},
		{
			description: "Other Error",
			testCase: func(m *Metrics) {
				performTest(m, map[pbsmetrics.AdapterError]struct{}{
					pbsmetrics.AdapterErrorBadServerResponse: {},
				})
			},
			expectedErrorsCount:         1,
			expectedBadInputErrorsCount: 0,
		},
	}

	for _, test := range testCases {
		m := createMetricsForTesting()

		test.testCase(m)

		var errorsCount float64
		var badInputErrorsCount float64
		processMetrics(m.adapterErrors, func(m dto.Metric) {
			isMetricForAdapter := false
			for _, label := range m.GetLabel() {
				if label.GetName() == adapterLabel && label.GetValue() == adapterName {
					isMetricForAdapter = true
				}
			}

			if isMetricForAdapter {
				value := m.GetCounter().GetValue()
				errorsCount += value
				for _, label := range m.GetLabel() {
					if label.GetName() == adapterErrorLabel && label.GetValue() == string(pbsmetrics.AdapterErrorBadInput) {
						badInputErrorsCount += value
					}
				}
			}
		})
		assert.Equal(t, test.expectedErrorsCount, errorsCount, test.description+":errors")
		assert.Equal(t, test.expectedBadInputErrorsCount, badInputErrorsCount, test.description+":badInputErrors")
	}
}

func TestAdapterTimeMetric(t *testing.T) {
	adapterName := "anyName"
	performTest := func(m *Metrics, timeInMs float64, adapterErrors map[pbsmetrics.AdapterError]struct{}) {
		m.RecordAdapterTime(pbsmetrics.AdapterLabels{
			Adapter:       openrtb_ext.BidderName(adapterName),
			AdapterErrors: adapterErrors,
		}, time.Duration(timeInMs)*time.Millisecond)
	}

	testCases := []struct {
		description   string
		testCase      func(m *Metrics)
		expectedCount uint64
		expectedSum   float64
	}{
		{
			description: "Success",
			testCase: func(m *Metrics) {
				performTest(m, 500, map[pbsmetrics.AdapterError]struct{}{})
			},
			expectedCount: 1,
			expectedSum:   0.5,
		},
		{
			description: "Error",
			testCase: func(m *Metrics) {
				performTest(m, 500, map[pbsmetrics.AdapterError]struct{}{
					pbsmetrics.AdapterErrorTimeout: {},
				})
			},
			expectedCount: 0,
			expectedSum:   0,
		},
	}

	for _, test := range testCases {
		m := createMetricsForTesting()

		test.testCase(m)

		result := getHistogramFromHistogramVec(m.adapterRequestsTimer, adapterLabel, adapterName)
		assertHistogram(t, test.description, result, test.expectedCount, test.expectedSum)
	}
}

func TestAdapterCookieSyncMetric(t *testing.T) {
	m := createMetricsForTesting()
	adapterName := "anyName"
	privacyBlocked := true

	m.RecordAdapterCookieSync(openrtb_ext.BidderName(adapterName), privacyBlocked)

	expectedCount := float64(1)
	assertCounterVecValue(t, "", "adapterCookieSync", m.adapterCookieSync,
		expectedCount,
		prometheus.Labels{
			adapterLabel:        adapterName,
			privacyBlockedLabel: "true",
		})
}

func TestUserIDSetMetric(t *testing.T) {
	m := createMetricsForTesting()
	adapterName := "anyName"
	action := pbsmetrics.RequestActionSet

	m.RecordUserIDSet(pbsmetrics.UserLabels{
		Bidder: openrtb_ext.BidderName(adapterName),
		Action: action,
	})

	expectedCount := float64(1)
	assertCounterVecValue(t, "", "adapterUserSync", m.adapterUserSync,
		expectedCount,
		prometheus.Labels{
			adapterLabel: adapterName,
			actionLabel:  string(action),
		})
}

func TestUserIDSetMetricWhenBidderEmpty(t *testing.T) {
	m := createMetricsForTesting()
	action := pbsmetrics.RequestActionErr

	m.RecordUserIDSet(pbsmetrics.UserLabels{
		Bidder: openrtb_ext.BidderName(""),
		Action: action,
	})

	expectedTotalCount := float64(0)
	actualTotalCount := float64(0)
	processMetrics(m.adapterUserSync, func(m dto.Metric) {
		actualTotalCount += m.GetCounter().GetValue()
	})
	assert.Equal(t, expectedTotalCount, actualTotalCount, "total count")
}

func TestAdapterPanicMetric(t *testing.T) {
	m := createMetricsForTesting()
	adapterName := "anyName"

	m.RecordAdapterPanic(pbsmetrics.AdapterLabels{
		Adapter: openrtb_ext.BidderName(adapterName),
	})

	expectedCount := float64(1)
	assertCounterVecValue(t, "", "adapterPanics", m.adapterPanics,
		expectedCount,
		prometheus.Labels{
			adapterLabel: adapterName,
		})
}

func TestStoredReqCacheResultMetric(t *testing.T) {
	m := createMetricsForTesting()

	hitCount := 42
	missCount := 108
	m.RecordStoredReqCacheResult(pbsmetrics.CacheHit, hitCount)
	m.RecordStoredReqCacheResult(pbsmetrics.CacheMiss, missCount)

	assertCounterVecValue(t, "", "storedRequestCacheResult:hit", m.storedRequestCacheResult,
		float64(hitCount),
		prometheus.Labels{
			cacheResultLabel: string(pbsmetrics.CacheHit),
		})
	assertCounterVecValue(t, "", "storedRequestCacheResult:miss", m.storedRequestCacheResult,
		float64(missCount),
		prometheus.Labels{
			cacheResultLabel: string(pbsmetrics.CacheMiss),
		})
}

func TestStoredImpCacheResultMetric(t *testing.T) {
	m := createMetricsForTesting()

	hitCount := 42
	missCount := 108
	m.RecordStoredImpCacheResult(pbsmetrics.CacheHit, hitCount)
	m.RecordStoredImpCacheResult(pbsmetrics.CacheMiss, missCount)

	assertCounterVecValue(t, "", "storedImpressionsCacheResult:hit", m.storedImpressionsCacheResult,
		float64(hitCount),
		prometheus.Labels{
			cacheResultLabel: string(pbsmetrics.CacheHit),
		})
	assertCounterVecValue(t, "", "storedImpressionsCacheResult:miss", m.storedImpressionsCacheResult,
		float64(missCount),
		prometheus.Labels{
			cacheResultLabel: string(pbsmetrics.CacheMiss),
		})
}

func TestCookieMetric(t *testing.T) {
	m := createMetricsForTesting()

	m.RecordCookieSync()

	expectedCount := float64(1)
	assertCounterValue(t, "", "cookieSync", m.cookieSync,
		expectedCount)
}

func TestPrebidCacheRequestTimeMetric(t *testing.T) {
	m := createMetricsForTesting()

	m.RecordPrebidCacheRequestTime(true, time.Duration(100)*time.Millisecond)
	m.RecordPrebidCacheRequestTime(false, time.Duration(200)*time.Millisecond)

	successExpectedCount := uint64(1)
	successExpectedSum := float64(0.1)
	successResult := getHistogramFromHistogramVec(m.prebidCacheWriteTimer, successLabel, "true")
	assertHistogram(t, "Success", successResult, successExpectedCount, successExpectedSum)

	errorExpectedCount := uint64(1)
	errorExpectedSum := float64(0.2)
	errorResult := getHistogramFromHistogramVec(m.prebidCacheWriteTimer, successLabel, "false")
	assertHistogram(t, "Error", errorResult, errorExpectedCount, errorExpectedSum)
}

func TestMetricAccumulationSpotCheck(t *testing.T) {
	m := createMetricsForTesting()

	m.RecordLegacyImps(pbsmetrics.Labels{}, 1)
	m.RecordLegacyImps(pbsmetrics.Labels{}, 2)
	m.RecordLegacyImps(pbsmetrics.Labels{}, 3)

	expectedValue := float64(1 + 2 + 3)
	assertCounterValue(t, "", "impressionsLegacy", m.impressionsLegacy,
		expectedValue)
}

func TestRecordRequestQueueTimeMetric(t *testing.T) {
	performTest := func(m *Metrics, requestStatus bool, requestType pbsmetrics.RequestType, timeInSec float64) {
		m.RecordRequestQueueTime(requestStatus, requestType, time.Duration(timeInSec*float64(time.Second)))
	}

	testCases := []struct {
		description   string
		status        string
		testCase      func(m *Metrics)
		expectedCount uint64
		expectedSum   float64
	}{
		{
			description: "Success",
			status:      requestSuccessLabel,
			testCase: func(m *Metrics) {
				performTest(m, true, pbsmetrics.ReqTypeVideo, 2)
			},
			expectedCount: 1,
			expectedSum:   2,
		},
		{
			description: "TimeoutError",
			status:      requestRejectLabel,
			testCase: func(m *Metrics) {
				performTest(m, false, pbsmetrics.ReqTypeVideo, 50)
			},
			expectedCount: 1,
			expectedSum:   50,
		},
	}

	m := createMetricsForTesting()
	for _, test := range testCases {

		test.testCase(m)

		result := getHistogramFromHistogramVecByTwoKeys(m.requestsQueueTimer, requestTypeLabel, "video", requestStatusLabel, test.status)
		assertHistogram(t, test.description, result, test.expectedCount, test.expectedSum)
	}
}

func TestTimeoutNotifications(t *testing.T) {
	m := createMetricsForTesting()

	m.RecordTimeoutNotice(true)
	m.RecordTimeoutNotice(true)
	m.RecordTimeoutNotice(false)

	assertCounterVecValue(t, "", "timeout_notifications:ok", m.timeoutNotifications,
		float64(2),
		prometheus.Labels{
			successLabel: requestSuccessful,
		})

	assertCounterVecValue(t, "", "timeout_notifications:fail", m.timeoutNotifications,
		float64(1),
		prometheus.Labels{
			successLabel: requestFailed,
		})

}

func TestRecordDNSTime(t *testing.T) {
	type testIn struct {
		dnsLookupDuration time.Duration
	}
	type testOut struct {
		expDuration float64
		expCount    uint64
	}
	testCases := []struct {
		description string
		in          testIn
		out         testOut
	}{
		{
			description: "Five second DNS lookup time",
			in: testIn{
				dnsLookupDuration: time.Second * 5,
			},
			out: testOut{
				expDuration: 5,
				expCount:    1,
			},
		},
		{
			description: "Zero DNS lookup time",
			in:          testIn{},
			out: testOut{
				expDuration: 0,
				expCount:    1,
			},
		},
	}
	for i, test := range testCases {
		pm := createMetricsForTesting()
		pm.RecordDNSTime(test.in.dnsLookupDuration)

		m := dto.Metric{}
		pm.dnsLookupTimer.Write(&m)
		histogram := *m.GetHistogram()

		assert.Equal(t, test.out.expCount, histogram.GetSampleCount(), "[%d] Incorrect number of histogram entries. Desc: %s\n", i, test.description)
		assert.Equal(t, test.out.expDuration, histogram.GetSampleSum(), "[%d] Incorrect number of histogram cumulative values. Desc: %s\n", i, test.description)
	}
}

func TestRecordAdapterConnections(t *testing.T) {

	type testIn struct {
		adapterName   openrtb_ext.BidderName
		connWasReused bool
		connWait      time.Duration
	}

	type testOut struct {
		expectedConnReusedCount  int64
		expectedConnCreatedCount int64
		expectedConnWaitCount    uint64
		expectedConnWaitTime     float64
	}

	testCases := []struct {
		description string
		in          testIn
		out         testOut
	}{
		{
			description: "[1] Successful, new connection created, was idle, has connection wait",
			in: testIn{
				adapterName:   openrtb_ext.BidderAppnexus,
				connWasReused: false,
				connWait:      time.Second * 5,
			},
			out: testOut{
				expectedConnReusedCount:  0,
				expectedConnCreatedCount: 1,
				expectedConnWaitCount:    1,
				expectedConnWaitTime:     5,
			},
		},
		{
			description: "[2] Successful, new connection created, not idle, has connection wait",
			in: testIn{
				adapterName:   openrtb_ext.BidderAppnexus,
				connWasReused: false,
				connWait:      time.Second * 4,
			},
			out: testOut{
				expectedConnReusedCount:  0,
				expectedConnCreatedCount: 1,
				expectedConnWaitCount:    1,
				expectedConnWaitTime:     4,
			},
		},
		{
			description: "[3] Successful, was reused, was idle, no connection wait",
			in: testIn{
				adapterName:   openrtb_ext.BidderAppnexus,
				connWasReused: true,
			},
			out: testOut{
				expectedConnReusedCount:  1,
				expectedConnCreatedCount: 0,
				expectedConnWaitCount:    1,
				expectedConnWaitTime:     0,
			},
		},
		{
			description: "[4] Successful, was reused, not idle, has connection wait",
			in: testIn{
				adapterName:   openrtb_ext.BidderAppnexus,
				connWasReused: true,
				connWait:      time.Second * 5,
			},
			out: testOut{
				expectedConnReusedCount:  1,
				expectedConnCreatedCount: 0,
				expectedConnWaitCount:    1,
				expectedConnWaitTime:     5,
			},
		},
	}

	for i, test := range testCases {
		m := createMetricsForTesting()
		assertDesciptions := []string{
			fmt.Sprintf("[%d] Metric: adapterReusedConnections; Desc: %s", i+1, test.description),
			fmt.Sprintf("[%d] Metric: adapterCreatedConnections; Desc: %s", i+1, test.description),
			fmt.Sprintf("[%d] Metric: adapterWaitConnectionCount; Desc: %s", i+1, test.description),
			fmt.Sprintf("[%d] Metric: adapterWaitConnectionTime; Desc: %s", i+1, test.description),
		}

		m.RecordAdapterConnections(test.in.adapterName, test.in.connWasReused, test.in.connWait)

		// Assert number of reused connections
		assertCounterVecValue(t,
			assertDesciptions[0],
			"adapter_connection_reused",
			m.adapterReusedConnections,
			float64(test.out.expectedConnReusedCount),
			prometheus.Labels{adapterLabel: string(test.in.adapterName)})

		// Assert number of new created connections
		assertCounterVecValue(t,
			assertDesciptions[1],
			"adapter_connection_created",
			m.adapterCreatedConnections,
			float64(test.out.expectedConnCreatedCount),
			prometheus.Labels{adapterLabel: string(test.in.adapterName)})

		// Assert connection wait time
		histogram := getHistogramFromHistogramVec(m.adapterConnectionWaitTime, adapterLabel, string(test.in.adapterName))
		assert.Equal(t, test.out.expectedConnWaitCount, histogram.GetSampleCount(), assertDesciptions[2])
		assert.Equal(t, test.out.expectedConnWaitTime, histogram.GetSampleSum(), assertDesciptions[3])
	}
}

func TestDisableAdapterConnections(t *testing.T) {
	prometheusMetrics := NewMetrics(config.PrometheusMetrics{
		Port:      8080,
		Namespace: "prebid",
		Subsystem: "server",
	}, config.DisabledMetrics{AdapterConnectionMetrics: true})

	// Assert counter vector was not initialized
	assert.Nil(t, prometheusMetrics.adapterReusedConnections, "Counter Vector adapterReusedConnections should be nil")
	assert.Nil(t, prometheusMetrics.adapterCreatedConnections, "Counter Vector adapterCreatedConnections should be nil")
	assert.Nil(t, prometheusMetrics.adapterConnectionWaitTime, "Counter Vector adapterConnectionWaitTime should be nil")
}

func TestRecordRequestPrivacy(t *testing.T) {
	m := createMetricsForTesting()

	// CCPA
	m.RecordRequestPrivacy(pbsmetrics.PrivacyLabels{
		CCPAEnforced: true,
		CCPAProvided: true,
	})
	m.RecordRequestPrivacy(pbsmetrics.PrivacyLabels{
		CCPAEnforced: true,
		CCPAProvided: false,
	})
	m.RecordRequestPrivacy(pbsmetrics.PrivacyLabels{
		CCPAEnforced: false,
		CCPAProvided: true,
	})

	// COPPA
	m.RecordRequestPrivacy(pbsmetrics.PrivacyLabels{
		COPPAEnforced: true,
	})

	// LMT
	m.RecordRequestPrivacy(pbsmetrics.PrivacyLabels{
		LMTEnforced: true,
	})

	// GDPR
	m.RecordRequestPrivacy(pbsmetrics.PrivacyLabels{
		GDPREnforced:   true,
		GDPRTCFVersion: pbsmetrics.TCFVersionErr,
	})
	m.RecordRequestPrivacy(pbsmetrics.PrivacyLabels{
		GDPREnforced:   true,
		GDPRTCFVersion: pbsmetrics.TCFVersionV1,
	})
	m.RecordRequestPrivacy(pbsmetrics.PrivacyLabels{
		GDPREnforced:   true,
		GDPRTCFVersion: pbsmetrics.TCFVersionV2,
	})
	m.RecordRequestPrivacy(pbsmetrics.PrivacyLabels{
		GDPREnforced:   true,
		GDPRTCFVersion: pbsmetrics.TCFVersionV1,
	})

	assertCounterVecValue(t, "", "privacy_ccpa", m.privacyCCPA,
		float64(1),
		prometheus.Labels{
			sourceLabel: sourceRequest,
			optOutLabel: "true",
		})

	assertCounterVecValue(t, "", "privacy_ccpa", m.privacyCCPA,
		float64(1),
		prometheus.Labels{
			sourceLabel: sourceRequest,
			optOutLabel: "false",
		})

	assertCounterVecValue(t, "", "privacy_coppa", m.privacyCOPPA,
		float64(1),
		prometheus.Labels{
			sourceLabel: sourceRequest,
		})

	assertCounterVecValue(t, "", "privacy_lmt", m.privacyLMT,
		float64(1),
		prometheus.Labels{
			sourceLabel: sourceRequest,
		})

	assertCounterVecValue(t, "", "privacy_tcf:err", m.privacyTCF,
		float64(1),
		prometheus.Labels{
			sourceLabel:  sourceRequest,
			versionLabel: "err",
		})

	assertCounterVecValue(t, "", "privacy_tcf:v1", m.privacyTCF,
		float64(2),
		prometheus.Labels{
			sourceLabel:  sourceRequest,
			versionLabel: "v1",
		})

	assertCounterVecValue(t, "", "privacy_tcf:v2", m.privacyTCF,
		float64(1),
		prometheus.Labels{
			sourceLabel:  sourceRequest,
			versionLabel: "v2",
		})
}

func assertCounterValue(t *testing.T, description, name string, counter prometheus.Counter, expected float64) {
	m := dto.Metric{}
	counter.Write(&m)
	actual := *m.GetCounter().Value

	assert.Equal(t, expected, actual, description)
}

func assertCounterVecValue(t *testing.T, description, name string, counterVec *prometheus.CounterVec, expected float64, labels prometheus.Labels) {
	counter := counterVec.With(labels)
	assertCounterValue(t, description, name, counter, expected)
}

func getHistogramFromHistogramVec(histogram *prometheus.HistogramVec, labelKey, labelValue string) dto.Histogram {
	var result dto.Histogram
	processMetrics(histogram, func(m dto.Metric) {
		for _, label := range m.GetLabel() {
			if label.GetName() == labelKey && label.GetValue() == labelValue {
				result = *m.GetHistogram()
			}
		}
	})
	return result
}

func getHistogramFromHistogramVecByTwoKeys(histogram *prometheus.HistogramVec, label1Key, label1Value, label2Key, label2Value string) dto.Histogram {
	var result dto.Histogram
	processMetrics(histogram, func(m dto.Metric) {
		for ind, label := range m.GetLabel() {
			if label.GetName() == label1Key && label.GetValue() == label1Value {
				valInd := ind
				if ind == 1 {
					valInd = 0
				}
				if m.Label[valInd].GetName() == label2Key && m.Label[valInd].GetValue() == label2Value {
					result = *m.GetHistogram()
				}
			}
		}
	})
	return result
}

func processMetrics(collector prometheus.Collector, handler func(m dto.Metric)) {
	collectorChan := make(chan prometheus.Metric)
	go func() {
		collector.Collect(collectorChan)
		close(collectorChan)
	}()

	for metric := range collectorChan {
		dtoMetric := dto.Metric{}
		metric.Write(&dtoMetric)
		handler(dtoMetric)
	}
}

func assertHistogram(t *testing.T, name string, histogram dto.Histogram, expectedCount uint64, expectedSum float64) {
	assert.Equal(t, expectedCount, histogram.GetSampleCount(), name+":count")
	assert.Equal(t, expectedSum, histogram.GetSampleSum(), name+":sum")
}
