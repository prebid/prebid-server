package timeutil

import (
	"testing"
	"time"
	_ "time/tzdata"

	"github.com/stretchr/testify/assert"
)

func TestLocationCacheLoadLocation(t *testing.T) {
	_, err := LoadLocation("Asia/Shanghai")
	assert.Nil(t, err, "should load location Asia/Shanghai")

	c := NewLocationCache()
	_, ok := c.cache["America/New_York"]
	assert.False(t, ok, "cache should not contain America/New_York")

	newyork, err := c.LoadLocation("America/New_York")
	assert.Nil(t, err)
	assert.NotNil(t, newyork)

	_, ok = c.cache["America/New_York"]
	assert.True(t, ok, "cache should contain America/New_York")

	cacheNewyork, _ := c.LoadLocation("America/New_York")
	assert.Equal(t, newyork, cacheNewyork)
}

func TestLocationCacheLoadLocationUnknown(t *testing.T) {
	c := NewLocationCache()
	_, ok := c.cache["America/Unknown"]
	assert.False(t, ok, "cache should not contain America/Unknown")

	unknown, err := c.LoadLocation("America/Unknown")
	assert.NotNil(t, err, "should return error")
	assert.Nil(t, unknown, "should return nil location")

	result, ok := c.cache["America/Unknown"]
	assert.True(t, ok, "cache should contain America/Unknown")
	assert.NotNil(t, result.err, "cache should contain error")
}

// goos: darwin
// goarch: amd64
// pkg: github.com/prebid/prebid-server/v2/util/timeutil
// cpu: Intel(R) Core(TM) i5-8257U CPU @ 1.40GHz
// BenchmarkLocationCacheLoadLocation-8   	66584589	        18.2 ns/op	       0 B/op	       0 allocs/op
func BenchmarkLocationCacheLoadLocation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = LoadLocation("America/New_York")
	}
}

// goos: darwin
// goarch: amd64
// pkg: github.com/prebid/prebid-server/v2/util/timeutil
// cpu: Intel(R) Core(TM) i5-8257U CPU @ 1.40GHz
// BenchmarkTimeLoadLocation-8   	   51571	     23117 ns/op	    8635 B/op	      13 allocs/op
func BenchmarkTimeLoadLocation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = time.LoadLocation("America/New_York")
	}
}
