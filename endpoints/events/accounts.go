package events

import (
	"context"
	"encoding/json"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/stored_requests"
)

/**
 * Retrieves an account and merges it with AccountDefaults
 */
func GetAccount(ctx context.Context, cfg *config.Configuration, fetcher stored_requests.AccountFetcher, id string) (*config.Account, []error) {
	// account defaults is our baseline
	account := cfg.AccountDefaults
	account.ID = id

	// attempt to retrieve account details using an AccountFetcher
	if accountJSON, errs := fetcher.FetchAccount(ctx, id); len(errs) > 0 {
		// tolerate account not found/empty and use account defaults
		// if that's not the case return the errors that should be handler by the callers
		if !accountNotFound(errs) {
			return nil, errs
		}
	} else {
		// merge with account defaults
		// id resolved to a valid account, merge with AccountDefaults for a complete config
		completeJSON, err := jsonpatch.MergePatch(cfg.AccountDefaultsJSON(), accountJSON)
		if err == nil {
			err = json.Unmarshal(completeJSON, &account)
		}
		if err != nil {
			errs = append(errs, err)
			return nil, errs
		}
		account.ID = id
	}

	return &account, nil
}
