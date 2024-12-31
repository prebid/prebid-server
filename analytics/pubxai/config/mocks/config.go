package mocks

import (
	"github.com/prebid/prebid-server/v3/analytics/pubxai/config"
	"github.com/stretchr/testify/mock"
)

// MockConfigService is a mock of ConfigService interface using testify
type MockConfigService struct {
	mock.Mock
}

// NewMockConfigService creates a new mock instance
func NewMockConfigService() *MockConfigService {
	return &MockConfigService{}
}

// IsSameAs provides a mock function
func (m *MockConfigService) IsSameAs(a, b *config.Configuration) bool {
	args := m.Called(a, b)
	return args.Bool(0)
}

// Start provides a mock function
func (m *MockConfigService) Start(stop <-chan struct{}) <-chan *config.Configuration {
	args := m.Called(stop)
	return args.Get(0).(<-chan *config.Configuration)
}
