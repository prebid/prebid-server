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
func GetAccount(ctx context.Context, cfg *config.Configuration, fetcher stored_requests.AccountFetcher, accountID string, me metrics.MetricsEngine) (account *config.Account, errs []error) {
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
	if accountJSON, accErrs := fetcher.FetchAccount(ctx, cfg.AccountDefaultsJSON(), accountID); len(accErrs) > 0 || accountJSON == nil {
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
		err := json.Unmarshal(accountJSON, account)

		// this logic exists for backwards compatibility. If the initial unmarshal fails above, we attempt to
		// resolve it by converting the GDPR enforce purpose fields and then attempting an unmarshal again before
		// declaring a malformed account error.
		// unmarshal fetched account to determine if it is well-formed
		var deprecatedPurposeFields []string
		if _, ok := err.(*json.UnmarshalTypeError); ok {
			// attempt to convert deprecated GDPR enforce purpose fields to resolve issue
			accountJSON, err, deprecatedPurposeFields = ConvertGDPREnforcePurposeFields(accountJSON)
			// unmarshal again to check if unmarshal error still exists after GDPR field conversion
			err = json.Unmarshal(accountJSON, account)

			if _, ok := err.(*json.UnmarshalTypeError); ok {
				return nil, []error{&errortypes.MalformedAcct{
					Message: fmt.Sprintf("The prebid-server account config for account id \"%s\" is malformed. Please reach out to the prebid server host.", accountID),
				}}
			}
		}
		usingGDPRChannelEnabled := useGDPRChannelEnabled(account)
		usingCCPAChannelEnabled := useCCPAChannelEnabled(account)

		if usingGDPRChannelEnabled {
			me.RecordAccountGDPRChannelEnabledWarning(accountID)
		}
		if usingCCPAChannelEnabled {
			me.RecordAccountCCPAChannelEnabledWarning(accountID)
		}
		for _, purposeName := range deprecatedPurposeFields {
			me.RecordAccountGDPRPurposeWarning(accountID, purposeName)
		}
		if len(deprecatedPurposeFields) > 0 || usingGDPRChannelEnabled || usingCCPAChannelEnabled {
			me.RecordAccountUpgradeStatus(accountID)
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

	errs = account.Privacy.IPv6Config.Validate(errs)
	if len(errs) > 0 {
		return nil, errs
	}

	errs = account.Privacy.IPv4Config.Validate(errs)
	if len(errs) > 0 {
		return nil, errs
	}

	// set the value of events.enabled field based on deprecated events_enabled field and ensure backward compatibility
	deprecateEventsEnabledField(account)

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
func ConvertGDPREnforcePurposeFields(config []byte) (newConfig []byte, err error, deprecatedPurposeFields []string) {
	gdprJSON, _, _, err := jsonparser.Get(config, "gdpr")
	if err != nil && err == jsonparser.KeyPathNotFoundError {
		return config, nil, deprecatedPurposeFields
	}
	if err != nil {
		return nil, err, deprecatedPurposeFields
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
			return nil, err, deprecatedPurposeFields
		}
		if purposeDataType != jsonparser.String {
			continue
		} else {
			deprecatedPurposeFields = append(deprecatedPurposeFields, purposeName)
		}

		_, _, _, err = jsonparser.Get(gdprJSON, purposeName, "enforce_algo")
		if err != nil && err != jsonparser.KeyPathNotFoundError {
			return nil, err, deprecatedPurposeFields
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
		return nil, err, deprecatedPurposeFields
	}

	newConfig, err = jsonpatch.MergePatch(config, patchConfig)
	if err != nil {
		return nil, err, deprecatedPurposeFields
	}

	return newConfig, nil, deprecatedPurposeFields
}

func useGDPRChannelEnabled(account *config.Account) bool {
	return account.GDPR.ChannelEnabled.IsSet() && !account.GDPR.IntegrationEnabled.IsSet()
}

func useCCPAChannelEnabled(account *config.Account) bool {
	return account.CCPA.ChannelEnabled.IsSet() && !account.CCPA.IntegrationEnabled.IsSet()
}

// deprecateEventsEnabledField is responsible for ensuring backwards compatibility of "events_enabled" field.
// This function favors "events.enabled" field over deprecated "events_enabled" field, if values for both are set.
// If only deprecated "events_enabled" field is set then it sets the same value to "events.enabled" field.
func deprecateEventsEnabledField(account *config.Account) {
	if account != nil {
		if account.Events.Enabled == nil {
			account.Events.Enabled = account.EventsEnabled
		}
		// assign the old value to the new value so old and new are always the same even though the new value is what is used in the application code.
		account.EventsEnabled = account.Events.Enabled
	}
}
