package geolocationtest

import (
	"context"
	"errors"
	"sync"

	"github.com/prebid/prebid-server/v3/geolocation"
)

type MockGeoLocation struct {
	mu   sync.RWMutex
	data map[string]*geolocation.GeoInfo
}

func NewMockGeoLocation(data map[string]*geolocation.GeoInfo) *MockGeoLocation {
	if data == nil {
		data = make(map[string]*geolocation.GeoInfo)
	}
	return &MockGeoLocation{
		data: data,
	}
}

func (m *MockGeoLocation) Add(ip string, info *geolocation.GeoInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[ip] = info
}

func (m *MockGeoLocation) Remove(ip string, info *geolocation.GeoInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, ip)
}

func (m *MockGeoLocation) Lookup(ctx context.Context, ip string) (*geolocation.GeoInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if info, ok := m.data[ip]; ok {
		return info, nil
	}
	return nil, errors.New("not found")
}
