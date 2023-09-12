package privacy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComponentMatchesName(t *testing.T) {
	testCases := []struct {
		name   string
		given  Component
		target string
		result bool
	}{
		{
			name:   "match",
			given:  Component{Type: "a", Name: "b"},
			target: "b",
			result: true,
		},
		{
			name:   "wrong-field",
			given:  Component{Type: "a", Name: "b"},
			target: "a",
			result: false,
		},
		{
			name:   "different-value",
			given:  Component{Type: "a", Name: "b"},
			target: "foo",
			result: false,
		},
		{
			name:   "different-case",
			given:  Component{Type: "a", Name: "b"},
			target: "B",
			result: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.result, test.given.MatchesName(test.target))
		})
	}
}

func TestComponentMatchesType(t *testing.T) {
	testCases := []struct {
		name   string
		given  Component
		target string
		result bool
	}{
		{
			name:   "match",
			given:  Component{Type: "a", Name: "b"},
			target: "a",
			result: true,
		},
		{
			name:   "wrong-field",
			given:  Component{Type: "a", Name: "b"},
			target: "b",
			result: false,
		},
		{
			name:   "different-value",
			given:  Component{Type: "a", Name: "b"},
			target: "foo",
			result: false,
		},
		{
			name:   "different-case",
			given:  Component{Type: "a", Name: "b"},
			target: "A",
			result: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.result, test.given.MatchesType(test.target))
		})
	}
}
