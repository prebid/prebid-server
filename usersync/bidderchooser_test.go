package usersync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// These tests use a deterministic version of the shuffler, where the order
// is reversed rather than randomized. Most tests specify at least 2 elements
// to implicitly verify the shuffler is invoked (or not invoked).

func TestBidderChooserChoose(t *testing.T) {
	shuffler := reverseShuffler{}
	available := []string{"a1", "a2"}

	testCases := []struct {
		description      string
		givenRequested   []string
		givenCooperative Cooperative
		expected         []string
	}{
		{
			description:      "No Coop - Nil",
			givenRequested:   nil,
			givenCooperative: Cooperative{Enabled: false},
			expected:         []string{"a2", "a1"},
		},
		{
			description:      "No Coop - Empty",
			givenRequested:   []string{},
			givenCooperative: Cooperative{Enabled: false},
			expected:         []string{"a2", "a1"},
		},
		{
			description:      "No Coop - One",
			givenRequested:   []string{"r"},
			givenCooperative: Cooperative{Enabled: false},
			expected:         []string{"r"},
		},
		{
			description:      "No Coop - Many",
			givenRequested:   []string{"r1", "r2"},
			givenCooperative: Cooperative{Enabled: false},
			expected:         []string{"r2", "r1"},
		},
		{
			description:      "Coop - Nil",
			givenRequested:   nil,
			givenCooperative: Cooperative{Enabled: true, PriorityGroups: [][]string{{"pr1A", "pr1B"}, {"pr2A", "pr2B"}}},
			expected:         []string{"pr1B", "pr1A", "pr2B", "pr2A", "a2", "a1"},
		},
		{
			description:      "Coop - Empty",
			givenRequested:   nil,
			givenCooperative: Cooperative{Enabled: true, PriorityGroups: [][]string{{"pr1A", "pr1B"}, {"pr2A", "pr2B"}}},
			expected:         []string{"pr1B", "pr1A", "pr2B", "pr2A", "a2", "a1"},
		},
		{
			description:      "Coop - Integration Test",
			givenRequested:   []string{"r1", "r2"},
			givenCooperative: Cooperative{Enabled: true, PriorityGroups: [][]string{{"pr1A", "pr1B"}, {"pr2A", "pr2B"}}},
			expected:         []string{"r2", "r1", "pr1B", "pr1A", "pr2B", "pr2A", "a2", "a1"},
		},
	}

	for _, test := range testCases {
		chooser := standardBidderChooser{shuffler: shuffler}
		result := chooser.choose(test.givenRequested, available, test.givenCooperative)

		assert.Equal(t, test.expected, result, test.description)
	}
}

func TestBidderChooserCooperative(t *testing.T) {
	shuffler := reverseShuffler{}
	available := []string{"a1", "a2"}

	testCases := []struct {
		description         string
		givenRequested      []string
		givenPriorityGroups [][]string
		expected            []string
	}{
		{
			description:         "Nil",
			givenRequested:      nil,
			givenPriorityGroups: nil,
			expected:            []string{"a2", "a1"},
		},
		{
			description:         "Empty",
			givenRequested:      []string{},
			givenPriorityGroups: [][]string{},
			expected:            []string{"a2", "a1"},
		},
		{
			description:         "Requested",
			givenRequested:      []string{"r1", "r2"},
			givenPriorityGroups: nil,
			expected:            []string{"r2", "r1", "a2", "a1"},
		},
		{
			description:         "Priority Groups - One",
			givenRequested:      nil,
			givenPriorityGroups: [][]string{{"pr1A", "pr1B"}},
			expected:            []string{"pr1B", "pr1A", "a2", "a1"},
		},
		{
			description:         "Priority Groups - Many",
			givenRequested:      nil,
			givenPriorityGroups: [][]string{{"pr1A", "pr1B"}, {"pr2A", "pr2B"}},
			expected:            []string{"pr1B", "pr1A", "pr2B", "pr2A", "a2", "a1"},
		},
		{
			description:         "Requested + Priority Groups",
			givenRequested:      []string{"r1", "r2"},
			givenPriorityGroups: [][]string{{"pr1A", "pr1B"}, {"pr2A", "pr2B"}},
			expected:            []string{"r2", "r1", "pr1B", "pr1A", "pr2B", "pr2A", "a2", "a1"},
		},
	}

	for _, test := range testCases {
		chooser := standardBidderChooser{shuffler: shuffler}
		result := chooser.chooseCooperative(test.givenRequested, available, test.givenPriorityGroups)

		assert.Equal(t, test.expected, result, test.description)
	}
}

