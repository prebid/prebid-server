package randomutil

import (
	"math/rand"
	"time"
)

type RandomGenerator interface {
	GenerateInt63() int64
}

type RandomNumberGenerator struct{}

func (RandomNumberGenerator) GenerateInt63() int64 {
	rand.Seed(time.Now().UnixNano())
	return rand.Int63()
}
