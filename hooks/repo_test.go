package hooks

import (
	"context"
	"fmt"
	"testing"

	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	hook := hook{}
	repo := hookRepository{}
	expectedErr := fmt.Errorf(`hook of type "%T" with id "%s" already registered`, new(hookstage.Entrypoint), id)

	err := repo.add(id, hook)
	require.NoError(t, err, "failed to add hook")

	err = repo.add(id, hook)
	assert.Equal(t, expectedErr, err)
}

type hook struct{}

func (h hook) HandleEntrypointHook(ctx context.Context, context hookstage.ModuleInvocationContext, payload hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	return hookstage.HookResult[hookstage.EntrypointPayload]{}, nil
}
