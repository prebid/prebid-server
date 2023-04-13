package ortb

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloneData(t *testing.T) {
	t.Run("populated", func(t *testing.T) {
		given := openrtb2.Data{
			ID:      "anyID",
			Name:    "anyName",
			Segment: []openrtb2.Segment{},
			Ext:     json.RawMessage(`{"anyField":1}`),
		}
		result := CloneData(given)
		assert.Equal(t, given, result, "equality")
		assert.NotSame(t, given, result, "pointer")
		assert.NotSame(t, given.Ext, result.Ext, "ext")
	})

	// how to test segment copy is properly called?
	// - shallow copy comparison? check at least one deep copy level?

	t.Run("assumptions", func(t *testing.T) {
		assert.ElementsMatch(t, discoverDirectPointerFields(reflect.TypeOf(openrtb2.Data{})),
			[]string{
				"Segment",
				"Ext",
			})
	})
}

func TestCloneSegmentSlice(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		result := CloneSegmentSlice(nil)
		assert.Nil(t, result)
	})

	t.Run("empty", func(t *testing.T) {
		result := CloneSegmentSlice([]openrtb2.Segment{})
		assert.Empty(t, result)
	})

	t.Run("one", func(t *testing.T) {
		given := []openrtb2.Segment{
			{ID: "1", Ext: json.RawMessage(`{"anyField":1}`)},
		}
		result := CloneSegmentSlice(given)
		require.Len(t, result, 1)
		assert.Equal(t, given[0], result[0])
		assert.NotSame(t, given[0], result[0])
	})

	t.Run("many", func(t *testing.T) {
		given := []openrtb2.Segment{
			{Ext: json.RawMessage(`{"anyField":1}`)},
			{Ext: json.RawMessage(`{"anyField":2}`)},
		}
		result := CloneSegmentSlice(given)
		require.Len(t, result, 2)
		assert.Equal(t, given[0], result[0])
		assert.NotSame(t, given[0], result[0])
		assert.NotSame(t, given[0].Ext, result[0].Ext)
		assert.Equal(t, given[1], result[1])
		assert.NotSame(t, given[1], result[1])
		assert.NotSame(t, given[1].Ext, result[1].Ext)
	})
}

func TestCloneSegment(t *testing.T) {
	t.Run("populated", func(t *testing.T) {
		given := openrtb2.Segment{
			ID:    "anyID",
			Name:  "anyName",
			Value: "anyValue",
			Ext:   json.RawMessage(`{"anyField":1}`),
		}
		result := CloneSegment(given)
		assert.Equal(t, given, result, "equality")
		assert.NotSame(t, given, result, "pointer")
		assert.NotSame(t, given.Ext, result.Ext, "ext")
	})

	t.Run("assumptions", func(t *testing.T) {
		assert.ElementsMatch(t, discoverDirectPointerFields(reflect.TypeOf(openrtb2.Segment{})),
			[]string{
				"Ext",
			})
	})
}

func TestCloneNetwork(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		result := CloneNetwork(nil)
		assert.Nil(t, result)
	})

	t.Run("populated", func(t *testing.T) {
		given := &openrtb2.Network{
			ID:     "anyID",
			Name:   "anyName",
			Domain: "anyDomain",
			Ext:    json.RawMessage(`{"anyField":1}`),
		}
		result := CloneNetwork(given)
		assert.Equal(t, given, result, "equality")
		assert.NotSame(t, given, result, "pointer")
		assert.NotSame(t, given.Ext, result.Ext, "ext")
	})

	t.Run("assumptions", func(t *testing.T) {
		assert.ElementsMatch(t, discoverDirectPointerFields(reflect.TypeOf(openrtb2.Network{})),
			[]string{
				"Ext",
			})
	})
}

func TestCloneChannel(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		result := CloneChannel(nil)
		assert.Nil(t, result)
	})

	t.Run("populated", func(t *testing.T) {
		given := &openrtb2.Channel{
			ID:     "anyID",
			Name:   "anyName",
			Domain: "anyDomain",
			Ext:    json.RawMessage(`{"anyField":1}`),
		}
		result := CloneChannel(given)
		assert.Equal(t, given, result, "equality")
		assert.NotSame(t, given, result, "pointer")
		assert.NotSame(t, given.Ext, result.Ext, "ext")
	})

	t.Run("assumptions", func(t *testing.T) {
		assert.ElementsMatch(t, discoverDirectPointerFields(reflect.TypeOf(openrtb2.Channel{})),
			[]string{
				"Ext",
			})
	})
}

// discoverDirectPointerFields returns the names of all fields directly on an object
// which is a pointer and would need to be cloned specially. This method is optimized
// for only types which can appear within an OpenRTB data model object.
func discoverDirectPointerFields(t reflect.Type) []string {
	var fields []string
	for _, f := range reflect.VisibleFields(t) {
		if f.Type.Kind() == reflect.Slice || f.Type.Kind() == reflect.Pointer {
			fields = append(fields, f.Name)
		}
	}
	return fields
}
