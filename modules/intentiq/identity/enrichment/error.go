package enrichment

import (
	"context"
	"errors"
	"strconv"
)

type ErrorKind string

const (
	KindRequest   ErrorKind = "request"
	KindTransport ErrorKind = "transport"
	KindTimeout   ErrorKind = "timeout"
	KindStatus    ErrorKind = "status"
	KindBodyRead  ErrorKind = "body_read"
	KindParse     ErrorKind = "parse"
)

type Error struct {
	Kind   ErrorKind
	Status int // 0 when no response was received
	Err    error

	// ResponseSnippet is a length-capped response body, set for a non-2xx used for debugging
	ResponseSnippet string
}

func (e *Error) Error() string {
	kind := string(e.Kind)
	if kind == "" {
		kind = "unknown"
	}
	if e.Err == nil {
		return kind
	}
	return kind + ": " + e.Err.Error()
}
func (e *Error) Unwrap() error {
	return e.Err
}

func ErrorLabels(err error) (kind, statusCode string) {
	var ae *Error
	if errors.As(err, &ae) {
		if ae.Status > 0 {
			return string(ae.Kind), strconv.Itoa(ae.Status)
		}
		return string(ae.Kind), ""
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return string(KindTimeout), ""
	}
	return string(KindTransport), ""
}
