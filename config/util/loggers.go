package util

import (
	"math/rand"
)

type LogMsg func(string, ...interface{})

type randomGenerator func() float32

// LogRandomSample will log a randam sample of the messages it is sent, based on the chance to log
// chance = 1.0 => always log,
// chance = 0.0 => never log
func LogRandomSample(msg string, logger LogMsg, chance float32) {
	logRandomSampleImpl(msg, logger, chance, rand.Float32)
}

func logRandomSampleImpl(msg string, logger LogMsg, chance float32, randGenerator randomGenerator) {
	if chance < 1.0 && randGenerator() > chance {
		// this is the chance we don't log anything
		return
	}
	logger(msg)
}
