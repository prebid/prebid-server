package errortypes

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type stubError struct{ severity Severity }

func (e *stubError) Error() string      { return "anyMessage" }
func (e *stubError) Code() int          { return 42 }
func (e *stubError) Severity() Severity { return e.severity }

func TestContainsFatalError(t *testing.T) {
	fatalError := &stubError{severity: SeverityFatal}
	notFatalError := &stubError{severity: SeverityWarning}
	unknownSeverityError := errors.New("anyError")

	testCases := []struct {
		description   string
		errors        []error
		shouldBeFatal bool
	}{
		{
			description:   "None",
			errors:        []error{},
			shouldBeFatal: false,
		},
		{
			description:   "One - Fatal",
			errors:        []error{fatalError},
			shouldBeFatal: true,
		},
		{
			description:   "One - Not Fatal",
			errors:        []error{notFatalError},
			shouldBeFatal: false,
		},
		{
			description:   "One - Unknown Severity Same As Fatal",
			errors:        []error{unknownSeverityError},
			shouldBeFatal: true,
		},
		{
			description:   "Mixed",
			errors:        []error{fatalError, notFatalError, unknownSeverityError},
			shouldBeFatal: true,
		},
	}

	for _, tc := range testCases {
		result := ContainsFatalError(tc.errors)
		assert.Equal(t, tc.shouldBeFatal, result)
	}
}

func TestFatalOnly(t *testing.T) {
	fatalError := &stubError{severity: SeverityFatal}
	notFatalError := &stubError{severity: SeverityWarning}
	unknownSeverityError := errors.New("anyError")

	testCases := []struct {
		description       string
		errs              []error
		errsShouldBeFatal []error
	}{
		{
			description:       "None",
			errs:              []error{},
			errsShouldBeFatal: []error{},
		},
		{
			description:       "One - Fatal",
			errs:              []error{fatalError},
			errsShouldBeFatal: []error{fatalError},
		},
		{
			description:       "One - Not Fatal",
			errs:              []error{notFatalError},
			errsShouldBeFatal: []error{},
		},
		{
			description:       "One - Unknown Severity Same As Fatal",
			errs:              []error{unknownSeverityError},
			errsShouldBeFatal: []error{unknownSeverityError},
		},
		{
			description:       "Mixed",
			errs:              []error{fatalError, notFatalError, unknownSeverityError},
			errsShouldBeFatal: []error{fatalError, unknownSeverityError},
		},
	}

	for _, tc := range testCases {
		result := FatalOnly(tc.errs)
		assert.ElementsMatch(t, tc.errsShouldBeFatal, result)
	}
}

func TestWarningOnly(t *testing.T) {
	warningError := &stubError{severity: SeverityWarning}
	notWarningError := &stubError{severity: SeverityFatal}
	unknownSeverityError := errors.New("anyError")

	testCases := []struct {
		description         string
		errs                []error
		errsShouldBeWarning []error
	}{
		{
			description:         "None",
			errs:                []error{},
			errsShouldBeWarning: []error{},
		},
		{
			description:         "One - Warning",
			errs:                []error{warningError},
			errsShouldBeWarning: []error{warningError},
		},
		{
			description:         "One - Not Warning",
			errs:                []error{notWarningError},
			errsShouldBeWarning: []error{},
		},
		{
			description:         "One - Unknown Severity Not Warning",
			errs:                []error{unknownSeverityError},
			errsShouldBeWarning: []error{},
		},
		{
			description:         "One - Mixed",
			errs:                []error{warningError, notWarningError, unknownSeverityError},
			errsShouldBeWarning: []error{warningError},
		},
	}

	for _, tc := range testCases {
		result := WarningOnly(tc.errs)
		assert.ElementsMatch(t, tc.errsShouldBeWarning, result)
	}
}
