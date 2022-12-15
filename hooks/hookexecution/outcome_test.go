package hookexecution

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func assertEqualStageOutcomes(t *testing.T, expected StageOutcome, actual StageOutcome) {
	t.Helper()

	assert.Equal(t, len(actual.Groups), len(expected.Groups), "Stage outcomes contain different number of groups")

	// calculate expected timings from actual outcome
	for i, group := range actual.Groups {
		expected.ExecutionTimeMillis += group.ExecutionTimeMillis
		for _, hook := range group.InvocationResults {
			if hook.ExecutionTimeMillis > expected.Groups[i].ExecutionTimeMillis {
				expected.Groups[i].ExecutionTimeMillis = hook.ExecutionTimeMillis
			}
		}
	}

	assert.Equal(t, expected.ExecutionTimeMillis, actual.ExecutionTimeMillis, "Incorrect stage execution time")
	assert.Equal(t, expected.Stage, actual.Stage, "Incorrect stage name")
	assert.Equal(t, expected.Entity, actual.Entity, "Incorrect stage entity name")

	for i, expGroup := range expected.Groups {
		gotGroup := actual.Groups[i]
		assert.Equal(t, len(expGroup.InvocationResults), len(gotGroup.InvocationResults), "Group outcomes #%d contain different number of invocation results", i)
		assert.Equal(t, expGroup.ExecutionTimeMillis, gotGroup.ExecutionTimeMillis, "Incorrect group #%d execution time", i)

		for _, expHook := range expGroup.InvocationResults {
			gotHook := findCorrespondingHookResult(expHook.HookID, gotGroup)
			assert.NotNil(t, gotHook, "Expected to get hook, got nil: group #%d, hookID %v", i, expHook.HookID)

			gotHook.ExecutionTimeMillis = 0 // reset hook execution time, we cannot predict it
			assert.Equal(t, expHook, *gotHook, "Incorrect hook outcome: group #%d, hookID %v", i, expHook.HookID)
		}
	}
}

func findCorrespondingHookResult(hookID HookID, group GroupOutcome) *HookOutcome {
	for _, hook := range group.InvocationResults {
		if hook.HookID.ModuleCode == hookID.ModuleCode &&
			hook.HookID.HookImplCode == hookID.HookImplCode {
			return &hook
		}
	}
	return nil
}
