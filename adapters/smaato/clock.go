package smaato

import "time"

type clock interface {
	Now() time.Time
}

type realClock struct{}
type mockClock time.Time

func (_ realClock) Now() time.Time {
	return time.Now()
}

func (mockClock mockClock) Now() time.Time {
	return time.Time(mockClock)
}
