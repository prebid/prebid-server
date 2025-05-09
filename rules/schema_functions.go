package rules

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"slices"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
)

const (
	DeviceCountry    = "deviceCountry"
	DeviceCountryIn  = "deviceCountryIn"
	DataCenter       = "dataCenter"
	DataCenterIn     = "dataCenterIn"
	Channel          = "channel"
	EidAvailable     = "eidAvailable"
	UserFpdAvailable = "userFpdAvailable"
	FpdAvail         = "fpdAvail"
	GppSid           = "gppSid"
	TcfInScope       = "tcfInScope"
	Percent          = "percent"
	PrebidKey        = "prebidKey"
	Domain           = "domain"
	Bundle           = "bundle"
	DeviceType       = "deviceType"
)

// SchemaFunction...
type SchemaFunction[T any] interface {
	Call(payload *T) (string, string, error)
}

// NewRequestSchemaFunction returns the specified schema function that operates on a request payload along with
// any schema function args validation errors that occurred during instantiation
func NewRequestSchemaFunction(name string, params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	switch name {
	case DeviceCountry:
		return NewDeviceCountry(params)
	case DeviceCountryIn:
		return NewDeviceCountryIn(params)
	case DataCenter:
		return NewDataCenter(params)
	case DataCenterIn:
		return NewDataCenterIn(params)
	case Channel:
		return NewChannel()
	case EidAvailable:
		return NewAidAvailable(params)
	case UserFpdAvailable:
		return NewUserFpdAvailable()
	case FpdAvail:
		return NewFpdAvail()
	case GppSid:
		return NewGppSid(params)
	case TcfInScope:
		return NewTcfInScope()
	case Percent:
		return NewPercent(params)
	case PrebidKey:
		return NewPrebidKey(params)
	case Domain:
		return NewDomain(params)
	case Bundle:
		return NewBundle(params)
	case DeviceType:
		return NewDeviceType(params)

	default:
		return nil, fmt.Errorf("Schema function %s was not created", name)
	}
}

type deviceCountryIn struct {
	funcName     string
	CountryCodes []string
}

func NewDeviceCountryIn(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	var args []interface{}

	if err := jsonutil.Unmarshal(params, &args); err != nil {
		return nil, err
	}
	if len(args) != 1 {
		return nil, errors.New("deviceCountryIn expects one argument")
	}
	countryCodes, ok := args[0].([]string)
	if !ok {
		return nil, errors.New("deviceCountryIn arg 0 must be an array of strings")
	}
	return &deviceCountryIn{CountryCodes: countryCodes, funcName: DeviceCountryIn}, nil
}

func (dci *deviceCountryIn) Call(wrapper *openrtb_ext.RequestWrapper) (string, string, error) {
	if wrapper.Device == nil && wrapper.Device.Geo != nil && len(wrapper.Device.Geo.Country) == 0 {
		return "false", dci.funcName, nil
	}
	if contains := slices.Contains(dci.CountryCodes, wrapper.Device.Geo.Country); contains {
		return "true", dci.funcName, nil
	}
	return "false", dci.funcName, nil
}

type deviceCountry struct {
	funcName string
}

func NewDeviceCountry(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	var args []interface{}

	if err := jsonutil.Unmarshal(params, &args); err != nil {
		return nil, err
	}
	if len(args) > 0 {
		return nil, errors.New("deviceCountry expects 0 arguments")
	}
	return &deviceCountry{funcName: DeviceCountry}, nil
}

func (dc *deviceCountry) Call(wrapper *openrtb_ext.RequestWrapper) (string, string, error) {
	if wrapper.Device == nil && wrapper.Device.Geo != nil && len(wrapper.Device.Geo.Country) == 0 {
		return "", dc.funcName, fmt.Errorf("request.Device.Geo.Country is not present in request")
	}
	return wrapper.Device.Geo.Country, dc.funcName, nil
}

