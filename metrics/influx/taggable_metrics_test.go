package metrics

import (
	"github.com/rcrowley/go-metrics"
	"reflect"
	"testing"
)

func doStoreMeterTest(t *testing.T, name string, tags map[string]string) {
	var registry = taggableRegistry{
		delegate: metrics.NewRegistry(),
	}

	var meter1 = registry.getOrRegisterMeter(name, tags)
	var meter2 = registry.getOrRegisterMeter(name, tags)

	if meter1 != meter2 {
		t.Error("The registry did not return the same meter in both cases")
	}
}

func doStoreHistogramTest(t *testing.T, name string, tags map[string]string) {
	var registry = taggableRegistry{
		delegate: metrics.NewRegistry(),
	}

	var h1 = registry.getOrRegisterHistogram(name, tags, metrics.NewUniformSample(50))
	var h2 = registry.getOrRegisterHistogram(name, tags, metrics.NewUniformSample(100))

	if h1 != h2 {
		t.Error("The registry did not return the same histogram in both cases")
	}
}

func doStoreTimerTest(t *testing.T, name string, tags map[string]string) {
	var registry = taggableRegistry{
		delegate: metrics.NewRegistry(),
	}

	var h1 = registry.getOrRegisterTimer(name, tags)
	var h2 = registry.getOrRegisterTimer(name, tags)

	if h1 != h2 {
		t.Error("The registry did not return the same timer in both cases")
	}
}

func TestTaglessMeter(t *testing.T) {
	doStoreMeterTest(t, "some_name", nil)
}

func TestTaggedMeter(t *testing.T) {
	doStoreMeterTest(t, "some_name", map[string]string{"trick\"y": "tag}s"})
}

func TestTaglessHistogram(t *testing.T) {
	doStoreHistogramTest(t, "some_name", nil)
}

func TestTaggedHistogram(t *testing.T) {
	doStoreHistogramTest(t, "some_name", map[string]string{"trick\"y": "tag}s"})
}

func TestTaglessTimer(t *testing.T) {
	doStoreTimerTest(t, "some_name", nil)
}

func TestTaggedTimer(t *testing.T) {
	doStoreTimerTest(t, "some_name", map[string]string{
		"tag1": "value1",
		"tag2": "value2",
	})
}

func TestEach(t *testing.T) {
	var registry = taggableRegistry{
		delegate: metrics.NewRegistry(),
	}

	var name = "some_name"
	var tags = map[string]string{"tag1": "value"}
	var meter = registry.getOrRegisterMeter(name, tags)

	registry.each(func(name2 string, tags2 map[string]string, metric interface{}) {
		if name != name2 {
			t.Errorf("%s does not match %s", name, name2)
		}
		if !reflect.DeepEqual(tags, tags2) {
			t.Errorf("%v does not match %v", tags, tags2)
		}
		if meter != metric {
			t.Error("The metrics don't match.")
		}
	})

}
