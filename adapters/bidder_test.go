package adapters

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestExtraRequestInfoConvertCurrency(t *testing.T) {
	var (
		givenValue float64 = 2
		givenFrom  string  = "AAA"
		givenTo    string  = "BBB"
	)

	testCases := []struct {
		description   string
		setMock       func(m *mock.Mock)
		expectedValue float64
		expectedError error
	}{
		{
			description:   "Success",
			setMock:       func(m *mock.Mock) { m.On("GetRate", "AAA", "BBB").Return(2.5, nil) },
			expectedValue: 5,
			expectedError: nil,
		},
		{
			description:   "Error",
			setMock:       func(m *mock.Mock) { m.On("GetRate", "AAA", "BBB").Return(2.5, errors.New("some error")) },
			expectedValue: 0,
			expectedError: errors.New("some error"),
		},
	}

	for _, test := range testCases {
		mockConversions := &mockConversions{}
		test.setMock(&mockConversions.Mock)

		extraRequestInfo := NewExtraRequestInfo(mockConversions)
		result, err := extraRequestInfo.ConvertCurrency(givenValue, givenFrom, givenTo)

		mockConversions.AssertExpectations(t)
		assert.Equal(t, test.expectedValue, result, test.description+":result")
		assert.Equal(t, test.expectedError, err, test.description+":err")
	}
}

type mockConversions struct {
	mock.Mock
}

func (m mockConversions) GetRate(from string, to string) (float64, error) {
	args := m.Called(from, to)
	return args.Get(0).(float64), args.Error(1)
}

func (m mockConversions) GetRates() *map[string]map[string]float64 {
	args := m.Called()
	return args.Get(0).(*map[string]map[string]float64)
}
