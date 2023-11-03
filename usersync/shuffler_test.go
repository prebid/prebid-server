package usersync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShuffler(t *testing.T) {
	testCases := []struct {
		description string
		given       []string
	}{
		{
			description: "Nil",
			given:       nil,
		},
		{
			description: "Empty",
			given:       []string{},
		},
		{
			description: "One",
			given:       []string{"a"},
		},
		{
			// at least 3 elements are required to test the swap logic.
			description: "Many",
			given:       []string{"a", "b", "c"},
		},
	}

	for _, test := range testCases {
		givenCopy := make([]string, len(test.given))
		copy(givenCopy, test.given)

		randomShuffler{}.shuffle(test.given)

		// ignores order of elements. we're testing the swap logic, not the rand shuffle algorithm.
		assert.ElementsMatch(t, givenCopy, test.given, test.description)
	}
}
