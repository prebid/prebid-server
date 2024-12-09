package hookexecution

import (
	"fmt"

	"github.com/prebid/prebid-server/v3/errortypes"
)

// TimeoutError indicates exceeding of the max execution time allotted for hook.
type TimeoutError struct{}

func (e TimeoutError) Error() string {
	return "Hook execution timeout"
}

func NewFailure(format string, a ...any) FailureError {
	return FailureError{Message: fmt.Sprintf(format, a...)}
}

// FailureError indicates expected error occurred during hook execution on the module-side.
// A moduleFailed metric will be sent in such case.
type FailureError struct {
	Message string
}

func (e FailureError) Error() string {
	return fmt.Sprintf("hook execution failed: %s", e.Message)
}

// RejectError indicates stage rejection requested by specific hook.
// Implements errortypes.Coder interface for compatibility only,
// so as not to be recognized as a fatal error
type RejectError struct {
	NBR   int
	Hook  HookID
	Stage string
}

func (e RejectError) Code() int {
	return errortypes.ModuleRejectionErrorCode
}

func (e RejectError) Severity() errortypes.Severity {
	return errortypes.SeverityWarning
}

func (e RejectError) Error() string {
	return fmt.Sprintf(
		`Module %s (hook: %s) rejected request with code %d at %s stage`,
		e.Hook.ModuleCode,
		e.Hook.HookImplCode,
		e.NBR,
		e.Stage,
	)
}

func FindFirstRejectOrNil(errors []error) *RejectError {
	for _, err := range errors {
		if rejectErr, ok := CastRejectErr(err); ok {
			return rejectErr
		}
	}
	return nil
}

func CastRejectErr(err error) (*RejectError, bool) {
	rejectErr, ok := err.(*RejectError)
	return rejectErr, ok
}
