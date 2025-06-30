package hookstage

import (
	"encoding/json"

	"github.com/prebid/prebid-server/v3/hooks/hookanalytics"
)

// HookResult represents the result of execution the concrete hook instance.
type HookResult[T any] struct {
	Reject        bool         // true value indicates rejection of the program execution at the specific stage
	NbrCode       int          // hook must provide NbrCode if the field Reject set to true
	Message       string       // holds arbitrary message added by hook
	ChangeSet     ChangeSet[T] // set of changes the module wants to apply to hook payload in case of successful execution
	Errors        []string
	Warnings      []string
	DebugMessages []string
	AnalyticsTags hookanalytics.Analytics
	ModuleContext ModuleContext // holds values that the module wants to pass to itself at later stages
}

// ModuleInvocationContext holds data passed to the module hook during invocation.
type ModuleInvocationContext struct {
	// AccountID holds the account ID
	AccountID string
	// AccountConfig represents module config rewritten at the account-level.
	AccountConfig json.RawMessage
	// Endpoint represents the path of the current endpoint.
	Endpoint string
	// ModuleContext holds values that the module passes to itself from the previous stages.
	ModuleContext ModuleContext
}

// ModuleContext holds arbitrary data passed between module hooks at different stages.
// We use interface as we do not know exactly how the modules will use their inner context.
type ModuleContext map[string]interface{}
