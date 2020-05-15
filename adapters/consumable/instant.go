package consumable

import "time"

type instant interface {
	Now() time.Time
}

// Send a real instance when you construct it in adapter_map.go
type realInstant struct{}

func (_ realInstant) Now() time.Time {
	return time.Now()
}

// Use this for tests e.g. knownInstant(time.Date(y, m, ..., time.UTC))
type knownInstant time.Time

func (i knownInstant) Now() time.Time {
	return time.Time(i)
}
