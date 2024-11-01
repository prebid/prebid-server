package hooks

import (
	"fmt"

	"github.com/prebid/prebid-server/v3/hooks/hookstage"
)

// HookRepository is the interface that exposes methods
// that return instance of the certain hook interface.
//
// Each method accepts hook ID and returns hook interface
// registered under this ID and true if hook found
// otherwise nil value returned with the false,
// indicating not found hook for this ID.
type HookRepository interface {
	GetEntrypointHook(id string) (hookstage.Entrypoint, bool)
	GetRawAuctionHook(id string) (hookstage.RawAuctionRequest, bool)
	GetProcessedAuctionHook(id string) (hookstage.ProcessedAuctionRequest, bool)
	GetBidderRequestHook(id string) (hookstage.BidderRequest, bool)
	GetRawBidderResponseHook(id string) (hookstage.RawBidderResponse, bool)
	GetAllProcessedBidResponsesHook(id string) (hookstage.AllProcessedBidResponses, bool)
	GetAuctionResponseHook(id string) (hookstage.AuctionResponse, bool)
}

// NewHookRepository returns a new instance of the HookRepository interface.
//
// The hooks argument represents a mapping of hook IDs to types
// implementing at least one of the available hook interfaces, see [hookstage] pkg.
//
// Error returned if provided interface doesn't implement any hook interface
// or hook with same ID already exists.
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
	rawAuctionHooks              map[string]hookstage.RawAuctionRequest
	processedAuctionHooks        map[string]hookstage.ProcessedAuctionRequest
	bidderRequestHooks           map[string]hookstage.BidderRequest
	rawBidderResponseHooks       map[string]hookstage.RawBidderResponse
	allProcessedBidResponseHooks map[string]hookstage.AllProcessedBidResponses
	auctionResponseHooks         map[string]hookstage.AuctionResponse
}

func (r *hookRepository) GetEntrypointHook(id string) (hookstage.Entrypoint, bool) {
	return getHook(r.entrypointHooks, id)
}

func (r *hookRepository) GetRawAuctionHook(id string) (hookstage.RawAuctionRequest, bool) {
	return getHook(r.rawAuctionHooks, id)
}

func (r *hookRepository) GetProcessedAuctionHook(id string) (hookstage.ProcessedAuctionRequest, bool) {
	return getHook(r.processedAuctionHooks, id)
}

func (r *hookRepository) GetBidderRequestHook(id string) (hookstage.BidderRequest, bool) {
	return getHook(r.bidderRequestHooks, id)
}

func (r *hookRepository) GetRawBidderResponseHook(id string) (hookstage.RawBidderResponse, bool) {
	return getHook(r.rawBidderResponseHooks, id)
}

func (r *hookRepository) GetAllProcessedBidResponsesHook(id string) (hookstage.AllProcessedBidResponses, bool) {
	return getHook(r.allProcessedBidResponseHooks, id)
}

func (r *hookRepository) GetAuctionResponseHook(id string) (hookstage.AuctionResponse, bool) {
	return getHook(r.auctionResponseHooks, id)
}

func (r *hookRepository) add(id string, hook interface{}) error {
	var hasAnyHooks bool
	var err error

	if h, ok := hook.(hookstage.Entrypoint); ok {
		hasAnyHooks = true
		if r.entrypointHooks, err = addHook(r.entrypointHooks, h, id); err != nil {
			return err
		}
	}

	if h, ok := hook.(hookstage.RawAuctionRequest); ok {
		hasAnyHooks = true
		if r.rawAuctionHooks, err = addHook(r.rawAuctionHooks, h, id); err != nil {
			return err
		}
	}

	if h, ok := hook.(hookstage.ProcessedAuctionRequest); ok {
		hasAnyHooks = true
		if r.processedAuctionHooks, err = addHook(r.processedAuctionHooks, h, id); err != nil {
			return err
		}
	}

	if h, ok := hook.(hookstage.BidderRequest); ok {
		hasAnyHooks = true
		if r.bidderRequestHooks, err = addHook(r.bidderRequestHooks, h, id); err != nil {
			return err
		}
	}

	if h, ok := hook.(hookstage.RawBidderResponse); ok {
		hasAnyHooks = true
		if r.rawBidderResponseHooks, err = addHook(r.rawBidderResponseHooks, h, id); err != nil {
			return err
		}
	}

	if h, ok := hook.(hookstage.AllProcessedBidResponses); ok {
		hasAnyHooks = true
		if r.allProcessedBidResponseHooks, err = addHook(r.allProcessedBidResponseHooks, h, id); err != nil {
			return err
		}
	}

	if h, ok := hook.(hookstage.AuctionResponse); ok {
		hasAnyHooks = true
		if r.auctionResponseHooks, err = addHook(r.auctionResponseHooks, h, id); err != nil {
			return err
		}
	}

	if !hasAnyHooks {
		return fmt.Errorf(`hook "%s" does not implement any supported hook interface`, id)
	}

	return nil
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
