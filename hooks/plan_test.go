package hooks

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/hooks/hookstage"
	"github.com/stretchr/testify/assert"
)

func TestNewExecutionPlanBuilder(t *testing.T) {
	enabledConfig := config.Hooks{Enabled: true}
	testCases := map[string]struct {
		givenConfig         config.Hooks
		expectedPlanBuilder ExecutionPlanBuilder
	}{
		"Real plan builder returned when hooks enabled": {
			givenConfig:         enabledConfig,
			expectedPlanBuilder: PlanBuilder{hooks: enabledConfig},
		},
		"Empty plan builder returned when hooks disabled": {
			givenConfig:         config.Hooks{Enabled: false},
			expectedPlanBuilder: EmptyPlanBuilder{},
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			gotPlanBuilder := NewExecutionPlanBuilder(test.givenConfig, nil)
			assert.Equal(t, test.expectedPlanBuilder, gotPlanBuilder)
		})
	}
}

func TestPlanForEntrypointStage(t *testing.T) {
	testCases := map[string]struct {
		givenEndpoint               string
		givenHostPlanData           []byte
		givenDefaultAccountPlanData []byte
		givenHooks                  map[string]interface{}
		expectedPlan                Plan[hookstage.Entrypoint]
	}{
		"Host and default-account execution plans successfully merged": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"entrypoint":{"groups":[{"timeout":5,"hook_sequence":[{"module_code":"foobar","hook_impl_code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"entrypoint": {"groups": [{"timeout": 10, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "bar"}, {"module_code": "ortb2blocking", "hook_impl_code": "block_request"}]}, {"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [{"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}}}`),
			givenHooks: map[string]interface{}{
				"foobar":        fakeEntrypointHook{},
				"ortb2blocking": fakeEntrypointHook{},
			},
			expectedPlan: Plan[hookstage.Entrypoint]{
				// first group from host-level plan
				Group[hookstage.Entrypoint]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.Entrypoint]{
						{Module: "foobar", Code: "foo", Hook: fakeEntrypointHook{}},
					},
				},
				// then groups from the account-level plan
				Group[hookstage.Entrypoint]{
					Timeout: 10 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.Entrypoint]{
						{Module: "foobar", Code: "bar", Hook: fakeEntrypointHook{}},
						{Module: "ortb2blocking", Code: "block_request", Hook: fakeEntrypointHook{}},
					},
				},
				Group[hookstage.Entrypoint]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.Entrypoint]{
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
			expectedPlan: Plan[hookstage.Entrypoint]{
				Group[hookstage.Entrypoint]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.Entrypoint]{
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
			expectedPlan: Plan[hookstage.Entrypoint]{
				Group[hookstage.Entrypoint]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.Entrypoint]{
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
			expectedPlan:                Plan[hookstage.Entrypoint]{},
		},
		"Empty plan if hook repository empty": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"entrypoint":{"groups":[{"timeout":5,"hook_sequence":[{"module_code":"foobar","hook_impl_code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{}`),
			givenHooks:                  nil,
			expectedPlan:                Plan[hookstage.Entrypoint]{},
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
		expectedPlan                Plan[hookstage.RawAuction]
	}{
		"Account-specific execution plan rewrites default-account execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"rawauction":{"groups":[{"timeout":5,"hook_sequence":[{"module_code":"foobar","hook_impl_code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"rawauction": {"groups": [{"timeout": 10, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "bar"}, {"module_code": "ortb2blocking", "hook_impl_code": "block_request"}]}, {"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [{"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}}}`),
			giveAccountPlanData:         []byte(`{"execution_plan": {"endpoints": {"/openrtb2/auction": {"stages": {"rawauction": {"groups": [{"timeout": 15, "hook_sequence": [{"module_code": "prebid", "hook_impl_code": "baz"}]}]}}}}}}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[hookstage.RawAuction]{
				// first group from host-level plan
				Group[hookstage.RawAuction]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.RawAuction]{
						{Module: "foobar", Code: "foo", Hook: fakeRawAuctionHook{}},
					},
				},
				// then come groups from account-level plan (default-account-level plan ignored)
				Group[hookstage.RawAuction]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.RawAuction]{
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
			expectedPlan: Plan[hookstage.RawAuction]{
				Group[hookstage.RawAuction]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.RawAuction]{
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
			expectedPlan: Plan[hookstage.RawAuction]{
				Group[hookstage.RawAuction]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.RawAuction]{
						{Module: "foobar", Code: "foo", Hook: fakeRawAuctionHook{}},
					},
				},
				Group[hookstage.RawAuction]{
					Timeout: 10 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.RawAuction]{
						{Module: "foobar", Code: "bar", Hook: fakeRawAuctionHook{}},
						{Module: "ortb2blocking", Code: "block_request", Hook: fakeRawAuctionHook{}},
					},
				},
				Group[hookstage.RawAuction]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.RawAuction]{
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
		expectedPlan                Plan[hookstage.ProcessedAuction]
	}{
		"Account-specific execution plan rewrites default-account execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"procauction":{"groups":[{"timeout":5,"hook_sequence":[{"module_code":"foobar","hook_impl_code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"procauction": {"groups": [{"timeout": 10, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "bar"}, {"module_code": "ortb2blocking", "hook_impl_code": "block_request"}]}, {"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [{"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}}}`),
			giveAccountPlanData:         []byte(`{"execution_plan": {"endpoints": {"/openrtb2/auction": {"stages": {"procauction": {"groups": [{"timeout": 15, "hook_sequence": [{"module_code": "prebid", "hook_impl_code": "baz"}]}]}}}}}}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[hookstage.ProcessedAuction]{
				// first group from host-level plan
				Group[hookstage.ProcessedAuction]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.ProcessedAuction]{
						{Module: "foobar", Code: "foo", Hook: fakeProcessedAuctionHook{}},
					},
				},
				// then come groups from account-level plan (default-account-level plan ignored)
				Group[hookstage.ProcessedAuction]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.ProcessedAuction]{
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
			expectedPlan: Plan[hookstage.ProcessedAuction]{
				Group[hookstage.ProcessedAuction]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.ProcessedAuction]{
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
			expectedPlan: Plan[hookstage.ProcessedAuction]{
				Group[hookstage.ProcessedAuction]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.ProcessedAuction]{
						{Module: "foobar", Code: "foo", Hook: fakeProcessedAuctionHook{}},
					},
				},
				Group[hookstage.ProcessedAuction]{
					Timeout: 10 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.ProcessedAuction]{
						{Module: "foobar", Code: "bar", Hook: fakeProcessedAuctionHook{}},
						{Module: "ortb2blocking", Code: "block_request", Hook: fakeProcessedAuctionHook{}},
					},
				},
				Group[hookstage.ProcessedAuction]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.ProcessedAuction]{
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
		expectedPlan                Plan[hookstage.BidRequest]
	}{
		"Account-specific execution plan rewrites default-account execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"bidrequest":{"groups":[{"timeout":5,"hook_sequence":[{"module_code":"foobar","hook_impl_code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"bidrequest": {"groups": [{"timeout": 10, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "bar"}, {"module_code": "ortb2blocking", "hook_impl_code": "block_request"}]}, {"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [{"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}}}`),
			giveAccountPlanData:         []byte(`{"execution_plan": {"endpoints": {"/openrtb2/auction": {"stages": {"bidrequest": {"groups": [{"timeout": 15, "hook_sequence": [{"module_code": "prebid", "hook_impl_code": "baz"}]}]}}}}}}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[hookstage.BidRequest]{
				// first group from host-level plan
				Group[hookstage.BidRequest]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.BidRequest]{
						{Module: "foobar", Code: "foo", Hook: fakeBidRequestHook{}},
					},
				},
				// then come groups from account-level plan (default-account-level plan ignored)
				Group[hookstage.BidRequest]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.BidRequest]{
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
			expectedPlan: Plan[hookstage.BidRequest]{
				Group[hookstage.BidRequest]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.BidRequest]{
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
			expectedPlan: Plan[hookstage.BidRequest]{
				Group[hookstage.BidRequest]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.BidRequest]{
						{Module: "foobar", Code: "foo", Hook: fakeBidRequestHook{}},
					},
				},
				Group[hookstage.BidRequest]{
					Timeout: 10 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.BidRequest]{
						{Module: "foobar", Code: "bar", Hook: fakeBidRequestHook{}},
						{Module: "ortb2blocking", Code: "block_request", Hook: fakeBidRequestHook{}},
					},
				},
				Group[hookstage.BidRequest]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.BidRequest]{
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
		expectedPlan                Plan[hookstage.RawBidResponse]
	}{
		"Account-specific execution plan rewrites default-account execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"rawbidresponse":{"groups":[{"timeout":5,"hook_sequence":[{"module_code":"foobar","hook_impl_code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"rawbidresponse": {"groups": [{"timeout": 10, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "bar"}, {"module_code": "ortb2blocking", "hook_impl_code": "block_request"}]}, {"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [{"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}}}`),
			giveAccountPlanData:         []byte(`{"execution_plan": {"endpoints": {"/openrtb2/auction": {"stages": {"rawbidresponse": {"groups": [{"timeout": 15, "hook_sequence": [{"module_code": "prebid", "hook_impl_code": "baz"}]}]}}}}}}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[hookstage.RawBidResponse]{
				// first group from host-level plan
				Group[hookstage.RawBidResponse]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.RawBidResponse]{
						{Module: "foobar", Code: "foo", Hook: fakeRawBidResponseHook{}},
					},
				},
				// then come groups from account-level plan (default-account-level plan ignored)
				Group[hookstage.RawBidResponse]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.RawBidResponse]{
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
			expectedPlan: Plan[hookstage.RawBidResponse]{
				Group[hookstage.RawBidResponse]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.RawBidResponse]{
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
			expectedPlan: Plan[hookstage.RawBidResponse]{
				Group[hookstage.RawBidResponse]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.RawBidResponse]{
						{Module: "foobar", Code: "foo", Hook: fakeRawBidResponseHook{}},
					},
				},
				Group[hookstage.RawBidResponse]{
					Timeout: 10 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.RawBidResponse]{
						{Module: "foobar", Code: "bar", Hook: fakeRawBidResponseHook{}},
						{Module: "ortb2blocking", Code: "block_request", Hook: fakeRawBidResponseHook{}},
					},
				},
				Group[hookstage.RawBidResponse]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.RawBidResponse]{
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
		expectedPlan                Plan[hookstage.AllProcessedBidResponses]
	}{
		"Account-specific execution plan rewrites default-account execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"allprocbidresponses":{"groups":[{"timeout":5,"hook_sequence":[{"module_code":"foobar","hook_impl_code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"allprocbidresponses": {"groups": [{"timeout": 10, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "bar"}, {"module_code": "ortb2blocking", "hook_impl_code": "block_request"}]}, {"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [{"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}}}`),
			giveAccountPlanData:         []byte(`{"execution_plan": {"endpoints": {"/openrtb2/auction": {"stages": {"allprocbidresponses": {"groups": [{"timeout": 15, "hook_sequence": [{"module_code": "prebid", "hook_impl_code": "baz"}]}]}}}}}}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[hookstage.AllProcessedBidResponses]{
				// first group from host-level plan
				Group[hookstage.AllProcessedBidResponses]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.AllProcessedBidResponses]{
						{Module: "foobar", Code: "foo", Hook: fakeAllProcBidResponsesHook{}},
					},
				},
				// then come groups from account-level plan (default-account-level plan ignored)
				Group[hookstage.AllProcessedBidResponses]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.AllProcessedBidResponses]{
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
			expectedPlan: Plan[hookstage.AllProcessedBidResponses]{
				Group[hookstage.AllProcessedBidResponses]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.AllProcessedBidResponses]{
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
			expectedPlan: Plan[hookstage.AllProcessedBidResponses]{
				Group[hookstage.AllProcessedBidResponses]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.AllProcessedBidResponses]{
						{Module: "foobar", Code: "foo", Hook: fakeAllProcBidResponsesHook{}},
					},
				},
				Group[hookstage.AllProcessedBidResponses]{
					Timeout: 10 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.AllProcessedBidResponses]{
						{Module: "foobar", Code: "bar", Hook: fakeAllProcBidResponsesHook{}},
						{Module: "ortb2blocking", Code: "block_request", Hook: fakeAllProcBidResponsesHook{}},
					},
				},
				Group[hookstage.AllProcessedBidResponses]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.AllProcessedBidResponses]{
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
		expectedPlan                Plan[hookstage.AuctionResponse]
	}{
		"Account-specific execution plan rewrites default-account execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"auctionresponse":{"groups":[{"timeout":5,"hook_sequence":[{"module_code":"foobar","hook_impl_code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"auctionresponse": {"groups": [{"timeout": 10, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "bar"}, {"module_code": "ortb2blocking", "hook_impl_code": "block_request"}]}, {"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [{"timeout": 5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}]}}}}}`),
			giveAccountPlanData:         []byte(`{"execution_plan": {"endpoints": {"/openrtb2/auction": {"stages": {"auctionresponse": {"groups": [{"timeout": 15, "hook_sequence": [{"module_code": "prebid", "hook_impl_code": "baz"}]}]}}}}}}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[hookstage.AuctionResponse]{
				// first group from host-level plan
				Group[hookstage.AuctionResponse]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.AuctionResponse]{
						{Module: "foobar", Code: "foo", Hook: fakeAuctionResponseHook{}},
					},
				},
				// then come groups from account-level plan (default-account-level plan ignored)
				Group[hookstage.AuctionResponse]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.AuctionResponse]{
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
			expectedPlan: Plan[hookstage.AuctionResponse]{
				Group[hookstage.AuctionResponse]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.AuctionResponse]{
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
			expectedPlan: Plan[hookstage.AuctionResponse]{
				Group[hookstage.AuctionResponse]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.AuctionResponse]{
						{Module: "foobar", Code: "foo", Hook: fakeAuctionResponseHook{}},
					},
				},
				Group[hookstage.AuctionResponse]{
					Timeout: 10 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.AuctionResponse]{
						{Module: "foobar", Code: "bar", Hook: fakeAuctionResponseHook{}},
						{Module: "ortb2blocking", Code: "block_request", Hook: fakeAuctionResponseHook{}},
					},
				},
				Group[hookstage.AuctionResponse]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.AuctionResponse]{
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

	hooks.Enabled = true
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
	_ hookstage.InvocationContext,
	_ hookstage.EntrypointPayload,
) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	return hookstage.HookResult[hookstage.EntrypointPayload]{}, nil
}

type fakeRawAuctionHook struct{}

func (f fakeRawAuctionHook) HandleRawAuctionHook(
	_ context.Context,
	_ hookstage.InvocationContext,
	_ hookstage.RawAuctionPayload,
) (hookstage.HookResult[hookstage.RawAuctionPayload], error) {
	return hookstage.HookResult[hookstage.RawAuctionPayload]{}, nil
}

type fakeProcessedAuctionHook struct{}

func (f fakeProcessedAuctionHook) HandleProcessedAuctionHook(
	_ context.Context,
	_ hookstage.InvocationContext,
	_ hookstage.ProcessedAuctionPayload,
) (hookstage.HookResult[hookstage.ProcessedAuctionPayload], error) {
	return hookstage.HookResult[hookstage.ProcessedAuctionPayload]{}, nil
}

type fakeBidRequestHook struct{}

func (f fakeBidRequestHook) HandleBidRequestHook(
	_ context.Context,
	_ hookstage.InvocationContext,
	_ hookstage.BidRequestPayload,
) (hookstage.HookResult[hookstage.BidRequestPayload], error) {
	return hookstage.HookResult[hookstage.BidRequestPayload]{}, nil
}

type fakeRawBidResponseHook struct{}

func (f fakeRawBidResponseHook) HandleRawBidResponseHook(
	_ context.Context,
	_ hookstage.InvocationContext,
	_ hookstage.RawBidResponsePayload,
) (hookstage.HookResult[hookstage.RawBidResponsePayload], error) {
	return hookstage.HookResult[hookstage.RawBidResponsePayload]{}, nil
}

type fakeAllProcBidResponsesHook struct{}

func (f fakeAllProcBidResponsesHook) HandleAllProcBidResponsesHook(
	_ context.Context,
	_ hookstage.InvocationContext,
	_ hookstage.AllProcessedBidResponsesPayload,
) (hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload], error) {
	return hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload]{}, nil
}

type fakeAuctionResponseHook struct{}

func (f fakeAuctionResponseHook) HandleAuctionResponseHook(
	_ context.Context,
	_ hookstage.InvocationContext,
	_ *openrtb2.BidResponse,
) (hookstage.HookResult[*openrtb2.BidResponse], error) {
	return hookstage.HookResult[*openrtb2.BidResponse]{}, nil
}
