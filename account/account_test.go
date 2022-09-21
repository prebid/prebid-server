package account

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/buger/jsonparser"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/metrics"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/stored_requests"
	"github.com/stretchr/testify/assert"
)

var mockAccountData = map[string]json.RawMessage{
	"valid_acct":        json.RawMessage(`{"disabled":false}`),
	"disabled_acct":     json.RawMessage(`{"disabled":true}`),
	"malformed_acct":    json.RawMessage(`{"disabled":"invalid type"}`),
	"gdpr_convert_acct": json.RawMessage(`{"disabled":false,"gdpr":{"purpose5":{"enforce_purpose":"full"}}}`),
}

type mockAccountFetcher struct {
}

func (af mockAccountFetcher) FetchAccount(ctx context.Context, accountID string) (json.RawMessage, []error) {
	if account, ok := mockAccountData[accountID]; ok {
		return account, nil
	}
	return nil, []error{stored_requests.NotFoundError{ID: accountID, DataType: "Account"}}
}

func TestGetAccount(t *testing.T) {
	unknown := metrics.PublisherUnknown
	testCases := []struct {
		accountID string
		// account_required
		required bool
		// account_defaults.disabled
		disabled bool
		// expected error, or nil if account should be found
		err error
	}{
		// Blacklisted account is always rejected even in permissive setup
		{accountID: "bad_acct", required: false, disabled: false, err: &errortypes.BlacklistedAcct{}},

		// empty pubID
		{accountID: unknown, required: false, disabled: false, err: nil},
		{accountID: unknown, required: true, disabled: false, err: &errortypes.AcctRequired{}},
		{accountID: unknown, required: false, disabled: true, err: &errortypes.BlacklistedAcct{}},
		{accountID: unknown, required: true, disabled: true, err: &errortypes.AcctRequired{}},

		// pubID given but is not a valid host account (does not exist)
		{accountID: "doesnt_exist_acct", required: false, disabled: false, err: nil},
		{accountID: "doesnt_exist_acct", required: true, disabled: false, err: nil},
		{accountID: "doesnt_exist_acct", required: false, disabled: true, err: &errortypes.BlacklistedAcct{}},
		{accountID: "doesnt_exist_acct", required: true, disabled: true, err: &errortypes.AcctRequired{}},

		// pubID given and matches a valid host account with Disabled: false
		{accountID: "valid_acct", required: false, disabled: false, err: nil},
		{accountID: "valid_acct", required: true, disabled: false, err: nil},
		{accountID: "valid_acct", required: false, disabled: true, err: nil},
		{accountID: "valid_acct", required: true, disabled: true, err: nil},

		// pubID given and matches a host account explicitly disabled (Disabled: true on account json)
		{accountID: "disabled_acct", required: false, disabled: false, err: &errortypes.BlacklistedAcct{}},
		{accountID: "disabled_acct", required: true, disabled: false, err: &errortypes.BlacklistedAcct{}},
		{accountID: "disabled_acct", required: false, disabled: true, err: &errortypes.BlacklistedAcct{}},
		{accountID: "disabled_acct", required: true, disabled: true, err: &errortypes.BlacklistedAcct{}},

		// pubID given and matches a host account with Disabled: false and GDPR purpose data to convert
		{accountID: "gdpr_convert_acct", required: false, disabled: false, err: nil},
		{accountID: "gdpr_convert_acct", required: true, disabled: false, err: nil},
		{accountID: "gdpr_convert_acct", required: false, disabled: true, err: nil},
		{accountID: "gdpr_convert_acct", required: true, disabled: true, err: nil},

		// pubID given and matches a host account that has a malformed config
		{accountID: "malformed_acct", required: false, disabled: false, err: &errortypes.MalformedAcct{}},
		{accountID: "malformed_acct", required: true, disabled: false, err: &errortypes.MalformedAcct{}},
		{accountID: "malformed_acct", required: false, disabled: true, err: &errortypes.MalformedAcct{}},
		{accountID: "malformed_acct", required: true, disabled: true, err: &errortypes.MalformedAcct{}},

		// account not provided (does not exist)
		{accountID: "", required: false, disabled: false, err: nil},
		{accountID: "", required: true, disabled: false, err: nil},
		{accountID: "", required: false, disabled: true, err: &errortypes.BlacklistedAcct{}},
		{accountID: "", required: true, disabled: true, err: &errortypes.AcctRequired{}},
	}

	for _, test := range testCases {
		description := fmt.Sprintf(`ID=%s/required=%t/disabled=%t`, test.accountID, test.required, test.disabled)
		t.Run(description, func(t *testing.T) {
			cfg := &config.Configuration{
				BlacklistedAcctMap: map[string]bool{"bad_acct": true},
				AccountRequired:    test.required,
				AccountDefaults:    config.Account{Disabled: test.disabled},
			}
			fetcher := &mockAccountFetcher{}
			assert.NoError(t, cfg.MarshalAccountDefaults())

			account, errors := GetAccount(context.Background(), cfg, fetcher, test.accountID)

			if test.err == nil {
				assert.Empty(t, errors)
				assert.Equal(t, test.accountID, account.ID, "account.ID must match requested ID")
				assert.Equal(t, false, account.Disabled, "returned account must not be disabled")
			} else {
				assert.NotEmpty(t, errors, "expected errors but got success")
				assert.Nil(t, account, "return account must be nil on error")
				assert.IsType(t, test.err, errors[0], "error is of unexpected type")
			}
		})
	}
}

