package hooks

import (
	"fmt"

	"github.com/prebid/prebid-server/hooks/stages"
)

type HookRepository interface {
	GetEntrypointHook(id string) (stages.EntrypointHook, bool)
	GetRawAuctionHook(id string) (stages.RawAuctionHook, bool)
	GetProcessedAuctionHook(id string) (stages.ProcessedAuctionHook, bool)
	GetBidRequestHook(id string) (stages.BidRequestHook, bool)
	GetRawBidResponseHook(id string) (stages.RawBidResponseHook, bool)
	GetAllProcessedBidResponsesHook(id string) (stages.AllProcBidResponsesHook, bool)
	GetAuctionResponseHook(id string) (stages.AuctionResponseHook, bool)
}

func NewHookRepository(hooks map[string]interface{}) (HookRepository, error) {
	repo := new(hookRepository)
	for id, hook := range hooks {
		if err := repo.add(id, hook); err != nil {
			return nil, err
		}
	}

	return repo, nil
}

type hookRepository struct {
	entrypointHooks              map[string]stages.EntrypointHook
	rawAuctionHooks              map[string]stages.RawAuctionHook
	processedAuctionHooks        map[string]stages.ProcessedAuctionHook
	bidRequestHooks              map[string]stages.BidRequestHook
	rawBidResponseHooks          map[string]stages.RawBidResponseHook
	allProcessedBidResponseHooks map[string]stages.AllProcBidResponsesHook
	auctionResponseHooks         map[string]stages.AuctionResponseHook
}

func (r *hookRepository) GetEntrypointHook(id string) (h stages.EntrypointHook, ok bool) {
	return getHook(r.entrypointHooks, id)
}

func (r *hookRepository) GetRawAuctionHook(id string) (stages.RawAuctionHook, bool) {
	return getHook(r.rawAuctionHooks, id)
}

func (r *hookRepository) GetProcessedAuctionHook(id string) (stages.ProcessedAuctionHook, bool) {
	return getHook(r.processedAuctionHooks, id)
}

func (r *hookRepository) GetBidRequestHook(id string) (stages.BidRequestHook, bool) {
	return getHook(r.bidRequestHooks, id)
}

func (r *hookRepository) GetRawBidResponseHook(id string) (stages.RawBidResponseHook, bool) {
	return getHook(r.rawBidResponseHooks, id)
}

func (r *hookRepository) GetAllProcessedBidResponsesHook(id string) (stages.AllProcBidResponsesHook, bool) {
	return getHook(r.allProcessedBidResponseHooks, id)
}

func (r *hookRepository) GetAuctionResponseHook(id string) (stages.AuctionResponseHook, bool) {
	return getHook(r.auctionResponseHooks, id)
}

func (r *hookRepository) add(id string, hook interface{}) (err error) {
	var isCompatible bool

	if h, ok := hook.(stages.EntrypointHook); ok {
		isCompatible = true
		if r.entrypointHooks, err = addHook(r.entrypointHooks, h, id); err != nil {
			return
		}
	}

	if h, ok := hook.(stages.RawAuctionHook); ok {
		isCompatible = true
		if r.rawAuctionHooks, err = addHook(r.rawAuctionHooks, h, id); err != nil {
			return
		}
	}

	if h, ok := hook.(stages.ProcessedAuctionHook); ok {
		isCompatible = true
		if r.processedAuctionHooks, err = addHook(r.processedAuctionHooks, h, id); err != nil {
			return
		}
	}

	if h, ok := hook.(stages.BidRequestHook); ok {
		isCompatible = true
		if r.bidRequestHooks, err = addHook(r.bidRequestHooks, h, id); err != nil {
			return
		}
	}

	if h, ok := hook.(stages.RawBidResponseHook); ok {
		isCompatible = true
		if r.rawBidResponseHooks, err = addHook(r.rawBidResponseHooks, h, id); err != nil {
			return
		}
	}

	if h, ok := hook.(stages.AllProcBidResponsesHook); ok {
		isCompatible = true
		if r.allProcessedBidResponseHooks, err = addHook(r.allProcessedBidResponseHooks, h, id); err != nil {
			return
		}
	}

	if h, ok := hook.(stages.AuctionResponseHook); ok {
		isCompatible = true
		if r.auctionResponseHooks, err = addHook(r.auctionResponseHooks, h, id); err != nil {
			return
		}
	}

	if !isCompatible {
		return fmt.Errorf(`hook "%s" does not implement any supported hook interface`, id)
	}

	return
}

func getHook[T any](hooks map[string]T, id string) (T, bool) {
	hook, ok := hooks[id]
	return hook, ok
}

func addHook[T any](hooks map[string]T, hook T, id string) (map[string]T, error) {
	if hooks == nil {
		hooks = make(map[string]T)
	}

	if _, ok := hooks[id]; ok {
		return nil, fmt.Errorf(`hook of type "%T" with id "%s" already registered`, new(T), id)
	}

	hooks[id] = hook

	return hooks, nil
}
