package account

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/metrics"
	"github.com/prebid/prebid-server/stored_requests"
	"github.com/stretchr/testify/assert"
)

var mockAccountData = map[string]json.RawMessage{
	"valid_acct":    json.RawMessage(`{"disabled":false}`),
	"disabled_acct": json.RawMessage(`{"disabled":true}`),
}

type mockAccountFetcher struct {
}

func (af mockAccountFetcher) FetchAccount(ctx context.Context, accountID string) (json.RawMessage, []error) {
	if account, ok := mockAccountData[accountID]; ok {
		return account, nil
	}
	return nil, []error{stored_requests.NotFoundError{accountID, "Account"}}
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
