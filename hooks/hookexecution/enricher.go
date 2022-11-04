package hookexecution

import (
	"encoding/json"

	"github.com/prebid/openrtb/v17/openrtb2"
	jsonpatch "gopkg.in/evanphx/json-patch.v4"
)

type ModulesResponse struct {
	Ext Ext `json:"ext"`
}

type Ext struct {
	Prebid Prebid `json:"prebid"`
}

type Prebid struct {
	Modules ModulesOutcome `json:"modules"`
}

func EnrichResponse(response *openrtb2.BidResponse, stageOutcomes []StageOutcome) error {
	var err error
	var modulesOutcome ModulesOutcome
	var trace TraceOutcome
	var stages map[string]Stage

	// group all stage outcomes by stages
	for _, stageOutcome := range stageOutcomes {
		stage, ok := stages[stageOutcome.Stage]
		if !ok {
			stage = Stage{
				ExecutionTime: ExecutionTime{stageOutcome.ExecutionTimeMillis},
				Stage:         stageOutcome.Stage,
				Outcomes:      []StageOutcome{},
			}
		}

		if stageOutcome.ExecutionTimeMillis > stage.ExecutionTimeMillis {
			stage.ExecutionTimeMillis = stageOutcome.ExecutionTimeMillis
		}

		stage.Outcomes = append(stage.Outcomes, stageOutcome)
		stages[stageOutcome.Stage] = stage

		// fill errors and warnings
		for _, group := range stageOutcome.Groups {
			for _, hookOutcome := range group.InvocationResults {
				if len(hookOutcome.Errors) > 0 {
					errors := modulesOutcome.Errors[hookOutcome.HookID.ModuleCode][hookOutcome.HookID.HookCode]
					errors = append(errors, hookOutcome.Errors...)
					modulesOutcome.Errors[hookOutcome.HookID.ModuleCode][hookOutcome.HookID.HookCode] = errors
				}

				if len(hookOutcome.Warnings) > 0 {
					warnings := modulesOutcome.Warnings[hookOutcome.HookID.ModuleCode][hookOutcome.HookID.HookCode]
					warnings = append(warnings, hookOutcome.Warnings...)
					modulesOutcome.Warnings[hookOutcome.HookID.ModuleCode][hookOutcome.HookID.HookCode] = warnings
				}
			}
		}
	}

	for _, stage := range stages {
		trace.ExecutionTimeMillis += stage.ExecutionTimeMillis
		trace.Stages = append(trace.Stages, stage)
	}

	modulesOutcome.Trace = trace

	patch, err := json.Marshal(map[string]map[string]map[string]ModulesOutcome{
		"ext": {
			"prebid": {
				"modules": modulesOutcome,
			},
		},
	})
	if err == nil {
		return err
	}

	response.Ext, err = jsonpatch.MergePatch(response.Ext, patch)

	return nil

	/*patch := json.RawMessage(`{}`)

	for _, data := range *r {
		patch, err = jsonpatch.MergeMergePatches(patch, data)
		if err != nil {
			return err
		}
	}*/
}
