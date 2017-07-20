package metrics

import (
	"encoding/json"
	"github.com/rcrowley/go-metrics"
)

// TaggedRegistry is a pseudo-Registry which allows metrics-level Influx Tags.
//
// This piggybacks on the Registry for its threadsafety...
// but exposes Tag-based APIs, and only implements the subset of the features we actually use.
type taggableRegistry struct {
	delegate metrics.Registry
}

func (r *taggableRegistry) each(f func(string, map[string]string, interface{})) {
	var decodeNamesBeforeF = func(encodedName string, metric interface{}) {
		var measurementData = decode(encodedName)
		f(measurementData.Name, measurementData.Tags, metric)
	}
	r.delegate.Each(decodeNamesBeforeF)
}

func (r *taggableRegistry) getOrRegisterMeter(name string, tags map[string]string) metrics.Meter {
	var encodedName = encode(name, tags)
	return r.delegate.GetOrRegister(encodedName, metrics.NewMeter).(metrics.Meter)
}

func (r *taggableRegistry) getOrRegisterTimer(name string, tags map[string]string) metrics.Timer {
	var encodedName = encode(name, tags)
	return r.delegate.GetOrRegister(encodedName, metrics.NewTimer).(metrics.Timer)
}

func (r *taggableRegistry) getOrRegisterHistogram(name string, tags map[string]string, sample metrics.Sample) metrics.Histogram {
	var encodedName = encode(name, tags)
	return r.delegate.GetOrRegister(encodedName, func() metrics.Histogram { return metrics.NewHistogram(sample) }).(metrics.Histogram)
}

type fieldMetadata struct {
	Name string            `json:"n"`
	Tags map[string]string `json:"t"`
}

func encode(name string, tags map[string]string) string {
	if tags == nil || len(tags) == 0 {
		return name
	}

	var fieldMetadata = fieldMetadata{
		Name: name,
		Tags: tags,
	}
	var encoded, err = json.Marshal(fieldMetadata)
	if err != nil {
		return name
	} else {
		return string(encoded)
	}
}

func decode(name string) *fieldMetadata {
	var dat fieldMetadata
	var err = json.Unmarshal([]byte(name), &dat)

	if err == nil {
		return &dat
	} else {
		return &fieldMetadata{
			Name: name,
			Tags: map[string]string{},
		}
	}
}
