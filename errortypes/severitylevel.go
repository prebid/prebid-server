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