// ------------datacenters------------------

type dataCenter struct {
	funcName    string
	DataCenters []string
}

func NewDataCenter(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	var dc []string
	if err := jsonutil.Unmarshal(params, &dc); err != nil {
		return nil, err
	}
	return &dataCenter{DataCenters: dc, funcName: DataCenter}, nil
}

func (dc *dataCenter) Call(wrapper *openrtb_ext.RequestWrapper) (string, string, error) {

	// where is datacenter in bid request?
	// logic should be the same, but read a data center value from a proper location, not wrapper.Device.Geo.Region
	if wrapper.Device == nil && wrapper.Device.Geo != nil && len(wrapper.Device.Geo.Region) == 0 {
		return "", dc.funcName, fmt.Errorf("reqiuest.Device.Geo.Country is not present in request")
	}

	if len(dc.DataCenters) == 0 {
		return wrapper.Device.Geo.Region, dc.funcName, nil
	}

	if contains := slices.Contains(dc.DataCenters, wrapper.Device.Geo.Region); contains {
		return "true", dc.funcName, nil
	}
	return "false", dc.funcName, nil
}

// TODO
func NewDataCenterIn(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	var dc []string
	if err := jsonutil.Unmarshal(params, &dc); err != nil {
		return nil, err
	}
	return &dataCenter{DataCenters: dc}, nil
}

// ------------channel------------------
type channel struct {
	funcName string
	// no params
}

func NewChannel() (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	return &channel{funcName: Channel}, nil
}

func (c *channel) Call(wrapper *openrtb_ext.RequestWrapper) (string, string, error) {
	reqExt, err := wrapper.GetRequestExt()
	if err != nil {
		return "", c.funcName, err
	}
	reqExtPrebid := reqExt.GetPrebid()
	if reqExtPrebid == nil || reqExtPrebid.Channel == nil {
		return "", c.funcName, fmt.Errorf("reqiuest.ext.prebid or req.ext.prebid.channel is not present in request")
	}
	chName := reqExtPrebid.Channel.Name
	if chName == "pbjs" {
		return "web", c.funcName, nil
	}
	return chName, c.funcName, nil
}

// ------------eidAvailable------------------

type eidAvailable struct {
	funcName string
	eids     []string
}

// New
func NewAidAvailable(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	var eidsParam []string
	if err := jsonutil.Unmarshal(params, &eidsParam); err != nil {
		return nil, err
	}
	return &eidAvailable{eids: eidsParam, funcName: EidAvailable}, nil
}

