package stored_requests

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMultiFetcher(t *testing.T) {
	f1 := &mockFetcher{}
	f2 := &mockFetcher{}
	fetcher := &MultiFetcher{f1, f2}
	ctx := context.Background()
	reqIDs := []string{"abc", "def"}
	impIDs := []string{"imp-1", "imp-2"}

	f1.On("FetchRequests", ctx, reqIDs, impIDs).Return(
		map[string]json.RawMessage{
			"abc": json.RawMessage(`{"req_id": "abc"}`),
		},
		map[string]json.RawMessage{
			"imp-1": json.RawMessage(`{"imp_id": "imp-1"}`),
		},
		[]error{NotFoundError{"def", "Request"}, NotFoundError{"imp-2", "Imp"}},
	)
	f2.On("FetchRequests", ctx, []string{"def"}, []string{"imp-2"}).Return(
		map[string]json.RawMessage{
			"def": json.RawMessage(`{"req_id": "def"}`),
		},
		map[string]json.RawMessage{
			"imp-2": json.RawMessage(`{"imp_id": "imp-2"}`),
		},
		[]error{},
	)

	reqData, impData, errs := fetcher.FetchRequests(ctx, reqIDs, impIDs)

	f1.AssertExpectations(t)
	f2.AssertExpectations(t)
	assert.Len(t, reqData, 2, "MultiFetcher should return all the requested stored req data that exists")
	assert.Len(t, impData, 2, "MultiFetcher should return all the requested stored imp data that exists")
	assert.Len(t, errs, 0, "MultiFetcher shouldn't return an error")
	assert.JSONEq(t, `{"req_id": "abc"}`, string(reqData["abc"]), "MultiFetcher should return the right request data")
	assert.JSONEq(t, `{"req_id": "def"}`, string(reqData["def"]), "MultiFetcher should return the right request data")
	assert.JSONEq(t, `{"imp_id": "imp-1"}`, string(impData["imp-1"]), "MultiFetcher should return the right imp data")
	assert.JSONEq(t, `{"imp_id": "imp-2"}`, string(impData["imp-2"]), "MultiFetcher should return the right imp data")
}

func TestMissingID(t *testing.T) {
	f1 := &mockFetcher{}
	f2 := &mockFetcher{}
	fetcher := &MultiFetcher{f1, f2}
	ctx := context.Background()
	reqIDs := []string{"abc", "def", "ghi"}
	impIDs := []string{"imp-1", "imp-2", "imp-3"}

	f1.On("FetchRequests", ctx, reqIDs, impIDs).Return(
		map[string]json.RawMessage{
			"abc": json.RawMessage(`{"req_id": "abc"}`),
		},
		map[string]json.RawMessage{
			"imp-1": json.RawMessage(`{"imp_id": "imp-1"}`),
		},
		[]error{NotFoundError{"def", "Request"}, NotFoundError{"imp-2", "Imp"}},
	)
	f2.On("FetchRequests", ctx, []string{"def", "ghi"}, []string{"imp-2", "imp-3"}).Return(
		map[string]json.RawMessage{
			"def": json.RawMessage(`{"req_id": "def"}`),
		},
		map[string]json.RawMessage{
			"imp-2": json.RawMessage(`{"imp_id": "imp-2"}`),
		},
		[]error{},
	)

	reqData, impData, errs := fetcher.FetchRequests(ctx, reqIDs, impIDs)

	f1.AssertExpectations(t)
	f2.AssertExpectations(t)
	assert.Len(t, reqData, 2, "MultiFetcher should return all the requested stored req data that exists")
	assert.Len(t, impData, 2, "MultiFetcher should return all the requested stored imp data that exists")
	assert.Len(t, errs, 2, "MultiFetcher should return an error if there are missing IDs")
	assert.JSONEq(t, `{"req_id": "abc"}`, string(reqData["abc"]), "MultiFetcher should return the right request data")
	assert.JSONEq(t, `{"req_id": "def"}`, string(reqData["def"]), "MultiFetcher should return the right request data")
	assert.JSONEq(t, `{"imp_id": "imp-1"}`, string(impData["imp-1"]), "MultiFetcher should return the right imp data")
	assert.JSONEq(t, `{"imp_id": "imp-2"}`, string(impData["imp-2"]), "MultiFetcher should return the right imp data")
}

