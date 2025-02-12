package hooks

import (
	"context"
	"testing"
	"time"

	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
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
	const group1 string = `{"timeout":  5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}`
	const group2 string = `{"timeout": 10, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "bar"}, {"module_code": "ortb2blocking", "hook_impl_code": "block_request"}]}`
	const planData1 string = `{"endpoints": {"/openrtb2/auction": {"stages": {"entrypoint": {"groups": [` + group1 + `]}}}}}`
	const planData2 string = `{"endpoints": {"/openrtb2/auction": {"stages": {"entrypoint": {"groups": [` + group2 + `,` + group1 + `]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [` + group1 + `]}}}}}`

	testCases := map[string]struct {
		givenEndpoint               string
		givenHostPlanData           []byte
		givenDefaultAccountPlanData []byte
		givenHooks                  map[string]interface{}
		expectedPlan                Plan[hookstage.Entrypoint]
	}{
		"Host and default-account execution plans successfully merged": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(planData1),
			givenDefaultAccountPlanData: []byte(planData2),
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
			givenHostPlanData:           []byte(planData1),
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
			givenDefaultAccountPlanData: []byte(planData1),
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
			givenHostPlanData:           []byte(planData1),
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
	const group1 string = `{"timeout":  5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}`
	const group2 string = `{"timeout": 10, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "bar"}, {"module_code": "ortb2blocking", "hook_impl_code": "block_request"}]}`
	const group3 string = `{"timeout": 15, "hook_sequence": [{"module_code": "prebid", "hook_impl_code": "baz"}]}`
	const hostPlanData string = `{"endpoints": {"/openrtb2/auction": {"stages": {"raw_auction_request": {"groups": [` + group1 + `]}}}}}`
	const defaultAccountPlanData string = `{"endpoints": {"/openrtb2/auction": {"stages": {"raw_auction_request": {"groups": [` + group2 + `,` + group1 + `]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [` + group1 + `]}}}}}`
	const accountPlanData string = `{"execution_plan": {"endpoints": {"/openrtb2/auction": {"stages": {"raw_auction_request": {"groups": [` + group3 + `]}}}}}}`

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
		expectedPlan                Plan[hookstage.RawAuctionRequest]
	}{
		"Account-specific execution plan rewrites default-account execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(hostPlanData),
			givenDefaultAccountPlanData: []byte(defaultAccountPlanData),
			giveAccountPlanData:         []byte(accountPlanData),
			givenHooks:                  hooks,
			expectedPlan: Plan[hookstage.RawAuctionRequest]{
				// first group from host-level plan
				Group[hookstage.RawAuctionRequest]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.RawAuctionRequest]{
						{Module: "foobar", Code: "foo", Hook: fakeRawAuctionHook{}},
					},
				},
				// then come groups from account-level plan (default-account-level plan ignored)
				Group[hookstage.RawAuctionRequest]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.RawAuctionRequest]{
						{Module: "prebid", Code: "baz", Hook: fakeRawAuctionHook{}},
					},
				},
			},
		},
		"Works with only account-specific plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{}`),
			givenDefaultAccountPlanData: []byte(`{}`),
			giveAccountPlanData:         []byte(accountPlanData),
			givenHooks:                  hooks,
			expectedPlan: Plan[hookstage.RawAuctionRequest]{
				Group[hookstage.RawAuctionRequest]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.RawAuctionRequest]{
						{Module: "prebid", Code: "baz", Hook: fakeRawAuctionHook{}},
					},
				},
			},
		},
		"Works with empty account-specific execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(hostPlanData),
			givenDefaultAccountPlanData: []byte(defaultAccountPlanData),
			giveAccountPlanData:         []byte(`{}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[hookstage.RawAuctionRequest]{
				Group[hookstage.RawAuctionRequest]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.RawAuctionRequest]{
						{Module: "foobar", Code: "foo", Hook: fakeRawAuctionHook{}},
					},
				},
				Group[hookstage.RawAuctionRequest]{
					Timeout: 10 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.RawAuctionRequest]{
						{Module: "foobar", Code: "bar", Hook: fakeRawAuctionHook{}},
						{Module: "ortb2blocking", Code: "block_request", Hook: fakeRawAuctionHook{}},
					},
				},
				Group[hookstage.RawAuctionRequest]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.RawAuctionRequest]{
						{Module: "foobar", Code: "foo", Hook: fakeRawAuctionHook{}},
					},
				},
			},
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			account := new(config.Account)
			if err := jsonutil.UnmarshalValid(test.giveAccountPlanData, &account.Hooks); err != nil {
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
	const group1 string = `{"timeout":  5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}`
	const group2 string = `{"timeout": 10, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "bar"}, {"module_code": "ortb2blocking", "hook_impl_code": "block_request"}]}`
	const group3 string = `{"timeout": 15, "hook_sequence": [{"module_code": "prebid", "hook_impl_code": "baz"}]}`
	const hostPlanData string = `{"endpoints": {"/openrtb2/auction": {"stages": {"processed_auction_request": {"groups": [` + group1 + `]}}}}}`
	const defaultAccountPlanData string = `{"endpoints": {"/openrtb2/auction": {"stages": {"processed_auction_request": {"groups": [` + group2 + `,` + group1 + `]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [` + group1 + `]}}}}}`
	const accountPlanData string = `{"execution_plan": {"endpoints": {"/openrtb2/auction": {"stages": {"processed_auction_request": {"groups": [` + group3 + `]}}}}}}`

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
		expectedPlan                Plan[hookstage.ProcessedAuctionRequest]
	}{
		"Account-specific execution plan rewrites default-account execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(hostPlanData),
			givenDefaultAccountPlanData: []byte(defaultAccountPlanData),
			giveAccountPlanData:         []byte(accountPlanData),
			givenHooks:                  hooks,
			expectedPlan: Plan[hookstage.ProcessedAuctionRequest]{
				// first group from host-level plan
				Group[hookstage.ProcessedAuctionRequest]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.ProcessedAuctionRequest]{
						{Module: "foobar", Code: "foo", Hook: fakeProcessedAuctionHook{}},
					},
				},
				// then come groups from account-level plan (default-account-level plan ignored)
				Group[hookstage.ProcessedAuctionRequest]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.ProcessedAuctionRequest]{
						{Module: "prebid", Code: "baz", Hook: fakeProcessedAuctionHook{}},
					},
				},
			},
		},
		"Works with only account-specific plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{}`),
			givenDefaultAccountPlanData: []byte(`{}`),
			giveAccountPlanData:         []byte(accountPlanData),
			givenHooks:                  hooks,
			expectedPlan: Plan[hookstage.ProcessedAuctionRequest]{
				Group[hookstage.ProcessedAuctionRequest]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.ProcessedAuctionRequest]{
						{Module: "prebid", Code: "baz", Hook: fakeProcessedAuctionHook{}},
					},
				},
			},
		},
		"Works with empty account-specific execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(hostPlanData),
			givenDefaultAccountPlanData: []byte(defaultAccountPlanData),
			giveAccountPlanData:         []byte(`{}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[hookstage.ProcessedAuctionRequest]{
				Group[hookstage.ProcessedAuctionRequest]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.ProcessedAuctionRequest]{
						{Module: "foobar", Code: "foo", Hook: fakeProcessedAuctionHook{}},
					},
				},
				Group[hookstage.ProcessedAuctionRequest]{
					Timeout: 10 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.ProcessedAuctionRequest]{
						{Module: "foobar", Code: "bar", Hook: fakeProcessedAuctionHook{}},
						{Module: "ortb2blocking", Code: "block_request", Hook: fakeProcessedAuctionHook{}},
					},
				},
				Group[hookstage.ProcessedAuctionRequest]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.ProcessedAuctionRequest]{
						{Module: "foobar", Code: "foo", Hook: fakeProcessedAuctionHook{}},
					},
				},
			},
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			account := new(config.Account)
			if err := jsonutil.UnmarshalValid(test.giveAccountPlanData, &account.Hooks); err != nil {
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

func TestPlanForBidderRequestStage(t *testing.T) {
	const group1 string = `{"timeout":  5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}`
	const group2 string = `{"timeout": 10, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "bar"}, {"module_code": "ortb2blocking", "hook_impl_code": "block_request"}]}`
	const group3 string = `{"timeout": 15, "hook_sequence": [{"module_code": "prebid", "hook_impl_code": "baz"}]}`
	const hostPlanData string = `{"endpoints": {"/openrtb2/auction": {"stages": {"bidder_request": {"groups": [` + group1 + `]}}}}}`
	const defaultAccountPlanData string = `{"endpoints": {"/openrtb2/auction": {"stages": {"bidder_request": {"groups": [` + group2 + `,` + group1 + `]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [` + group1 + `]}}}}}`
	const accountPlanData string = `{"execution_plan": {"endpoints": {"/openrtb2/auction": {"stages": {"bidder_request": {"groups": [` + group3 + `]}}}}}}`

	hooks := map[string]interface{}{
		"foobar":        fakeBidderRequestHook{},
		"ortb2blocking": fakeBidderRequestHook{},
		"prebid":        fakeBidderRequestHook{},
	}

	testCases := map[string]struct {
		givenEndpoint               string
		givenHostPlanData           []byte
		givenDefaultAccountPlanData []byte
		giveAccountPlanData         []byte
		givenHooks                  map[string]interface{}
		expectedPlan                Plan[hookstage.BidderRequest]
	}{
		"Account-specific execution plan rewrites default-account execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(hostPlanData),
			givenDefaultAccountPlanData: []byte(defaultAccountPlanData),
			giveAccountPlanData:         []byte(accountPlanData),
			givenHooks:                  hooks,
			expectedPlan: Plan[hookstage.BidderRequest]{
				// first group from host-level plan
				Group[hookstage.BidderRequest]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.BidderRequest]{
						{Module: "foobar", Code: "foo", Hook: fakeBidderRequestHook{}},
					},
				},
				// then come groups from account-level plan (default-account-level plan ignored)
				Group[hookstage.BidderRequest]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.BidderRequest]{
						{Module: "prebid", Code: "baz", Hook: fakeBidderRequestHook{}},
					},
				},
			},
		},
		"Works with only account-specific plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{}`),
			givenDefaultAccountPlanData: []byte(`{}`),
			giveAccountPlanData:         []byte(accountPlanData),
			givenHooks:                  hooks,
			expectedPlan: Plan[hookstage.BidderRequest]{
				Group[hookstage.BidderRequest]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.BidderRequest]{
						{Module: "prebid", Code: "baz", Hook: fakeBidderRequestHook{}},
					},
				},
			},
		},
		"Works with empty account-specific execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(hostPlanData),
			givenDefaultAccountPlanData: []byte(defaultAccountPlanData),
			giveAccountPlanData:         []byte(`{}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[hookstage.BidderRequest]{
				Group[hookstage.BidderRequest]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.BidderRequest]{
						{Module: "foobar", Code: "foo", Hook: fakeBidderRequestHook{}},
					},
				},
				Group[hookstage.BidderRequest]{
					Timeout: 10 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.BidderRequest]{
						{Module: "foobar", Code: "bar", Hook: fakeBidderRequestHook{}},
						{Module: "ortb2blocking", Code: "block_request", Hook: fakeBidderRequestHook{}},
					},
				},
				Group[hookstage.BidderRequest]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.BidderRequest]{
						{Module: "foobar", Code: "foo", Hook: fakeBidderRequestHook{}},
					},
				},
			},
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			account := new(config.Account)
			if err := jsonutil.UnmarshalValid(test.giveAccountPlanData, &account.Hooks); err != nil {
				t.Fatal(err)
			}

			planBuilder, err := getPlanBuilder(test.givenHooks, test.givenHostPlanData, test.givenDefaultAccountPlanData)
			if assert.NoError(t, err, "Failed to init hook execution plan builder") {
				plan := planBuilder.PlanForBidderRequestStage(test.givenEndpoint, account)
				assert.Equal(t, test.expectedPlan, plan)
			}
		})
	}
}

func TestPlanForRawBidderResponseStage(t *testing.T) {
	const group1 string = `{"timeout":  5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}`
	const group2 string = `{"timeout": 10, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "bar"}, {"module_code": "ortb2blocking", "hook_impl_code": "block_request"}]}`
	const group3 string = `{"timeout": 15, "hook_sequence": [{"module_code": "prebid", "hook_impl_code": "baz"}]}`
	const hostPlanData string = `{"endpoints": {"/openrtb2/auction": {"stages": {"raw_bidder_response": {"groups": [` + group1 + `]}}}}}`
	const defaultAccountPlanData string = `{"endpoints": {"/openrtb2/auction": {"stages": {"raw_bidder_response": {"groups": [` + group2 + `,` + group1 + `]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [` + group1 + `]}}}}}`
	const accountPlanData string = `{"execution_plan": {"endpoints": {"/openrtb2/auction": {"stages": {"raw_bidder_response": {"groups": [` + group3 + `]}}}}}}`

	hooks := map[string]interface{}{
		"foobar":        fakeRawBidderResponseHook{},
		"ortb2blocking": fakeRawBidderResponseHook{},
		"prebid":        fakeRawBidderResponseHook{},
	}

	testCases := map[string]struct {
		givenEndpoint               string
		givenHostPlanData           []byte
		givenDefaultAccountPlanData []byte
		giveAccountPlanData         []byte
		givenHooks                  map[string]interface{}
		expectedPlan                Plan[hookstage.RawBidderResponse]
	}{
		"Account-specific execution plan rewrites default-account execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(hostPlanData),
			givenDefaultAccountPlanData: []byte(defaultAccountPlanData),
			giveAccountPlanData:         []byte(accountPlanData),
			givenHooks:                  hooks,
			expectedPlan: Plan[hookstage.RawBidderResponse]{
				// first group from host-level plan
				Group[hookstage.RawBidderResponse]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.RawBidderResponse]{
						{Module: "foobar", Code: "foo", Hook: fakeRawBidderResponseHook{}},
					},
				},
				// then come groups from account-level plan (default-account-level plan ignored)
				Group[hookstage.RawBidderResponse]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.RawBidderResponse]{
						{Module: "prebid", Code: "baz", Hook: fakeRawBidderResponseHook{}},
					},
				},
			},
		},
		"Works with only account-specific plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{}`),
			givenDefaultAccountPlanData: []byte(`{}`),
			giveAccountPlanData:         []byte(accountPlanData),
			givenHooks:                  hooks,
			expectedPlan: Plan[hookstage.RawBidderResponse]{
				Group[hookstage.RawBidderResponse]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.RawBidderResponse]{
						{Module: "prebid", Code: "baz", Hook: fakeRawBidderResponseHook{}},
					},
				},
			},
		},
		"Works with empty account-specific execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(hostPlanData),
			givenDefaultAccountPlanData: []byte(defaultAccountPlanData),
			giveAccountPlanData:         []byte(`{}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[hookstage.RawBidderResponse]{
				Group[hookstage.RawBidderResponse]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.RawBidderResponse]{
						{Module: "foobar", Code: "foo", Hook: fakeRawBidderResponseHook{}},
					},
				},
				Group[hookstage.RawBidderResponse]{
					Timeout: 10 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.RawBidderResponse]{
						{Module: "foobar", Code: "bar", Hook: fakeRawBidderResponseHook{}},
						{Module: "ortb2blocking", Code: "block_request", Hook: fakeRawBidderResponseHook{}},
					},
				},
				Group[hookstage.RawBidderResponse]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.RawBidderResponse]{
						{Module: "foobar", Code: "foo", Hook: fakeRawBidderResponseHook{}},
					},
				},
			},
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			account := new(config.Account)
			if err := jsonutil.UnmarshalValid(test.giveAccountPlanData, &account.Hooks); err != nil {
				t.Fatal(err)
			}

			planBuilder, err := getPlanBuilder(test.givenHooks, test.givenHostPlanData, test.givenDefaultAccountPlanData)
			if assert.NoError(t, err, "Failed to init hook execution plan builder") {
				plan := planBuilder.PlanForRawBidderResponseStage(test.givenEndpoint, account)
				assert.Equal(t, test.expectedPlan, plan)
			}
		})
	}
}

func TestPlanForAllProcessedBidResponsesStage(t *testing.T) {
	const group1 string = `{"timeout":  5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}`
	const group2 string = `{"timeout": 10, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "bar"}, {"module_code": "ortb2blocking", "hook_impl_code": "block_request"}]}`
	const group3 string = `{"timeout": 15, "hook_sequence": [{"module_code": "prebid", "hook_impl_code": "baz"}]}`
	const hostPlanData string = `{"endpoints": {"/openrtb2/auction": {"stages": {"all_processed_bid_responses": {"groups": [` + group1 + `]}}}}}`
	const defaultAccountPlanData string = `{"endpoints": {"/openrtb2/auction": {"stages": {"all_processed_bid_responses": {"groups": [` + group2 + `,` + group1 + `]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [` + group1 + `]}}}}}`
	const accountPlanData string = `{"execution_plan": {"endpoints": {"/openrtb2/auction": {"stages": {"all_processed_bid_responses": {"groups": [` + group3 + `]}}}}}}`

	hooks := map[string]interface{}{
		"foobar":        fakeAllProcessedBidResponsesHook{},
		"ortb2blocking": fakeAllProcessedBidResponsesHook{},
		"prebid":        fakeAllProcessedBidResponsesHook{},
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
			givenHostPlanData:           []byte(hostPlanData),
			givenDefaultAccountPlanData: []byte(defaultAccountPlanData),
			giveAccountPlanData:         []byte(accountPlanData),
			givenHooks:                  hooks,
			expectedPlan: Plan[hookstage.AllProcessedBidResponses]{
				// first group from host-level plan
				Group[hookstage.AllProcessedBidResponses]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.AllProcessedBidResponses]{
						{Module: "foobar", Code: "foo", Hook: fakeAllProcessedBidResponsesHook{}},
					},
				},
				// then come groups from account-level plan (default-account-level plan ignored)
				Group[hookstage.AllProcessedBidResponses]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.AllProcessedBidResponses]{
						{Module: "prebid", Code: "baz", Hook: fakeAllProcessedBidResponsesHook{}},
					},
				},
			},
		},
		"Works with only account-specific plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{}`),
			givenDefaultAccountPlanData: []byte(`{}`),
			giveAccountPlanData:         []byte(accountPlanData),
			givenHooks:                  hooks,
			expectedPlan: Plan[hookstage.AllProcessedBidResponses]{
				Group[hookstage.AllProcessedBidResponses]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.AllProcessedBidResponses]{
						{Module: "prebid", Code: "baz", Hook: fakeAllProcessedBidResponsesHook{}},
					},
				},
			},
		},
		"Works with empty account-specific execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(hostPlanData),
			givenDefaultAccountPlanData: []byte(defaultAccountPlanData),
			giveAccountPlanData:         []byte(`{}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[hookstage.AllProcessedBidResponses]{
				Group[hookstage.AllProcessedBidResponses]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.AllProcessedBidResponses]{
						{Module: "foobar", Code: "foo", Hook: fakeAllProcessedBidResponsesHook{}},
					},
				},
				Group[hookstage.AllProcessedBidResponses]{
					Timeout: 10 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.AllProcessedBidResponses]{
						{Module: "foobar", Code: "bar", Hook: fakeAllProcessedBidResponsesHook{}},
						{Module: "ortb2blocking", Code: "block_request", Hook: fakeAllProcessedBidResponsesHook{}},
					},
				},
				Group[hookstage.AllProcessedBidResponses]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[hookstage.AllProcessedBidResponses]{
						{Module: "foobar", Code: "foo", Hook: fakeAllProcessedBidResponsesHook{}},
					},
				},
			},
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			account := new(config.Account)
			if err := jsonutil.UnmarshalValid(test.giveAccountPlanData, &account.Hooks); err != nil {
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
	const group1 string = `{"timeout":  5, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "foo"}]}`
	const group2 string = `{"timeout": 10, "hook_sequence": [{"module_code": "foobar", "hook_impl_code": "bar"}, {"module_code": "ortb2blocking", "hook_impl_code": "block_request"}]}`
	const group3 string = `{"timeout": 15, "hook_sequence": [{"module_code": "prebid", "hook_impl_code": "baz"}]}`
	const hostPlanData string = `{"endpoints": {"/openrtb2/auction": {"stages": {"auction_response": {"groups": [` + group1 + `]}}}}}`
	const defaultAccountPlanData string = `{"endpoints": {"/openrtb2/auction": {"stages": {"auction_response": {"groups": [` + group2 + `,` + group1 + `]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [` + group1 + `]}}}}}`
	const accountPlanData string = `{"execution_plan": {"endpoints": {"/openrtb2/auction": {"stages": {"auction_response": {"groups": [` + group3 + `]}}}}}}`

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
			givenHostPlanData:           []byte(hostPlanData),
			givenDefaultAccountPlanData: []byte(defaultAccountPlanData),
			giveAccountPlanData:         []byte(accountPlanData),
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
			giveAccountPlanData:         []byte(accountPlanData),
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
			givenHostPlanData:           []byte(hostPlanData),
			givenDefaultAccountPlanData: []byte(defaultAccountPlanData),
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
			if err := jsonutil.UnmarshalValid(test.giveAccountPlanData, &account.Hooks); err != nil {
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

	err = jsonutil.UnmarshalValid(hostPlanData, &hostPlan)
	if err != nil {
		return nil, err
	}

	err = jsonutil.UnmarshalValid(accountPlanData, &defaultAccountPlan)
	if err != nil {
		return nil, err
	}

	hooks.Enabled = true
	hooks.HostExecutionPlan = hostPlan
	hooks.DefaultAccountExecutionPlan = defaultAccountPlan

	repo, err := NewHookRepository(moduleHooks)
	if err != nil {
		return nil, err
	}

	return NewExecutionPlanBuilder(hooks, repo), nil
}

type fakeEntrypointHook struct{}

func (h fakeEntrypointHook) HandleEntrypointHook(
	_ context.Context,
	_ hookstage.ModuleInvocationContext,
	_ hookstage.EntrypointPayload,
) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	return hookstage.HookResult[hookstage.EntrypointPayload]{}, nil
}

type fakeRawAuctionHook struct{}

func (f fakeRawAuctionHook) HandleRawAuctionHook(
	_ context.Context,
	_ hookstage.ModuleInvocationContext,
	_ hookstage.RawAuctionRequestPayload,
) (hookstage.HookResult[hookstage.RawAuctionRequestPayload], error) {
	return hookstage.HookResult[hookstage.RawAuctionRequestPayload]{}, nil
}

type fakeProcessedAuctionHook struct{}

func (f fakeProcessedAuctionHook) HandleProcessedAuctionHook(
	_ context.Context,
	_ hookstage.ModuleInvocationContext,
	_ hookstage.ProcessedAuctionRequestPayload,
) (hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload], error) {
	return hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload]{}, nil
}

type fakeBidderRequestHook struct{}

func (f fakeBidderRequestHook) HandleBidderRequestHook(
	_ context.Context,
	_ hookstage.ModuleInvocationContext,
	_ hookstage.BidderRequestPayload,
) (hookstage.HookResult[hookstage.BidderRequestPayload], error) {
	return hookstage.HookResult[hookstage.BidderRequestPayload]{}, nil
}

type fakeRawBidderResponseHook struct{}

func (f fakeRawBidderResponseHook) HandleRawBidderResponseHook(
	_ context.Context,
	_ hookstage.ModuleInvocationContext,
	_ hookstage.RawBidderResponsePayload,
) (hookstage.HookResult[hookstage.RawBidderResponsePayload], error) {
	return hookstage.HookResult[hookstage.RawBidderResponsePayload]{}, nil
}

type fakeAllProcessedBidResponsesHook struct{}

func (f fakeAllProcessedBidResponsesHook) HandleAllProcessedBidResponsesHook(
	_ context.Context,
	_ hookstage.ModuleInvocationContext,
	_ hookstage.AllProcessedBidResponsesPayload,
) (hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload], error) {
	return hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload]{}, nil
}

type fakeAuctionResponseHook struct{}

func (f fakeAuctionResponseHook) HandleAuctionResponseHook(
	_ context.Context,
	_ hookstage.ModuleInvocationContext,
	_ hookstage.AuctionResponsePayload,
) (hookstage.HookResult[hookstage.AuctionResponsePayload], error) {
	return hookstage.HookResult[hookstage.AuctionResponsePayload]{}, nil
}
