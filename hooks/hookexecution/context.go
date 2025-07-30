package hookexecution

import (
	"sync"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/privacy"
)

// executionContext holds information passed to module's hook during hook execution.
type executionContext struct {
	endpoint        string
	stage           string
	accountID       string
	account         *config.Account
	moduleContexts  *moduleContexts
	activityControl privacy.ActivityControl
}

func (ctx executionContext) getModuleContext(moduleName string) hookstage.ModuleInvocationContext {
	moduleInvocationCtx := hookstage.ModuleInvocationContext{Endpoint: ctx.endpoint}
	if ctx.moduleContexts != nil {
		if mc, ok := ctx.moduleContexts.get(moduleName); ok {
			moduleInvocationCtx.ModuleContext = mc
		}
	}

	if ctx.account != nil {
		cfg, err := ctx.account.Hooks.Modules.ModuleConfig(moduleName)
		if err != nil {
			glog.Warningf("Failed to get account config for %s module: %s", moduleName, err)
		}

		moduleInvocationCtx.AccountID = ctx.accountID
		moduleInvocationCtx.AccountConfig = cfg
	}

	return moduleInvocationCtx
}

// moduleContexts preserves data the module wants to pass to itself from earlier stages to later stages.
type moduleContexts struct {
	sync.RWMutex
	ctxs map[string]hookstage.ModuleContext // format: {"module_name": hookstage.ModuleContext}
}

func (mc *moduleContexts) put(moduleName string, mCtx hookstage.ModuleContext) {
	mc.Lock()
	defer mc.Unlock()

	newCtx := mCtx
	if existingCtx, ok := mc.ctxs[moduleName]; ok && existingCtx != nil {
		for k, v := range mCtx {
			existingCtx[k] = v
		}
		newCtx = existingCtx
	}
	mc.ctxs[moduleName] = newCtx
}

func (mc *moduleContexts) get(moduleName string) (hookstage.ModuleContext, bool) {
	mc.RLock()
	defer mc.RUnlock()
	mCtx, ok := mc.ctxs[moduleName]

	return mCtx, ok
}

type stageModuleContext struct {
	groupCtx []groupModuleContext
}

type groupModuleContext map[string]hookstage.ModuleContext
