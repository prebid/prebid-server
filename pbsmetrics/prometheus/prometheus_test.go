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
	})
}

// test accumulation

func TestConnectionMetrics(t *testing.T) {
	testCases := []struct {
		description         string
		testCase            func(m *Metrics)
		expectedOpened      float64
		expectedOpenedError float64
		expectedClosed      float64
		expectedClosedError float64
	}{
		{
			description: "Open Success",
			testCase: func(m *Metrics) {
				m.RecordConnectionAccept(true)
			},
			expectedOpened:      1,
			expectedOpenedError: 0,
			expectedClosed:      0,
			expectedClosedError: 0,
		},
		{
			description: "Open Error",
			testCase: func(m *Metrics) {
				m.RecordConnectionAccept(false)
			},
			expectedOpened:      0,
			expectedOpenedError: 1,
			expectedClosed:      0,
			expectedClosedError: 0,
		},
		{
			description: "Closed Success",
			testCase: func(m *Metrics) {
				m.RecordConnectionClose(true)
			},
			expectedOpened:      0,
			expectedOpenedError: 0,
			expectedClosed:      1,
			expectedClosedError: 0,
		},
		{
			description: "Closed Error",
			testCase: func(m *Metrics) {
				m.RecordConnectionClose(false)
			},
			expectedOpened:      0,
			expectedOpenedError: 0,
			expectedClosed:      0,
			expectedClosedError: 1,
		},
	}

	for _, test := range testCases {
		m := createMetricsForTesting()

		test.testCase(m)

		assertCounterValue(t, test.description, "connectionsClosed", m.connectionsClosed,
			test.expectedClosed)
		assertCounterValue(t, test.description, "connectionsOpened", m.connectionsOpened,
			test.expectedOpened)
		assertCounterVecValue(t, test.description, "connectionsError[accept]", m.connectionsError,
			test.expectedOpenedError, prometheus.Labels{
				connectionErrorLabel: connectionAcceptError,
			})
		assertCounterVecValue(t, test.description, "connectionsError[close]", m.connectionsError,
			test.expectedClosedError, prometheus.Labels{
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

	expected := float64(1)
	assertCounterVecValue(t, "", fmt.Sprintf("requests[%s,%s]", requestType, requestStatus), m.requests,
		expected,
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
		description string
		testCase    func(m *Metrics)
		cookieFlag  pbsmetrics.CookieFlag
		expected    float64
	}{
		{
			description: "Yes",
			testCase: func(m *Metrics) {
				performTest(m, pbsmetrics.CookieFlagYes)
			},
			expected: 0,
		},
		{
			description: "No",
			testCase: func(m *Metrics) {
				performTest(m, pbsmetrics.CookieFlagNo)
			},
			expected: 1,
		},
		{
			description: "Unknown",
			testCase: func(m *Metrics) {
				performTest(m, pbsmetrics.CookieFlagUnknown)
			},
			expected: 0,
		},
	}

	for _, test := range testCases {
		m := createMetricsForTesting()

		test.testCase(m)

		assertCounterVecValue(t, test.description, fmt.Sprintf("requestsWithoutCookie[%s]", requestType), m.requestsWithoutCookie,
			test.expected,
			prometheus.Labels{
				requestTypeLabel: string(requestType),
			})
	}
}

func TestAccountMetric(t *testing.T) {
	knownPubID := "knownPublisher"
	performTest := func(m *Metrics, pubID string) {
		requestType := pbsmetrics.ReqTypeORTB2Web
		requestStatus := pbsmetrics.RequestStatusBlacklisted
		m.RecordRequest(pbsmetrics.Labels{
			RType:         requestType,
			RequestStatus: requestStatus,
			PubID:         pubID,
		})
	}

	testCases := []struct {
		description string
		testCase    func(m *Metrics)
		expected    float64
	}{
		{
			description: "Known",
			testCase: func(m *Metrics) {
				performTest(m, knownPubID)
			},
			expected: 1,
		},
		{
			description: "Unknown",
			testCase: func(m *Metrics) {
				performTest(m, pbsmetrics.PublisherUnknown)
			},
			expected: 0,
		},
	}

	for _, test := range testCases {
		m := createMetricsForTesting()

		test.testCase(m)

		assertCounterVecValue(t, test.description, fmt.Sprintf("accountRequests[%s]", knownPubID), m.accountRequests,
			test.expected,
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
		description    string
		testCase       func(m *Metrics)
		expectedBanner float64
		expectedVideo  float64
		expectedAudio  float64
		expectedNative float64
	}{
		{
			description: "Banner Only",
			testCase: func(m *Metrics) {
				performTest(m, true, false, false, false)
			},
			expectedBanner: 1,
			expectedVideo:  0,
			expectedAudio:  0,
			expectedNative: 0,
		},
		{
			description: "Video Only",
			testCase: func(m *Metrics) {
				performTest(m, false, true, false, false)
			},
			expectedBanner: 0,
			expectedVideo:  1,
			expectedAudio:  0,
			expectedNative: 0,
		},
		{
			description: "Audio Only",
			testCase: func(m *Metrics) {
				performTest(m, false, false, true, false)
			},
			expectedBanner: 0,
			expectedVideo:  0,
			expectedAudio:  1,
			expectedNative: 0,
		},
		{
			description: "Native Only",
			testCase: func(m *Metrics) {
				performTest(m, false, false, false, true)
			},
			expectedBanner: 0,
			expectedVideo:  0,
			expectedAudio:  0,
			expectedNative: 1,
		},
		{
			description: "Multiple Types",
			testCase: func(m *Metrics) {
				performTest(m, true, false, false, true)
			},
			expectedBanner: 1,
			expectedVideo:  0,
			expectedAudio:  0,
			expectedNative: 1,
		},
	}

	for _, test := range testCases {
		m := createMetricsForTesting()

		test.testCase(m)

		var totalBanner float64
		var totalVideo float64
		var totalAudio float64
		var totalNative float64
		processMetrics(m.impressions, func(m dto.Metric) {
			value := m.GetCounter().GetValue()
			for _, label := range m.GetLabel() {
				if label.GetValue() == "true" {
					switch label.GetName() {
					case isBannerLabel:
						totalBanner = totalBanner + value
					case isVideoLabel:
						totalVideo = totalVideo + value
					case isAudioLabel:
						totalAudio = totalAudio + value
					case isNativeLabel:
						totalNative = totalNative + value
					}
				}
			}
		})
		assert.Equal(t, test.expectedBanner, totalBanner, test.description)
		assert.Equal(t, test.expectedVideo, totalVideo, test.description)
		assert.Equal(t, test.expectedAudio, totalAudio, test.description)
		assert.Equal(t, test.expectedNative, totalNative, test.description)
	}
}

func TestLegacyImpressionsMetric(t *testing.T) {
	m := createMetricsForTesting()

	m.RecordLegacyImps(pbsmetrics.Labels{}, 42)

	expected := float64(42)
	assertCounterValue(t, "", "impressionsLegacy", m.impressionsLegacy,
		expected)
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

func TestAdapterRequestMetrics(t *testing.T) {

	// primary test
	// cookie / no cookie
	// has bids / no bids

	// erors

	//	m.adapterRequests.With(prometheus.Labels{
	//	adapterLabel:   string(labels.Adapter),
	//	hasCookieLabel: strconv.FormatBool(labels.CookieFlag != pbsmetrics.CookieFlagNo),
	//		hasBidsLabel:   strconv.FormatBool(labels.AdapterBids == pbsmetrics.AdapterBidPresent),
	//}).Inc()

	//for err := range labels.AdapterErrors {
	//	m.adapterErrors.With(prometheus.Labels{
	//		adapterLabel:      string(labels.Adapter),
	//		adapterErrorLabel: string(err),
	//	}).Inc()
	//}
}

// adapter bids, adm vs nurl

// adapter price, same as request time

// adapter time, same as request time (if no errors)

func TestAdapterPanicMetric(t *testing.T) {
	m := createMetricsForTesting()
	adapterName := "anyName"

	m.RecordAdapterPanic(pbsmetrics.AdapterLabels{
		Adapter: openrtb_ext.BidderName(adapterName),
	})

	expectedCount := float64(1)
	assertCounterVecValue(t, "", fmt.Sprintf("adapterPanics[%s]", adapterName), m.adapterPanics,
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

	assertCounterVecValue(t, "", "storedRequestCacheResult[hit]", m.storedRequestCacheResult,
		float64(hitCount),
		prometheus.Labels{
			cacheResultLabel: string(pbsmetrics.CacheHit),
		})
	assertCounterVecValue(t, "", "storedRequestCacheResult[miss]", m.storedRequestCacheResult,
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

	assertCounterVecValue(t, "", "storedRequestCacheResult[hit]", m.storedImpressionsCacheResult,
		float64(hitCount),
		prometheus.Labels{
			cacheResultLabel: string(pbsmetrics.CacheHit),
		})
	assertCounterVecValue(t, "", "storedRequestCacheResult[miss]", m.storedImpressionsCacheResult,
		float64(missCount),
		prometheus.Labels{
			cacheResultLabel: string(pbsmetrics.CacheMiss),
		})
}

func TestCookieMetric(t *testing.T) {
	m := createMetricsForTesting()

	m.RecordCookieSync()

	expected := float64(1)
	assertCounterValue(t, "", "cookieSync", m.cookieSync,
		expected)
}

// user metrics

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
	assert.Equal(t, expectedCount, histogram.GetSampleCount(), name+":Count")
	assert.Equal(t, expectedSum, histogram.GetSampleSum(), name+":Sum")
}
