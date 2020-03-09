package errortypes

// SeverityLevel represents the severity level of a bid processing error.
type SeverityLevel int

const (
	// SeverityLevelUnknown represents an unknown severity level.
	SeverityLevelUnknown SeverityLevel = iota

	// SeverityLevelFatal represents a fatal bid processing error which prevents a bid response.
	SeverityLevelFatal

	// SeverityLevelWarning represents a non-fatal bid processing error where invalid or ambiguous
	// data in the bid request was ignored.
	SeverityLevelWarning
)

// SeverityLeveler provides a bid processing error severity level.
type SeverityLeveler interface {
	SeverityLevel() SeverityLevel
}

func isFatal(err error) bool {
	s, ok := err.(SeverityLeveler)
	return !ok || s.SeverityLevel() == SeverityLevelFatal
}

func isWarning(err error) bool {
	s, ok := err.(SeverityLeveler)
	return ok && s.SeverityLevel() == SeverityLevelWarning
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
		if isWarning(err) {
			errsWarning = append(errsWarning, err)
		}
	}

	return errsWarning
}
