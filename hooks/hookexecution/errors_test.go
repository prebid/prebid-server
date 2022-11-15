package hookexecution

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetReject(t *testing.T) {
	customError := errors.New("error message")
	rejectError := &RejectError{NBR: 123}

	testCases := []struct {
		description string
		errs        []error
		expectedErr *RejectError
	}{
		{
			description: "Returns reject error",
			errs:        []error{rejectError},
			expectedErr: rejectError,
		},
		{
			description: "Finds reject error in slice of errors and returns it",
			errs:        []error{customError, rejectError},
			expectedErr: rejectError,
		},
		{
			description: "No reject error if there are no errors",
			errs:        []error{},
			expectedErr: nil,
		},
		{
			description: "No reject error if it not found",
			errs:        []error{customError},
			expectedErr: nil,
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			result := FindReject(test.errs)
			assert.Equal(t, test.expectedErr, result)
		})
	}
}
