package hep

import (
	"fmt"
	"github.com/prebid/prebid-server/hooks/stages"
)

type HookRepository interface {
	GetEntrypointHook(module, hook string) (stages.EntrypointHook, bool)
	GetRawAuctionHook(module, hook string) (stages.RawAuctionHook, bool)
	GetProcessedAuctionHook(module, hook string) (stages.ProcessedAuctionHook, bool)
	GetBidRequestHook(module, hook string) (stages.BidRequestHook, bool)
	GetRawBidResponseHook(module, hook string) (stages.RawBidResponseHook, bool)
	GetProcessedBidResponseHook(module, hook string) (stages.ProcessedBidResponseHook, bool)
	GetAllProcBidResponsesHook(module, hook string) (stages.AllProcBidResponsesHook, bool)
	GetAuctionResponseHook(module, hook string) (stages.AuctionResponseHook, bool)
}

func NewHookRepository(moduleHooks map[string]map[string]interface{}) (HookRepository, error) {
	repo := new(hookRepository)
	for moduleName, hooks := range moduleHooks {
		for hookCode, hook := range hooks {
			if err := repo.add(moduleName, hookCode, hook); err != nil {
				return nil, err
			}
		}
	}

	return repo, nil
}

type hookRepository struct {
	entrypointHooks         map[string]map[string]stages.EntrypointHook
	rawauctionHooks         map[string]map[string]stages.RawAuctionHook
	procauctionHooks        map[string]map[string]stages.ProcessedAuctionHook
	bidrequestHooks         map[string]map[string]stages.BidRequestHook
	rawbidresponseHooks     map[string]map[string]stages.RawBidResponseHook
	procbidresponseHooks    map[string]map[string]stages.ProcessedBidResponseHook
	allprocbidresponseHooks map[string]map[string]stages.AllProcBidResponsesHook
	auctionresponseHooks    map[string]map[string]stages.AuctionResponseHook
}

func (r *hookRepository) GetEntrypointHook(module, hook string) (h stages.EntrypointHook, ok bool) {
	return getHook(r.entrypointHooks, module, hook)
}

func (r *hookRepository) GetRawAuctionHook(module, hook string) (stages.RawAuctionHook, bool) {
	return getHook(r.rawauctionHooks, module, hook)
}

func (r *hookRepository) GetProcessedAuctionHook(module, hook string) (stages.ProcessedAuctionHook, bool) {
	return getHook(r.procauctionHooks, module, hook)
}

func (r *hookRepository) GetBidRequestHook(module, hook string) (stages.BidRequestHook, bool) {
	return getHook(r.bidrequestHooks, module, hook)
}

func (r *hookRepository) GetRawBidResponseHook(module, hook string) (stages.RawBidResponseHook, bool) {
	return getHook(r.rawbidresponseHooks, module, hook)
}

func (r *hookRepository) GetProcessedBidResponseHook(module, hook string) (stages.ProcessedBidResponseHook, bool) {
	return getHook(r.procbidresponseHooks, module, hook)
}

func (r *hookRepository) GetAllProcBidResponsesHook(module, hook string) (stages.AllProcBidResponsesHook, bool) {
	return getHook(r.allprocbidresponseHooks, module, hook)
}

func (r *hookRepository) GetAuctionResponseHook(module, hook string) (stages.AuctionResponseHook, bool) {
	return getHook(r.auctionresponseHooks, module, hook)
}

func (r *hookRepository) add(module, code string, hook interface{}) (err error) {
	switch hookType := hook.(type) {
	case stages.EntrypointHook:
		r.entrypointHooks, err = addHook(r.entrypointHooks, hookType, module, code)
	case stages.RawAuctionHook:
		r.rawauctionHooks, err = addHook(r.rawauctionHooks, hookType, module, code)
	case stages.ProcessedAuctionHook:
		r.procauctionHooks, err = addHook(r.procauctionHooks, hookType, module, code)
	case stages.BidRequestHook:
		r.bidrequestHooks, err = addHook(r.bidrequestHooks, hookType, module, code)
	case stages.RawBidResponseHook:
		r.rawbidresponseHooks, err = addHook(r.rawbidresponseHooks, hookType, module, code)
	case stages.ProcessedBidResponseHook:
		r.procbidresponseHooks, err = addHook(r.procbidresponseHooks, hookType, module, code)
	case stages.AllProcBidResponsesHook:
		r.allprocbidresponseHooks, err = addHook(r.allprocbidresponseHooks, hookType, module, code)
	case stages.AuctionResponseHook:
		r.auctionresponseHooks, err = addHook(r.auctionresponseHooks, hookType, module, code)
	default:
		return fmt.Errorf(`trying to register invalid hook type: %s %s`, module, code)
	}
	return
}

func getHook[T any](moduleHooks map[string]map[string]T, module, hook string) (T, bool) {
	var h T
	if hooks, ok := moduleHooks[module]; ok {
		h, ok = hooks[hook]
		return h, ok
	}
	return h, false
}

func addHook[T any](moduleHooks map[string]map[string]T, hook T, module, hookCode string) (map[string]map[string]T, error) {
	if moduleHooks == nil {
		moduleHooks = make(map[string]map[string]T)
	}

	if moduleHooks[module] == nil {
		moduleHooks[module] = make(map[string]T)
	}

	if _, ok := moduleHooks[module][hookCode]; ok {
		return nil, fmt.Errorf(`hook with code "%s" already registered for module "%s"`, hookCode, module)
	}

	moduleHooks[module][hookCode] = hook

	return moduleHooks, nil
}
