package account

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/stored_requests"
	"github.com/prebid/prebid-server/v3/util/iputil"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	validDSA   = `{\"dsarequired\":1,\"pubrender\":2,\"transparency\":[{\"domain\":\"test.com\"}]}`
	invalidDSA = `{\"dsarequired\":\"invalid\",\"pubrender\":2,\"transparency\":[{\"domain\":\"test.com\"}]}`
)

var mockAccountData = map[string]json.RawMessage{
	"valid_acct":                json.RawMessage(`{"disabled":false}`),
	"valid_acct_dsa":            json.RawMessage(`{"disabled":false, "privacy": {"dsa": {"default": "` + validDSA + `"}}}`),
	"invalid_acct_dsa":          json.RawMessage(`{"disabled":false, "privacy": {"dsa": {"default": "` + invalidDSA + `"}}}`),
	"invalid_acct_ipv6_ipv4":    json.RawMessage(`{"disabled":false, "privacy": {"ipv6": {"anon_keep_bits": -32}, "ipv4": {"anon_keep_bits": -16}}}`),
	"disabled_acct":             json.RawMessage(`{"disabled":true}`),
	"malformed_acct":            json.RawMessage(`{"disabled":"invalid type"}`),
	"gdpr_channel_enabled_acct": json.RawMessage(`{"disabled":false,"gdpr":{"channel_enabled":{"amp":true}}}`),
	"ccpa_channel_enabled_acct": json.RawMessage(`{"disabled":false,"ccpa":{"channel_enabled":{"amp":true}}}`),
}

type mockAccountFetcher struct {
}

func (af mockAccountFetcher) FetchAccount(ctx context.Context, accountDefaultsJSON json.RawMessage, accountID string) (json.RawMessage, []error) {
	if account, ok := mockAccountData[accountID]; ok {
		return account, nil
	}
	return nil, []error{stored_requests.NotFoundError{ID: accountID, DataType: "Account"}}
}

