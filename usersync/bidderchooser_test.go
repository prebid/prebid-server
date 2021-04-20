package usersync

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/stretchr/testify/assert"
)

// These tests use a deterministic test version of the shuffler, where the order
// is reversed rather than randomized.

func TestChoose(t *testing.T) {
	shuffler := reverseShuffler{}
	available := []string{"a", "b"}

	testCases := []struct {
		description      string
		givenRequested   []string
		givenCooperative config.UserSyncCooperative
		expected         []string
	}{
		{
			description:      "No Coop - Nil",
			givenRequested:   nil,
			givenCooperative: config.UserSyncCooperative{Enabled: false},
			expected:         []string{"b", "a"},
		},
		{
			description:      "No Coop - Empty",
			givenRequested:   []string{},
			givenCooperative: config.UserSyncCooperative{Enabled: false},
			expected:         []string{},
		},
		{
			description:      "No Coop - One",
			givenRequested:   []string{"c"},
			givenCooperative: config.UserSyncCooperative{Enabled: false},
			expected:         []string{"c"},
		},
		{
			description:      "No Coop - Many",
			givenRequested:   []string{"c", "d"},
			givenCooperative: config.UserSyncCooperative{Enabled: false},
			expected:         []string{"d", "c"},
		},
		{
			description:      "Coop - Integration",
			givenRequested:   []string{"c", "d"},
			givenCooperative: config.UserSyncCooperative{Enabled: true, PriorityGroups: [][]string{{"1", "2"}, {"3", "4"}}},
			expected:         []string{"d", "c", "2", "1", "4", "3", "b", "a"},
		},
	}

	for _, test := range testCases {
		chooser := randomBidderChooser{shuffler: shuffler}
		chosen := chooser.choose(test.givenRequested, available, test.givenCooperative)

		assert.Equal(t, test.expected, chosen, test.description)
	}
}

func TestChooseCooperative(t *testing.T) {
	shuffler := reverseShuffler{}
	available := []string{"a", "b"}

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
			expected:            []string{"b", "a"},
		},
		{
			description:         "Empty",
			givenRequested:      []string{},
			givenPriorityGroups: [][]string{},
			expected:            []string{"b", "a"},
		},
		{
			description:         "Requested",
			givenRequested:      []string{"c", "d"},
			givenPriorityGroups: nil,
			expected:            []string{"d", "c", "b", "a"},
		},
		{
			description:         "Priority Groups - One",
			givenRequested:      nil,
			givenPriorityGroups: [][]string{{"c", "d"}},
			expected:            []string{"d", "c", "b", "a"},
		},
		{
			description:         "Priority Groups - Many",
			givenRequested:      nil,
			givenPriorityGroups: [][]string{{"c", "d"}, {"e", "f", "g"}},
			expected:            []string{"d", "c", "g", "f", "e", "b", "a"},
		},
		{
			description:         "Requested + Priority Groups",
			givenRequested:      []string{"c", "d"},
			givenPriorityGroups: [][]string{{"e", "f"}, {"g", "h", "i"}},
			expected:            []string{"d", "c", "f", "e", "i", "h", "g", "b", "a"},
		},
	}

	for _, test := range testCases {
		chooser := randomBidderChooser{shuffler: shuffler}
		chosen := chooser.chooseCooperative(test.givenRequested, available, test.givenPriorityGroups)

		assert.Equal(t, test.expected, chosen, test.description)
	}
}

func TestShuffledCopy(t *testing.T) {
	shuffler := reverseShuffler{}

	testCases := []struct {
		description string
		given       []string
		expected    []string
	}{
		{
			description: "Nil",
			given:       nil,
			expected:    nil,
		},
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

		chooser := randomBidderChooser{shuffler: shuffler}
		shuffled := chooser.shuffledCopy(test.given)

		assert.Equal(t, givenCopy, test.given, test.description+":input unchanged")
		assert.Equal(t, test.expected, shuffled, test.description+":expected")
	}
}

func TestShuffledAppend(t *testing.T) {
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
			givenB:      []string{"1"},
			expected:    []string{"1"},
		},
		{
			description: "Empty - Append Many",
			givenA:      []string{},
			givenB:      []string{"1", "2"},
			expected:    []string{"2", "1"},
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
			givenA:      []string{"a"},
			givenB:      []string{"1"},
			expected:    []string{"a", "1"},
		},
		{
			description: "One - Append Many",
			givenA:      []string{"a"},
			givenB:      []string{"1", "2"},
			expected:    []string{"a", "2", "1"},
		},
		{
			description: "Many - Append Nil",
			givenA:      []string{"a", "b"},
			givenB:      nil,
			expected:    []string{"a", "b"},
		},
		{
			description: "Many - Append Empty",
			givenA:      []string{"a", "b"},
			givenB:      []string{},
			expected:    []string{"a", "b"},
		},
		{
			description: "Many - Append One",
			givenA:      []string{"a", "b"},
			givenB:      []string{"1"},
			expected:    []string{"a", "b", "1"},
		},
		{
			description: "Many - Append Many",
			givenA:      []string{"a", "b"},
			givenB:      []string{"1", "2"},
			expected:    []string{"a", "b", "2", "1"},
		},
	}

	for _, test := range testCases {
		givenBCopy := copySlice(test.givenB)

		chooser := randomBidderChooser{shuffler: shuffler}
		shuffled := chooser.shuffledAppend(test.givenA, test.givenB)

		assert.Equal(t, givenBCopy, test.givenB, test.description+":append input unchanged")
		assert.Equal(t, test.expected, shuffled, test.description+":expected")
	}
}

// copySlice clones a slice with proper nil handling.
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