func TestOtherError(t *testing.T) {
	f1 := &mockFetcher{}
	f2 := &mockFetcher{}
	fetcher := &MultiFetcher{f1, f2}
	ctx := context.Background()
	reqIDs := []string{"abc", "def"}
	impIDs := []string{"imp-1"}

	f1.On("FetchRequests", ctx, reqIDs, impIDs).Return(
		map[string]json.RawMessage{
			"abc": json.RawMessage(`{"req_id": "abc"}`),
		},
		map[string]json.RawMessage{},
		[]error{NotFoundError{"def", "Request"}, errors.New("Other error")},
	)
	f2.On("FetchRequests", ctx, []string{"def"}, []string{"imp-1"}).Return(
		map[string]json.RawMessage{
			"def": json.RawMessage(`{"req_id": "def"}`),
		},
		map[string]json.RawMessage{
			"imp-1": json.RawMessage(`{"imp_id": "imp-1"}`),
		},
		[]error{},
	)

	reqData, impData, errs := fetcher.FetchRequests(ctx, reqIDs, impIDs)

	f1.AssertExpectations(t)
	f2.AssertExpectations(t)
	assert.Len(t, reqData, 2, "MultiFetcher should return all the requested stored req data that exists")
	assert.Len(t, impData, 1, "MultiFetcher should return all the requested stored imp data that exists")
	assert.Len(t, errs, 1, "MultiFetcher should return an error if one of the fetcher returns an error other than NotFoundError")
	assert.JSONEq(t, `{"req_id": "abc"}`, string(reqData["abc"]), "MultiFetcher should return the right request data")
	assert.JSONEq(t, `{"req_id": "def"}`, string(reqData["def"]), "MultiFetcher should return the right request data")
	assert.JSONEq(t, `{"imp_id": "imp-1"}`, string(impData["imp-1"]), "MultiFetcher should return the right imp data")
}

func TestMultiFetcherAccountFoundInFirstFetcher(t *testing.T) {
	f1 := &mockFetcher{}
	f2 := &mockFetcher{}
	fetcher := &MultiFetcher{f1, f2}
	ctx := context.Background()

	f1.On("FetchAccount", ctx, json.RawMessage("{}"), "ONE").Once().Return(json.RawMessage(`{"id": "ONE"}`), []error{})

	account, errs := fetcher.FetchAccount(ctx, json.RawMessage("{}"), "ONE")

	f1.AssertExpectations(t)
	f2.AssertNotCalled(t, "FetchAccount")
	assert.Empty(t, errs)
	assert.JSONEq(t, `{"id": "ONE"}`, string(account))
}

func TestMultiFetcherAccountFoundInSecondFetcher(t *testing.T) {
	f1 := &mockFetcher{}
	f2 := &mockFetcher{}
	fetcher := &MultiFetcher{f1, f2}
	ctx := context.Background()

	f1.On("FetchAccount", ctx, json.RawMessage("{}"), "TWO").Once().Return(json.RawMessage(``), []error{NotFoundError{"TWO", "Account"}})
	f2.On("FetchAccount", ctx, json.RawMessage("{}"), "TWO").Once().Return(json.RawMessage(`{"id": "TWO"}`), []error{})

	account, errs := fetcher.FetchAccount(ctx, json.RawMessage("{}"), "TWO")

	f1.AssertExpectations(t)
	f2.AssertExpectations(t)
	assert.Empty(t, errs)
	assert.JSONEq(t, `{"id": "TWO"}`, string(account))
}

