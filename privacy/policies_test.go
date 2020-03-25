package privacy

import (
	"errors"
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestWritePoliciesNone(t *testing.T) {
	request := &openrtb.BidRequest{}
	policyWriters := []policyWriter{}

	err := writePolicies(request, policyWriters)

	assert.NoError(t, err)
}

func TestWritePoliciesOne(t *testing.T) {
	request := &openrtb.BidRequest{}
	mockWriter := new(mockPolicyWriter)
	policyWriters := []policyWriter{
		mockWriter,
	}

	mockWriter.On("Write", request).Return(nil).Once()

	err := writePolicies(request, policyWriters)

	assert.NoError(t, err)
	mockWriter.AssertExpectations(t)
}

func TestWritePoliciesMany(t *testing.T) {
	request := &openrtb.BidRequest{}
	mockWriter1 := new(mockPolicyWriter)
	mockWriter2 := new(mockPolicyWriter)
	policyWriters := []policyWriter{
		mockWriter1, mockWriter2,
	}

	mockWriter1.On("Write", request).Return(nil).Once()
	mockWriter2.On("Write", request).Return(nil).Once()

	err := writePolicies(request, policyWriters)

	assert.NoError(t, err)
	mockWriter1.AssertExpectations(t)
	mockWriter2.AssertExpectations(t)
}

func TestWritePoliciesError(t *testing.T) {
	request := &openrtb.BidRequest{}
	mockWriter := new(mockPolicyWriter)
	policyWriters := []policyWriter{
		mockWriter,
	}

	expectedErr := errors.New("anyError")
	mockWriter.On("Write", request).Return(expectedErr).Once()

	err := writePolicies(request, policyWriters)

	assert.Error(t, err, expectedErr)
	mockWriter.AssertExpectations(t)
}

type mockPolicyWriter struct {
	mock.Mock
}

func (m *mockPolicyWriter) Write(req *openrtb.BidRequest) error {
	args := m.Called(req)
	return args.Error(0)
}
