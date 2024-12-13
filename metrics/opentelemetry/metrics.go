package opentelemetry

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/gobeam/stringy"
	"github.com/golang/glog"
	"go.opentelemetry.io/otel/metric"
)

type (
	// Elem represents a metric element.
	Elem struct {
		Name  string
		Value reflect.Value
		Tag   reflect.StructTag
	}
)

// FindAllMetrics finds all metrics in the given struct.
func FindAllMetrics(m any, prefix string) <-chan Elem {
	ret := make(chan Elem, 16)

	go func() {
		defer close(ret)

		v := reflect.Indirect(reflect.ValueOf(m))
		q := []Elem{{Value: v, Name: prefix}}
		for len(q) > 0 {
			e := q[0]
			q = q[1:]
			if e.Value.Kind() == reflect.Struct {
				t := e.Value.Type()
				for i := 0; i < t.NumField(); i++ {
					f := t.Field(i)
					if f.Type.Kind() == reflect.Struct {
						q = append(q, Elem{Name: e.Name + f.Name + ".", Value: e.Value.Field(i)})
					} else {
						ret <- Elem{Name: e.Name + f.Name, Value: e.Value.Field(i), Tag: f.Tag}
					}
				}
			}
		}
	}()

	return ret
}

// CreateMetric creates a metric.
func CreateMetric[F func(string, ...O) (M, error), O, M any](name string, f F, opts ...metric.InstrumentOption) (M, error) {
	fOpts := make([]O, len(opts))
	for i, o := range opts {
		fOpts[i] = o.(O)
	}
	return f(name, fOpts...)
}

// CreateHistogramMetric creates a histogram metric.
func CreateHistogramMetric[F func(string, ...O) (M, error), O, M any](name string, f F, histOpts []metric.HistogramOption, opts ...metric.InstrumentOption) (M, error) {
	histOptsLen := len(histOpts)
	fOpts := make([]O, histOptsLen+len(opts))
	for i, o := range histOpts {
		fOpts[i] = o.(O)
	}
	for i, o := range opts {
		fOpts[histOptsLen+i] = o.(O)
	}
	return f(name, fOpts...)
}

// MetricName returns the metric name in snake case.
func MetricName(name string) string {
	return stringy.New(name).SnakeCase(".", ".").ToLower()
}

// HistogramOptions returns the options for a histogram metric.
func HistogramOptions(elem Elem) []metric.HistogramOption {
	if bb, ok := elem.Tag.Lookup("buckets"); ok {
		floatStrings := strings.Split(bb, ",")
		floats := make([]float64, len(floatStrings))
		var err error
		for i, s := range floatStrings {
			if floats[i], err = strconv.ParseFloat(s, 64); err != nil {
				glog.Error("failed to parse bucket boundary: %s: %v", s, err)
				return nil
			}
		}
		return []metric.HistogramOption{metric.WithExplicitBucketBoundaries(floats...)}
	}
	return nil
}

// InitMetrics initializes all metrics in the given struct (except those tagged with metric="-").
func InitMetrics(meter metric.Meter, m any, prefix string) error {
	if prefix != "" && !strings.HasSuffix(prefix, ".") {
		prefix += "."
	}
	var err error
	for elem := range FindAllMetrics(m, prefix) {
		err = errors.Join(err, InitMetricElem(meter, elem, prefix))
	}
	if err != nil {
		return err
	}
	return nil
}

// InitMetricElem initializes a single metric element - intended for callers that want to initialize a single metric
// possibly with altered Name or Tags.
func InitMetricElem(meter metric.Meter, elem Elem, prefix string) error {
	if elem.Tag.Get("metric") == "-" {
		glog.Info("skipping metric: %s", elem.Name)
		return nil
	}
	var opts []metric.InstrumentOption
	if description, ok := elem.Tag.Lookup("description"); ok {
		opts = append(opts, metric.WithDescription(description))
	} else {
		return fmt.Errorf("missing description tag from %s", elem.Name)
	}
	if u, ok := elem.Tag.Lookup("unit"); ok {
		opts = append(opts, metric.WithUnit(u))
	}
	var m any
	var err error
	metricName := MetricName(elem.Name)
	switch elem.Value.Type().String() {
	case "metric.Int64Counter":
		m, err = CreateMetric(metricName, meter.Int64Counter, opts...)
	case "metric.Int64UpDownCounter":
		m, err = CreateMetric(metricName, meter.Int64UpDownCounter, opts...)
	case "metric.Int64Histogram":
		m, err = CreateHistogramMetric(metricName, meter.Int64Histogram, HistogramOptions(elem), opts...)
	case "metric.Float64Counter":
		m, err = CreateMetric(metricName, meter.Float64Counter, opts...)
	case "metric.Float64UpDownCounter":
		m, err = CreateMetric(metricName, meter.Float64UpDownCounter, opts...)
	case "metric.Float64Histogram":
		m, err = CreateHistogramMetric(metricName, meter.Float64Histogram, HistogramOptions(elem), opts...)
	default:
		glog.Error("unknown metric type; skipping: %s", elem.Value.Type().String())
		return nil
	}
	if err != nil {
		return err
	}
	elem.Value.Set(reflect.ValueOf(m))
	return nil
}

// GetMetricName returns the name assigned to the metric.
func GetMetricName[M any](m M) string {
	return reflect.Indirect(reflect.ValueOf(m)).FieldByName("name").String()
}
