package rulesengine

import (
	"encoding/json"
	"fmt"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"slices"
)

type SchemaFunction interface {
	Call(wrapper *openrtb_ext.RequestWrapper) (string, error)
}

func NewSchemaFunctionFactory(name string, params json.RawMessage) (SchemaFunction, error) {
	switch name {
	case "deviceCountry":
		return NewDeviceCountry(params)
	case "dataCenters":
		return NewDataCenter(params)
	case "channel":
		return NewChannel()
	case "eidAvailable":
		return NewAidAvailable(params)
	default:
		return nil, fmt.Errorf("Schema function %s was not created", name)
	}
}

// ------------deviceCountry------------------
type deviceCountry struct {
	CountryCodes []string
}

func NewDeviceCountry(params json.RawMessage) (SchemaFunction, error) {
	var devCountry []string
	if err := jsonutil.Unmarshal(params, &devCountry); err != nil {
		return nil, err
	}
	return &deviceCountry{CountryCodes: devCountry}, nil
}

func (dc *deviceCountry) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	if wrapper.Device == nil && wrapper.Device.Geo != nil && len(wrapper.Device.Geo.Country) == 0 {
		return "", fmt.Errorf("reqiuest.Device.Geo.Country is not present in request")
	}

	if len(dc.CountryCodes) == 0 {
		return wrapper.Device.Geo.Country, nil
	}

	if contains := slices.Contains(dc.CountryCodes, wrapper.Device.Geo.Country); contains {
		return "true", nil
	}
	return "false", nil
}

// ------------datacenters------------------

type dataCenter struct {
	DataCenters []string
}

func NewDataCenter(params json.RawMessage) (SchemaFunction, error) {
	var dc []string
	if err := jsonutil.Unmarshal(params, &dc); err != nil {
		return nil, err
	}
	return &dataCenter{DataCenters: dc}, nil
}

func (dc *dataCenter) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {

	// where is datacenter in bid request?
	// logic should be the same, but read a data center value from a proper location, not wrapper.Device.Geo.Region
	if wrapper.Device == nil && wrapper.Device.Geo != nil && len(wrapper.Device.Geo.Region) == 0 {
		return "", fmt.Errorf("reqiuest.Device.Geo.Country is not present in request")
	}

	if len(dc.DataCenters) == 0 {
		return wrapper.Device.Geo.Region, nil
	}

	if contains := slices.Contains(dc.DataCenters, wrapper.Device.Geo.Region); contains {
		return "true", nil
	}
	return "false", nil
}

// ------------channel------------------
type channel struct {
	// no params
}

func NewChannel() (SchemaFunction, error) {
	return &channel{}, nil
}

func (c *channel) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	reqExt, err := wrapper.GetRequestExt()
	if err != nil {
		return "", err
	}
	reqExtPrebid := reqExt.GetPrebid()
	if reqExtPrebid == nil || reqExtPrebid.Channel == nil {
		return "", fmt.Errorf("reqiuest.ext.prebid or req.ext.prebid.channel is not present in request")
	}
	chName := reqExtPrebid.Channel.Name
	if chName == "pbjs" {
		return "web", nil
	}
	return chName, nil
}

// ------------eidAvailable------------------

type eidAvailable struct {
	eids []string
}

func NewAidAvailable(params json.RawMessage) (SchemaFunction, error) {
	var eidsParam []string
	if err := jsonutil.Unmarshal(params, &eidsParam); err != nil {
		return nil, err
	}
	return &eidAvailable{eids: eidsParam}, nil
}

func (ae *eidAvailable) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	// From Requirements doc:
	// Arguments may be null in which case it returns true if user.eids array exists and is non-empty.
	// False if user.eids doesn't exist or is empty.
	if wrapper.User == nil || len(wrapper.User.EIDs) == 0 {
		return "false", nil
	}

	if len(ae.eids) == 0 {
		return "true", nil
	}

	// unit test this
	var eidExists string
	for _, eidParam := range ae.eids {
		eidExists = "false"
		for _, eid := range wrapper.User.EIDs {
			if eidParam == eid.Source {
				eidExists = "true"
				break
			}
		}
	}
	return eidExists, nil
}

// ------------userFpdAvailable------------------
// ------------fpdAvail------------------
// ------------gppSid------------------
// ------------tcfInScope------------------
// ------------percent------------------
// ------------prebidKey------------------
// ------------domain------------------
// ------------bundle------------------
// ------------percent------------------
// ------------mediaTypes------------------
// ------------adUnitCode------------------
// ------------deviceType------------------
// ------------bidPrice------------------
