// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/prebid/prebid-server/v2/analytics/pubxai/queue (interfaces: AuctionBidsQueueInterface)

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	utils "github.com/prebid/prebid-server/v2/analytics/pubxai/utils"
)

// MockAuctionBidsQueueInterface is a mock of AuctionBidsQueueInterface interface.
type MockAuctionBidsQueueInterface struct {
	ctrl     *gomock.Controller
	recorder *MockAuctionBidsQueueInterfaceMockRecorder
}

// MockAuctionBidsQueueInterfaceMockRecorder is the mock recorder for MockAuctionBidsQueueInterface.
type MockAuctionBidsQueueInterfaceMockRecorder struct {
	mock *MockAuctionBidsQueueInterface
}

// NewMockAuctionBidsQueueInterface creates a new mock instance.
func NewMockAuctionBidsQueueInterface(ctrl *gomock.Controller) *MockAuctionBidsQueueInterface {
	mock := &MockAuctionBidsQueueInterface{ctrl: ctrl}
	mock.recorder = &MockAuctionBidsQueueInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAuctionBidsQueueInterface) EXPECT() *MockAuctionBidsQueueInterfaceMockRecorder {
	return m.recorder
}

// Enqueue mocks base method.
func (m *MockAuctionBidsQueueInterface) Enqueue(arg0 utils.AuctionBids) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Enqueue", arg0)
}

// Enqueue indicates an expected call of Enqueue.
func (mr *MockAuctionBidsQueueInterfaceMockRecorder) Enqueue(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Enqueue", reflect.TypeOf((*MockAuctionBidsQueueInterface)(nil).Enqueue), arg0)
}

// UpdateConfig mocks base method.
func (m *MockAuctionBidsQueueInterface) UpdateConfig(arg0, arg1 string) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "UpdateConfig", arg0, arg1)
}

// UpdateConfig indicates an expected call of UpdateConfig.
func (mr *MockAuctionBidsQueueInterfaceMockRecorder) UpdateConfig(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateConfig", reflect.TypeOf((*MockAuctionBidsQueueInterface)(nil).UpdateConfig), arg0, arg1)
}
