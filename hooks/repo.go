package hooks

import (
	"fmt"

	"github.com/prebid/prebid-server/hooks/hookstage"
)

type HookRepository interface {
	GetEntrypointHook(id string) (hookstage.Entrypoint, bool)
	GetRawAuctionHook(id string) (hookstage.RawAuction, bool)
	GetProcessedAuctionHook(id string) (hookstage.ProcessedAuction, bool)
	GetBidRequestHook(id string) (hookstage.BidRequest, bool)
	GetRawBidResponseHook(id string) (hookstage.RawBidResponse, bool)
	GetAllProcessedBidResponsesHook(id string) (hookstage.AllProcessedBidResponses, bool)
	GetAuctionResponseHook(id string) (hookstage.AuctionResponse, bool)
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
	entrypointHooks              map[string]hookstage.Entrypoint
	rawAuctionHooks              map[string]hookstage.RawAuction
	processedAuctionHooks        map[string]hookstage.ProcessedAuction
	bidRequestHooks              map[string]hookstage.BidRequest
	rawBidResponseHooks          map[string]hookstage.RawBidResponse
	allProcessedBidResponseHooks map[string]hookstage.AllProcessedBidResponses
	auctionResponseHooks         map[string]hookstage.AuctionResponse
}

func (r *hookRepository) GetEntrypointHook(id string) (h hookstage.Entrypoint, ok bool) {
	return getHook(r.entrypointHooks, id)
}

func (r *hookRepository) GetRawAuctionHook(id string) (hookstage.RawAuction, bool) {
	return getHook(r.rawAuctionHooks, id)
}

func (r *hookRepository) GetProcessedAuctionHook(id string) (hookstage.ProcessedAuction, bool) {
	return getHook(r.processedAuctionHooks, id)
}

func (r *hookRepository) GetBidRequestHook(id string) (hookstage.BidRequest, bool) {
	return getHook(r.bidRequestHooks, id)
}

func (r *hookRepository) GetRawBidResponseHook(id string) (hookstage.RawBidResponse, bool) {
	return getHook(r.rawBidResponseHooks, id)
}

func (r *hookRepository) GetAllProcessedBidResponsesHook(id string) (hookstage.AllProcessedBidResponses, bool) {
	return getHook(r.allProcessedBidResponseHooks, id)
}

func (r *hookRepository) GetAuctionResponseHook(id string) (hookstage.AuctionResponse, bool) {
	return getHook(r.auctionResponseHooks, id)
}

func (r *hookRepository) add(id string, hook interface{}) (err error) {
	var isCompatible bool

	if h, ok := hook.(hookstage.Entrypoint); ok {
		isCompatible = true
		if r.entrypointHooks, err = addHook(r.entrypointHooks, h, id); err != nil {
			return
		}
	}

	if h, ok := hook.(hookstage.RawAuction); ok {
		isCompatible = true
		if r.rawAuctionHooks, err = addHook(r.rawAuctionHooks, h, id); err != nil {
			return
		}
	}

	if h, ok := hook.(hookstage.ProcessedAuction); ok {
		isCompatible = true
		if r.processedAuctionHooks, err = addHook(r.processedAuctionHooks, h, id); err != nil {
			return
		}
	}

	if h, ok := hook.(hookstage.BidRequest); ok {
		isCompatible = true
		if r.bidRequestHooks, err = addHook(r.bidRequestHooks, h, id); err != nil {
			return
		}
	}

	if h, ok := hook.(hookstage.RawBidResponse); ok {
		isCompatible = true
		if r.rawBidResponseHooks, err = addHook(r.rawBidResponseHooks, h, id); err != nil {
			return
		}
	}

	if h, ok := hook.(hookstage.AllProcessedBidResponses); ok {
		isCompatible = true
		if r.allProcessedBidResponseHooks, err = addHook(r.allProcessedBidResponseHooks, h, id); err != nil {
			return
		}
	}

	if h, ok := hook.(hookstage.AuctionResponse); ok {
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
