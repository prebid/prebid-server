package main

import (
	"fmt"
	"net/http"
)

// HookCatalog created during module loading, consists of hook implementations
type HookCatalog struct {
	entrypointHooks map[string]map[string]EntrypointHook
	// other hooks
}

func (c HookCatalog) getEntrypointHook(moduleCode, hookCode string) EntrypointHook {
	if mh, ok := c.entrypointHooks[moduleCode]; ok {
		if v, ok := mh[hookCode]; ok {
			return v
		}
	}

	return nil
}

type EntrypointHook interface {
	HandleEntrypointHook(req *http.Request, body []byte)
}

type EntrypointHookImpl struct {
	callback string // for simplification
}

func (i EntrypointHookImpl) HandleEntrypointHook(_ *http.Request, _ []byte) {
	fmt.Println(i.callback)
}

type HookStageExecutor struct {
	hostExecutionPlan           ExecutionPlan
	defaultAccountExecutionPlan ExecutionPlan
	hookCatalog                 HookCatalog
}

// ExecutionPlan execution plan hierarchy
type ExecutionPlan struct {
	endpoints map[string]EndpointExecutionPlan
}

type EndpointExecutionPlan struct {
	stages map[string]StageExecutionPlan
}

type StageExecutionPlan struct {
	groups []ExecutionGroup
}

type ExecutionGroup struct {
	timeout      int
	hookSequence []HookId
}

type HookId struct {
	moduleCode   string
	hookImplCode string
}

// CreateExecutor called during application configuration
func CreateExecutor(hostExecutionPlan string, defaultAccountExecutionPlan string, hookCatalog HookCatalog) HookStageExecutor {
	return HookStageExecutor{
		hostExecutionPlan:           parseAndValidateExecutionPlan(hostExecutionPlan),
		defaultAccountExecutionPlan: parseAndValidateExecutionPlan(defaultAccountExecutionPlan),
		hookCatalog:                 hookCatalog,
	}
}

type Endpoint string

const (
	openrtb2Auction Endpoint = "/openrtb2/auction"
	//other endpoints
)

func (e Endpoint) String() string {
	return string(e)
}

type Stage string

const (
	entrypoint Stage = "entrypoint"
	//other stages
)

func (s Stage) String() string {
	return string(s)
}

type Context struct {
	endpoint Endpoint
}

type HookStageExecutionResult struct {
	reject bool
	result interface{}
}

func (exec HookStageExecutor) ExecuteEntrypointStage(ctx Context, req *http.Request, body []byte) (HookStageExecutionResult, error) {
	groups := exec.resolveEntrypointGroups(ctx.endpoint)

	for _, g := range groups {
		for _, hook := range g {
			hook.HandleEntrypointHook(req, body)
		}
	}

	//TODO: prepare result
	return HookStageExecutionResult{}, nil
}

func parseAndValidateExecutionPlan(_ string) ExecutionPlan {
	//TODO: implement real parsing and validation from string config
	// for illustration purpose use mock
	return ExecutionPlan{
		map[string]EndpointExecutionPlan{
			openrtb2Auction.String(): {
				stages: map[string]StageExecutionPlan{
					entrypoint.String(): {
						groups: []ExecutionGroup{
							{
								timeout: 5,
								hookSequence: []HookId{
									{
										moduleCode:   "module1",
										hookImplCode: "hook-code1",
									},
									{
										moduleCode:   "module1",
										hookImplCode: "hook-code4",
									},
								},
							},
							{
								timeout: 5,
								hookSequence: []HookId{
									{
										moduleCode:   "module1",
										hookImplCode: "hook-code3",
									},
									{
										moduleCode:   "module1",
										hookImplCode: "hook-code2",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (exec HookStageExecutor) resolveEntrypointGroups(e Endpoint) [][]EntrypointHook {
	var res [][]EntrypointHook
	var ires []EntrypointHook

	if endpoint, ok := exec.hostExecutionPlan.endpoints[e.String()]; ok {
		if stage, ok := endpoint.stages[entrypoint.String()]; ok {
			groups := stage.groups
			for _, g := range groups {
				for _, hId := range g.hookSequence {
					ires = append(ires, exec.hookCatalog.getEntrypointHook(hId.moduleCode, hId.hookImplCode))
				}
				res = append(res, ires)
				ires = nil
			}
		}
	}

	return res
}

func main() {
	// simulating module loading
	c := HookCatalog{
		entrypointHooks: map[string]map[string]EntrypointHook{
			"module1": {
				"hook-code1": EntrypointHookImpl{"Executing callback for hook 1"},
				"hook-code2": EntrypointHookImpl{"Executing callback for hook 2"},
				"hook-code3": EntrypointHookImpl{"Executing callback for hook 3"},
				"hook-code4": EntrypointHookImpl{"Executing callback for hook 4"},
			},
		},
	}

	// passing empty values as we are using mocks later
	executor := CreateExecutor("", "", c)

	// simulating openrtb2Auction request
	ctx := Context{endpoint: openrtb2Auction}
	_, err := executor.ExecuteEntrypointStage(ctx, nil, []byte{})
	if err != nil {
		fmt.Println(err)
	}
}
