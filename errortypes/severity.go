package errortypes

// Severity defines the severity level of a bid processing error.
type SeverityLevel int

const (
	// SeverityUnknown defines an unknown severity level.
	SeverityUnknown Severity = iota

	// SeverityFatal defines a fatal bid processing error which prevents a bid response.
	SeverityFatal

	// SeverityWarning defines a non-fatal bid processing error.
	SeverityWarning
)

// SeverityLeveler provides a bid processing error severity level.
type SeverityProvider interface {
	Severity() Severity
}
