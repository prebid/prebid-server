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

func FatalOnly(errs []error) []error {
	errsFatal := make([]error, 0, len(errs))

	for _, err := range errs {
		if s, ok := err.(SeverityLeveler); !ok || s.SeverityLevel() == SeverityLevelFatal {
			errsFatal = append(errsFatal, err)
		}
	}

	return errsFatal
}

func WarningOnly(errs []error) []error {
	errsWarning := make([]error, 0, len(errs))

	for _, err := range errs {
		if s, ok := err.(SeverityLeveler); ok && s.SeverityLevel() == SeverityLevelWarning {
			errsWarning = append(errsWarning, err)
		}
	}

	return errsWarning
}
