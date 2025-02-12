package hookexecution

import (
	"time"

	"github.com/prebid/prebid-server/v3/hooks/hookanalytics"
)

// Status indicates the result of hook execution.
type Status string

const (
	StatusSuccess          Status = "success"           // successful hook execution
	StatusTimeout          Status = "timeout"           // hook was not completed in the allotted time
	StatusFailure          Status = "failure"           // expected module-side failure occurred during hook execution
	StatusExecutionFailure Status = "execution_failure" // unexpected failure occurred during hook execution
)

// Action indicates the type of taken behaviour after the successful hook execution.
type Action string

const (
	ActionUpdate Action = "update"    // the hook returned mutations that were successfully applied
	ActionReject Action = "reject"    // the hook decided to reject the stage
	ActionNone   Action = "no_action" // the hook does not want to take any action
)

// Messages in format: {"module": {"hook": ["msg1", "msg2"]}}
type Messages map[string]map[string][]string

// ModulesOutcome represents result of hooks execution
// ready to be added to BidResponse.
//
// Errors and Warnings hold the error and warning
// messages returned from executing individual hooks.
type ModulesOutcome struct {
	Errors   Messages      `json:"errors,omitempty"`
	Warnings Messages      `json:"warnings,omitempty"`
	Trace    *TraceOutcome `json:"trace,omitempty"`
}

// TraceOutcome holds the result of executing hooks at all stages.
type TraceOutcome struct {
	// ExecutionTime is the sum of ExecutionTime of all stages.
	ExecutionTime
	Stages []Stage `json:"stages"`
}

// Stage holds the result of executing hooks at specific stage.
// May contain multiple StageOutcome results for stages executed
// multiple times (ex. for each bidder-request/response stages).
type Stage struct {
	// ExecutionTime is set to the longest ExecutionTime of its children.
	ExecutionTime
	Stage    string         `json:"stage"`
	Outcomes []StageOutcome `json:"outcomes"`
}

// StageOutcome represents the result of executing specific stage.
type StageOutcome struct {
	// ExecutionTime is the sum of ExecutionTime of all its groups
	ExecutionTime
	// An Entity specifies the type of object that was processed during the execution of the stage.
	Entity entity         `json:"entity"`
	Groups []GroupOutcome `json:"groups"`
	Stage  string         `json:"-"`
}

// GroupOutcome represents the result of executing specific group of hooks.
type GroupOutcome struct {
	// ExecutionTime is set to the longest ExecutionTime of its children.
	ExecutionTime
	InvocationResults []HookOutcome `json:"invocation_results"`
}

// HookOutcome represents the result of executing specific hook.
type HookOutcome struct {
	// ExecutionTime is the execution time of a specific hook without applying its result.
	ExecutionTime
	AnalyticsTags hookanalytics.Analytics `json:"analytics_tags"`
	HookID        HookID                  `json:"hook_id"`
	Status        Status                  `json:"status"`
	Action        Action                  `json:"action"`
	Message       string                  `json:"message"` // arbitrary string value returned from hook execution
	DebugMessages []string                `json:"debug_messages,omitempty"`
	Errors        []string                `json:"-"`
	Warnings      []string                `json:"-"`
}

// HookID points to the specific hook defined by the hook execution plan.
type HookID struct {
	ModuleCode   string `json:"module_code"`
	HookImplCode string `json:"hook_impl_code"`
}

type ExecutionTime struct {
	ExecutionTimeMillis time.Duration `json:"execution_time_millis,omitempty"`
}