func TestMultiFetcherAccountNotFound(t *testing.T) {
	f1 := &mockFetcher{}
	f2 := &mockFetcher{}
	fetcher := &MultiFetcher{f1, f2}
	ctx := context.Background()

	f1.On("FetchAccount", ctx, json.RawMessage("{}"), "MISSING").Once().Return(json.RawMessage(``), []error{NotFoundError{"TWO", "Account"}})
	f2.On("FetchAccount", ctx, json.RawMessage("{}"), "MISSING").Once().Return(json.RawMessage(``), []error{NotFoundError{"TWO", "Account"}})

	account, errs := fetcher.FetchAccount(ctx, json.RawMessage("{}"), "MISSING")

	f1.AssertExpectations(t)
	f2.AssertExpectations(t)
	assert.Len(t, errs, 1)
	assert.Nil(t, account)
	assert.EqualError(t, errs[0], NotFoundError{"MISSING", "Account"}.Error())
}

// noAccountsFetcher implements AllFetcher but not AllAccountsFetcher, so MultiFetcher
// should skip it during bulk enumeration.
type noAccountsFetcher struct{}

func (noAccountsFetcher) FetchRequests(ctx context.Context, requestIDs []string, impIDs []string) (map[string]json.RawMessage, map[string]json.RawMessage, []error) {
	return nil, nil, nil
}
func (noAccountsFetcher) FetchResponses(ctx context.Context, ids []string) (map[string]json.RawMessage, []error) {
	return nil, nil
}
func (noAccountsFetcher) FetchAccount(ctx context.Context, def json.RawMessage, id string) (json.RawMessage, []error) {
	return nil, []error{NotFoundError{id, "Account"}}
}
func (noAccountsFetcher) FetchCategories(ctx context.Context, primaryAdServer, publisherId, iabCategory string) (string, error) {
	return "", nil
}

// bulkAccountsFetcher additionally supports bulk account enumeration.
type bulkAccountsFetcher struct {
	noAccountsFetcher
	accounts map[string]json.RawMessage
	errs     []error
}

func (b bulkAccountsFetcher) FetchAllAccounts(ctx context.Context) (map[string]json.RawMessage, []error) {
	return b.accounts, b.errs
}

func TestMultiFetcherFetchAllAccounts(t *testing.T) {
	first := bulkAccountsFetcher{accounts: map[string]json.RawMessage{
		"a": json.RawMessage(`{"src":"first"}`),
		"b": json.RawMessage(`{"src":"first"}`),
	}}
	second := bulkAccountsFetcher{accounts: map[string]json.RawMessage{
		"b": json.RawMessage(`{"src":"second"}`), // conflicts with first; first wins
		"c": json.RawMessage(`{"src":"second"}`),
	}}
	fetcher := &MultiFetcher{first, noAccountsFetcher{}, second}

	accounts, errs := fetcher.FetchAllAccounts(context.Background())

	assert.Empty(t, errs)
	assert.Len(t, accounts, 3, "should merge accounts from all bulk-capable fetchers")
	assert.JSONEq(t, `{"src":"first"}`, string(accounts["a"]))
	assert.JSONEq(t, `{"src":"first"}`, string(accounts["b"]), "earlier fetcher wins on ID conflict")
	assert.JSONEq(t, `{"src":"second"}`, string(accounts["c"]))
}

func TestMultiFetcherFetchAllAccountsPropagatesErrors(t *testing.T) {
	failing := bulkAccountsFetcher{errs: []error{errors.New("db down")}}
	ok := bulkAccountsFetcher{accounts: map[string]json.RawMessage{"a": json.RawMessage(`{}`)}}
	fetcher := &MultiFetcher{failing, ok}

	accounts, errs := fetcher.FetchAllAccounts(context.Background())

	assert.Len(t, errs, 1, "a failing fetcher's error should be surfaced")
	assert.Contains(t, accounts, "a", "healthy fetchers still contribute their accounts")
}
