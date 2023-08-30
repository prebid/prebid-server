package randomutil

import (
	"math/rand"
)

type RandomGenerator interface {
	GenerateInt63() int64
}

type RandomNumberGenerator struct{}

func (RandomNumberGenerator) GenerateInt63() int64 {
	return rand.Int63()
}
