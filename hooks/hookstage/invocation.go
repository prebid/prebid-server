package hookstage

import "github.com/prebid/prebid-server/hooks/hookanalytics"

// InvocationContext holds information passed to module's hook during hook execution.
type InvocationContext struct {
}

// HookResult represents the result of execution the concrete hook instance.
type HookResult[T any] struct {
	// Reject indicates that the hook rejects execution of the program logic at the specific stage.
	Reject        bool
	AnalyticsTags hookanalytics.Analytics
}
