package hookexecution

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/hooks"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/ortb"
	"github.com/prebid/prebid-server/v3/privacy"
	"github.com/prebid/prebid-server/v3/util/iputil"
)

type hookResponse[T any] struct {
	Err           error
	ExecutionTime time.Duration
	HookID        HookID
	Result        hookstage.HookResult[T]
}

type hookHandler[H any, P any] func(
	context.Context,
	hookstage.ModuleInvocationContext,
	H,
	P,
) (hookstage.HookResult[P], error)

func executeStage[H any, P any](
	executionCtx executionContext,
	plan hooks.Plan[H],
	payload P,
	hookHandler hookHandler[H, P],
	metricEngine metrics.MetricsEngine,
) (StageOutcome, P, stageModuleContext, *RejectError) {
	stageOutcome := StageOutcome{}
	stageOutcome.Groups = make([]GroupOutcome, 0, len(plan))
	stageModuleCtx := stageModuleContext{}
	stageModuleCtx.groupCtx = make([]groupModuleContext, 0, len(plan))

	for _, group := range plan {
		groupOutcome, newPayload, moduleContexts, rejectErr := executeGroup(executionCtx, group, payload, hookHandler, metricEngine)
		stageOutcome.ExecutionTimeMillis += groupOutcome.ExecutionTimeMillis
		stageOutcome.Groups = append(stageOutcome.Groups, groupOutcome)
		stageModuleCtx.groupCtx = append(stageModuleCtx.groupCtx, moduleContexts)
		if rejectErr != nil {
			return stageOutcome, payload, stageModuleCtx, rejectErr
		}

		payload = newPayload
	}

	return stageOutcome, payload, stageModuleCtx, nil
}

func executeGroup[H any, P any](
	executionCtx executionContext,
	group hooks.Group[H],
	payload P,
	hookHandler hookHandler[H, P],
	metricEngine metrics.MetricsEngine,
) (GroupOutcome, P, groupModuleContext, *RejectError) {
	var wg sync.WaitGroup
	rejected := make(chan struct{})
	resp := make(chan hookResponse[P], len(group.Hooks))

	for _, hook := range group.Hooks {
		mCtx := executionCtx.getModuleContext(hook.Module)
		newPayload := handleModuleActivities(hook.Code, executionCtx.activityControl, payload, executionCtx.account)
		wg.Add(1)
		go func(hw hooks.HookWrapper[H], moduleCtx hookstage.ModuleInvocationContext) {
			defer wg.Done()
			executeHook(moduleCtx, hw, newPayload, hookHandler, group.Timeout, resp, rejected)
		}(hook, mCtx)
	}

	go func() {
		wg.Wait()
		close(resp)
	}()

	hookResponses := collectHookResponses(resp, rejected)

	return handleHookResponses(executionCtx, hookResponses, payload, metricEngine)
}

func executeHook[H any, P any](
	moduleCtx hookstage.ModuleInvocationContext,
	hw hooks.HookWrapper[H],
	payload P,
	hookHandler hookHandler[H, P],
	timeout time.Duration,
	resp chan<- hookResponse[P],
	rejected <-chan struct{},
) {
	hookRespCh := make(chan hookResponse[P], 1)
	startTime := time.Now()
	hookId := HookID{ModuleCode: hw.Module, HookImplCode: hw.Code}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				glog.Errorf("OpenRTB auction recovered panic in module hook %s.%s: %v, Stack trace is: %v",
					hw.Module, hw.Code, r, string(debug.Stack()))
			}
		}()

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		result, err := hookHandler(ctx, moduleCtx, hw.Hook, payload)
		hookRespCh <- hookResponse[P]{
			Result: result,
			Err:    err,
		}
	}()

	select {
	case res := <-hookRespCh:
		res.HookID = hookId
		res.ExecutionTime = time.Since(startTime)
		resp <- res
	case <-time.After(timeout):
		resp <- hookResponse[P]{
			Err:           TimeoutError{},
			ExecutionTime: time.Since(startTime),
			HookID:        hookId,
			Result:        hookstage.HookResult[P]{},
		}
	case <-rejected:
		return
	}
}

