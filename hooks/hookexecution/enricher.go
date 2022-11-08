package hookexecution

import (
	"encoding/json"

	"github.com/buger/jsonparser"
	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/hooks/hookanalytics"
	jsonpatch "gopkg.in/evanphx/json-patch.v4"
)

const (
	traceLevelBasic   trace = "basic"
	traceLevelVerbose trace = "verbose"
)

type trace string

func (t trace) isBasicOrHigher() bool {
	return t == traceLevelBasic || t.isVerbose()
}

func (t trace) isVerbose() bool {
	return t == traceLevelVerbose
}

type modulesResponse struct {
	Prebid struct {
		Modules ModulesOutcome `json:"modules"`
	} `json:"prebid"`
}

func EnrichResponse(
	bidResponse *openrtb2.BidResponse,
	stageOutcomes []StageOutcome,
	bidRequest *openrtb2.BidRequest,
	account *config.Account,
) error {
	trace, isDebugEnabled := getDebugContext(bidRequest, account)

	modResponse := modulesResponse{}
	modResponse.Prebid.Modules = getModulesOutcome(stageOutcomes, trace, isDebugEnabled)
	patch, err := json.Marshal(modResponse)
	if err != nil {
		return err
	}

	if bidResponse.Ext == nil {
		bidResponse.Ext = patch
	} else {
		bidResponse.Ext, err = jsonpatch.MergePatch(bidResponse.Ext, patch)
	}

	return err
}

func getDebugContext(bidRequest *openrtb2.BidRequest, account *config.Account) (trace, bool) {
	var traceLevel string
	var isDebugEnabled bool

	if bidRequest != nil {
		traceLevel, _ = jsonparser.GetString(bidRequest.Ext, "prebid", "trace")
		if account != nil {
			isDebug, _ := jsonparser.GetBoolean(bidRequest.Ext, "prebid", "debug")
			isDebugEnabled = (bidRequest.Test == 1 || isDebug) && account.DebugAllow
		}
	}

	return trace(traceLevel), isDebugEnabled
}

func getModulesOutcome(stageOutcomes []StageOutcome, trace trace, isDebugEnabled bool) ModulesOutcome {
	var modulesOutcome ModulesOutcome
	stages := make(map[string]Stage)
	stageNames := make([]string, 0)

	for _, stageOutcome := range stageOutcomes {
		if len(stageOutcome.Groups) == 0 {
			continue
		}

		prepareModulesOutcome(&modulesOutcome, stageOutcome.Groups, trace, isDebugEnabled)
		if !trace.isBasicOrHigher() {
			continue
		}

		stage, ok := stages[stageOutcome.Stage]
		if !ok {
			stageNames = append(stageNames, stageOutcome.Stage)
			stage = Stage{
				Stage:    stageOutcome.Stage,
				Outcomes: []StageOutcome{},
			}
		}

		stage.Outcomes = append(stage.Outcomes, stageOutcome)
		if stageOutcome.ExecutionTimeMillis > stage.ExecutionTimeMillis {
			stage.ExecutionTimeMillis = stageOutcome.ExecutionTimeMillis
		}

		stages[stageOutcome.Stage] = stage
	}

	if len(stages) == 0 {
		return modulesOutcome
	}

	modulesOutcome.Trace = &TraceOutcome{}
	modulesOutcome.Trace.Stages = make([]Stage, 0, len(stages))
	// iterate through slice of names to keep order of stages
	for _, stage := range stageNames {
		modulesOutcome.Trace.ExecutionTimeMillis += stages[stage].ExecutionTimeMillis
		modulesOutcome.Trace.Stages = append(modulesOutcome.Trace.Stages, stages[stage])
	}

	return modulesOutcome
}

func prepareModulesOutcome(modulesOutcome *ModulesOutcome, groups []GroupOutcome, trace trace, isDebugEnabled bool) {
	for _, group := range groups {
		for _, hookOutcome := range group.InvocationResults {
			if !trace.isVerbose() {
				hookOutcome.DebugMessages = nil
				hookOutcome.AnalyticsTags = hookanalytics.Analytics{}
			}

			if !isDebugEnabled {
				continue
			}

			modulesOutcome.Errors = fillMessages(modulesOutcome.Errors, hookOutcome.Errors, hookOutcome.HookID)
			modulesOutcome.Warnings = fillMessages(modulesOutcome.Warnings, hookOutcome.Warnings, hookOutcome.HookID)
		}
	}
}

func fillMessages(messages Messages, values []string, hookID HookID) Messages {
	if len(values) == 0 {
		return messages
	}

	if messages == nil {
		return Messages{hookID.ModuleCode: {hookID.HookCode: values}}
	}

	if _, ok := messages[hookID.ModuleCode]; !ok {
		messages[hookID.ModuleCode] = map[string][]string{hookID.HookCode: values}
		return messages
	}

	prevValues, ok := messages[hookID.ModuleCode][hookID.HookCode]
	if ok {
		values = append(prevValues, values...)
	}

	messages[hookID.ModuleCode][hookID.HookCode] = values

	return messages
}
