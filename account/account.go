package account

import (
	"context"
	"encoding/json"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/metrics"
	"github.com/prebid/prebid-server/stored_requests"
)

// GetAccount looks up the config.Account object referenced by the given accountID, with access rules applied
func GetAccount(ctx context.Context, cfg *config.Configuration, fetcher stored_requests.AccountFetcher, accountID string) (account *config.Account, errs []error) {
	// Check BlacklistedAcctMap until we have deprecated it
	if _, found := cfg.BlacklistedAcctMap[accountID]; found {
		return nil, []error{&errortypes.BlacklistedAcct{
			Message: fmt.Sprintf("Prebid-server has disabled Account ID: %s, please reach out to the prebid server host.", accountID),
		}}
	}
	if cfg.AccountRequired && accountID == metrics.PublisherUnknown {
		return nil, []error{&errortypes.AcctRequired{
			Message: fmt.Sprintf("Prebid-server has been configured to discard requests without a valid Account ID. Please reach out to the prebid server host."),
		}}
	}
	if accountJSON, accErrs := fetcher.FetchAccount(ctx, accountID); len(accErrs) > 0 || accountJSON == nil {
		// accountID does not reference a valid account
		for _, e := range accErrs {
			if _, ok := e.(stored_requests.NotFoundError); !ok {
				errs = append(errs, e)
			}
		}
		if cfg.AccountRequired && cfg.AccountDefaults.Disabled {
			errs = append(errs, &errortypes.AcctRequired{
				Message: fmt.Sprintf("Prebid-server could not verify the Account ID. Please reach out to the prebid server host."),
			})
			return nil, errs
		}
		// Make a copy of AccountDefaults instead of taking a reference,
		// to preserve original accountID in case is needed to check NonStandardPublisherMap
		pubAccount := cfg.AccountDefaults
		pubAccount.ID = accountID
		account = &pubAccount
	} else {
		// accountID resolved to a valid account, merge with AccountDefaults for a complete config
		account = &config.Account{}
		completeJSON, err := jsonpatch.MergePatch(cfg.AccountDefaultsJSON(), accountJSON)
		if err == nil {
			err = json.Unmarshal(completeJSON, account)
		}
		if err != nil {
			errs = append(errs, err)
			return nil, errs
		}
		// Fill in ID if needed, so it can be left out of account definition
		if len(account.ID) == 0 {
			account.ID = accountID
		}
	}
	if account.Disabled {
		errs = append(errs, &errortypes.BlacklistedAcct{
			Message: fmt.Sprintf("Prebid-server has disabled Account ID: %s, please reach out to the prebid server host.", accountID),
		})
		return nil, errs
	}
	return account, nil
}
