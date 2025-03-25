package iterators

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestWalkGjsonLeaves(t *testing.T) {
	tests := []struct {
		name      string
		json      string
		wantPaths []string
	}{
		{
			name:      "true",
			json:      "true",
			wantPaths: []string{""},
		},
		{
			name:      "three fields",
			json:      `{"foo": "bar", "num": 42, "bool": true}`,
			wantPaths: []string{"bool", "foo", "num"},
		},
		{
			name:      "deep struct",
			json:      `{"foo": {"bar": {"baz": "qux"}}, "num": 42, "bool": true}`,
			wantPaths: []string{"bool", "foo.bar.baz", "num"},
		},
		{
			name: "deep struct with array",
			json: `{"foo": {"bar": {"baz": "qux", "biz": [1,2,3,4]}}, "num": 42, "bool": true}`,
			wantPaths: []string{
				"bool",
				"foo.bar.baz",
				"foo.bar.biz.0",
				"foo.bar.biz.1",
				"foo.bar.biz.2",
				"foo.bar.biz.3",
				"num",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gjsonResult := gjson.Parse(tt.json)
			gotPaths := slices.Collect(Firsts(WalkGjsonLeaves(gjsonResult)))
			slices.Sort(gotPaths)
			assert.Equal(t, tt.wantPaths, gotPaths)
		})
	}
}
