package hookexecution

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/stretchr/testify/assert"
)

// AssertEqualModulesData is the test helper function which asserts
// that expected modules data fully corresponds the actual modules data.
// Dynamic data for execution time is calculated from actual modules data.
func AssertEqualModulesData(t *testing.T, expectedData, actualData json.RawMessage) {
	t.Helper()

	var expectedModulesOutcome ModulesOutcome
	var actualModulesOutcome ModulesOutcome

	assert.NoError(t, jsonutil.UnmarshalValid(expectedData, &expectedModulesOutcome), "Failed to unmarshal expected modules data.")
	assert.NoError(t, jsonutil.UnmarshalValid(actualData, &actualModulesOutcome), "Failed to unmarshal actual modules data.")
	assert.Equal(t, expectedModulesOutcome.Errors, actualModulesOutcome.Errors, "Invalid error messages.")
	assert.Equal(t, expectedModulesOutcome.Warnings, actualModulesOutcome.Warnings, "Invalid warning messages.")

	assertEqualTraces(t, expectedModulesOutcome.Trace, actualModulesOutcome.Trace)
}

func assertEqualTraces(t *testing.T, expectedTrace *TraceOutcome, actualTrace *TraceOutcome) {
	if expectedTrace == nil {
		assert.Nil(t, actualTrace, "Nil trace not expected.")
	}

	// calculate expected timings from actual modules outcome
	for i, actualStage := range actualTrace.Stages {
		expectedStage := expectedTrace.Stages[i]
		expectedTrace.ExecutionTimeMillis += actualStage.ExecutionTimeMillis

		for _, actualOutcome := range actualStage.Outcomes {
			if expectedStage.ExecutionTimeMillis < actualOutcome.ExecutionTimeMillis {
				expectedStage.ExecutionTimeMillis = actualOutcome.ExecutionTimeMillis
			}

			expectedOutcome := findCorrespondingStageOutcome(expectedStage, actualOutcome)
			assert.NotNil(t, expectedOutcome, "Not found corresponding stage outcome, actual:`", actualOutcome)
			assertEqualStageOutcomes(t, *expectedOutcome, actualOutcome)
		}

		assert.Equal(t, expectedStage.Stage, actualStage.Stage, "Invalid stage name.")
		assert.Equal(t, expectedStage.ExecutionTimeMillis, actualStage.ExecutionTimeMillis, "Invalid stage execution time.")
	}

	assert.Equal(t, expectedTrace.ExecutionTimeMillis, actualTrace.ExecutionTimeMillis, "Invalid trace execution time.")
}

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

func findCorrespondingStageOutcome(stage Stage, outcome StageOutcome) *StageOutcome {
	for _, out := range stage.Outcomes {
		if out.Entity == outcome.Entity {
			return &out
		}
	}
	return nil
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
