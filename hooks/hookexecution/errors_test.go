package hookexecution

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindFirstRejectOrNil(t *testing.T) {
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
			result := FindFirstRejectOrNil(test.errs)
			assert.Equal(t, test.expectedErr, result)
		})
	}
}

func TestCastRejectErr(t *testing.T) {
	rejectError := &RejectError{NBR: 123}
	testCases := []struct {
		description    string
		err            error
		expectedErr    *RejectError
		expectedResult bool
	}{
		{
			description:    "Returns reject error and true if reject error provided",
			err:            rejectError,
			expectedErr:    rejectError,
			expectedResult: true,
		},
		{
			description:    "Returns nil and false if no error provided",
			err:            nil,
			expectedErr:    nil,
			expectedResult: false,
		},
		{
			description:    "Returns nil and false if custom error type provided",
			err:            errors.New("error message"),
			expectedErr:    nil,
			expectedResult: false,
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			rejectErr, isRejectErr := CastRejectErr(test.err)
			assert.Equal(t, test.expectedErr, rejectErr, "Invalid error returned.")
			assert.Equal(t, test.expectedResult, isRejectErr, "Invalid casting result.")
		})
	}
}
