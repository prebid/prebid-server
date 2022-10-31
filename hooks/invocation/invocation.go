package invocation

import (
	"encoding/json"
	"time"

	"github.com/prebid/prebid-server/metrics"
)

type InvocationContext struct {
	Endpoint          string
	Stage             string
	DebugEnabled      bool
	RequestTypeMetric metrics.RequestType
	moduleContexts    map[string]*ModuleContext
}

func (ctx *InvocationContext) ModuleContextFor(moduleCode string) *ModuleContext {
	if mc, ok := ctx.moduleContexts[moduleCode]; ok {
		return mc
	}

	emptyCtx := ModuleContext{}

	if ctx.moduleContexts == nil {
		ctx.moduleContexts = map[string]*ModuleContext{}
	}
	ctx.moduleContexts[moduleCode] = &emptyCtx

	return &emptyCtx
}

type HookResponse[T any] struct {
	Result HookResult[T]
	Err    error
}

type HookResult[T any] struct {
	ModuleCode    string
	ExecutionTime time.Duration
	Reject        bool
	NbrCode       int
	Message       string
	ChangeSet     *ChangeSet[T]
	Errors        []string
	Warnings      []string
	DebugMessages []string
	//todo: think on adding next fields
	// analyticTags
}

type ModuleContext struct {
	Ctx           map[string]interface{} // interface as we do not know exactly how the modules will use their inner context
	AccountConfig json.RawMessage
}

type StageResult[T any] struct {
	GroupsResults [][]HookResult[T]
}
