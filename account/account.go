package account

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/buger/jsonparser"
	"github.com/prebid/go-gdpr/consentconstants"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/metrics"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/stored_requests"
	jsonpatch "gopkg.in/evanphx/json-patch.v4"
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

			// this logic exists for backwards compatibility. If the initial unmarshal fails above, we attempt to
			// resolve it by converting the GDPR enforce purpose fields and then attempting an unmarshal again before
			// declaring a malformed account error.
			// unmarshal fetched account to determine if it is well-formed
			if _, ok := err.(*json.UnmarshalTypeError); ok {
				// attempt to convert deprecated GDPR enforce purpose fields to resolve issue
				completeJSON, err = ConvertGDPREnforcePurposeFields(completeJSON)
				// unmarshal again to check if unmarshal error still exists after GDPR field conversion
				err = json.Unmarshal(completeJSON, account)

				if _, ok := err.(*json.UnmarshalTypeError); ok {
					return nil, []error{&errortypes.MalformedAcct{
						Message: fmt.Sprintf("The prebid-server account config for account id \"%s\" is malformed. Please reach out to the prebid server host.", accountID),
					}}
				}
			}
		}

		if err != nil {
			errs = append(errs, err)
			return nil, errs
		}
		// Fill in ID if needed, so it can be left out of account definition
		if len(account.ID) == 0 {
			account.ID = accountID
		}

		// Set derived fields
		setDerivedConfig(account)
	}
	if account.Disabled {
		errs = append(errs, &errortypes.BlacklistedAcct{
			Message: fmt.Sprintf("Prebid-server has disabled Account ID: %s, please reach out to the prebid server host.", accountID),
		})
		return nil, errs
	}
	return account, nil
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

	// To look for a purpose's vendor exceptions in O(1) time, for each purpose we fill this hash table with bidders
	// located in the VendorExceptions field of the GDPR.PurposeX struct
	for _, pc := range account.GDPR.PurposeConfigs {
		if pc.VendorExceptions == nil {
			continue
		}
		pc.VendorExceptionMap = make(map[openrtb_ext.BidderName]struct{})
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

// PatchAccount represents the GDPR portion of a publisher account configuration that can be mutated
// for backwards compatibility reasons
type PatchAccount struct {
	GDPR map[string]*PatchAccountGDPRPurpose `json:"gdpr"`
}

// PatchAccountGDPRPurpose represents account-specific GDPR purpose configuration data that can be mutated
// for backwards compatibility reasons
type PatchAccountGDPRPurpose struct {
	EnforceAlgo    string `json:"enforce_algo,omitempty"`
	EnforcePurpose *bool  `json:"enforce_purpose,omitempty"`
}

// ConvertGDPREnforcePurposeFields is responsible for ensuring account GDPR config backwards compatibility
// given the recent type change of gdpr.purpose{1-10}.enforce_purpose from a string to a bool. This function
// iterates over each GDPR purpose config and sets enforce_purpose and enforce_algo to the appropriate
// bool and string values respectively if enforce_purpose is a string and enforce_algo is not set
func ConvertGDPREnforcePurposeFields(config []byte) (newConfig []byte, err error) {
	gdprJSON, _, _, err := jsonparser.Get(config, "gdpr")
	if err != nil && err == jsonparser.KeyPathNotFoundError {
		return config, nil
	}
	if err != nil {
		return nil, err
	}

	newAccount := PatchAccount{
		GDPR: map[string]*PatchAccountGDPRPurpose{},
	}

	for i := 1; i <= 10; i++ {
		purposeName := fmt.Sprintf("purpose%d", i)

		enforcePurpose, purposeDataType, _, err := jsonparser.Get(gdprJSON, purposeName, "enforce_purpose")
		if err != nil && err == jsonparser.KeyPathNotFoundError {
			continue
		}
		if err != nil {
			return nil, err
		}
		if purposeDataType != jsonparser.String {
			continue
		}

		_, _, _, err = jsonparser.Get(gdprJSON, purposeName, "enforce_algo")
		if err != nil && err != jsonparser.KeyPathNotFoundError {
			return nil, err
		}
		if err == nil {
			continue
		}

		newEnforcePurpose := false
		if string(enforcePurpose) == "full" {
			newEnforcePurpose = true
		}

		newAccount.GDPR[purposeName] = &PatchAccountGDPRPurpose{
			EnforceAlgo:    "full",
			EnforcePurpose: &newEnforcePurpose,
		}
	}

	patchConfig, err := json.Marshal(newAccount)
	if err != nil {
		return nil, err
	}

	newConfig, err = jsonpatch.MergePatch(config, patchConfig)
	if err != nil {
		return nil, err
	}

	return newConfig, nil
}
