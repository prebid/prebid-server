package hooks

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/hooks/invocation"
	"github.com/prebid/prebid-server/hooks/stages"
	"github.com/stretchr/testify/assert"
)

func TestPlanForEntrypointStage(t *testing.T) {
	testCases := map[string]struct {
		givenEndpoint               string
		givenHostPlanData           []byte
		givenDefaultAccountPlanData []byte
		givenHooks                  map[string]interface{}
		expectedPlan                Plan[stages.EntrypointHook]
	}{
		"Host and default-account execution plans successfully merged": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"entrypoint":{"groups":[{"timeout":5,"hook_sequence":[{"module_code":"foobar","hook_impl_code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"entrypoint": {"groups": [{"timeout": 10, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "bar"}, {"module_code": "ortb2blocking", "hook_impl_code": "block_request"}]}, {"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [{"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}}}`),
			givenHooks: map[string]interface{}{
				"foobar":        fakeEntrypointHook{},
				"ortb2blocking": fakeEntrypointHook{},
			},
			expectedPlan: Plan[stages.EntrypointHook]{
				// first group from host-level plan
				Group[stages.EntrypointHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.EntrypointHook]{
						{Module: "foobar", Code: "foo", Hook: fakeEntrypointHook{}},
					},
				},
				// then groups from the account-level plan
				Group[stages.EntrypointHook]{
					Timeout: 10 * time.Millisecond,
					Hooks: []HookWrapper[stages.EntrypointHook]{
						{Module: "foobar", Code: "bar", Hook: fakeEntrypointHook{}},
						{Module: "ortb2blocking", Code: "block_request", Hook: fakeEntrypointHook{}},
					},
				},
				Group[stages.EntrypointHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.EntrypointHook]{
						{Module: "foobar", Code: "foo", Hook: fakeEntrypointHook{}},
					},
				},
			},
		},
		"Works with empty default-account-execution_plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"entrypoint":{"groups":[{"timeout":5,"hook_sequence":[{"module_code":"foobar","hook_impl_code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{}`),
			givenHooks:                  map[string]interface{}{"foobar": fakeEntrypointHook{}},
			expectedPlan: Plan[stages.EntrypointHook]{
				Group[stages.EntrypointHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.EntrypointHook]{
						{Module: "foobar", Code: "foo", Hook: fakeEntrypointHook{}},
					},
				},
			},
		},
		"Works with empty host-execution_plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{}`),
			givenDefaultAccountPlanData: []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"entrypoint":{"groups":[{"timeout":5,"hook_sequence":[{"module_code":"foobar","hook_impl_code":"foo"}]}]}}}}}`),
			givenHooks:                  map[string]interface{}{"foobar": fakeEntrypointHook{}},
			expectedPlan: Plan[stages.EntrypointHook]{
				Group[stages.EntrypointHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.EntrypointHook]{
						{Module: "foobar", Code: "foo", Hook: fakeEntrypointHook{}},
					},
				},
			},
		},
		"Empty plan if hooks config not defined": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{}`),
			givenDefaultAccountPlanData: []byte(`{}`),
			givenHooks:                  map[string]interface{}{"foobar": fakeEntrypointHook{}},
			expectedPlan:                Plan[stages.EntrypointHook]{},
		},
		"Empty plan if hook repository empty": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"entrypoint":{"groups":[{"timeout":5,"hook_sequence":[{"module_code":"foobar","hook_impl_code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{}`),
			givenHooks:                  nil,
			expectedPlan:                Plan[stages.EntrypointHook]{},
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			planBuilder, err := getPlanBuilder(test.givenHooks, test.givenHostPlanData, test.givenDefaultAccountPlanData)
			if assert.NoError(t, err, "Failed to init hook execution plan builder") {
				assert.Equal(t, test.expectedPlan, planBuilder.PlanForEntrypointStage(test.givenEndpoint))
			}
		})
	}
}

func TestPlanForRawAuctionStage(t *testing.T) {
	hooks := map[string]interface{}{
		"foobar":        fakeRawAuctionHook{},
		"ortb2blocking": fakeRawAuctionHook{},
		"prebid":        fakeRawAuctionHook{},
	}

	testCases := map[string]struct {
		givenEndpoint               string
		givenHostPlanData           []byte
		givenDefaultAccountPlanData []byte
		giveAccountPlanData         []byte
		givenHooks                  map[string]interface{}
		expectedPlan                Plan[stages.RawAuctionHook]
	}{
		"Account-specific execution plan rewrites default-account execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"rawauction":{"groups":[{"timeout":5,"hook_sequence":[{"module_code":"foobar","hook_impl_code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"rawauction": {"groups": [{"timeout": 10, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "bar"}, {"module_code": "ortb2blocking", "hook_impl_code": "block_request"}]}, {"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [{"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}}}`),
			giveAccountPlanData:         []byte(`{"execution_plan": {"endpoints": {"/openrtb2/auction": {"stages": {"rawauction": {"groups": [{"timeout": 15, "hook_sequence": [{"module_code": "prebid", "hook_impl_code": "baz"}]}]}}}}}}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.RawAuctionHook]{
				// first group from host-level plan
				Group[stages.RawAuctionHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.RawAuctionHook]{
						{Module: "foobar", Code: "foo", Hook: fakeRawAuctionHook{}},
					},
				},
				// then come groups from account-level plan (default-account-level plan ignored)
				Group[stages.RawAuctionHook]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[stages.RawAuctionHook]{
						{Module: "prebid", Code: "baz", Hook: fakeRawAuctionHook{}},
					},
				},
			},
		},
		"Works with only account-specific plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{}`),
			givenDefaultAccountPlanData: []byte(`{}`),
			giveAccountPlanData:         []byte(`{"execution_plan": {"endpoints": {"/openrtb2/auction": {"stages": {"rawauction": {"groups": [{"timeout": 15, "hook_sequence": [{"module_code": "prebid", "hook_impl_code": "baz"}]}]}}}}}}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.RawAuctionHook]{
				Group[stages.RawAuctionHook]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[stages.RawAuctionHook]{
						{Module: "prebid", Code: "baz", Hook: fakeRawAuctionHook{}},
					},
				},
			},
		},
		"Works with empty account-specific execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"rawauction":{"groups":[{"timeout":5,"hook_sequence":[{"module_code":"foobar","hook_impl_code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"rawauction": {"groups": [{"timeout": 10, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "bar"}, {"module_code": "ortb2blocking", "hook_impl_code": "block_request"}]}, {"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [{"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}}}`),
			giveAccountPlanData:         []byte(`{}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.RawAuctionHook]{
				Group[stages.RawAuctionHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.RawAuctionHook]{
						{Module: "foobar", Code: "foo", Hook: fakeRawAuctionHook{}},
					},
				},
				Group[stages.RawAuctionHook]{
					Timeout: 10 * time.Millisecond,
					Hooks: []HookWrapper[stages.RawAuctionHook]{
						{Module: "foobar", Code: "bar", Hook: fakeRawAuctionHook{}},
						{Module: "ortb2blocking", Code: "block_request", Hook: fakeRawAuctionHook{}},
					},
				},
				Group[stages.RawAuctionHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.RawAuctionHook]{
						{Module: "foobar", Code: "foo", Hook: fakeRawAuctionHook{}},
					},
				},
			},
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			account := new(config.Account)
			if err := json.Unmarshal(test.giveAccountPlanData, &account.Hooks); err != nil {
				t.Fatal(err)
			}

			planBuilder, err := getPlanBuilder(test.givenHooks, test.givenHostPlanData, test.givenDefaultAccountPlanData)
			if assert.NoError(t, err, "Failed to init hook execution plan builder") {
				plan := planBuilder.PlanForRawAuctionStage(test.givenEndpoint, account)
				assert.Equal(t, test.expectedPlan, plan)
			}
		})
	}
}

func TestPlanForProcessedAuctionStage(t *testing.T) {
	hooks := map[string]interface{}{
		"foobar":        fakeProcessedAuctionHook{},
		"ortb2blocking": fakeProcessedAuctionHook{},
		"prebid":        fakeProcessedAuctionHook{},
	}

	testCases := map[string]struct {
		givenEndpoint               string
		givenHostPlanData           []byte
		givenDefaultAccountPlanData []byte
		giveAccountPlanData         []byte
		givenHooks                  map[string]interface{}
		expectedPlan                Plan[stages.ProcessedAuctionHook]
	}{
		"Account-specific execution plan rewrites default-account execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"procauction":{"groups":[{"timeout":5,"hook_sequence":[{"module_code":"foobar","hook_impl_code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"procauction": {"groups": [{"timeout": 10, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "bar"}, {"module_code": "ortb2blocking", "hook_impl_code": "block_request"}]}, {"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [{"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}}}`),
			giveAccountPlanData:         []byte(`{"execution_plan": {"endpoints": {"/openrtb2/auction": {"stages": {"procauction": {"groups": [{"timeout": 15, "hook_sequence": [{"module_code": "prebid", "hook_impl_code": "baz"}]}]}}}}}}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.ProcessedAuctionHook]{
				// first group from host-level plan
				Group[stages.ProcessedAuctionHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.ProcessedAuctionHook]{
						{Module: "foobar", Code: "foo", Hook: fakeProcessedAuctionHook{}},
					},
				},
				// then come groups from account-level plan (default-account-level plan ignored)
				Group[stages.ProcessedAuctionHook]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[stages.ProcessedAuctionHook]{
						{Module: "prebid", Code: "baz", Hook: fakeProcessedAuctionHook{}},
					},
				},
			},
		},
		"Works with only account-specific plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{}`),
			givenDefaultAccountPlanData: []byte(`{}`),
			giveAccountPlanData:         []byte(`{"execution_plan": {"endpoints": {"/openrtb2/auction": {"stages": {"procauction": {"groups": [{"timeout": 15, "hook_sequence": [{"module_code": "prebid", "hook_impl_code": "baz"}]}]}}}}}}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.ProcessedAuctionHook]{
				Group[stages.ProcessedAuctionHook]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[stages.ProcessedAuctionHook]{
						{Module: "prebid", Code: "baz", Hook: fakeProcessedAuctionHook{}},
					},
				},
			},
		},
		"Works with empty account-specific execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"procauction":{"groups":[{"timeout":5,"hook_sequence":[{"module_code":"foobar","hook_impl_code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"procauction": {"groups": [{"timeout": 10, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "bar"}, {"module_code": "ortb2blocking", "hook_impl_code": "block_request"}]}, {"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [{"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}}}`),
			giveAccountPlanData:         []byte(`{}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.ProcessedAuctionHook]{
				Group[stages.ProcessedAuctionHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.ProcessedAuctionHook]{
						{Module: "foobar", Code: "foo", Hook: fakeProcessedAuctionHook{}},
					},
				},
				Group[stages.ProcessedAuctionHook]{
					Timeout: 10 * time.Millisecond,
					Hooks: []HookWrapper[stages.ProcessedAuctionHook]{
						{Module: "foobar", Code: "bar", Hook: fakeProcessedAuctionHook{}},
						{Module: "ortb2blocking", Code: "block_request", Hook: fakeProcessedAuctionHook{}},
					},
				},
				Group[stages.ProcessedAuctionHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.ProcessedAuctionHook]{
						{Module: "foobar", Code: "foo", Hook: fakeProcessedAuctionHook{}},
					},
				},
			},
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			account := new(config.Account)
			if err := json.Unmarshal(test.giveAccountPlanData, &account.Hooks); err != nil {
				t.Fatal(err)
			}

			planBuilder, err := getPlanBuilder(test.givenHooks, test.givenHostPlanData, test.givenDefaultAccountPlanData)
			if assert.NoError(t, err, "Failed to init hook execution plan builder") {
				plan := planBuilder.PlanForProcessedAuctionStage(test.givenEndpoint, account)
				assert.Equal(t, test.expectedPlan, plan)
			}
		})
	}
}

func TestPlanForBidRequestStage(t *testing.T) {
	hooks := map[string]interface{}{
		"foobar":        fakeBidRequestHook{},
		"ortb2blocking": fakeBidRequestHook{},
		"prebid":        fakeBidRequestHook{},
	}

	testCases := map[string]struct {
		givenEndpoint               string
		givenHostPlanData           []byte
		givenDefaultAccountPlanData []byte
		giveAccountPlanData         []byte
		givenHooks                  map[string]interface{}
		expectedPlan                Plan[stages.BidRequestHook]
	}{
		"Account-specific execution plan rewrites default-account execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"bidrequest":{"groups":[{"timeout":5,"hook_sequence":[{"module_code":"foobar","hook_impl_code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"bidrequest": {"groups": [{"timeout": 10, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "bar"}, {"module_code": "ortb2blocking", "hook_impl_code": "block_request"}]}, {"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [{"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}}}`),
			giveAccountPlanData:         []byte(`{"execution_plan": {"endpoints": {"/openrtb2/auction": {"stages": {"bidrequest": {"groups": [{"timeout": 15, "hook_sequence": [{"module_code": "prebid", "hook_impl_code": "baz"}]}]}}}}}}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.BidRequestHook]{
				// first group from host-level plan
				Group[stages.BidRequestHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.BidRequestHook]{
						{Module: "foobar", Code: "foo", Hook: fakeBidRequestHook{}},
					},
				},
				// then come groups from account-level plan (default-account-level plan ignored)
				Group[stages.BidRequestHook]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[stages.BidRequestHook]{
						{Module: "prebid", Code: "baz", Hook: fakeBidRequestHook{}},
					},
				},
			},
		},
		"Works with only account-specific plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{}`),
			givenDefaultAccountPlanData: []byte(`{}`),
			giveAccountPlanData:         []byte(`{"execution_plan": {"endpoints": {"/openrtb2/auction": {"stages": {"bidrequest": {"groups": [{"timeout": 15, "hook_sequence": [{"module_code": "prebid", "hook_impl_code": "baz"}]}]}}}}}}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.BidRequestHook]{
				Group[stages.BidRequestHook]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[stages.BidRequestHook]{
						{Module: "prebid", Code: "baz", Hook: fakeBidRequestHook{}},
					},
				},
			},
		},
		"Works with empty account-specific execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"bidrequest":{"groups":[{"timeout":5,"hook_sequence":[{"module_code":"foobar","hook_impl_code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"bidrequest": {"groups": [{"timeout": 10, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "bar"}, {"module_code": "ortb2blocking", "hook_impl_code": "block_request"}]}, {"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [{"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}}}`),
			giveAccountPlanData:         []byte(`{}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.BidRequestHook]{
				Group[stages.BidRequestHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.BidRequestHook]{
						{Module: "foobar", Code: "foo", Hook: fakeBidRequestHook{}},
					},
				},
				Group[stages.BidRequestHook]{
					Timeout: 10 * time.Millisecond,
					Hooks: []HookWrapper[stages.BidRequestHook]{
						{Module: "foobar", Code: "bar", Hook: fakeBidRequestHook{}},
						{Module: "ortb2blocking", Code: "block_request", Hook: fakeBidRequestHook{}},
					},
				},
				Group[stages.BidRequestHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.BidRequestHook]{
						{Module: "foobar", Code: "foo", Hook: fakeBidRequestHook{}},
					},
				},
			},
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			account := new(config.Account)
			if err := json.Unmarshal(test.giveAccountPlanData, &account.Hooks); err != nil {
				t.Fatal(err)
			}

			planBuilder, err := getPlanBuilder(test.givenHooks, test.givenHostPlanData, test.givenDefaultAccountPlanData)
			if assert.NoError(t, err, "Failed to init hook execution plan builder") {
				plan := planBuilder.PlanForBidRequestStage(test.givenEndpoint, account)
				assert.Equal(t, test.expectedPlan, plan)
			}
		})
	}
}

func TestPlanForRawBidResponseStage(t *testing.T) {
	hooks := map[string]interface{}{
		"foobar":        fakeRawBidResponseHook{},
		"ortb2blocking": fakeRawBidResponseHook{},
		"prebid":        fakeRawBidResponseHook{},
	}

	testCases := map[string]struct {
		givenEndpoint               string
		givenHostPlanData           []byte
		givenDefaultAccountPlanData []byte
		giveAccountPlanData         []byte
		givenHooks                  map[string]interface{}
		expectedPlan                Plan[stages.RawBidResponseHook]
	}{
		"Account-specific execution plan rewrites default-account execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"rawbidresponse":{"groups":[{"timeout":5,"hook_sequence":[{"module_code":"foobar","hook_impl_code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"rawbidresponse": {"groups": [{"timeout": 10, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "bar"}, {"module_code": "ortb2blocking", "hook_impl_code": "block_request"}]}, {"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [{"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}}}`),
			giveAccountPlanData:         []byte(`{"execution_plan": {"endpoints": {"/openrtb2/auction": {"stages": {"rawbidresponse": {"groups": [{"timeout": 15, "hook_sequence": [{"module_code": "prebid", "hook_impl_code": "baz"}]}]}}}}}}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.RawBidResponseHook]{
				// first group from host-level plan
				Group[stages.RawBidResponseHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.RawBidResponseHook]{
						{Module: "foobar", Code: "foo", Hook: fakeRawBidResponseHook{}},
					},
				},
				// then come groups from account-level plan (default-account-level plan ignored)
				Group[stages.RawBidResponseHook]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[stages.RawBidResponseHook]{
						{Module: "prebid", Code: "baz", Hook: fakeRawBidResponseHook{}},
					},
				},
			},
		},
		"Works with only account-specific plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{}`),
			givenDefaultAccountPlanData: []byte(`{}`),
			giveAccountPlanData:         []byte(`{"execution_plan": {"endpoints": {"/openrtb2/auction": {"stages": {"rawbidresponse": {"groups": [{"timeout": 15, "hook_sequence": [{"module_code": "prebid", "hook_impl_code": "baz"}]}]}}}}}}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.RawBidResponseHook]{
				Group[stages.RawBidResponseHook]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[stages.RawBidResponseHook]{
						{Module: "prebid", Code: "baz", Hook: fakeRawBidResponseHook{}},
					},
				},
			},
		},
		"Works with empty account-specific execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"rawbidresponse":{"groups":[{"timeout":5,"hook_sequence":[{"module_code":"foobar","hook_impl_code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"rawbidresponse": {"groups": [{"timeout": 10, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "bar"}, {"module_code": "ortb2blocking", "hook_impl_code": "block_request"}]}, {"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [{"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}}}`),
			giveAccountPlanData:         []byte(`{}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.RawBidResponseHook]{
				Group[stages.RawBidResponseHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.RawBidResponseHook]{
						{Module: "foobar", Code: "foo", Hook: fakeRawBidResponseHook{}},
					},
				},
				Group[stages.RawBidResponseHook]{
					Timeout: 10 * time.Millisecond,
					Hooks: []HookWrapper[stages.RawBidResponseHook]{
						{Module: "foobar", Code: "bar", Hook: fakeRawBidResponseHook{}},
						{Module: "ortb2blocking", Code: "block_request", Hook: fakeRawBidResponseHook{}},
					},
				},
				Group[stages.RawBidResponseHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.RawBidResponseHook]{
						{Module: "foobar", Code: "foo", Hook: fakeRawBidResponseHook{}},
					},
				},
			},
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			account := new(config.Account)
			if err := json.Unmarshal(test.giveAccountPlanData, &account.Hooks); err != nil {
				t.Fatal(err)
			}

			planBuilder, err := getPlanBuilder(test.givenHooks, test.givenHostPlanData, test.givenDefaultAccountPlanData)
			if assert.NoError(t, err, "Failed to init hook execution plan builder") {
				plan := planBuilder.PlanForRawBidResponseStage(test.givenEndpoint, account)
				assert.Equal(t, test.expectedPlan, plan)
			}
		})
	}
}

func TestPlanForAllProcBidResponsesStage(t *testing.T) {
	hooks := map[string]interface{}{
		"foobar":        fakeAllProcBidResponsesHook{},
		"ortb2blocking": fakeAllProcBidResponsesHook{},
		"prebid":        fakeAllProcBidResponsesHook{},
	}

	testCases := map[string]struct {
		givenEndpoint               string
		givenHostPlanData           []byte
		givenDefaultAccountPlanData []byte
		giveAccountPlanData         []byte
		givenHooks                  map[string]interface{}
		expectedPlan                Plan[stages.AllProcBidResponsesHook]
	}{
		"Account-specific execution plan rewrites default-account execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"allprocbidresponses":{"groups":[{"timeout":5,"hook_sequence":[{"module_code":"foobar","hook_impl_code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"allprocbidresponses": {"groups": [{"timeout": 10, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "bar"}, {"module_code": "ortb2blocking", "hook_impl_code": "block_request"}]}, {"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [{"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}}}`),
			giveAccountPlanData:         []byte(`{"execution_plan": {"endpoints": {"/openrtb2/auction": {"stages": {"allprocbidresponses": {"groups": [{"timeout": 15, "hook_sequence": [{"module_code": "prebid", "hook_impl_code": "baz"}]}]}}}}}}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.AllProcBidResponsesHook]{
				// first group from host-level plan
				Group[stages.AllProcBidResponsesHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.AllProcBidResponsesHook]{
						{Module: "foobar", Code: "foo", Hook: fakeAllProcBidResponsesHook{}},
					},
				},
				// then come groups from account-level plan (default-account-level plan ignored)
				Group[stages.AllProcBidResponsesHook]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[stages.AllProcBidResponsesHook]{
						{Module: "prebid", Code: "baz", Hook: fakeAllProcBidResponsesHook{}},
					},
				},
			},
		},
		"Works with only account-specific plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{}`),
			givenDefaultAccountPlanData: []byte(`{}`),
			giveAccountPlanData:         []byte(`{"execution_plan": {"endpoints": {"/openrtb2/auction": {"stages": {"allprocbidresponses": {"groups": [{"timeout": 15, "hook_sequence": [{"module_code": "prebid", "hook_impl_code": "baz"}]}]}}}}}}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.AllProcBidResponsesHook]{
				Group[stages.AllProcBidResponsesHook]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[stages.AllProcBidResponsesHook]{
						{Module: "prebid", Code: "baz", Hook: fakeAllProcBidResponsesHook{}},
					},
				},
			},
		},
		"Works with empty account-specific execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"allprocbidresponses":{"groups":[{"timeout":5,"hook_sequence":[{"module_code":"foobar","hook_impl_code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"allprocbidresponses": {"groups": [{"timeout": 10, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "bar"}, {"module_code": "ortb2blocking", "hook_impl_code": "block_request"}]}, {"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [{"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}}}`),
			giveAccountPlanData:         []byte(`{}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.AllProcBidResponsesHook]{
				Group[stages.AllProcBidResponsesHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.AllProcBidResponsesHook]{
						{Module: "foobar", Code: "foo", Hook: fakeAllProcBidResponsesHook{}},
					},
				},
				Group[stages.AllProcBidResponsesHook]{
					Timeout: 10 * time.Millisecond,
					Hooks: []HookWrapper[stages.AllProcBidResponsesHook]{
						{Module: "foobar", Code: "bar", Hook: fakeAllProcBidResponsesHook{}},
						{Module: "ortb2blocking", Code: "block_request", Hook: fakeAllProcBidResponsesHook{}},
					},
				},
				Group[stages.AllProcBidResponsesHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.AllProcBidResponsesHook]{
						{Module: "foobar", Code: "foo", Hook: fakeAllProcBidResponsesHook{}},
					},
				},
			},
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			account := new(config.Account)
			if err := json.Unmarshal(test.giveAccountPlanData, &account.Hooks); err != nil {
				t.Fatal(err)
			}

			planBuilder, err := getPlanBuilder(test.givenHooks, test.givenHostPlanData, test.givenDefaultAccountPlanData)
			if assert.NoError(t, err, "Failed to init hook execution plan builder") {
				plan := planBuilder.PlanForAllProcessedBidResponsesStage(test.givenEndpoint, account)
				assert.Equal(t, test.expectedPlan, plan)
			}
		})
	}
}

func TestPlanForAuctionResponseStage(t *testing.T) {
	hooks := map[string]interface{}{
		"foobar":        fakeAuctionResponseHook{},
		"ortb2blocking": fakeAuctionResponseHook{},
		"prebid":        fakeAuctionResponseHook{},
	}

	testCases := map[string]struct {
		givenEndpoint               string
		givenHostPlanData           []byte
		givenDefaultAccountPlanData []byte
		giveAccountPlanData         []byte
		givenHooks                  map[string]interface{}
		expectedPlan                Plan[stages.AuctionResponseHook]
	}{
		"Account-specific execution plan rewrites default-account execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"auctionresponse":{"groups":[{"timeout":5,"hook_sequence":[{"module_code":"foobar","hook_impl_code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"auctionresponse": {"groups": [{"timeout": 10, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "bar"}, {"module_code": "ortb2blocking", "hook_impl_code": "block_request"}]}, {"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [{"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}}}`),
			giveAccountPlanData:         []byte(`{"execution_plan": {"endpoints": {"/openrtb2/auction": {"stages": {"auctionresponse": {"groups": [{"timeout": 15, "hook_sequence": [{"module_code": "prebid", "hook_impl_code": "baz"}]}]}}}}}}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.AuctionResponseHook]{
				// first group from host-level plan
				Group[stages.AuctionResponseHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.AuctionResponseHook]{
						{Module: "foobar", Code: "foo", Hook: fakeAuctionResponseHook{}},
					},
				},
				// then come groups from account-level plan (default-account-level plan ignored)
				Group[stages.AuctionResponseHook]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[stages.AuctionResponseHook]{
						{Module: "prebid", Code: "baz", Hook: fakeAuctionResponseHook{}},
					},
				},
			},
		},
		"Works with only account-specific plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{}`),
			givenDefaultAccountPlanData: []byte(`{}`),
			giveAccountPlanData:         []byte(`{"execution_plan": {"endpoints": {"/openrtb2/auction": {"stages": {"auctionresponse": {"groups": [{"timeout": 15, "hook_sequence": [{"module_code": "prebid", "hook_impl_code": "baz"}]}]}}}}}}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.AuctionResponseHook]{
				Group[stages.AuctionResponseHook]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[stages.AuctionResponseHook]{
						{Module: "prebid", Code: "baz", Hook: fakeAuctionResponseHook{}},
					},
				},
			},
		},
		"Works with empty account-specific execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"auctionresponse":{"groups":[{"timeout":5,"hook_sequence":[{"module_code":"foobar","hook_impl_code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"auctionresponse": {"groups": [{"timeout": 10, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "bar"}, {"module_code": "ortb2blocking", "hook_impl_code": "block_request"}]}, {"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [{"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}}}`),
			giveAccountPlanData:         []byte(`{}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.AuctionResponseHook]{
				Group[stages.AuctionResponseHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.AuctionResponseHook]{
						{Module: "foobar", Code: "foo", Hook: fakeAuctionResponseHook{}},
					},
				},
				Group[stages.AuctionResponseHook]{
					Timeout: 10 * time.Millisecond,
					Hooks: []HookWrapper[stages.AuctionResponseHook]{
						{Module: "foobar", Code: "bar", Hook: fakeAuctionResponseHook{}},
						{Module: "ortb2blocking", Code: "block_request", Hook: fakeAuctionResponseHook{}},
					},
				},
				Group[stages.AuctionResponseHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.AuctionResponseHook]{
						{Module: "foobar", Code: "foo", Hook: fakeAuctionResponseHook{}},
					},
				},
			},
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			account := new(config.Account)
			if err := json.Unmarshal(test.giveAccountPlanData, &account.Hooks); err != nil {
				t.Fatal(err)
			}

			planBuilder, err := getPlanBuilder(test.givenHooks, test.givenHostPlanData, test.givenDefaultAccountPlanData)
			if assert.NoError(t, err, "Failed to init hook execution plan builder") {
				plan := planBuilder.PlanForAuctionResponseStage(test.givenEndpoint, account)
				assert.Equal(t, test.expectedPlan, plan)
			}
		})
	}
}

func getPlanBuilder(
	moduleHooks map[string]interface{},
	hostPlanData, accountPlanData []byte,
) (ExecutionPlanBuilder, error) {
	var err error
	var hooks config.Hooks
	var hostPlan config.HookExecutionPlan
	var defaultAccountPlan config.HookExecutionPlan

	err = json.Unmarshal(hostPlanData, &hostPlan)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(accountPlanData, &defaultAccountPlan)
	if err != nil {
		return nil, err
	}

	hooks.HostExecutionPlan = hostPlan
	hooks.AccountExecutionPlan = defaultAccountPlan

	repo, err := NewHookRepository(moduleHooks)
	if err != nil {
		return nil, err
	}

	return NewExecutionPlanBuilder(hooks, repo), nil
}

type fakeEntrypointHook struct{}

func (h fakeEntrypointHook) HandleEntrypointHook(
	_ context.Context,
	_ invocation.Context,
	_ stages.EntrypointPayload,
) (invocation.HookResult[stages.EntrypointPayload], error) {
	return invocation.HookResult[stages.EntrypointPayload]{}, nil
}

type fakeRawAuctionHook struct{}

func (f fakeRawAuctionHook) HandleRawAuctionHook(
	_ context.Context,
	_ invocation.Context,
	_ stages.BidRequest,
) (invocation.HookResult[stages.BidRequest], error) {
	return invocation.HookResult[stages.BidRequest]{}, nil
}

type fakeProcessedAuctionHook struct{}

func (f fakeProcessedAuctionHook) HandleProcessedAuctionHook(
	_ context.Context,
	_ invocation.Context,
	_ stages.ProcessedAuctionPayload,
) (invocation.HookResult[stages.ProcessedAuctionPayload], error) {
	return invocation.HookResult[stages.ProcessedAuctionPayload]{}, nil
}

type fakeBidRequestHook struct{}

func (f fakeBidRequestHook) HandleBidRequestHook(
	_ context.Context,
	_ invocation.Context,
	_ stages.BidRequestPayload,
) (invocation.HookResult[stages.BidRequestPayload], error) {
	return invocation.HookResult[stages.BidRequestPayload]{}, nil
}

type fakeRawBidResponseHook struct{}

func (f fakeRawBidResponseHook) HandleRawBidResponseHook(
	_ context.Context,
	_ invocation.Context,
	_ stages.RawBidResponsePayload,
) (invocation.HookResult[stages.RawBidResponsePayload], error) {
	return invocation.HookResult[stages.RawBidResponsePayload]{}, nil
}

type fakeAllProcBidResponsesHook struct{}

func (f fakeAllProcBidResponsesHook) HandleAllProcBidResponsesHook(
	_ context.Context,
	_ invocation.Context,
	_ stages.AllProcBidResponsesPayload,
) (invocation.HookResult[stages.AllProcBidResponsesPayload], error) {
	return invocation.HookResult[stages.AllProcBidResponsesPayload]{}, nil
}

type fakeAuctionResponseHook struct{}

func (f fakeAuctionResponseHook) HandleAuctionResponseHook(
	_ context.Context,
	_ invocation.Context,
	_ *openrtb2.BidResponse,
) (invocation.HookResult[*openrtb2.BidResponse], error) {
	return invocation.HookResult[*openrtb2.BidResponse]{}, nil
}
