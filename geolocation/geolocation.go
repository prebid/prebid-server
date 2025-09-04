package geolocation

import (
	"context"
	"errors"
	"time"

	"github.com/prebid/prebid-server/v3/util/timeutil"
)

var (
	ErrDatabaseUnavailable = errors.New("database is unavailable")
	ErrLookupIPInvalid     = errors.New("lookup IP is invalid")
	ErrLookupTimeout       = errors.New("lookup timeout")
)

// Retrieves geolocation information by IP address.
//
// Provided default implementation - MaxMind
// Each vendor (host company) might provide its own implementation.
type GeoLocation interface {
	Lookup(ctx context.Context, ip string) (*GeoInfo, error)
}

type NilGeoLocation struct{}

func (g *NilGeoLocation) Lookup(ctx context.Context, ip string) (*GeoInfo, error) {
	return &GeoInfo{}, nil
}

func NewNilGeoLocation() *NilGeoLocation {
	return &NilGeoLocation{}
}

// TimezoneToUTCOffset returns UTC offset of timezone in minutes.
func TimezoneToUTCOffset(name string) (int, error) {
	loc, err := timeutil.LoadLocation(name)
	if err != nil {
		return 0, err
	}
	_, offset := time.Now().In(loc).Zone()
	return offset / 60, nil
}
