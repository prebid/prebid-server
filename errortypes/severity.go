package errortypes

// Severity represents the severity level of a bid processing error.
type Severity int

const (
	// SeverityUnknown represents an unknown severity level.
	SeverityUnknown Severity = iota

	// SeverityFatal represents a fatal bid processing error which prevents a bid response.
	SeverityFatal

	// SeverityWarning represents a non-fatal bid processing error where invalid or ambiguous
	// data in the bid request was ignored.
	SeverityWarning

	// SeverityDebugWarning represents a non-fatal bid processing error where invalid or ambiguous
	// data in the bid request was ignored and to be logged in debug mode.
	SeverityDebugWarning
)

func isFatal(err error) bool {
	s, ok := err.(Coder)
	return !ok || s.Severity() == SeverityFatal
}

// IsWarning returns true if an error is labeled with a Severity of SeverityWarning
// Throughout the codebase, errors with SeverityWarning are of the type Warning
// defined in this package
func IsWarning(err error) bool {
	s, ok := err.(Coder)
	return ok && s.Severity() == SeverityWarning
}

// IsDebugWarning returns true if an error is labeled with a Severity of SeverityDebugWarning
// Throughout the codebase, errors with SeverityDebugWarning are of the type Warning
// defined in this package
func IsDebugWarning(err error) bool {
	_, ok := err.(*DebugWarning)
	return ok
}

// ContainsFatalError checks if the error list contains a fatal error.
func ContainsFatalError(errors []error) bool {
	for _, err := range errors {
		if isFatal(err) {
			return true
		}
	}

	return false
}

// FatalOnly returns a new error list with only the fatal severity errors.
func FatalOnly(errs []error) []error {
	errsFatal := make([]error, 0, len(errs))

	for _, err := range errs {
		if isFatal(err) {
			errsFatal = append(errsFatal, err)
		}
	}

	return errsFatal
}

// WarningOnly returns a new error list with only the warning severity errors.
func WarningOnly(errs []error) []error {
	errsWarning := make([]error, 0, len(errs))

	for _, err := range errs {
		if IsWarning(err) {
			errsWarning = append(errsWarning, err)
		}
	}

	return errsWarning
}

// DebugWarningOnly returns a new error list with only the debug warning severity errors.
func DebugWarningOnly(errs []error) []error {
	errsDebugWarning := make([]error, 0, len(errs))

	for _, err := range errs {
		if _, ok := err.(*DebugWarning); ok {
			errsDebugWarning = append(errsDebugWarning, err)
		}
	}

	return errsDebugWarning
}
