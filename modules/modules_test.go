package modules

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/hooks"
	"github.com/prebid/prebid-server/hooks/hookstage"
	"github.com/stretchr/testify/assert"
)

func TestModuleBuilderBuild(t *testing.T) {
	vendor := "acme"
	moduleName := "foobar"

	testCases := map[string]struct {
		isHookFound          bool
		expectedModStageColl map[string][]string
		expectedHook         interface{}
		givenModule          interface{}
		givenConfig          config.Modules
		expectedErr          error
		givenHookBuilderErr  error
		givenGetHookFn       func(repo hooks.HookRepository, module string) (interface{}, bool)
	}{
		"Can build with entrypoint hook without config": {
			isHookFound:          true,
			expectedModStageColl: map[string][]string{vendor + "_" + moduleName: {hooks.StageEntrypoint.String(), hooks.StageAuctionResponse.String()}},
			expectedHook:         module{},
			givenModule:          module{},
			givenGetHookFn: func(repo hooks.HookRepository, module string) (interface{}, bool) {
				return repo.GetEntrypointHook(module)
			},
		},
		"Can build with entrypoint hook with config": {
			isHookFound:          true,
			expectedModStageColl: map[string][]string{vendor + "_" + moduleName: {hooks.StageEntrypoint.String(), hooks.StageAuctionResponse.String()}},
			expectedHook:         module{},
			givenModule:          module{},
			givenConfig:          map[string]map[string]interface{}{vendor: {moduleName: map[string]bool{"enabled": true}}},
			givenGetHookFn: func(repo hooks.HookRepository, module string) (interface{}, bool) {
				return repo.GetEntrypointHook(module)
			},
		},
		"Can build with auction response hook": {
			isHookFound:          true,
			expectedModStageColl: map[string][]string{vendor + "_" + moduleName: {hooks.StageEntrypoint.String(), hooks.StageAuctionResponse.String()}},
			expectedHook:         module{},
			givenModule:          module{},
			givenConfig:          map[string]map[string]interface{}{"vendor": {"module": map[string]bool{"enabled": true}}},
			givenGetHookFn: func(repo hooks.HookRepository, module string) (interface{}, bool) {
				return repo.GetAuctionResponseHook(module)
			},
		},
		"Fails to find not registered hook": {
			isHookFound:          false,
			expectedModStageColl: map[string][]string{vendor + "_" + moduleName: {hooks.StageEntrypoint.String(), hooks.StageAuctionResponse.String()}},
			expectedHook:         nil,
			givenModule:          module{},
			givenConfig:          map[string]map[string]interface{}{vendor: {"module": map[string]bool{"enabled": true}}},
			givenGetHookFn: func(repo hooks.HookRepository, module string) (interface{}, bool) {
				return repo.GetAllProcessedBidResponsesHook(module) // ask for hook not implemented in module
			},
		},
		"Fails if module does not implement any hook interface": {
			expectedHook: struct{}{},
			expectedErr:  fmt.Errorf(`hook "%s.%s" does not implement any supported hook interface`, vendor, moduleName),
		},
		"Fails if module builder function returns error": {
			givenModule:         module{},
			givenConfig:         map[string]map[string]interface{}{vendor: {moduleName: map[string]string{"media_type": "video"}}},
			givenHookBuilderErr: errors.New("failed to build module"),
			expectedErr:         fmt.Errorf(`failed to init "%s.%s" module: %s`, vendor, moduleName, "failed to build module"),
		},
		"Fails if config marshaling returns error": {
			givenModule: module{},
			givenConfig: map[string]map[string]interface{}{vendor: {moduleName: math.Inf(1)}},
			expectedErr: fmt.Errorf(`failed to marshal "%s.%s" module config: json: unsupported value: +Inf`, vendor, moduleName),
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			builder := &builder{
				builders: ModuleBuilders{
					vendor: {
						moduleName: func(cfg json.RawMessage, client *http.Client) (interface{}, error) {
							return test.givenModule, test.givenHookBuilderErr
						},
					},
				},
			}

			repo, coll, err := builder.Build(test.givenConfig, http.DefaultClient)
			assert.Equal(t, test.expectedErr, err)
			if test.expectedErr == nil {
				hook, found := test.givenGetHookFn(repo, fmt.Sprintf("%s.%s", vendor, moduleName))
				assert.Equal(t, test.isHookFound, found)
				assert.IsType(t, test.expectedHook, hook)
				assert.Equal(t, test.expectedModStageColl, coll)
			}
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
