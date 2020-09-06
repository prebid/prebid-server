package events

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/stored_requests"
	"github.com/stretchr/testify/assert"
	"testing"
)

// Mock AccountsFetcher
type mockAccountsFetcher struct {
	Fail          bool
	Error         error
	EventsEnabled bool
}

func (e *mockAccountsFetcher) FetchAccount(ctx context.Context, accountID string) (json.RawMessage, []error) {
	if e.Fail {
		return nil, []error{e.Error}
	}

	var acc = &config.Account{
		ID:            accountID,
		EventsEnabled: false,
	}

	if e.EventsEnabled {
		acc.EventsEnabled = true
	}

	s, _ := json.Marshal(acc)

	return s, []error{}
}

func TestShouldReturnDefaultAccountWithSpecifiedIdWhenAccountNotFound(t *testing.T) {
	// prepare
	ctx := context.Background()

	cfg := &config.Configuration{
		AccountDefaults: config.Account{
			EventsEnabled: true,
		},
	}
	cfg.MarshalAccountDefaults()

	af := mockAccountsFetcher{
		Fail:  true,
		Error: stored_requests.NotFoundError{},
	}

	acc, errs := GetAccount(ctx, cfg, &af, "test")

	assert.Equal(t, 0, len(errs), "Expected 0 errors when account not found")
	assert.EqualValues(t, config.Account{
		EventsEnabled: true,
		ID:            "test",
	}, *acc)

}

func TestShouldReturnNilAccountWhenFetcherFailsAttemptingToGetAccount(t *testing.T) {
	// prepare
	ctx := context.Background()

	cfg := &config.Configuration{
		AccountDefaults: config.Account{
			EventsEnabled: true,
		},
	}
	cfg.MarshalAccountDefaults()

	af := mockAccountsFetcher{
		Fail:  true,
		Error: fmt.Errorf("test error"),
	}

	acc, errs := GetAccount(ctx, cfg, &af, "test")

	assert.Equal(t, 1, len(errs), "Expected 1 errors")
	assert.Nil(t, acc)
	assert.Equal(t, "test error", errs[0].Error())

}

func TestShouldReturnAccountMergedWithAccountsDefaults(t *testing.T) {
	// prepare
	ctx := context.Background()

	cfg := &config.Configuration{
		AccountDefaults: config.Account{
			EventsEnabled: false,
		},
	}
	cfg.MarshalAccountDefaults()

	af := mockAccountsFetcher{
		Fail:          false,
		EventsEnabled: true,
	}

	expectedAccount := config.Account{
		EventsEnabled: true,
		ID:            "test",
	}

	acc, errs := GetAccount(ctx, cfg, &af, "test")

	assert.Equal(t, 0, len(errs), "Expected 0 errors")
	assert.EqualValues(t, expectedAccount, *acc)

}

func TestShouldReturnAccountMergedWithEmptyAccountsDefaults(t *testing.T) {
	// prepare
	ctx := context.Background()

	cfg := &config.Configuration{
		AccountDefaults: config.Account{},
	}
	cfg.MarshalAccountDefaults()

	af := mockAccountsFetcher{
		Fail:          false,
		EventsEnabled: true,
	}

	expectedAccount := config.Account{
		EventsEnabled: true,
		ID:            "test",
	}

	acc, errs := GetAccount(ctx, cfg, &af, "test")

	assert.Equal(t, 0, len(errs), "Expected 0 errors")
	assert.EqualValues(t, expectedAccount, *acc)

}
