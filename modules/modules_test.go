package modules

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/hooks/hep"
	"github.com/prebid/prebid-server/hooks/invocation"
	"github.com/prebid/prebid-server/hooks/stages"
	"github.com/stretchr/testify/assert"
)

func TestModuleBuilderBuild(t *testing.T) {
	moduleName := "foobar"
	hookCode := "baz"

	testCases := map[string]struct {
		isHookFound         bool
		expectedHook        interface{}
		givenHook           interface{}
		expectedErr         error
		givenHookBuilderErr error
		givenGetHookFn      func(repo hep.HookRepository, module, hook string) (interface{}, bool)
	}{
		"Can register entrypoint hook": {
			givenHook:    fakeEntrypointHook{},
			expectedHook: fakeEntrypointHook{},
			isHookFound:  true,
			givenGetHookFn: func(repo hep.HookRepository, module, hook string) (interface{}, bool) {
				return repo.GetEntrypointHook(module, hook)
			},
		},
		"Can register auction response hook": {
			givenHook:    fakeAuctionResponseHook{},
			expectedHook: fakeAuctionResponseHook{},
			isHookFound:  true,
			givenGetHookFn: func(repo hep.HookRepository, module, hook string) (interface{}, bool) {
				return repo.GetAuctionResponseHook(module, hook)
			},
		},
		"Cannot find not registered hook": {
			givenHook:    fakeEntrypointHook{},
			expectedHook: nil,
			isHookFound:  false,
			givenGetHookFn: func(repo hep.HookRepository, module, hook string) (interface{}, bool) {
				return repo.GetAuctionResponseHook(module, hook) // ask for not registered hook
			},
		},
		"Cannot register invalid hook type": {
			expectedHook: struct{}{},
			expectedErr:  fmt.Errorf(`trying to register invalid hook type: %s %s`, moduleName, hookCode),
		},
		"Cannot build when hook builder fails": {
			givenHook:           fakeEntrypointHook{},
			givenHookBuilderErr: errors.New("failed to build hook"),
			expectedErr:         fmt.Errorf(`failed to init "%s" module: %s`, moduleName, "failed to build hook"),
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			builder := NewModuleBuilder().SetModuleBuilderFn(func() map[string]HookBuilderFn {
				return map[string]HookBuilderFn{
					moduleName: func(cfg json.RawMessage, client *http.Client) (map[string]interface{}, error) {
						return map[string]interface{}{
							hookCode: test.givenHook,
						}, test.givenHookBuilderErr
					},
				}
			})

			repo, err := builder.Build(map[string]interface{}{}, http.DefaultClient)
			assert.Equal(t, test.expectedErr, err)

			if err == nil {
				hook, found := test.givenGetHookFn(repo, moduleName, hookCode)
				assert.Equal(t, test.isHookFound, found)
				assert.IsType(t, test.expectedHook, hook)
			}
		})
	}
}

type fakeEntrypointHook struct{}

func (h fakeEntrypointHook) Call(ctx context.Context, context invocation.Context, payload stages.EntrypointPayload) (invocation.HookResult[stages.EntrypointPayload], error) {
	return invocation.HookResult[stages.EntrypointPayload]{}, nil
}

type fakeAuctionResponseHook struct{}

func (f fakeAuctionResponseHook) Call(ctx context.Context, i invocation.Context, response *openrtb2.BidResponse) (invocation.HookResult[*openrtb2.BidResponse], error) {
	return invocation.HookResult[*openrtb2.BidResponse]{}, nil
}
