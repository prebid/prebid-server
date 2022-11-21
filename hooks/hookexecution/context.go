package hookexecution

import (
	"sync"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/hooks/hookstage"
)

// executionContext holds information passed to module's hook during hook execution.
type executionContext struct {
	endpoint       string
	stage          string
	accountId      string
	account        *config.Account
	moduleContexts *moduleContexts
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

		moduleInvocationCtx.AccountConfig = cfg
	}

	return moduleInvocationCtx
}

type moduleContexts struct {
	sync.RWMutex
	ctxs map[string]hookstage.ModuleContext
}

func (mc *moduleContexts) put(moduleName string, mCtx hookstage.ModuleContext) {
	mc.Lock()
	mc.ctxs[moduleName] = mCtx
	mc.Unlock()
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