func TestGetAccount(t *testing.T) {
	validDSA := &openrtb_ext.ExtRegsDSA{
		Required:  ptrutil.ToPtr[int8](1),
		PubRender: ptrutil.ToPtr[int8](2),
		Transparency: []openrtb_ext.ExtBidDSATransparency{
			{
				Domain: "test.com",
			},
		},
	}

	unknown := metrics.PublisherUnknown
	testCases := []struct {
		accountID string
		// account_required
		required bool
		// account_defaults.disabled
		disabled bool
		// checkDefaultIP indicates IPv6 and IPv6 should be set to default values
		wantDefaultIP bool
		wantDSA       *openrtb_ext.ExtRegsDSA
		// expected error, or nil if account should be found
		err error
	}{
		// empty pubID
		{accountID: unknown, required: false, disabled: false, err: nil},
		{accountID: unknown, required: true, disabled: false, err: &errortypes.AcctRequired{}},
		{accountID: unknown, required: false, disabled: true, err: &errortypes.AccountDisabled{}},
		{accountID: unknown, required: true, disabled: true, err: &errortypes.AcctRequired{}},

		// pubID given but is not a valid host account (does not exist)
		{accountID: "doesnt_exist_acct", required: false, disabled: false, err: nil},
		{accountID: "doesnt_exist_acct", required: true, disabled: false, err: nil},
		{accountID: "doesnt_exist_acct", required: false, disabled: true, err: &errortypes.AccountDisabled{}},
		{accountID: "doesnt_exist_acct", required: true, disabled: true, err: &errortypes.AcctRequired{}},

		// pubID given and matches a valid host account with Disabled: false
		{accountID: "valid_acct", required: false, disabled: false, err: nil},
		{accountID: "valid_acct", required: true, disabled: false, err: nil},
		{accountID: "valid_acct", required: false, disabled: true, err: nil},
		{accountID: "valid_acct", required: true, disabled: true, err: nil},

		{accountID: "valid_acct_dsa", required: false, disabled: false, wantDSA: validDSA, err: nil},
		{accountID: "valid_acct_dsa", required: true, disabled: false, wantDSA: validDSA, err: nil},
		{accountID: "valid_acct_dsa", required: false, disabled: true, wantDSA: validDSA, err: nil},
		{accountID: "valid_acct_dsa", required: true, disabled: true, wantDSA: validDSA, err: nil},

		{accountID: "invalid_acct_ipv6_ipv4", required: true, disabled: false, err: nil, wantDefaultIP: true},
		{accountID: "invalid_acct_dsa", required: false, disabled: false, err: &errortypes.MalformedAcct{}},

		// pubID given and matches a host account explicitly disabled (Disabled: true on account json)
		{accountID: "disabled_acct", required: false, disabled: false, err: &errortypes.AccountDisabled{}},
		{accountID: "disabled_acct", required: true, disabled: false, err: &errortypes.AccountDisabled{}},
		{accountID: "disabled_acct", required: false, disabled: true, err: &errortypes.AccountDisabled{}},
		{accountID: "disabled_acct", required: true, disabled: true, err: &errortypes.AccountDisabled{}},

		// pubID given and matches a host account that has a malformed config
		{accountID: "malformed_acct", required: false, disabled: false, err: &errortypes.MalformedAcct{}},
		{accountID: "malformed_acct", required: true, disabled: false, err: &errortypes.MalformedAcct{}},
		{accountID: "malformed_acct", required: false, disabled: true, err: &errortypes.MalformedAcct{}},
		{accountID: "malformed_acct", required: true, disabled: true, err: &errortypes.MalformedAcct{}},

		// account not provided (does not exist)
		{accountID: "", required: false, disabled: false, err: nil},
		{accountID: "", required: true, disabled: false, err: nil},
		{accountID: "", required: false, disabled: true, err: &errortypes.AccountDisabled{}},
		{accountID: "", required: true, disabled: true, err: &errortypes.AcctRequired{}},
	}

	for _, test := range testCases {
		description := fmt.Sprintf(`ID=%s/required=%t/disabled=%t`, test.accountID, test.required, test.disabled)
		t.Run(description, func(t *testing.T) {
			cfg := &config.Configuration{
				AccountRequired: test.required,
				AccountDefaults: config.Account{Disabled: test.disabled},
			}
			fetcher := &mockAccountFetcher{}
			assert.NoError(t, cfg.MarshalAccountDefaults())

			metrics := &metrics.MetricsEngineMock{}
			metrics.Mock.On("RecordAccountUpgradeStatus", mock.Anything, mock.Anything).Return()

			account, errors := GetAccount(context.Background(), cfg, fetcher, test.accountID, metrics)

			if test.err == nil {
				assert.Empty(t, errors)
				assert.Equal(t, test.accountID, account.ID, "account.ID must match requested ID")
				assert.Equal(t, false, account.Disabled, "returned account must not be disabled")
			} else {
				assert.NotEmpty(t, errors, "expected errors but got success")
				assert.Nil(t, account, "return account must be nil on error")
				assert.IsType(t, test.err, errors[0], "error is of unexpected type")
			}
			if test.wantDefaultIP {
				assert.Equal(t, account.Privacy.IPv6Config.AnonKeepBits, iputil.IPv6DefaultMaskingBitSize, "ipv6 should be set to default value")
				assert.Equal(t, account.Privacy.IPv4Config.AnonKeepBits, iputil.IPv4DefaultMaskingBitSize, "ipv4 should be set to default value")
			}
			if test.wantDSA != nil {
				assert.Equal(t, test.wantDSA, account.Privacy.DSA.DefaultUnpacked)
			}
		})
	}
}

func TestSetDerivedConfig(t *testing.T) {
	tests := []struct {
		description              string
		purpose1VendorExceptions []string
		feature1VendorExceptions []openrtb_ext.BidderName
		basicEnforcementVendors  []string
		enforceAlgo              string
		wantEnforceAlgoID        config.TCF2EnforcementAlgo
	}{
		{
			description:              "Nil purpose 1 vendor exceptions",
			purpose1VendorExceptions: nil,
		},
		{
			description:              "One purpose 1 vendor exception",
			purpose1VendorExceptions: []string{"appnexus"},
		},
		{
			description:              "Multiple purpose 1 vendor exceptions",
			purpose1VendorExceptions: []string{"appnexus", "rubicon"},
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
		{
			description:       "Basic Enforcement algorithm",
			enforceAlgo:       config.TCF2EnforceAlgoBasic,
			wantEnforceAlgoID: config.TCF2BasicEnforcement,
		},
		{
			description:       "Full Enforcement algorithm",
			enforceAlgo:       config.TCF2EnforceAlgoFull,
			wantEnforceAlgoID: config.TCF2FullEnforcement,
		},
	}

	for _, tt := range tests {
		account := config.Account{
			GDPR: config.AccountGDPR{
				Purpose1: config.AccountGDPRPurpose{
					VendorExceptions: tt.purpose1VendorExceptions,
					EnforceAlgo:      tt.enforceAlgo,
				},
				SpecialFeature1: config.AccountGDPRSpecialFeature{
					VendorExceptions: tt.feature1VendorExceptions,
				},
				BasicEnforcementVendors: tt.basicEnforcementVendors,
			},
		}

		setDerivedConfig(&account)

		purpose1ExceptionMapKeys := make([]string, 0)
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

		assert.Equal(t, account.GDPR.Purpose1.EnforceAlgoID, tt.wantEnforceAlgoID, tt.description)
	}
}
