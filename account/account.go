package account

import (
	"context"
	"fmt"

	"github.com/prebid/go-gdpr/consentconstants"

	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/metrics"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/stored_requests"
	"github.com/prebid/prebid-server/v4/util/iputil"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

// TypedAccountFetcher is an optional interface that a fetcher may implement to
// return a fully-derived, immutable *config.Account directly, skipping the
// per-request JSON unmarshal + defaults-merge + derive done on the legacy path.
// The Fetchers 2.0 (cachekit) account fetcher implements this; when present,
// GetAccount uses it. Legacy fetchers do not, and take the JSON path unchanged.
type TypedAccountFetcher interface {
	FetchAccountTyped(ctx context.Context, accountID string) (*config.Account, []error)
}

// GetAccount looks up the config.Account object referenced by the given accountID, with access rules applied
func GetAccount(ctx context.Context, cfg *config.Configuration, fetcher stored_requests.AccountFetcher, accountID string, me metrics.MetricsEngine) (account *config.Account, errs []error) {
	if cfg.AccountRequired && accountID == metrics.PublisherUnknown {
		return nil, []error{&errortypes.AcctRequired{
			Message: "Prebid-server has been configured to discard requests without a valid Account ID. Please reach out to the prebid server host.",
		}}
	}

	if typed, ok := fetcher.(TypedAccountFetcher); ok {
		return getAccountTyped(ctx, cfg, typed, accountID)
	}
	return getAccountJSON(ctx, cfg, fetcher, accountID)
}

// getAccountJSON is the legacy account resolution path: it fetches raw
// (defaults-merged) JSON, unmarshals it, unpacks DSA defaults and computes the
// derived config on every call.
func getAccountJSON(ctx context.Context, cfg *config.Configuration, fetcher stored_requests.AccountFetcher, accountID string) (account *config.Account, errs []error) {
	if accountJSON, accErrs := fetcher.FetchAccount(ctx, cfg.AccountDefaultsJSON(), accountID); len(accErrs) > 0 || accountJSON == nil {
		// accountID does not reference a valid account
		for _, e := range accErrs {
			if _, ok := e.(stored_requests.NotFoundError); !ok {
				errs = append(errs, e)
			}
		}
		if cfg.AccountRequired && cfg.AccountDefaults.Disabled {
			errs = append(errs, &errortypes.AcctRequired{
				Message: "Prebid-server could not verify the Account ID. Please reach out to the prebid server host.",
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
		if err := jsonutil.UnmarshalValid(accountJSON, account); err != nil {
			return nil, []error{&errortypes.MalformedAcct{
				Message: fmt.Sprintf("The prebid-server account config for account id \"%s\" is malformed. Please reach out to the prebid server host.", accountID),
			}}
		}
		if err := config.UnpackDSADefault(account.Privacy.DSA); err != nil {
			return nil, []error{&errortypes.MalformedAcct{
				Message: fmt.Sprintf("The prebid-server account config DSA for account id \"%s\" is malformed. Please reach out to the prebid server host.", accountID),
			}}
		}

		// Fill in ID if needed, so it can be left out of account definition
		if len(account.ID) == 0 {
			account.ID = accountID
		}

		// Set derived fields
		setDerivedConfig(account)
	}
	if account.Disabled {
		errs = append(errs, &errortypes.AccountDisabled{
			Message: fmt.Sprintf("Prebid-server has disabled Account ID: %s, please reach out to the prebid server host.", accountID),
		})
		return nil, errs
	}

	applyIPMaskingDefaults(account)
	return account, nil
}

// getAccountTyped is the Fetchers 2.0 account resolution path. The fetcher returns
// a fully-derived, immutable *config.Account (unmarshal + DSA + derive + IP masking
// were done once, at cache insert). This path only applies the not-found fallback
// and the per-request access gating.
func getAccountTyped(ctx context.Context, cfg *config.Configuration, fetcher TypedAccountFetcher, accountID string) (account *config.Account, errs []error) {
	fetched, accErrs := fetcher.FetchAccountTyped(ctx, accountID)
	if len(accErrs) > 0 {
		// A malformed account is a hard error, mirroring the legacy path where the
		// unmarshal/DSA failure returns immediately rather than falling back to defaults.
		for _, e := range accErrs {
			if _, ok := e.(*errortypes.MalformedAcct); ok {
				return nil, accErrs
			}
		}
		// Otherwise (not-found, or a swallowed backend error) fall through to the
		// AccountDefaults fallback, matching the legacy not-found branch.
		fetched = nil
	}
	if fetched == nil {
		if cfg.AccountRequired && cfg.AccountDefaults.Disabled {
			return nil, []error{&errortypes.AcctRequired{
				Message: "Prebid-server could not verify the Account ID. Please reach out to the prebid server host.",
			}}
		}
		// Make a copy of AccountDefaults instead of taking a reference,
		// to preserve original accountID in case is needed to check NonStandardPublisherMap
		pubAccount := cfg.AccountDefaults
		pubAccount.ID = accountID
		account = &pubAccount
	} else {
		// Fully-derived, immutable account returned straight from the cache.
		account = fetched
	}
	if account.Disabled {
		errs = append(errs, &errortypes.AccountDisabled{
			Message: fmt.Sprintf("Prebid-server has disabled Account ID: %s, please reach out to the prebid server host.", accountID),
		})
		return nil, errs
	}

	// No-op for the cached (already-masked) account; corrects the defaults copy.
	applyIPMaskingDefaults(account)
	return account, nil
}

// applyIPMaskingDefaults falls back to the default IPv4/IPv6 masking bit sizes when
// the configured values are invalid. Re-running it on an already-valid account is a
// read-only no-op.
func applyIPMaskingDefaults(account *config.Account) {
	if ipV6Err := account.Privacy.IPv6Config.Validate(nil); len(ipV6Err) > 0 {
		account.Privacy.IPv6Config.AnonKeepBits = iputil.IPv6DefaultMaskingBitSize
	}

	if ipV4Err := account.Privacy.IPv4Config.Validate(nil); len(ipV4Err) > 0 {
		account.Privacy.IPv4Config.AnonKeepBits = iputil.IPv4DefaultMaskingBitSize
	}
}

// TCF2Enforcements maps enforcement algo string values to their integer representation and is
// used to limit string compares
var TCF2Enforcements = map[string]config.TCF2EnforcementAlgo{
	config.TCF2EnforceAlgoBasic: config.TCF2BasicEnforcement,
	config.TCF2EnforceAlgoFull:  config.TCF2FullEnforcement,
}

// setDerivedConfig modifies an account object by setting fields derived from other fields set in the account configuration
func setDerivedConfig(account *config.Account) {
	account.GDPR.PurposeConfigs = map[consentconstants.Purpose]*config.AccountGDPRPurpose{
		1:  &account.GDPR.Purpose1,
		2:  &account.GDPR.Purpose2,
		3:  &account.GDPR.Purpose3,
		4:  &account.GDPR.Purpose4,
		5:  &account.GDPR.Purpose5,
		6:  &account.GDPR.Purpose6,
		7:  &account.GDPR.Purpose7,
		8:  &account.GDPR.Purpose8,
		9:  &account.GDPR.Purpose9,
		10: &account.GDPR.Purpose10,
	}

	for _, pc := range account.GDPR.PurposeConfigs {
		// To minimize the number of string compares per request, we set the integer representation
		// of the enforcement algorithm on each purpose config
		pc.EnforceAlgoID = config.TCF2UndefinedEnforcement
		if algo, exists := TCF2Enforcements[pc.EnforceAlgo]; exists {
			pc.EnforceAlgoID = algo
		}

		// To look for a purpose's vendor exceptions in O(1) time, for each purpose we fill this hash table with bidders
		// located in the VendorExceptions field of the GDPR.PurposeX struct
		if pc.VendorExceptions == nil {
			continue
		}
		pc.VendorExceptionMap = make(map[string]struct{})
		for _, v := range pc.VendorExceptions {
			pc.VendorExceptionMap[v] = struct{}{}
		}
	}

	// To look for special feature 1's vendor exceptions in O(1) time, we fill this hash table with bidders
	// located in the VendorExceptions field
	if account.GDPR.SpecialFeature1.VendorExceptions != nil {
		account.GDPR.SpecialFeature1.VendorExceptionMap = make(map[openrtb_ext.BidderName]struct{})

		for _, v := range account.GDPR.SpecialFeature1.VendorExceptions {
			account.GDPR.SpecialFeature1.VendorExceptionMap[v] = struct{}{}
		}
	}

	// To look for basic enforcement vendors in O(1) time, we fill this hash table with bidders
	// located in the BasicEnforcementVendors field
	if account.GDPR.BasicEnforcementVendors != nil {
		account.GDPR.BasicEnforcementVendorsMap = make(map[string]struct{})

		for _, v := range account.GDPR.BasicEnforcementVendors {
			account.GDPR.BasicEnforcementVendorsMap[v] = struct{}{}
		}
	}
}
