package metrics

import (
	"encoding/json"
	"github.com/rcrowley/go-metrics"
	"github.com/golang/glog"
)

// TaggedRegistry is a pseudo-Registry which allows metrics-level Influx Tags.
//
// This piggybacks on the Registry for its threadsafety...
// but exposes Tag-based APIs, and only implements the subset of the features we actually use.
type TaggableRegistry struct {
	delegate metrics.Registry
}

func (r *TaggableRegistry) Each(f func(string, map[string]string, interface{})) {
	var decodeNamesBeforeF = func(encodedName string, metric interface{}) {
		var measurementData = decode(encodedName)
		f(measurementData.Name, measurementData.Tags, metric)
	}
	r.delegate.Each(decodeNamesBeforeF)
}

func (r *TaggableRegistry) GetOrRegisterMeter(name string, tags map[string]string) metrics.Meter {
	var encodedName = encode(name, tags)
	return r.delegate.GetOrRegister(encodedName, metrics.NewMeter).(metrics.Meter)
}

func (r *TaggableRegistry) GetOrRegisterTimer(name string, tags map[string]string) metrics.Timer {
	var encodedName = encode(name, tags)
	return r.delegate.GetOrRegister(encodedName, metrics.NewTimer).(metrics.Timer)
}

func (r *TaggableRegistry) GetOrRegisterHistogram(name string, tags map[string]string) metrics.Histogram {
	var encodedName = encode(name, tags)
	return r.delegate.GetOrRegister(encodedName, metrics.NewHistogram).(metrics.Histogram)
}

type FieldMetadata struct {
	Name string            `json:"n"`
	Tags map[string]string `json:"t"`
}

func encode(name string, tags map[string]string) string {
	if tags == nil || len(tags) == 0 {
		return name
	}

	var fieldMetadata = FieldMetadata{
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

func decode(name string) *FieldMetadata {
	var dat FieldMetadata
	var err = json.Unmarshal([]byte(name), &dat)

	if err != nil {
		glog.Errorf("Failed to decode measurement: %s", name)
		return &FieldMetadata{
			Name: name,
			Tags: map[string]string{},
		}
	} else {
		return &dat
	}
}