func (ae *eidAvailable) Call(wrapper *openrtb_ext.RequestWrapper) (string, string, error) {
	// From Requirements doc:
	// Arguments may be null in which case it returns true if user.eids array exists and is non-empty.
	// False if user.eids doesn't exist or is empty.
	if wrapper.User == nil || len(wrapper.User.EIDs) == 0 {
		return "false", ae.funcName, nil
	}

	if len(ae.eids) == 0 {
		return "true", ae.funcName, nil
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
	return eidExists, ae.funcName, nil
}

// ------------userFpdAvailable------------------
type userFpdAvailable struct {
	funcName string
	// no params
}

func NewUserFpdAvailable() (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	return &userFpdAvailable{funcName: FpdAvail}, nil
}

func (ufpd *userFpdAvailable) Call(wrapper *openrtb_ext.RequestWrapper) (string, string, error) {
	res, err := checkUserDataAndUserExtData(wrapper)
	return res, ufpd.funcName, err
}

// ------------fpdAvail------------------
type fpdAvail struct {
	funcName string
	// no params
}

func NewFpdAvail() (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	return &fpdAvail{funcName: FpdAvail}, nil
}

func (fpd *fpdAvail) Call(wrapper *openrtb_ext.RequestWrapper) (string, string, error) {
	if wrapper.Site != nil {
		if wrapper.Site.Content != nil && len(wrapper.Site.Content.Data) > 0 {
			return "true", fpd.funcName, nil
		}
		siteExt, err := wrapper.GetSiteExt()
		if err != nil {
			return "false", fpd.funcName, err
		}

		ext := siteExt.GetExt()
		if extDataPresent(ext) {
			return "true", fpd.funcName, nil
		}
	}

	if wrapper.App != nil {
		if wrapper.App.Content != nil && len(wrapper.App.Content.Data) > 0 {
			return "true", fpd.funcName, nil
		}
		appExt, err := wrapper.GetAppExt()
		if err != nil {
			return "false", fpd.funcName, err
		}

		ext := appExt.GetExt()
		if extDataPresent(ext) {
			return "true", fpd.funcName, nil
		}
	}

	res, err := checkUserDataAndUserExtData(wrapper)
	return res, fpd.funcName, err
}

// ------------gppSid------------------
type gppSid struct {
	funcName string
	gppSids  []int8
}

func NewGppSid(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	var gppSidIn []int8
	if err := jsonutil.Unmarshal(params, &gppSidIn); err != nil {
		return nil, err
	}
	return &gppSid{gppSids: gppSidIn, funcName: GppSid}, nil
}

func (sid *gppSid) Call(wrapper *openrtb_ext.RequestWrapper) (string, string, error) {
	if len(sid.gppSids) > 0 && wrapper.Regs != nil && len(wrapper.Regs.GPPSID) > 0 {
		for _, s := range sid.gppSids {
			if contains := slices.Contains(wrapper.Regs.GPPSID, s); contains {
				return "true", sid.funcName, nil
			}
		}
	}
	return "false", sid.funcName, nil
}

// ------------tcfInScope------------------
type tcfInScope struct {
	funcName string
	// no params
}

func NewTcfInScope() (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	return &tcfInScope{funcName: TcfInScope}, nil
}

func (tcf *tcfInScope) Call(wrapper *openrtb_ext.RequestWrapper) (string, string, error) {
	if wrapper.Regs != nil && wrapper.Regs.GDPR == ptrutil.ToPtr[int8](1) {
		return "true", tcf.funcName, nil
	}
	return "false", tcf.funcName, nil
}

// ------------percent------------------
type percent struct {
	funcName string
	value    int
}

func NewPercent(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	var percentValue int
	if err := jsonutil.Unmarshal(params, &percentValue); err != nil {
		return nil, err
	}
	return &percent{value: percentValue, funcName: Percent}, nil
}

func (p *percent) Call(wrapper *openrtb_ext.RequestWrapper) (string, string, error) {
	percValue := 5 //default value
	if p.value < 0 {
		percValue = 0
	}
	if p.value > 100 {
		percValue = 100
	}
	randNum := randRange(0, 100)
	if randNum < percValue {
		return "true", p.funcName, nil
	}
	return "false", p.funcName, nil
}

// ------------prebidKey------------------
type prebidKey struct {
	funcName string
	key      string
}

func NewPrebidKey(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	var newKey string
	if err := jsonutil.Unmarshal(params, &newKey); err != nil {
		return nil, err
	}
	return &prebidKey{key: newKey, funcName: PrebidKey}, nil
}

func (p *prebidKey) Call(wrapper *openrtb_ext.RequestWrapper) (string, string, error) {
	reqExt, err := wrapper.GetRequestExt()
	if err != nil {
		return "", p.funcName, err
	}
	reqExtPrebid := reqExt.GetPrebid()
	// reqExtPrebid doesn't have kvps !
	// expected impl:
	// return reqExtPrebid.GetKVPs()[p.key], nil

	return reqExtPrebid.Integration, p.funcName, nil //stub
}

// ------------domain------------------
type domain struct {
	funcName    string
	domainNames []string
}

func NewDomain(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	var newdomains []string
	if err := jsonutil.Unmarshal(params, &newdomains); err != nil {
		return nil, err
	}
	return &domain{domainNames: newdomains, funcName: Domain}, nil
}

func (d *domain) Call(wrapper *openrtb_ext.RequestWrapper) (string, string, error) {
	reqDomain := ""
	if wrapper.Site != nil {
		reqDomain = wrapper.Site.Domain
	} else if wrapper.App != nil {
		reqDomain = wrapper.App.Domain
	} else if wrapper.DOOH != nil {
		reqDomain = wrapper.DOOH.Domain
	}

	if len(d.domainNames) == 0 {
		return reqDomain, d.funcName, nil
	}

	if contains := slices.Contains(d.domainNames, reqDomain); contains {
		return "true", d.funcName, nil
	}
	return "false", d.funcName, nil
}

// ------------bundle------------------
type bundle struct {
	funcName    string
	bundleNames []string
}

func NewBundle(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	var newBundles []string
	if err := jsonutil.Unmarshal(params, &newBundles); err != nil {
		return nil, err
	}
	return &bundle{bundleNames: newBundles, funcName: Bundle}, nil
}

func (b *bundle) Call(wrapper *openrtb_ext.RequestWrapper) (string, string, error) {
	bundleName := ""
	if wrapper.App != nil {
		bundleName = wrapper.App.Bundle
	}
	if len(b.bundleNames) == 0 {
		return bundleName, b.funcName, nil
	}

	if contains := slices.Contains(b.bundleNames, bundleName); contains {
		return "true", b.funcName, nil
	}
	return "false", b.funcName, nil
}

// ------------deviceType------------------
type deviceType struct {
	funcName string
	types    []string
}

func NewDeviceType(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	var deviceTypes []string
	if err := jsonutil.Unmarshal(params, &deviceTypes); err != nil {
		return nil, err
	}
	return &deviceType{types: deviceTypes, funcName: DeviceType}, nil
}

func (d *deviceType) Call(wrapper *openrtb_ext.RequestWrapper) (string, string, error) {
	devType := ""
	if wrapper.Device != nil {
		devTypeInt := wrapper.Device.DeviceType
		err := errors.New("")
		devType, err = convertDevTypeToString(devTypeInt)
		if err != nil {
			return "", d.funcName, err
		}
	}
	if len(d.types) == 0 {
		return devType, d.funcName, nil
	}

	if contains := slices.Contains(d.types, devType); contains {
		return "true", d.funcName, nil
	}
	return "false", d.funcName, nil
}

// ------------mediaTypes------------------
// ------------adUnitCode------------------
// ------------bidPrice------------------

// ----------helper functions---------
func checkUserDataAndUserExtData(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	if wrapper.User == nil {
		return "false", nil
	}
	if len(wrapper.User.Data) > 0 {
		return "true", nil
	}
	userExt, err := wrapper.GetUserExt()
	if err != nil {
		return "false", err
	}
	ext := userExt.GetExt()
	if extDataPresent(ext) {
		return "true", nil
	}
	return "false", nil
}

func extDataPresent(ext map[string]json.RawMessage) bool {
	val, ok := ext["data"]
	return ok && len(val) > 0
}

func randRange(min, max int) int {
	return rand.Intn(max-min) + min
}

func convertDevTypeToString(typeInt adcom1.DeviceType) (string, error) {
	switch typeInt {
	case adcom1.DeviceMobile:
		return "mobile", nil
	case adcom1.DevicePC:
		return "pc", nil
	case adcom1.DeviceTV:
		return "tv", nil
	case adcom1.DevicePhone:
		return "phone", nil
	case adcom1.DeviceTablet:
		return "tablet", nil
	case adcom1.DeviceConnected:
		return "connected device", nil
	case adcom1.DeviceSetTopBox:
		return "set top box", nil
	case adcom1.DeviceOOH:
		return "dooh", nil
	default:
		return "", fmt.Errorf("Device type %d was not found", typeInt)
	}
}
