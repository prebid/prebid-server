package hookexecution

import (
	"fmt"

	"github.com/prebid/prebid-server/errortypes"
)

// TimeoutError indicates exceeding of the max execution time allotted for hook.
type TimeoutError struct{}

func (e TimeoutError) Error() string {
	return fmt.Sprint("Hook execution timeout")
}

// FailureError indicates expected error occurred during hook execution on the module-side.
type FailureError struct{}

func (e FailureError) Error() string {
	return fmt.Sprint("Hook execution failed")
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
	return errortypes.UnknownErrorCode
}

func (e RejectError) Severity() errortypes.Severity {
	return errortypes.SeverityUnknown
}

func (e RejectError) Error() string {
	return fmt.Sprintf(
		`Module %s (hook: %s) rejected request with code %d at %s stage`,
		e.Hook.ModuleCode,
		e.Hook.HookCode,
		e.NBR,
		e.Stage,
	)
}

func FindReject(errors []error) *RejectError {
	for _, err := range errors {
		if reject, ok := err.(*RejectError); ok {
			return reject
		}
	}
	return nil
}
