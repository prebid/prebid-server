package iterators

import (
	"iter"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testIter() iter.Seq2[int, string] {
	return func(yield func(int, string) bool) {
		if !yield(1, "one") {
			return
		}
		if !yield(2, "two") {
			return
		}
		if !yield(3, "three") {
			return
		}
	}
}

func TestFirsts(t *testing.T) {
	firsts := Firsts(testIter())
	want := []int{1, 2, 3}
	got := slices.Collect(firsts)
	assert.Equal(t, want, got)
}

func TestSeconds(t *testing.T) {
	seconds := Seconds(testIter())
	want := []string{"one", "two", "three"}
	got := slices.Collect(seconds)
	assert.Equal(t, want, got)
}
