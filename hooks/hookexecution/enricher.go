package hookexecution

import (
	"encoding/json"

	"github.com/buger/jsonparser"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	jsonpatch "gopkg.in/evanphx/json-patch.v5"
)

const (
	// traceLevelBasic excludes debug_messages and analytic_tags from output
	traceLevelBasic trace = "basic"
	// traceLevelVerbose sets maximum level of output information
	traceLevelVerbose trace = "verbose"
)

// Trace controls the level of detail in the output information returned from executing hooks.
type trace string

func (t trace) isBasicOrHigher() bool {
	return t == traceLevelBasic || t.isVerbose()
}

func (t trace) isVerbose() bool {
	return t == traceLevelVerbose
}

type extPrebid struct {
	Prebid extModules `json:"prebid"`
}

type extModules struct {
	Modules json.RawMessage `json:"modules"`
}

// EnrichExtBidResponse adds debug and trace information returned from executing hooks to the ext argument.
// In response the outcome is visible under the key response.ext.prebid.modules.
//
// Debug information is added only if the debug mode is enabled by request and allowed by account (if provided).
// The details of the trace output depends on the value in the bidRequest.ext.prebid.trace field.
// Warnings returned if bidRequest contains unexpected types for debug fields controlling debug output.
func EnrichExtBidResponse(
	ext json.RawMessage,
	stageOutcomes []StageOutcome,
	bidRequest *openrtb2.BidRequest,
	account *config.Account,
) (json.RawMessage, []error, error) {
	modules, warnings, err := GetModulesJSON(stageOutcomes, bidRequest, account)
	if err != nil || modules == nil {
		return ext, warnings, err
	}

	response, err := jsonutil.Marshal(extPrebid{Prebid: extModules{Modules: modules}})
	if err != nil {
		return ext, warnings, err
	}

	if ext != nil {
		response, err = jsonpatch.MergePatch(ext, response)
	}

	return response, warnings, err
}

// GetModulesJSON returns debug and trace information produced from executing hooks.
// Debug information is returned only if the debug mode is enabled by request and allowed by account (if provided).
// The details of the trace output depends on the value in the bidRequest.ext.prebid.trace field.
// Warnings returned if bidRequest contains unexpected types for debug fields controlling debug output.
func GetModulesJSON(
	stageOutcomes []StageOutcome,
	bidRequest *openrtb2.BidRequest,
	account *config.Account,
) (json.RawMessage, []error, error) {
	if len(stageOutcomes) == 0 {
		return nil, nil, nil
	}

	trace, isDebugEnabled, warnings := getDebugContext(bidRequest, account)
	modulesOutcome := getModulesOutcome(stageOutcomes, trace, isDebugEnabled)
	if modulesOutcome == nil {
		return nil, warnings, nil
	}

	data, err := jsonutil.Marshal(modulesOutcome)

	return data, warnings, err
}

func getDebugContext(bidRequest *openrtb2.BidRequest, account *config.Account) (trace, bool, []error) {
	var traceLevel string
	var isDebugEnabled bool
	var warnings []error
	var err error

	if bidRequest != nil {
		traceLevel, err = jsonparser.GetString(bidRequest.Ext, "prebid", "trace")
		if err != nil && err != jsonparser.KeyPathNotFoundError {
			warnings = append(warnings, err)
		}

		isDebug, err := jsonparser.GetBoolean(bidRequest.Ext, "prebid", "debug")
		if err != nil && err != jsonparser.KeyPathNotFoundError {
			warnings = append(warnings, err)
		}

		isDebugEnabled = bidRequest.Test == 1 || isDebug
		if account != nil {
			isDebugEnabled = isDebugEnabled && account.DebugAllow
		}
	}

	return trace(traceLevel), isDebugEnabled, warnings
}

func getModulesOutcome(stageOutcomes []StageOutcome, trace trace, isDebugEnabled bool) *ModulesOutcome {
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

	if modulesOutcome.Errors == nil && modulesOutcome.Warnings == nil && len(stages) == 0 {
		return nil
	}

	if len(stages) > 0 {
		modulesOutcome.Trace = &TraceOutcome{}
		modulesOutcome.Trace.Stages = make([]Stage, 0, len(stages))

		for _, stage := range stageNames {
			modulesOutcome.Trace.ExecutionTimeMillis += stages[stage].ExecutionTimeMillis
			modulesOutcome.Trace.Stages = append(modulesOutcome.Trace.Stages, stages[stage])
		}
	}

	return &modulesOutcome
}

func prepareModulesOutcome(modulesOutcome *ModulesOutcome, groups []GroupOutcome, trace trace, isDebugEnabled bool) {
	for _, group := range groups {
		for i, hookOutcome := range group.InvocationResults {
			if !trace.isVerbose() {
				group.InvocationResults[i].DebugMessages = nil
			}

			if isDebugEnabled {
				modulesOutcome.Errors = fillMessages(modulesOutcome.Errors, hookOutcome.Errors, hookOutcome.HookID)
				modulesOutcome.Warnings = fillMessages(modulesOutcome.Warnings, hookOutcome.Warnings, hookOutcome.HookID)
			}
		}
	}
}

func fillMessages(messages Messages, values []string, hookID HookID) Messages {
	if len(values) == 0 {
		return messages
	}

	if messages == nil {
		return Messages{hookID.ModuleCode: {hookID.HookImplCode: values}}
	}

	if _, ok := messages[hookID.ModuleCode]; !ok {
		messages[hookID.ModuleCode] = map[string][]string{hookID.HookImplCode: values}
		return messages
	}

	if prevValues, ok := messages[hookID.ModuleCode][hookID.HookImplCode]; ok {
		values = append(prevValues, values...)
	}

	messages[hookID.ModuleCode][hookID.HookImplCode] = values

	return messages
}