func collectHookResponses[P any](resp <-chan hookResponse[P], rejected chan<- struct{}) []hookResponse[P] {
	hookResponses := make([]hookResponse[P], 0)
	for r := range resp {
		hookResponses = append(hookResponses, r)
		if r.Result.Reject {
			close(rejected)
			break
		}
	}

	return hookResponses
}

func handleHookResponses[P any](
	executionCtx executionContext,
	hookResponses []hookResponse[P],
	payload P,
	metricEngine metrics.MetricsEngine,
) (GroupOutcome, P, groupModuleContext, *RejectError) {
	groupOutcome := GroupOutcome{}
	groupOutcome.InvocationResults = make([]HookOutcome, 0, len(hookResponses))
	groupModuleCtx := make(groupModuleContext, len(hookResponses))

	for _, r := range hookResponses {
		groupModuleCtx[r.HookID.ModuleCode] = r.Result.ModuleContext
		if r.ExecutionTime > groupOutcome.ExecutionTimeMillis {
			groupOutcome.ExecutionTimeMillis = r.ExecutionTime
		}

		updatedPayload, hookOutcome, rejectErr := handleHookResponse(executionCtx, payload, r, metricEngine)
		groupOutcome.InvocationResults = append(groupOutcome.InvocationResults, hookOutcome)
		payload = updatedPayload

		if rejectErr != nil {
			return groupOutcome, payload, groupModuleCtx, rejectErr
		}
	}

	return groupOutcome, payload, groupModuleCtx, nil
}

// moduleReplacer changes unwanted symbols to be in compliance with metric naming requirements
var moduleReplacer = strings.NewReplacer(".", "_", "-", "_")

// handleHookResponse is a strategy function that selects and applies
// one of the available algorithms to handle hook response.
func handleHookResponse[P any](
	ctx executionContext,
	payload P,
	hr hookResponse[P],
	metricEngine metrics.MetricsEngine,
) (P, HookOutcome, *RejectError) {
	var rejectErr *RejectError
	labels := metrics.ModuleLabels{Module: moduleReplacer.Replace(hr.HookID.ModuleCode), Stage: ctx.stage, AccountID: ctx.accountID}
	metricEngine.RecordModuleCalled(labels, hr.ExecutionTime)

	hookOutcome := HookOutcome{
		Status:        StatusSuccess,
		HookID:        hr.HookID,
		Message:       hr.Result.Message,
		Errors:        hr.Result.Errors,
		Warnings:      hr.Result.Warnings,
		DebugMessages: hr.Result.DebugMessages,
		AnalyticsTags: hr.Result.AnalyticsTags,
		ExecutionTime: ExecutionTime{ExecutionTimeMillis: hr.ExecutionTime},
	}

	if hr.Err != nil || hr.Result.Reject {
		handleHookError(hr, &hookOutcome, metricEngine, labels)
		rejectErr = handleHookReject(ctx, hr, &hookOutcome, metricEngine, labels)
	} else {
		payload = handleHookMutations(payload, hr, &hookOutcome, metricEngine, labels)
	}

	return payload, hookOutcome, rejectErr
}

// handleHookError sets an appropriate status to HookOutcome depending on the type of hook execution error.
func handleHookError[P any](
	hr hookResponse[P],
	hookOutcome *HookOutcome,
	metricEngine metrics.MetricsEngine,
	labels metrics.ModuleLabels,
) {
	if hr.Err == nil {
		return
	}

	hookOutcome.Errors = append(hookOutcome.Errors, hr.Err.Error())
	switch hr.Err.(type) {
	case TimeoutError:
		metricEngine.RecordModuleTimeout(labels)
		hookOutcome.Status = StatusTimeout
	case FailureError:
		metricEngine.RecordModuleFailed(labels)
		hookOutcome.Status = StatusFailure
	default:
		metricEngine.RecordModuleExecutionError(labels)
		hookOutcome.Status = StatusExecutionFailure
	}
}

