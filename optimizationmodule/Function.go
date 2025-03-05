package optimizationmodule

import (
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"slices"
)

type Function interface {
	Call(wrapper *openrtb_ext.RequestWrapper) (string, error)
}

// will be used in Build rules trie
func NewFunction(name string, params []string) Function {
	switch name {
	case "deviceCountry":
		return NewDeviceCountry(params)
	case "setDeviceIP":
		return NewSetDevIp(params)
	default:
		return nil
	}
}

type DeviceCountry struct {
	CountryCodes []string
}

func (dc *DeviceCountry) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	contains := slices.Contains(dc.CountryCodes, wrapper.Device.Geo.Country)
	//convert result to string
	if contains {
		return "yes", nil
	}
	return "no", nil
}

func NewDeviceCountry(params []string) Function {
	//parse params to the format specific to this function
	return &DeviceCountry{CountryCodes: params}
}

type SetDeviceIp struct {
	IP string
}

func (sdip *SetDeviceIp) Call(rw *openrtb_ext.RequestWrapper) (string, error) {
	rw.Device.IP = sdip.IP
	return "", nil
}

func NewSetDevIp(params []string) Function {
	return &SetDeviceIp{IP: params[0]}
}
