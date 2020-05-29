package util

import (
	"math/rand"
)

type logMsg func(string, ...interface{})

// LogRandomSample will log a randam sample of the messages it is sent, based on the chance to log
// chance = 1.0 => always log,
// chance = 0.0 => never log
func LogRandomSample(msg string, logger logMsg, chance float32) {
	if chance < 1.0 && rand.Float32() > chance {
		// this is the chance we don't log anything
		return
	}
	logger(msg)
}
