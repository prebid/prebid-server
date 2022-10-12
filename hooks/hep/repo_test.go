package hep

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewHookRepository(t *testing.T) {
	moduleName := "foobar"
	hookCode := "baz"

	testCases := map[string]struct {
		isFound      bool
		providedHook interface{}
		expectedHook interface{}
		expectedErr  error
		getHookFn    func(HookRepository) (interface{}, bool)
	}{
		"AddedHookReturns": {
			isFound:      true,
			providedHook: fakeEntrypointHook{},
			expectedHook: fakeEntrypointHook{},
			expectedErr:  nil,
			getHookFn: func(repo HookRepository) (interface{}, bool) {
				return repo.GetEntrypointHook(moduleName, hookCode)
			},
		},
		"NotFoundHook": {
			isFound:      false,
			providedHook: fakeEntrypointHook{},
			expectedHook: nil,
			expectedErr:  nil,
			getHookFn: func(repo HookRepository) (interface{}, bool) {
				return repo.GetRawAuctionHook(moduleName, hookCode) // ask for not registered hook
			},
		},
		"FailsToAddInvalidHookType": {
			providedHook: struct{}{},
			expectedErr:  fmt.Errorf(`trying to register invalid hook type: %s %s`, moduleName, hookCode),
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			repo, err := NewHookRepository(map[string]map[string]interface{}{
				moduleName: {hookCode: test.providedHook},
			})
			assert.Equal(t, test.expectedErr, err)
			if err == nil {
				hook, found := test.getHookFn(repo)
				assert.Equal(t, test.isFound, found)
				assert.Equal(t, test.expectedHook, hook)
			}
		})
	}
}
