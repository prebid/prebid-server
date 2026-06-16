package randomutil

import (
	"math/rand"
)

type RandomGenerator interface {
	GenerateInt63() int64
	Intn(n int) int
}

type RandomNumberGenerator struct{}

func (RandomNumberGenerator) GenerateInt63() int64 {
	return rand.Int63()
}

func (r RandomNumberGenerator) Intn(n int) int {
	return rand.Intn(n)
}
