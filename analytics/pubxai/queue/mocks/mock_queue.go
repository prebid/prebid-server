package mocks

import (
	"testing"

	"github.com/stretchr/testify/mock"
)

type MockQueueService[T any] struct {
	mock.Mock
}

func NewMockQueueService[T any](t *testing.T) *MockQueueService[T] {
	mock := &MockQueueService[T]{}
	mock.Test(t)
	return mock
}

func (m *MockQueueService[T]) Enqueue(item T) {
	_ = m.Called(item)
	return
}

func (m *MockQueueService[T]) UpdateConfig(bufferInterval, bufferSize string) {
	_ = m.Called(bufferInterval, bufferSize)
	return
}
