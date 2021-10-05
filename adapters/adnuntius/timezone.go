package adnuntius

import (
	"time"
)

type timezone interface {
	Now() time.Time
}

// Send a real instance when you construct it in adapter_map.go
type realTzo struct{}

func (_ realTzo) Now() time.Time {
	return time.Now()
}

// Use this for tests e.g. knownTzo(time.Date(y, m, ..., time.UTC))
type knownTzo time.Time

func (i knownTzo) Now() time.Time {
	return time.Time(i)
}