func TestSetDerivedConfig(t *testing.T) {
	tests := []struct {
		description              string
		purpose1VendorExceptions []openrtb_ext.BidderName
		feature1VendorExceptions []openrtb_ext.BidderName
		basicEnforcementVendors  []string
	}{
		{
			description:              "Nil purpose 1 vendor exceptions",
			purpose1VendorExceptions: nil,
		},
		{
			description:              "One purpose 1 vendor exception",
			purpose1VendorExceptions: []openrtb_ext.BidderName{"appnexus"},
		},
		{
			description:              "Multiple purpose 1 vendor exceptions",
			purpose1VendorExceptions: []openrtb_ext.BidderName{"appnexus", "rubicon"},
		},
		{
			description:              "Nil feature 1 vendor exceptions",
			feature1VendorExceptions: nil,
		},
		{
			description:              "One feature 1 vendor exception",
			feature1VendorExceptions: []openrtb_ext.BidderName{"appnexus"},
		},
		{
			description:              "Multiple feature 1 vendor exceptions",
			feature1VendorExceptions: []openrtb_ext.BidderName{"appnexus", "rubicon"},
		},
		{
			description:             "Nil basic enforcement vendors",
			basicEnforcementVendors: nil,
		},
		{
			description:             "One basic enforcement vendor",
			basicEnforcementVendors: []string{"appnexus"},
		},
		{
			description:             "Multiple basic enforcement vendors",
			basicEnforcementVendors: []string{"appnexus", "rubicon"},
		},
	}

	for _, tt := range tests {
		account := config.Account{
			GDPR: config.AccountGDPR{
				Purpose1: config.AccountGDPRPurpose{
					VendorExceptions: tt.purpose1VendorExceptions,
				},
				SpecialFeature1: config.AccountGDPRSpecialFeature{
					VendorExceptions: tt.feature1VendorExceptions,
				},
				BasicEnforcementVendors: tt.basicEnforcementVendors,
			},
		}

		setDerivedConfig(&account)

		purpose1ExceptionMapKeys := make([]openrtb_ext.BidderName, 0)
		for k := range account.GDPR.Purpose1.VendorExceptionMap {
			purpose1ExceptionMapKeys = append(purpose1ExceptionMapKeys, k)
		}

		feature1ExceptionMapKeys := make([]openrtb_ext.BidderName, 0)
		for k := range account.GDPR.SpecialFeature1.VendorExceptionMap {
			feature1ExceptionMapKeys = append(feature1ExceptionMapKeys, k)
		}

		basicEnforcementMapKeys := make([]string, 0)
		for k := range account.GDPR.BasicEnforcementVendorsMap {
			basicEnforcementMapKeys = append(basicEnforcementMapKeys, k)
		}

		assert.ElementsMatch(t, purpose1ExceptionMapKeys, tt.purpose1VendorExceptions, tt.description)
		assert.ElementsMatch(t, feature1ExceptionMapKeys, tt.feature1VendorExceptions, tt.description)
		assert.ElementsMatch(t, basicEnforcementMapKeys, tt.basicEnforcementVendors, tt.description)
	}
}

