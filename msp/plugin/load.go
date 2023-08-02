package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"plugin"

	"github.com/golang/glog"
)

// Every plugin must export a symbol named `Builder`
func LoadBuilder[T any](name string, cfgData interface{}) (T, json.RawMessage, bool, error) {
	var builder T

	cfg, cfgJson := ParseConfig(name, cfgData)
	if !cfg.Enabled {
		glog.Infof("Skip loading plugin %s as it is disabled.", name)
		return builder, cfgJson, true, nil
	}

	if cfg.SoPath == "" {
		return builder, cfgJson, false, errors.New(fmt.Sprintf("The path to load plugin %s is empty.\n", name))
	}

	p, err := plugin.Open(cfg.SoPath)
	if err != nil {
		return builder, cfgJson, false, errors.New(fmt.Sprintf("Failed to open shared object of plugin %s, err: %+v.\n", name, err))
	}

	s, err := p.Lookup("Builder")
	if err != nil {
		return builder, cfgJson, false, errors.New(fmt.Sprintf("Failed to find Builder from plugin %s, err: %+v.\n", name, err))
	}

	builder, ok := s.(T)
	if !ok {
		return builder, cfgJson, false, errors.New(fmt.Sprintf("Failed to convert Builder from plugin %s, err: %+v.\n", name, err))
	}

	return builder, cfgJson, false, nil
}

func LoadBuilderFromPath[T any](name string, soPath string) (T, error) {
	var builder T

	p, err := plugin.Open(soPath)
	if err != nil {
		return builder, errors.New(fmt.Sprintf("Failed to open shared object of plugin %s, err: %+v.\n", name, err))
	}

	s, err := p.Lookup("Builder")
	if err != nil {
		return builder, errors.New(fmt.Sprintf("Failed to find Builder from plugin %s, err: %+v.\n", name, err))
	}

	builder, ok := s.(T)
	if !ok {
		return builder, errors.New(fmt.Sprintf("Failed to convert Builder from plugin %s, err: %+v.\n", name, err))
	}

	return builder, nil
}
