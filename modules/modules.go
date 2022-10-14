package modules

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/prebid-server/hooks/hep"
)

func NewModuleBuilder() ModuleBuilder {
	return &moduleBuilder{builders}
}

type ModuleBuilder interface {
	SetModuleBuilderFn(fn ModuleBuilderFn) ModuleBuilder
	Build(cfg map[string]interface{}, client *http.Client) (hep.HookRepository, error)
}

type (
	// ModuleBuilderFn returns mapping between module name and hook builders provided by module
	ModuleBuilderFn func() map[string]HookBuilderFn
	// HookBuilderFn returns mapping between hook code and implementation of a specific hook interface
	HookBuilderFn func(cfg json.RawMessage, client *http.Client) (map[string]interface{}, error)
)

type moduleBuilder struct {
	getBuildersFn func() map[string]HookBuilderFn
}

func (m *moduleBuilder) SetModuleBuilderFn(fn ModuleBuilderFn) ModuleBuilder {
	m.getBuildersFn = fn
	return m
}

func (m *moduleBuilder) Build(cfg map[string]interface{}, client *http.Client) (hep.HookRepository, error) {
	hooks := make(map[string]map[string]interface{})
	for module, builder := range m.getBuildersFn() {
		conf, err := json.Marshal(cfg[module])
		if err != nil {
			return nil, fmt.Errorf(`failed to marshal "%s" module config: %s`, module, err)
		}

		moduleHooks, err := builder(conf, client)
		if err != nil {
			return nil, fmt.Errorf(`failed to init "%s" module: %s`, module, err)
		}

		for code, hook := range moduleHooks {
			if hooks[module] == nil {
				hooks[module] = make(map[string]interface{})
			}

			hooks[module][code] = hook
		}
	}

	return hep.NewHookRepository(hooks)
}