func TestConvertGDPREnforcePurposeFields(t *testing.T) {
	enforcePurposeNo := `{"enforce_purpose":"no"}`
	enforcePurposeNoMapped := `{"enforce_algo":"full", "enforce_purpose":false}`
	enforcePurposeFull := `{"enforce_purpose":"full"}`
	enforcePurposeFullMapped := `{"enforce_algo":"full", "enforce_purpose":true}`

	tests := []struct {
		description string
		giveConfig  []byte
		wantConfig  []byte
		wantErr     error
	}{
		{
			description: "config is nil",
			giveConfig:  nil,
			wantConfig:  nil,
			wantErr:     nil,
		},
		{
			description: "config is empty - no gdpr key",
			giveConfig:  []byte(``),
			wantConfig:  []byte(``),
			wantErr:     nil,
		},
		{
			description: "gdpr present but empty",
			giveConfig:  []byte(`{"gdpr": {}}`),
			wantConfig:  []byte(`{"gdpr": {}}`),
			wantErr:     nil,
		},
		{
			description: "gdpr present but invalid",
			giveConfig:  []byte(`{"gdpr": {`),
			wantConfig:  nil,
			wantErr:     jsonparser.MalformedJsonError,
		},
		{
			description: "gdpr.purpose1 present but empty",
			giveConfig:  []byte(`{"gdpr":{"purpose1":{}}}`),
			wantConfig:  []byte(`{"gdpr":{"purpose1":{}}}`),
			wantErr:     nil,
		},
		{
			description: "gdpr.purpose1.enforce_purpose is set to bool",
			giveConfig:  []byte(`{"gdpr":{"purpose1":{"enforce_purpose":true}}}`),
			wantConfig:  []byte(`{"gdpr":{"purpose1":{"enforce_purpose":true}}}`),
			wantErr:     nil,
		},
		{
			description: "gdpr.purpose1.enforce_purpose is set to string full",
			giveConfig:  []byte(`{"gdpr":{"purpose1":{"enforce_purpose":"full"}}}`),
			wantConfig:  []byte(`{"gdpr":{"purpose1":{"enforce_algo":"full", "enforce_purpose":true}}}`),
			wantErr:     nil,
		},
		{
			description: "gdpr.purpose1.enforce_purpose is set to string no",
			giveConfig:  []byte(`{"gdpr":{"purpose1":{"enforce_purpose":"no"}}}`),
			wantConfig:  []byte(`{"gdpr":{"purpose1":{"enforce_algo":"full", "enforce_purpose":false}}}`),
			wantErr:     nil,
		},
		{
			description: "gdpr.purpose1.enforce_purpose is set to string no and other fields are untouched during conversion",
			giveConfig:  []byte(`{"gdpr":{"purpose1":{"enforce_purpose":"no", "enforce_vendors":true}}}`),
			wantConfig:  []byte(`{"gdpr":{"purpose1":{"enforce_algo":"full", "enforce_purpose":false, "enforce_vendors":true}}}`),
			wantErr:     nil,
		},
		{
			description: "gdpr.purpose1.enforce_purpose is set but invalid",
			giveConfig:  []byte(`{"gdpr":{"purpose1":{"enforce_purpose":}}}`),
			wantConfig:  nil,
			wantErr:     jsonparser.MalformedJsonError,
		},
		{
			description: "gdpr.purpose1.enforce_algo is set",
			giveConfig:  []byte(`{"gdpr":{"purpose1":{"enforce_algo":"full"}}}`),
			wantConfig:  []byte(`{"gdpr":{"purpose1":{"enforce_algo":"full"}}}`),
			wantErr:     nil,
		},
		{
			description: "gdpr.purpose1.enforce_purpose is set to string and enforce_algo is set",
			giveConfig:  []byte(`{"gdpr":{"purpose1":{"enforce_algo":"full", "enforce_purpose":"full"}}}`),
			wantConfig:  []byte(`{"gdpr":{"purpose1":{"enforce_algo":"full", "enforce_purpose":"full"}}}`),
			wantErr:     nil,
		},
		{
			description: "gdpr.purpose1.enforce_purpose is set to string and enforce_algo is set but invalid",
			giveConfig:  []byte(`{"gdpr":{"purpose1":{"enforce_algo":, "enforce_purpose":"full"}}}`),
			wantConfig:  nil,
			wantErr:     jsonparser.MalformedJsonError,
		},
		{
			description: "gdpr.purpose{1-10}.enforce_purpose are set to strings no and full alternating",
			giveConfig: []byte(`{"gdpr":{` +
				`"purpose1":` + enforcePurposeNo +
				`,"purpose2":` + enforcePurposeFull +
				`,"purpose3":` + enforcePurposeNo +
				`,"purpose4":` + enforcePurposeFull +
				`,"purpose5":` + enforcePurposeNo +
				`,"purpose6":` + enforcePurposeFull +
				`,"purpose7":` + enforcePurposeNo +
				`,"purpose8":` + enforcePurposeFull +
				`,"purpose9":` + enforcePurposeNo +
				`,"purpose10":` + enforcePurposeFull +
				`}}`),
			wantConfig: []byte(`{"gdpr":{` +
				`"purpose1":` + enforcePurposeNoMapped +
				`,"purpose2":` + enforcePurposeFullMapped +
				`,"purpose3":` + enforcePurposeNoMapped +
				`,"purpose4":` + enforcePurposeFullMapped +
				`,"purpose5":` + enforcePurposeNoMapped +
				`,"purpose6":` + enforcePurposeFullMapped +
				`,"purpose7":` + enforcePurposeNoMapped +
				`,"purpose8":` + enforcePurposeFullMapped +
				`,"purpose9":` + enforcePurposeNoMapped +
				`,"purpose10":` + enforcePurposeFullMapped +
				`}}`),
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		newConfig, err := ConvertGDPREnforcePurposeFields(tt.giveConfig)
		if tt.wantErr != nil {
			assert.Error(t, err, tt.description)
		}

		if len(tt.wantConfig) == 0 {
			assert.Equal(t, tt.wantConfig, newConfig, tt.description)
		} else {
			assert.JSONEq(t, string(tt.wantConfig), string(newConfig), tt.description)
		}
	}
}
