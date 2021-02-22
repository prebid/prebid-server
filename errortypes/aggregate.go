package errortypes

import (
	"bytes"
	"strconv"
)

// AggregateError represents one or more errors.
type AggregateError struct {
	Message string
	Errors  []error
}

// NewAggregateError builds a AggregateError struct.
func NewAggregateError(msg string, errs []error) AggregateError {
	return AggregateError{
		Message: msg,
		Errors:  errs,
	}
}

// Error implements the standard error interface.
func (e AggregateError) Error() string {
	if len(e.Errors) == 0 {
		return ""
	}

	b := bytes.Buffer{}
	b.WriteString(e.Message)

	if len(e.Errors) == 1 {
		b.WriteString(" (1 error):\n")
	} else {
		b.WriteString(" (")
		b.WriteString(strconv.Itoa(len(e.Errors)))
		b.WriteString(" errors):\n")
	}

	for i, err := range e.Errors {
		b.WriteString("  ")
		b.WriteString(strconv.Itoa(i + 1))
		b.WriteString(": ")
		b.WriteString(err.Error())
		b.WriteString("\n")
	}

	return b.String()
}
