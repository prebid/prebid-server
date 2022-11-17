package hookexecution

import (
	"time"

	"github.com/prebid/prebid-server/hooks/hookanalytics"
	"github.com/prebid/prebid-server/hooks/hookstage"
)

type Status string

const (
	StatusSuccess          Status = "success"
	StatusTimeout          Status = "timeout"
	StatusFailure          Status = "failure"           // expected module-side failure occurred during hook execution
	StatusExecutionFailure Status = "execution_failure" // unexpected failure occurred during hook execution
)

type Action string

const (
	ActionUpdate   Action = "update"
	ActionReject   Action = "reject"
	ActionNoAction Action = "no_action"
)

type Messages map[string]map[string][]string // Messages in format: {"module": {"hook": ["msg1", "msg2"]}}

type ModulesOutcome struct {
	Errors   Messages     `json:"errors"`
	Warnings Messages     `json:"warnings"`
	Trace    TraceOutcome `json:"trace"`
}

type TraceOutcome struct {
	ExecutionTime
	Stages []Stage `json:"stages"`
}

type Stage struct {
	ExecutionTime
	Stage    string         `json:"stage"`
	Outcomes []StageOutcome `json:"outcomes"`
}

type StageOutcome struct {
	ExecutionTime
	Entity hookstage.Entity `json:"entity"`
	Groups []GroupOutcome   `json:"groups"`
	Stage  string           `json:"-"`
}

type GroupOutcome struct {
	ExecutionTime
	InvocationResults []HookOutcome `json:"invocationresults"`
}

type HookOutcome struct {
	ExecutionTime
	AnalyticsTags hookanalytics.Analytics `json:"analyticstags"`
	HookID        HookID                  `json:"hookid"`
	Status        Status                  `json:"status"`
	Action        Action                  `json:"action"`
	Message       string                  `json:"message"`
	DebugMessages []string                `json:"debugmessages"`
	Errors        []string                `json:"-"`
	Warnings      []string                `json:"-"`
}

type HookID struct {
	ModuleCode string `json:"module-code"`
	HookCode   string `json:"hook-impl-code"`
}

type ExecutionTime struct {
	ExecutionTimeMillis time.Duration `json:"executiontimemillis"`
}
