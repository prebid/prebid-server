package account

import (
	"context"
	"fmt"

	"github.com/prebid/go-gdpr/consentconstants"

	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/stored_requests"
	"github.com/prebid/prebid-server/v3/util/iputil"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

// GetAccount looks up the config.Account object referenced by the given accountID, with access rules applied
func GetAccount(ctx context.Context, cfg *config.Configuration, fetcher stored_requests.AccountFetcher, accountID string, me metrics.MetricsEngine) (account *config.Account, errs []error) {
	if cfg.AccountRequired && accountID == metrics.PublisherUnknown {
		return nil, []error{&errortypes.AcctRequired{
			Message: "Prebid-server has been configured to discard requests without a valid Account ID. Please reach out to the prebid server host.",
		}}
	}

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

	if ipV6Err := account.Privacy.IPv6Config.Validate(nil); len(ipV6Err) > 0 {
		account.Privacy.IPv6Config.AnonKeepBits = iputil.IPv6DefaultMaskingBitSize
	}

	if ipV4Err := account.Privacy.IPv4Config.Validate(nil); len(ipV4Err) > 0 {
		account.Privacy.IPv4Config.AnonKeepBits = iputil.IPv4DefaultMaskingBitSize
	}

	return account, nil
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
