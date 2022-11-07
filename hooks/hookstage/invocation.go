package hookstage

import (
	"encoding/json"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/hooks/hookanalytics"
	"github.com/prebid/prebid-server/metrics"
)

type Entity string

const (
	EntityHttpRequest              Entity = "http-request"
	EntityAuctionRequest           Entity = "auction-request"
	EntityAuctionResponse          Entity = "auction-response"
	EntityAllProcessedBidResponses Entity = "all-processed-bid-responses"
)

type InvocationContext struct {
	Endpoint          string
	Stage             string
	AccountId         string
	DebugEnabled      bool
	RequestTypeMetric metrics.RequestType
	Account           *config.Account
	moduleContexts    map[string]*ModuleContext
}

func (ctx *InvocationContext) ModuleContextFor(moduleCode string) *ModuleContext {
	if mc, ok := ctx.moduleContexts[moduleCode]; ok {
		return mc
	}

	emptyCtx := ModuleContext{}
	if ctx.Account != nil {
		cfg, err := ctx.Account.Hooks.Modules.ModuleConfig(moduleCode)
		if err != nil {
			glog.Warningf("Failed to get account config for %s module: %s", moduleCode, err)
		}

		emptyCtx.AccountConfig = cfg
	}

	if ctx.moduleContexts == nil {
		ctx.moduleContexts = map[string]*ModuleContext{}
	}
	ctx.moduleContexts[moduleCode] = &emptyCtx

	return &emptyCtx
}

type HookResult[T any] struct {
	Reject        bool
	NbrCode       int
	Message       string
	ChangeSet     *ChangeSet[T]
	Errors        []string
	Warnings      []string
	DebugMessages []string
	AnalyticsTags hookanalytics.Analytics
}

type ModuleContext struct {
	Ctx           map[string]interface{} // interface as we do not know exactly how the modules will use their inner context
	AccountConfig json.RawMessage
}

type StageResult[T any] struct {
	GroupsResults [][]HookResult[T]
}
