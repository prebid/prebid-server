package modules

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/prebid/prebid-server/v3/di"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"math"
	"net/http"
	"testing"

	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/hooks"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/moduledeps"
	"github.com/stretchr/testify/assert"
)

func TestModuleBuilderBuild(t *testing.T) {
	vendor := "acme"
	moduleName := "foobar"
	defaultModulesConfig := map[string]map[string]interface{}{vendor: {moduleName: map[string]interface{}{"enabled": true}}}
	defaultHookRepository, err := hooks.NewHookRepository(map[string]interface{}{vendor + "." + moduleName: module{}})
	if err != nil {
		t.Fatalf("Failed to init default hook repository: %s", err)
	}
	emptyHookRepository, err := hooks.NewHookRepository(map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to init empty hook repository: %s", err)
	}

	testCases := map[string]struct {
		givenModule             interface{}
		givenConfig             config.Modules
		givenHookBuilderErr     error
		expectedHookRepo        hooks.HookRepository
		expectedModulesStages   map[string][]string
		expectedModulesDisabled []string
		expectedErr             error
	}{
		"Can build module with config": {
			givenModule:             module{},
			givenConfig:             defaultModulesConfig,
			expectedModulesStages:   map[string][]string{vendor + "_" + moduleName: {hooks.StageEntrypoint.String(), hooks.StageAuctionResponse.String()}},
			expectedModulesDisabled: []string{},
			expectedHookRepo:        defaultHookRepository,
			expectedErr:             nil,
		},
		"Module is not added to hook repository if it's disabled": {
			givenModule:             module{},
			givenConfig:             map[string]map[string]interface{}{vendor: {moduleName: map[string]interface{}{"enabled": false, "attr": "val"}}},
			expectedModulesStages:   map[string][]string{},
			expectedModulesDisabled: []string{fmt.Sprintf("%s.%s", vendor, moduleName)},
			expectedHookRepo:        emptyHookRepository,
			expectedErr:             nil,
		},
		"Module considered disabled if status property not defined in module config": {
			givenModule:             module{},
			givenConfig:             map[string]map[string]interface{}{vendor: {moduleName: map[string]interface{}{"foo": "bar"}}},
			expectedHookRepo:        emptyHookRepository,
			expectedModulesStages:   map[string][]string{},
			expectedModulesDisabled: []string{fmt.Sprintf("%s.%s", vendor, moduleName)},
			expectedErr:             nil,
		},
		"Module considered disabled if its config not provided and as a result skipped from execution": {
			givenModule:             module{},
			givenConfig:             nil,
			expectedHookRepo:        emptyHookRepository,
			expectedModulesStages:   map[string][]string{},
			expectedModulesDisabled: []string{fmt.Sprintf("%s.%s", vendor, moduleName)},
			expectedErr:             nil,
		},
		"Fails if module does not implement any hook interface": {
			givenModule:             struct{}{},
			givenConfig:             defaultModulesConfig,
			expectedHookRepo:        nil,
			expectedModulesStages:   nil,
			expectedModulesDisabled: nil,
			expectedErr:             fmt.Errorf(`hook "%s.%s" does not implement any supported hook interface`, vendor, moduleName),
		},
		"Fails if module builder function returns error": {
			givenModule:             module{},
			givenConfig:             defaultModulesConfig,
			givenHookBuilderErr:     errors.New("failed to build module"),
			expectedHookRepo:        nil,
			expectedModulesStages:   nil,
			expectedModulesDisabled: nil,
			expectedErr:             fmt.Errorf(`failed to init "%s.%s" module: %s`, vendor, moduleName, "failed to build module"),
		},
		"Fails if config marshaling returns error": {
			givenModule:             module{},
			givenConfig:             map[string]map[string]interface{}{vendor: {moduleName: math.Inf(1)}},
			expectedHookRepo:        nil,
			expectedModulesStages:   nil,
			expectedModulesDisabled: nil,
			expectedErr:             fmt.Errorf(`failed to marshal "%s.%s" module config: unsupported value: +Inf`, vendor, moduleName),
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			builder := &builder{
				builders: ModuleBuilders{
					vendor: {
						moduleName: func(cfg json.RawMessage, deps moduledeps.ModuleDeps) (interface{}, error) {
							return test.givenModule, test.givenHookBuilderErr
						},
					},
				},
			}

			repo, modulesStages, modulesDisabled, err := builder.Build(test.givenConfig, moduledeps.ModuleDeps{HTTPClient: http.DefaultClient})
			assert.Equal(t, test.expectedErr, err)
			assert.Equal(t, test.expectedModulesStages, modulesStages)
			assert.Equal(t, test.expectedModulesDisabled, modulesDisabled)
			assert.Equal(t, test.expectedHookRepo, repo)
		})
	}
}

