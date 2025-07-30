package modules

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
		expectedShutdownModules *ShutdownModules
		expectedErr             error
	}{
		"Can build module with config": {
			givenModule:             module{},
			givenConfig:             defaultModulesConfig,
			expectedModulesStages:   map[string][]string{vendor + "_" + moduleName: {hooks.StageEntrypoint.String(), hooks.StageAuctionResponse.String()}},
			expectedHookRepo:        defaultHookRepository,
			expectedShutdownModules: &ShutdownModules{modules: []Shutdowner{module{}}},
			expectedErr:             nil,
		},
		"Module is not added to hook repository if it's disabled": {
			givenModule:             module{},
			givenConfig:             map[string]map[string]interface{}{vendor: {moduleName: map[string]interface{}{"enabled": false, "attr": "val"}}},
			expectedModulesStages:   map[string][]string{},
			expectedHookRepo:        emptyHookRepository,
			expectedShutdownModules: &ShutdownModules{modules: []Shutdowner{}},
			expectedErr:             nil,
		},
		"Module considered disabled if status property not defined in module config": {
			givenModule:             module{},
			givenConfig:             map[string]map[string]interface{}{vendor: {moduleName: map[string]interface{}{"foo": "bar"}}},
			expectedHookRepo:        emptyHookRepository,
			expectedModulesStages:   map[string][]string{},
			expectedShutdownModules: &ShutdownModules{modules: []Shutdowner{}},
			expectedErr:             nil,
		},
		"Module considered disabled if its config not provided and as a result skipped from execution": {
			givenModule:             module{},
			givenConfig:             nil,
			expectedHookRepo:        emptyHookRepository,
			expectedModulesStages:   map[string][]string{},
			expectedShutdownModules: &ShutdownModules{modules: []Shutdowner{}},
			expectedErr:             nil,
		},
		"Fails if module does not implement any hook interface": {
			givenModule:             struct{}{},
			givenConfig:             defaultModulesConfig,
			expectedHookRepo:        nil,
			expectedModulesStages:   nil,
			expectedShutdownModules: nil,
			expectedErr:             fmt.Errorf(`hook "%s.%s" does not implement any supported hook interface`, vendor, moduleName),
		},
		"Fails if module builder function returns error": {
			givenModule:             module{},
			givenConfig:             defaultModulesConfig,
			givenHookBuilderErr:     errors.New("failed to build module"),
			expectedHookRepo:        nil,
			expectedModulesStages:   nil,
			expectedShutdownModules: nil,
			expectedErr:             fmt.Errorf(`failed to init "%s.%s" module: %s`, vendor, moduleName, "failed to build module"),
		},
		"Fails if config marshaling returns error": {
			givenModule:             module{},
			givenConfig:             map[string]map[string]interface{}{vendor: {moduleName: math.Inf(1)}},
			expectedHookRepo:        nil,
			expectedModulesStages:   nil,
			expectedShutdownModules: nil,
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

			repo, modulesStages, shutdownModules, err := builder.Build(test.givenConfig, moduledeps.ModuleDeps{HTTPClient: http.DefaultClient})
			assert.Equal(t, test.expectedErr, err)
			assert.Equal(t, test.expectedModulesStages, modulesStages)
			assert.Equal(t, test.expectedShutdownModules, shutdownModules)
			assert.Equal(t, test.expectedHookRepo, repo)
		})
	}
}

type module struct{}

func (h module) HandleEntrypointHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	return hookstage.HookResult[hookstage.EntrypointPayload]{}, nil
}

func (h module) HandleAuctionResponseHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.AuctionResponsePayload) (hookstage.HookResult[hookstage.AuctionResponsePayload], error) {
	return hookstage.HookResult[hookstage.AuctionResponsePayload]{}, nil
}

func (h module) Shutdown() error {
	return nil
}