// handleHookReject rejects execution at the current stage.
// In case the stage does not support rejection, hook execution marked as failed.
func handleHookReject[P any](
	ctx executionContext,
	hr hookResponse[P],
	hookOutcome *HookOutcome,
	metricEngine metrics.MetricsEngine,
	labels metrics.ModuleLabels,
) *RejectError {
	if !hr.Result.Reject {
		return nil
	}

	stage := hooks.Stage(ctx.stage)
	if !stage.IsRejectable() {
		metricEngine.RecordModuleExecutionError(labels)
		hookOutcome.Status = StatusExecutionFailure
		hookOutcome.Errors = append(
			hookOutcome.Errors,
			fmt.Sprintf(
				"Module (name: %s, hook code: %s) tried to reject request on the %s stage that does not support rejection",
				hr.HookID.ModuleCode,
				hr.HookID.HookImplCode,
				ctx.stage,
			),
		)
		return nil
	}

	rejectErr := &RejectError{NBR: hr.Result.NbrCode, Hook: hr.HookID, Stage: ctx.stage}
	hookOutcome.Action = ActionReject
	hookOutcome.Errors = append(hookOutcome.Errors, rejectErr.Error())
	metricEngine.RecordModuleSuccessRejected(labels)

	return rejectErr
}

// handleHookMutations applies mutations returned by hook to provided payload.
func handleHookMutations[P any](
	payload P,
	hr hookResponse[P],
	hookOutcome *HookOutcome,
	metricEngine metrics.MetricsEngine,
	labels metrics.ModuleLabels,
) P {
	if len(hr.Result.ChangeSet.Mutations()) == 0 {
		metricEngine.RecordModuleSuccessNooped(labels)
		hookOutcome.Action = ActionNone
		return payload
	}

	hookOutcome.Action = ActionUpdate
	successfulMutations := 0
	for _, mut := range hr.Result.ChangeSet.Mutations() {
		p, err := mut.Apply(payload)
		if err != nil {
			hookOutcome.Warnings = append(
				hookOutcome.Warnings,
				fmt.Sprintf("failed to apply hook mutation: %s", err),
			)
			continue
		}

		payload = p
		hookOutcome.DebugMessages = append(
			hookOutcome.DebugMessages,
			fmt.Sprintf(
				"Hook mutation successfully applied, affected key: %s, mutation type: %s",
				strings.Join(mut.Key(), "."),
				mut.Type(),
			),
		)
		successfulMutations++
	}

	// if at least one mutation from a given module was successfully applied
	// we consider that the module was processed successfully
	if successfulMutations > 0 {
		metricEngine.RecordModuleSuccessUpdated(labels)
	} else {
		hookOutcome.Status = StatusExecutionFailure
		metricEngine.RecordModuleExecutionError(labels)
	}

	return payload
}

func handleModuleActivities[P any](hookCode string, activityControl privacy.ActivityControl, payload P, account *config.Account) P {
	payloadData, ok := any(&payload).(hookstage.RequestUpdater)
	if !ok {
		return payload
	}

	scopeGeneral := privacy.Component{Type: privacy.ComponentTypeGeneral, Name: hookCode}
	transmitUserFPDActivityAllowed := activityControl.Allow(privacy.ActivityTransmitUserFPD, scopeGeneral, privacy.ActivityRequest{})
	transmitPreciseGeoActivityAllowed := activityControl.Allow(privacy.ActivityTransmitPreciseGeo, scopeGeneral, privacy.ActivityRequest{})

	if transmitUserFPDActivityAllowed && transmitPreciseGeoActivityAllowed {
		return payload
	}

	// changes need to be applied to new payload and leave original payload unchanged
	bidderReq := payloadData.GetBidderRequestPayload()

	bidderReqCopy := &openrtb_ext.RequestWrapper{
		BidRequest: ortb.CloneBidRequestPartial(bidderReq.BidRequest),
	}

	if !transmitUserFPDActivityAllowed {
		privacy.ScrubUserFPD(bidderReqCopy)
	}
	if !transmitPreciseGeoActivityAllowed {
		var ipConf privacy.IPConf
		if account != nil {
			ipConf = privacy.IPConf{IPV6: account.Privacy.IPv6Config, IPV4: account.Privacy.IPv4Config}
		} else {
			ipConf = privacy.IPConf{
				IPV6: config.IPv6{AnonKeepBits: iputil.IPv6DefaultMaskingBitSize},
				IPV4: config.IPv4{AnonKeepBits: iputil.IPv4DefaultMaskingBitSize}}
		}

		privacy.ScrubGeoAndDeviceIP(bidderReqCopy, ipConf)
	}

	var newPayload = payload
	var np = any(&newPayload).(hookstage.RequestUpdater)
	np.SetBidderRequestPayload(bidderReqCopy)
	return newPayload

}
