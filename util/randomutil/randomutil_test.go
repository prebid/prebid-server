package randomutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateInt63(t *testing.T) {
	randomNumberGenerator := RandomNumberGenerator{}
	num1 := randomNumberGenerator.GenerateInt63()
	num2 := randomNumberGenerator.GenerateInt63()
	assert.Equal(t, false, num1 == num2)
}
