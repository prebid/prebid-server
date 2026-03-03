package mathutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRoundTo4Decimals(t *testing.T) {
	t.Run("positive-number", func(t *testing.T) {
		r := RoundTo4Decimals(123.456789)
		assert.Equal(t, 123.4568, r)
	})

	t.Run("negative-number", func(t *testing.T) {
		r := RoundTo4Decimals(-123.456789)
		assert.Equal(t, -123.4568, r)
	})

	t.Run("already-rounded", func(t *testing.T) {
		r := RoundTo4Decimals(123.4567)
		assert.Equal(t, 123.4567, r)
	})

	t.Run("round-up", func(t *testing.T) {
		r := RoundTo4Decimals(123.45675)
		assert.Equal(t, 123.4568, r)
	})

	t.Run("round-down", func(t *testing.T) {
		r := RoundTo4Decimals(123.45674)
		assert.Equal(t, 123.4567, r)
	})

	t.Run("small-number", func(t *testing.T) {
		r := RoundTo4Decimals(0.00005)
		assert.Equal(t, 0.0001, r)
	})

	t.Run("negative-small-number", func(t *testing.T) {
		r := RoundTo4Decimals(-0.00005)
		assert.Equal(t, -0.0001, r)
	})

	t.Run("zero", func(t *testing.T) {
		r := RoundTo4Decimals(0)
		assert.Equal(t, 0.0, r)
	})
}
