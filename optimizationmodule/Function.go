package optimizationmodule

import (
	"encoding/json"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"slices"
)

type Function interface {
	Call(wrapper *openrtb_ext.RequestWrapper) (string, error)
}

// ---------schema functions-----------
type DeviceCountry struct {
	countryCodes []string
}

func NewDeviceCountry(params []string) Function {
	//parse params to the format specific to this function
	return &DeviceCountry{countryCodes: params}
}

func (dc *DeviceCountry) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	contains := slices.Contains(dc.countryCodes, wrapper.Device.Geo.Country)
	//convert result to string
	if contains {
		return "yes", nil
	}
	return "no", nil
}

type Datacenters struct {
	datacenters []string
}

func NewDatacenters(params []string) Function {
	return &Datacenters{
		datacenters: params,
	}
}

func (dc *Datacenters) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	//where to get datacenters from?
	for _, datacenter := range dc.datacenters {
		if wrapper.Device.Geo.Region == datacenter {
			return "true", nil
		}
	}
	return "false", nil
}

type Channel struct{}

func NewChannel() Function {
	return &Channel{}
}

func (c *Channel) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	reqExt, err := wrapper.GetRequestExt()
	return reqExt.GetPrebid().Channel.Name, err //channel.Name?
}

// ----------result functions---------
type ExcludeBidders struct {
	bidders []string
}

func NewExcludeBidders(params json.RawMessage) Function {
	//bidders, seatnonbid, ifSyncId, analytics value ...
	// convert params to the right format
	return &ExcludeBidders{bidders: []string{"appnexus"}}
}

func (eb *ExcludeBidders) Call(rw *openrtb_ext.RequestWrapper) (string, error) {
	// remove bidder from imp ext?
	// just modify App.Name for testing
	rw.App.Name = eb.bidders[0]
	return "", nil
}

// just for testing
type SetDeviceIp struct {
	ip string
}

func NewSetDevIp(params json.RawMessage) Function {
	// convert params to the right format
	return &SetDeviceIp{ip: "127.0.0.1"}
}

func (sdip *SetDeviceIp) Call(rw *openrtb_ext.RequestWrapper) (string, error) {
	rw.Device.IP = sdip.ip
	return "", nil
}