func TestBidderChooserShuffledCopy(t *testing.T) {
	shuffler := reverseShuffler{}

	testCases := []struct {
		description string
		given       []string
		expected    []string
	}{
		{
			description: "Empty",
			given:       []string{},
			expected:    []string{},
		},
		{
			description: "One",
			given:       []string{"a"},
			expected:    []string{"a"},
		},
		{
			description: "Many",
			given:       []string{"a", "b"},
			expected:    []string{"b", "a"},
		},
	}

	for _, test := range testCases {
		givenCopy := copySlice(test.given)

		chooser := standardBidderChooser{shuffler: shuffler}
		result := chooser.shuffledCopy(test.given)

		assert.Equal(t, givenCopy, test.given, test.description+":input")
		assert.Equal(t, test.expected, result, test.description+":result")
	}
}

func TestBidderChooserShuffledAppend(t *testing.T) {
	shuffler := reverseShuffler{}

	testCases := []struct {
		description string
		givenA      []string
		givenB      []string
		expected    []string
	}{
		{
			description: "Empty - Append Nil",
			givenA:      []string{},
			givenB:      nil,
			expected:    []string{},
		},
		{
			description: "Empty - Append Empty",
			givenA:      []string{},
			givenB:      []string{},
			expected:    []string{},
		},
		{
			description: "Empty - Append One",
			givenA:      []string{},
			givenB:      []string{"b"},
			expected:    []string{"b"},
		},
		{
			description: "Empty - Append Many",
			givenA:      []string{},
			givenB:      []string{"b1", "b2"},
			expected:    []string{"b2", "b1"},
		},
		{
			description: "One - Append Nil",
			givenA:      []string{"a"},
			givenB:      nil,
			expected:    []string{"a"},
		},
		{
			description: "One - Append Empty",
			givenA:      []string{"a"},
			givenB:      []string{},
			expected:    []string{"a"},
		},
		{
			description: "One - Append One",
			givenA:      []string{"a1"},
			givenB:      []string{"b1"},
			expected:    []string{"a1", "b1"},
		},
		{
			description: "One - Append Many",
			givenA:      []string{"a1"},
			givenB:      []string{"b1", "b2"},
			expected:    []string{"a1", "b2", "b1"},
		},
		{
			description: "Many - Append Nil",
			givenA:      []string{"a1", "a2"},
			givenB:      nil,
			expected:    []string{"a1", "a2"},
		},
		{
			description: "Many - Append Empty",
			givenA:      []string{"a1", "a2"},
			givenB:      []string{},
			expected:    []string{"a1", "a2"},
		},
		{
			description: "Many - Append One",
			givenA:      []string{"a1", "a2"},
			givenB:      []string{"b"},
			expected:    []string{"a1", "a2", "b"},
		},
		{
			description: "Many - Append Many",
			givenA:      []string{"a1", "a2"},
			givenB:      []string{"b1", "b2"},
			expected:    []string{"a1", "a2", "b2", "b1"},
		},
	}

	for _, test := range testCases {
		givenBCopy := copySlice(test.givenB)

		chooser := standardBidderChooser{shuffler: shuffler}
		result := chooser.shuffledAppend(test.givenA, test.givenB)

		assert.Equal(t, givenBCopy, test.givenB, test.description+":input")
		assert.Equal(t, test.expected, result, test.description+":result")
	}
}

// copySlice returns a cloned a slice or nil.
func copySlice(a []string) []string {
	var aCopy []string
	if a != nil {
		aCopy = make([]string, len(a))
		copy(aCopy, a)
	}
	return aCopy
}

type reverseShuffler struct{}

func (reverseShuffler) shuffle(a []string) {
	for i, j := 0, len(a)-1; i < j; i, j = i+1, j-1 {
		a[i], a[j] = a[j], a[i]
	}
}
