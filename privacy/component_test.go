package privacy

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComponentMatches(t *testing.T) {
	testCases := []struct {
		name   string
		given  Component
		target Component
		result bool
	}{
		{
			name:   "full",
			given:  Component{Type: "a", Name: "b"},
			target: Component{Type: "a", Name: "b"},
			result: true,
		},
		{
			name:   "name-wildcard",
			given:  Component{Type: "a", Name: "*"},
			target: Component{Type: "a", Name: "b"},
			result: true,
		},
		{
			name:   "different",
			given:  Component{Type: "a", Name: "b"},
			target: Component{Type: "foo", Name: "bar"},
			result: false,
		},
		{
			name:   "different-type",
			given:  Component{Type: "a", Name: "b"},
			target: Component{Type: "foo", Name: "b"},
			result: false,
		},
		{
			name:   "different-name",
			given:  Component{Type: "a", Name: "b"},
			target: Component{Type: "a", Name: "foo"},
			result: false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.result, test.given.Matches(test.target))
		})
	}
}

func TestParseComponent(t *testing.T) {
	testCases := []struct {
		name          string
		component     string
		expected      Component
		expectedError error
	}{
		{
			name:          "empty",
			component:     "",
			expected:      Component{},
			expectedError: errors.New("unable to parse empty component"),
		},
		{
			name:          "too-many-parts",
			component:     "bidder.bidderA.bidderB",
			expected:      Component{},
			expectedError: errors.New("unable to parse component: bidder.bidderA.bidderB"),
		},
		{
			name:          "type-bidder",
			component:     "bidder.bidderA",
			expected:      Component{Type: "bidder", Name: "bidderA"},
			expectedError: nil,
		},
		{
			name:          "type-analytics",
			component:     "analytics.bidderA",
			expected:      Component{Type: "analytics", Name: "bidderA"},
			expectedError: nil,
		},
		{
			name:          "type-no-type",
			component:     "bidderA",
			expected:      Component{Type: "", Name: "bidderA"},
			expectedError: nil,
		},
		{
			name:          "type-rtd",
			component:     "rtd.test",
			expected:      Component{Type: "rtd", Name: "test"},
			expectedError: nil,
		},
		{
			name:          "type-general",
			component:     "general.test",
			expected:      Component{Type: "general", Name: "test"},
			expectedError: nil,
		},
		{
			name:          "type-invalid",
			component:     "invalid.test",
			expected:      Component{},
			expectedError: errors.New("unable to parse component (invalid type): invalid.test"),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualSN, actualErr := ParseComponent(test.component)
			if test.expectedError == nil {
				assert.Equal(t, test.expected, actualSN)
				assert.NoError(t, actualErr)
			} else {
				assert.EqualError(t, actualErr, test.expectedError.Error())
			}
		})
	}
}
