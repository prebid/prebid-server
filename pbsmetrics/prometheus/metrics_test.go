package prometheusmetrics

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func createMetricsForTesting() *Metrics {
	return NewMetrics(config.PrometheusMetrics{
		Port:      8080,
		Namespace: "prebid",
		Subsystem: "server",
	})
}

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
			description: "Set of 1 open success.",
			testCase: func(m *Metrics) {
				m.RecordConnectionAccept(true)
			},
			expectedOpened:      1,
			expectedOpenedError: 0,
			expectedClosed:      0,
			expectedClosedError: 0,
		},
		{
			description: "Set of 1 open error.",
			testCase: func(m *Metrics) {
				m.RecordConnectionAccept(false)
			},
			expectedOpened:      0,
			expectedOpenedError: 1,
			expectedClosed:      0,
			expectedClosedError: 0,
		},
		{
			description: "Set of 1 closed success.",
			testCase: func(m *Metrics) {
				m.RecordConnectionClose(true)
			},
			expectedOpened:      0,
			expectedOpenedError: 0,
			expectedClosed:      1,
			expectedClosedError: 0,
		},
		{
			description: "Set of 1 closed error.",
			testCase: func(m *Metrics) {
				m.RecordConnectionClose(false)
			},
			expectedOpened:      0,
			expectedOpenedError: 0,
			expectedClosed:      0,
			expectedClosedError: 1,
		},
		{
			description: "Mixed set.",
			testCase: func(m *Metrics) {
				m.RecordConnectionAccept(true)
				m.RecordConnectionAccept(true)
				m.RecordConnectionClose(true)
				m.RecordConnectionAccept(true)
				m.RecordConnectionAccept(false)
				m.RecordConnectionClose(false)
			},
			expectedOpened:      3,
			expectedOpenedError: 1,
			expectedClosed:      1,
			expectedClosedError: 1,
		},
	}

	for _, test := range testCases {
		m := createMetricsForTesting()
		test.testCase(m)

		assertCounterValue(t, "connectionsClosed", m.connectionsClosed, test.expectedClosed)
		assertCounterValue(t, "connectionsOpened", m.connectionsOpened, test.expectedOpened)
		assertCounterVecValue(t, "connectionsError[accept]", m.connectionsError, test.expectedOpenedError, prometheus.Labels{
			connectionErrorLabel: connectionAcceptError,
		})
		assertCounterVecValue(t, "connectionsError[close]", m.connectionsError, test.expectedClosedError, prometheus.Labels{
			connectionErrorLabel: connectionCloseError,
		})
	}
}

// todo: all the other ones :)

func assertCounterValue(t *testing.T, name string, counter prometheus.Counter, expected float64) {
	output := dto.Metric{}
	counter.Write(&output)
	actual := *output.GetCounter().Value

	if actual != expected {
		t.Errorf("Incorrect value for metric '%s': expected=\"%f\", actual=\"%f\"", name, actual, expected)
	}
}

func assertCounterVecValue(t *testing.T, name string, counterVec *prometheus.CounterVec, expected float64, labels prometheus.Labels) {
	counter := counterVec.With(labels)
	assertCounterValue(t, name, counter, expected)
}
