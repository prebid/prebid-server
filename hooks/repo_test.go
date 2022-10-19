package hooks

import (
	"context"
	"fmt"
	"testing"

	"github.com/prebid/prebid-server/hooks/invocation"
	"github.com/prebid/prebid-server/hooks/stages"
	"github.com/stretchr/testify/assert"
)

func TestNewHookRepository(t *testing.T) {
	id := "foobar"
	testCases := map[string]struct {
		isFound      bool
		providedHook interface{}
		expectedHook interface{}
		expectedErr  error
		getHookFn    func(HookRepository) (interface{}, bool)
	}{
		"Added hook returns": {
			isFound:      true,
			providedHook: hook{},
			expectedHook: hook{},
			expectedErr:  nil,
			getHookFn: func(repo HookRepository) (interface{}, bool) {
				return repo.GetEntrypointHook(id)
			},
		},
		"Not found hook": {
			isFound:      false,
			providedHook: hook{},
			expectedHook: nil,
			expectedErr:  nil,
			getHookFn: func(repo HookRepository) (interface{}, bool) {
				return repo.GetRawAuctionHook(id) // ask for not implemented hook
			},
		},
		"Fails to add type that does not implement any hook interface": {
			providedHook: struct{}{},
			expectedErr:  fmt.Errorf(`hook "%s" does not implement any supported hook interface`, id),
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			repo, err := NewHookRepository(map[string]interface{}{id: test.providedHook})
			assert.Equal(t, test.expectedErr, err)
			if test.expectedErr == nil {
				hook, found := test.getHookFn(repo)
				assert.Equal(t, test.isFound, found)
				assert.Equal(t, test.expectedHook, hook)
			}
		})
	}
}

func TestAddHook_FailsToAddHookOfSameTypeAndIdTwice(t *testing.T) {
	id := "foobar"
	h := hook{}
	expectedErr := fmt.Errorf(`hook of type "%T" with id "%s" already registered`, new(stages.EntrypointHook), id)

	hooks, err := addHook[stages.EntrypointHook](nil, h, id)
	if assert.NoError(t, err, "failed to add hook") {
		_, err = addHook[stages.EntrypointHook](hooks, h, id)
		assert.Equal(t, expectedErr, err)
	}
}

type hook struct{}

func (h hook) HandleEntrypointHook(ctx context.Context, context invocation.Context, payload stages.EntrypointPayload) (invocation.HookResult[stages.EntrypointPayload], error) {
	return invocation.HookResult[stages.EntrypointPayload]{}, nil
}
