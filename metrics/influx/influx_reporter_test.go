package metrics

import (
	"errors"
	coreInflux "github.com/influxdata/influxdb/client/v2"
	"github.com/rcrowley/go-metrics"
	"testing"
	"time"
)

type TestableClient struct {
	IsClosed bool
	Points   []coreInflux.BatchPoints
}

// Ping checks that status of cluster, and will always return 0 time and no
// error for UDP clients.
func (c *TestableClient) Ping(timeout time.Duration) (time.Duration, string, error) {
	return 1 * time.Nanosecond, "Cluster is fine", nil
}

// Write takes a BatchPoints object and writes all Points to InfluxDB.
func (c *TestableClient) Write(bp coreInflux.BatchPoints) error {
	c.Points = append(c.Points, bp)
	return nil
}

// Query makes an InfluxDB Query on the database. This will fail if using
// the UDP client.
func (c *TestableClient) Query(q coreInflux.Query) (*coreInflux.Response, error) {
	return nil, errors.New("This test client doesn't support Queries")
}

// Close releases any resources a Client may be using.
func (c *TestableClient) Close() error {
	if c.IsClosed {
		return errors.New("The client shouldn't be closed twice.")
	} else {
		c.IsClosed = true
		return nil
	}
}

func NewTestableReporter(pointsCapacity int) (*reporter, *TestableClient) {
	var influxClient = &TestableClient{
		IsClosed: false,
		Points:   make([]coreInflux.BatchPoints, pointsCapacity),
	}

	var registry = taggableRegistry{metrics.NewRegistry()}

	var reporter = &reporter{
		client:   influxClient,
		database: "whatever",
		registry: &registry,
	}

	return reporter, influxClient
}

func TestChannelClose(t *testing.T) {
	var reporter, influxClient = NewTestableReporter(0)

	var sender = make(chan time.Time)
	close(sender)

	reporter.Run(nil, sender, nil)

	// If the code doesn't make it here, then the Run method isn't exiting when the channel closes like it should be.

	if !influxClient.IsClosed {
		t.Error("The influx client should be closed, since the channel was closed.")
	}

	reporter.registry.getOrRegisterMeter("metric_count", nil).Mark(1)

	if len(influxClient.Points) != 0 {
		t.Errorf("The client shouldn't have been sent any points. Received %d", len(influxClient.Points))
	}
}

type PointPair struct {
	actual   *coreInflux.Point
	expected *coreInflux.Point
}

func assertPointsMatch(t *testing.T, actual *coreInflux.Point, expected *coreInflux.Point, fields []string) {
	if actual.Name() != expected.Name() {
		t.Errorf("Point names differ. Actual: %s, Expected: %s", actual.Name(), expected.Name())
	}

	actualFields, _ := actual.Fields()
	expectedFields, _ := expected.Fields()

	for _, field := range fields {
		actualValue, _ := actualFields[field]
		expectedValue, _ := expectedFields[field]

		if expectedValue != actualValue {
			t.Errorf("Values for key %s differ. Actual: %v, Expected: %v", field, actualValue, expectedValue)
		}
	}
}

func assertPointArraysMatch(t *testing.T, actual []*coreInflux.Point, expected []*coreInflux.Point, fields []string) {
	if len(actual) != len(expected) {
		t.Errorf("The points arrays have different sizes. Actual: %d, Expected: %d", len(actual), len(expected))
	}

	pointPairs := make([]PointPair, 0, len(actual))
	for _, actualPoint := range actual {
		thisName := actualPoint.Name()
		for _, expectedPoint := range expected {
			if expectedPoint.Name() == thisName {
				pointPairs = append(pointPairs, PointPair{actualPoint, expectedPoint})
				break
			}
		}
	}

	if len(pointPairs) != len(actual) {
		t.Error("Actual and Expected point arrays don't have points with corresponding names")
	} else {
		for i := 0; i < len(pointPairs); i++ {
			assertPointsMatch(t, pointPairs[i].actual, pointPairs[i].expected, fields)
		}
	}
}

func TestSendMeters(t *testing.T) {
	var reporter, influxClient = NewTestableReporter(0)

	sender := make(chan time.Time)
	done := make(chan bool)
	go reporter.Run(nil, sender, done)
	var name1 = "metric_count"
	var name2 = "metric_count2"
	reporter.registry.getOrRegisterMeter(name1, nil).Mark(1)
	reporter.registry.getOrRegisterMeter(name2, nil).Mark(2)

	close(sender)
	<-done

	expectedPoint1, _ := coreInflux.NewPoint(name1, make(map[string]string), map[string]interface{}{"count": 1}, time.Now())
	expectedPoint2, _ := coreInflux.NewPoint(name2, make(map[string]string), map[string]interface{}{"count": 2}, time.Now())

	if len(influxClient.Points) != 1 {
		t.Errorf("The client should have been sent 1 point. Received %d", len(influxClient.Points))
	}
	assertPointArraysMatch(
		t,
		influxClient.Points[0].Points(),
		[]*coreInflux.Point{expectedPoint1, expectedPoint2},
		[]string{"count"},
	)
}

func TestSendHistograms(t *testing.T) {
	var reporter, influxClient = NewTestableReporter(0)

	sender := make(chan time.Time)
	done := make(chan bool)
	go reporter.Run(nil, sender, done)
	var name1 = "metric_count"
	var name2 = "metric_count2"
	reporter.registry.getOrRegisterHistogram(name1, nil, metrics.NewUniformSample(50)).Update(20)
	reporter.registry.getOrRegisterHistogram(name2, nil, metrics.NewUniformSample(50)).Update(25)

	close(sender)
	<-done

	expectedPoint1, _ := coreInflux.NewPoint(name1, make(map[string]string), map[string]interface{}{"max": 20}, time.Now())
	expectedPoint2, _ := coreInflux.NewPoint(name2, make(map[string]string), map[string]interface{}{"max": 25}, time.Now())

	if len(influxClient.Points) != 1 {
		t.Errorf("The client should have been sent 1 point. Received %d", len(influxClient.Points))
	}
	assertPointArraysMatch(
		t,
		influxClient.Points[0].Points(),
		[]*coreInflux.Point{expectedPoint1, expectedPoint2},
		[]string{"max"},
	)
}

func TestSendTimers(t *testing.T) {
	var reporter, influxClient = NewTestableReporter(0)

	sender := make(chan time.Time)
	done := make(chan bool)
	go reporter.Run(nil, sender, done)
	var name1 = "metric_count"
	var name2 = "metric_count2"
	reporter.registry.getOrRegisterTimer(name1, nil).Update(20)
	reporter.registry.getOrRegisterTimer(name2, nil).Update(25)

	close(sender)
	<-done

	expectedPoint1, _ := coreInflux.NewPoint(name1, make(map[string]string), map[string]interface{}{"max": 20}, time.Now())
	expectedPoint2, _ := coreInflux.NewPoint(name2, make(map[string]string), map[string]interface{}{"max": 25}, time.Now())

	if len(influxClient.Points) != 1 {
		t.Errorf("The client should have been sent 1 point. Received %d", len(influxClient.Points))
	}
	assertPointArraysMatch(
		t,
		influxClient.Points[0].Points(),
		[]*coreInflux.Point{expectedPoint1, expectedPoint2},
		[]string{"max"},
	)
}