func TestPlanForDisabledModule(t *testing.T) {
	testCases := map[string]struct {
		moduleCode        string
		enabled           bool
		expectedPlanLen   int
		expectedLogOutput string
	}{
		"Correct module_code and module enabled = plan contains one hook": {
			moduleCode:        "prebid.ortb2blocking",
			enabled:           true,
			expectedPlanLen:   1,
			expectedLogOutput: "",
		},
		"Incorrect module_code but module enabled = plan contains no hooks": {
			moduleCode:        "prebid_ortb2blocking",
			enabled:           true,
			expectedPlanLen:   0,
			expectedLogOutput: "Not found hook while building hook execution plan: prebid_ortb2blocking foo",
		},
		"Correct module_code but module disabled = plan contains no hooks": {
			moduleCode:        "prebid.ortb2blocking",
			enabled:           false,
			expectedPlanLen:   0,
			expectedLogOutput: "",
		},
	}

	old_logger := di.Logger
	testLogger := TestLogger{}
	di.Logger = &testLogger
	defer func() { di.Logger = old_logger }()

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			testLogger.log = ""
			hooksCfgData, accountCfgData := constructCfg(test.moduleCode, test.enabled)
			var hooksCfg config.Hooks
			var accountCfg config.Account
			var err = jsonutil.UnmarshalValid([]byte(hooksCfgData), &hooksCfg)
			assert.Nil(t, err)

			err = jsonutil.UnmarshalValid([]byte(accountCfgData), &accountCfg)
			assert.Nil(t, err)

			planBuilder, err := constructPlanBuilder(hooksCfg)
			assert.Nil(t, err)

			plan := planBuilder.PlanForRawBidderResponseStage("/openrtb2/auction", &accountCfg)
			assert.Equal(t, test.expectedPlanLen, len(plan))
			assert.Equal(t, test.expectedLogOutput, testLogger.log)

		})
	}
}

type TestLogger struct {
	log string
}

func (logger *TestLogger) Warningf(format string, args ...interface{}) {
	logger.log = logger.log + fmt.Sprintf(format, args...)
}

func constructCfg(module_code string, enabled bool) (string, string) {
	group := `{"timeout":  5, "hook_sequence": [{"module_code": "` + module_code + `", "hook_impl_code": "foo"}]}`
	executionPlanData := `{"endpoints": {"/openrtb2/auction": {"stages": {"raw_bidder_response": {"groups": [` + group + `]}}}}}`
	enabledS := `false`
	if enabled {
		enabledS = `true`
	}
	modules := `"modules": {"prebid": {"ortb2blocking": {"enabled": ` + enabledS + `}}}`
	hooksCfgData := `{"enabled":true, ` + modules + `, "execution_plan": ` + executionPlanData + `}`
	accountCfgData := `{"hooks":` + hooksCfgData + `}`
	return hooksCfgData, accountCfgData
}

func constructPlanBuilder(cfgHooks config.Hooks) (hooks.ExecutionPlanBuilder, error) {
	moduleDeps := moduledeps.ModuleDeps{}
	repo, _, disabledModuleCodes, err := NewBuilder().Build(cfgHooks.Modules, moduleDeps)
	if err != nil {
		return nil, err
	}

	planBuilder := hooks.NewExecutionPlanBuilder(cfgHooks, repo, disabledModuleCodes)
	return planBuilder, nil
}

type module struct{}

func (h module) HandleEntrypointHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	return hookstage.HookResult[hookstage.EntrypointPayload]{}, nil
}

func (h module) HandleAuctionResponseHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.AuctionResponsePayload) (hookstage.HookResult[hookstage.AuctionResponsePayload], error) {
	return hookstage.HookResult[hookstage.AuctionResponsePayload]{}, nil
}
