package hookexecution

import "fmt"

type TimeoutError struct{}

func (e TimeoutError) Error() string {
	return fmt.Sprint("Hook execution timeout")
}

// FailureError indicates expected error occurred during hook execution on the module-side
type FailureError struct{}

func (e FailureError) Error() string {
	return fmt.Sprint("Hook execution failed")
}

type RejectError struct {
	Code   int
	Reason string // is it needed or code is enough?
}

func (e RejectError) Error() string {
	return fmt.Sprintf(`Module rejected stage, reason: "%s"`, e.Reason)
}
