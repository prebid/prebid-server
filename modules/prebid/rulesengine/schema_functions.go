package rulesengine

import (
	"errors"
	"slices"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

type Function interface {
	Call(wrapper *openrtb_ext.RequestWrapper) (string, error)
}

func NewFunction(name string, params []string) (Function, error) {
	switch name {
	case "deviceCountry":
		return NewDeviceCountry(params)
	default:
		return nil, nil
	}
}

type deviceCountry struct {
	CountryCodes []string
}

func (dc *deviceCountry) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	if dc.CountryCodes == nil {
		return wrapper.Device.Geo.Country, nil
	}
	if contains := slices.Contains(dc.CountryCodes, wrapper.Device.Geo.Country); contains {
		return "true", nil
	}
	return "false", nil
}

func NewDeviceCountry(params []string) (Function, error) {
	if len(params) != 1 {
		return nil, errors.New("")
	}

	// validate and parse params
	return &deviceCountry{CountryCodes: params}, nil
}
