package randomutil

import (
	"math/rand"
)

type RandomGenerator interface {
	GenerateInt63() int64
	GenerateFloat64() float64
}

type RandomNumberGenerator struct{}

func (RandomNumberGenerator) GenerateInt63() int64 {
	return rand.Int63()
}

func (RandomNumberGenerator) GenerateFloat64() float64 {
	return rand.Float64()
}
