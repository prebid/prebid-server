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
	EidIn            = "eidAIn"
	UserFpdAvailable = "userFpdAvailable"
	FpdAvail         = "fpdAvail"
	GppSid           = "gppSid"
	GppSidIn         = "gppSidIn"
	TcfInScope       = "tcfInScope"
	Percent          = "percent"
	PrebidKey        = "prebidKey"
	Domain           = "domain"
	DomainIn         = "domainIn"
	Bundle           = "bundle"
	DeviceType       = "deviceType"
	AdUnitCode       = "adUnitCode"
	MediaTypes       = "mediaTypes"
)

// SchemaFunction...
type SchemaFunction[T any] interface {
	Call(payload *T) (string, error)
	Name() string
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
		return NewChannel(params)
	case EidAvailable:
		return NewEidAvailable(params)
	case EidIn:
		return NewEidIn(params)
	case UserFpdAvailable:
		return NewUserFpdAvailable(params)
	case FpdAvail:
		return NewFpdAvailable(params)
	case GppSid:
		return NewGppSidAvailable(params)
	case GppSidIn:
		return NewGppSidIn(params)
	case TcfInScope:
		return NewTcfInScope(params)
	case Percent:
		return NewPercent(params)
	case PrebidKey:
		return NewPrebidKey(params)
	case Domain:
		return NewDomain(params)
	case DomainIn:
		return NewDomainIn(params)
	case Bundle:
		return NewBundle(params)
	case DeviceType:
		return NewDeviceType(params)

	default:
		return nil, fmt.Errorf("Schema function %s was not created", name)
	}
}

type deviceCountryIn struct {
	CountryCodes []string
}

func NewDeviceCountryIn(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	countryCodes, err := checkArgsStringList(params, DeviceCountryIn)
	return &deviceCountryIn{CountryCodes: countryCodes}, err
}

func (dci *deviceCountryIn) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	if wrapper.Device == nil && wrapper.Device.Geo != nil && len(wrapper.Device.Geo.Country) == 0 {
		return "false", nil
	}
	if contains := slices.Contains(dci.CountryCodes, wrapper.Device.Geo.Country); contains {
		return "true", nil
	}
	return "false", nil
}

func (dci *deviceCountryIn) Name() string {
	return DeviceCountryIn
}

type deviceCountry struct{}

func NewDeviceCountry(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	return &deviceCountry{}, checkNilArgs(params, DeviceCountry)
}

func (dc *deviceCountry) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	if wrapper.Device == nil && wrapper.Device.Geo != nil && len(wrapper.Device.Geo.Country) == 0 {
		return "", fmt.Errorf("request.Device.Geo.Country is not present in request")
	}
	return wrapper.Device.Geo.Country, nil
}

func (dci *deviceCountry) Name() string {
	return DeviceCountry
}

// ------------datacenters------------------

type dataCenter struct{}

func NewDataCenter(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	return &dataCenter{}, checkNilArgs(params, DataCenter)
}

func (dc *dataCenter) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {

	// where is datacenter in bid request?
	// logic should be the same, but read a data center value from a proper location, not wrapper.Device.Geo.Region
	if wrapper.Device == nil && wrapper.Device.Geo != nil && len(wrapper.Device.Geo.Region) == 0 {
		return "", fmt.Errorf("dataCenter is not present in request")
	}
	return wrapper.Device.Geo.Region, nil
}

func (dci *dataCenter) Name() string {
	return DataCenter
}

type dataCenterIn struct {
	DataCenters []string
}

func NewDataCenterIn(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	dataCenters, err := checkArgsStringList(params, DataCenterIn)
	return &dataCenterIn{DataCenters: dataCenters}, err
}

func (dc *dataCenterIn) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {

	// where is datacenter in bid request?
	// logic should be the same, but read a data center value from a proper location, not wrapper.Device.Geo.Region
	if wrapper.Device == nil && wrapper.Device.Geo != nil && len(wrapper.Device.Geo.Region) == 0 {
		return "", fmt.Errorf("reqiuest.Device.Geo.Country is not present in request")
	}
	if contains := slices.Contains(dc.DataCenters, wrapper.Device.Geo.Region); contains {
		return "true", nil
	}
	return "false", nil
}
func (dci *dataCenterIn) Name() string {
	return DataCenterIn
}

// ------------channel------------------
type channel struct {
	// no params
}

func NewChannel(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	return &channel{}, checkNilArgs(params, DataCenter)
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

func (c *channel) Name() string {
	return Channel
}

// ------------eidAvailable------------------

type eidIn struct {
	eids []string
}

// New
func NewEidIn(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	eidsParam, err := checkArgsStringList(params, EidIn)
	return &eidIn{eids: eidsParam}, err
}

func (ei *eidIn) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	if wrapper.User == nil || len(wrapper.User.EIDs) == 0 {
		return "false", nil
	}
	for _, eidParam := range ei.eids {
		for _, eid := range wrapper.User.EIDs {
			if eidParam == eid.Source {
				return "true", nil
			}
		}
	}
	return "false", nil
}

func (ei *eidIn) Name() string {
	return EidIn
}

type eidAvailable struct {
}

func NewEidAvailable(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	return &eidAvailable{}, checkNilArgs(params, EidAvailable)
}

func (ea *eidAvailable) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	if wrapper.User == nil || len(wrapper.User.EIDs) == 0 {
		return "", fmt.Errorf("request.User.EIDs is not present in request")
	}
	return "true", nil
}
func (ea *eidAvailable) Name() string {
	return EidAvailable
}

// ------------userFpdAvailable------------------
type userFpdAvailable struct {
	// no params
}

func NewUserFpdAvailable(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	return &userFpdAvailable{}, checkNilArgs(params, UserFpdAvailable)
}

func (ufpd *userFpdAvailable) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	return checkUserDataAndUserExtData(wrapper)
}

func (ufpd *userFpdAvailable) Name() string {
	return UserFpdAvailable
}

// ------------fpdAvail------------------
type fpdAvailable struct {
	// no params
}

func NewFpdAvailable(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	return &fpdAvailable{}, checkNilArgs(params, FpdAvail)
}

func (fpd *fpdAvailable) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	if wrapper.Site != nil {
		if wrapper.Site.Content != nil && len(wrapper.Site.Content.Data) > 0 {
			return "true", nil
		}
		siteExt, err := wrapper.GetSiteExt()
		if err != nil {
			return "false", err
		}

		ext := siteExt.GetExt()
		if extDataPresent(ext) {
			return "true", nil
		}
	}

	if wrapper.App != nil {
		if wrapper.App.Content != nil && len(wrapper.App.Content.Data) > 0 {
			return "true", nil
		}
		appExt, err := wrapper.GetAppExt()
		if err != nil {
			return "false", err
		}

		ext := appExt.GetExt()
		if extDataPresent(ext) {
			return "true", nil
		}
	}

	return checkUserDataAndUserExtData(wrapper)
}

func (fpd *fpdAvailable) Name() string {
	return FpdAvail
}

// ------------gppSid------------------
type gppSidIn struct {
	gppSids []int8
}

func NewGppSidIn(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	gppSids, err := checkArgsInt8List(params, GppSidIn)
	return &gppSidIn{gppSids: gppSids}, err
}

func (sid *gppSidIn) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	if len(sid.gppSids) > 0 && wrapper.Regs != nil && len(wrapper.Regs.GPPSID) > 0 {
		for _, s := range sid.gppSids {
			if contains := slices.Contains(wrapper.Regs.GPPSID, s); contains {
				return "true", nil
			}
		}
	}
	return "false", nil
}

func (sid *gppSidIn) Name() string {
	return GppSidIn
}

type gppSidAvailable struct {
}

func NewGppSidAvailable(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	return &gppSidAvailable{}, checkNilArgs(params, GppSid)
}

func (sid *gppSidAvailable) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	if wrapper.Regs != nil && len(wrapper.Regs.GPPSID) > 0 {
		for _, gppSid := range wrapper.Regs.GPPSID {
			if gppSid > 0 {
				return "true", nil
			}
		}
	}
	return "false", nil
}

func (sid *gppSidAvailable) Name() string {
	return GppSid
}

// ------------tcfInScope------------------
type tcfInScope struct {
	// no params
}

func NewTcfInScope(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	return &tcfInScope{}, checkNilArgs(params, TcfInScope)
}

func (tcf *tcfInScope) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	if wrapper.Regs != nil && wrapper.Regs.GDPR == ptrutil.ToPtr[int8](1) {
		return "true", nil
	}
	return "false", nil
}

func (tcf *tcfInScope) Name() string {
	return TcfInScope
}

// ------------percent------------------
type percent struct {
	value int
}

func NewPercent(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	percentValue, err := checkArgsInt(params, Percent)
	return &percent{value: percentValue}, err
}

func (p *percent) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	percValue := p.value
	if percValue < 0 {
		percValue = 0
	}
	if percValue > 100 {
		percValue = 100
	}
	randNum := randRange(0, 100)
	if randNum < percValue {
		return "true", nil
	}
	return "false", nil
}

func (p *percent) Name() string {
	return Percent
}

// ------------prebidKey------------------
type prebidKey struct {
	key string
}

func NewPrebidKey(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	newKey, err := checkArgsString(params, PrebidKey)
	return &prebidKey{key: newKey}, err
}

func (p *prebidKey) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	reqExt, err := wrapper.GetRequestExt()
	if err != nil {
		return "", err
	}
	reqExtPrebid := reqExt.GetPrebid()
	// reqExtPrebid doesn't have kvps !
	// expected impl:
	// return reqExtPrebid.GetKVPs()[p.key], nil

	return reqExtPrebid.Integration, nil //stub
}

func (p *prebidKey) Name() string {
	return PrebidKey
}

// ------------domain------------------
type domain struct {
}

func NewDomain(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	return &domain{}, checkNilArgs(params, Domain)
}

func (d *domain) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	return getReqDomain(wrapper), nil
}

func (d *domain) Name() string {
	return Domain
}

type domainIn struct {
	domainNames []string
}

func NewDomainIn(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	newdomains, err := checkArgsStringList(params, DomainIn)
	return &domainIn{domainNames: newdomains}, err
}

func (d *domainIn) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	reqDomain := getReqDomain(wrapper)
	if contains := slices.Contains(d.domainNames, reqDomain); contains {
		return "true", nil
	}
	return "false", nil
}

func (d *domainIn) Name() string {
	return DomainIn
}

// TODO: from here
// ------------bundle------------------
type bundle struct {
	bundleNames []string
}

func NewBundle(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	var newBundles []string
	if err := jsonutil.Unmarshal(params, &newBundles); err != nil {
		return nil, err
	}
	return &bundle{bundleNames: newBundles}, nil
}

func (b *bundle) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	bundleName := ""
	if wrapper.App != nil {
		bundleName = wrapper.App.Bundle
	}
	if len(b.bundleNames) == 0 {
		return bundleName, nil
	}

	if contains := slices.Contains(b.bundleNames, bundleName); contains {
		return "true", nil
	}
	return "false", nil
}
func (b *bundle) Name() string {
	return Bundle
}

// ------------deviceType------------------
type deviceType struct {
	types []string
}

func NewDeviceType(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	var deviceTypes []string
	if err := jsonutil.Unmarshal(params, &deviceTypes); err != nil {
		return nil, err
	}
	return &deviceType{types: deviceTypes}, nil
}

func (d *deviceType) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	devType := ""
	if wrapper.Device != nil {
		devTypeInt := wrapper.Device.DeviceType
		err := errors.New("")
		devType, err = convertDevTypeToString(devTypeInt)
		if err != nil {
			return "", err
		}
	}
	if len(d.types) == 0 {
		return devType, nil
	}

	if contains := slices.Contains(d.types, devType); contains {
		return "true", nil
	}
	return "false", nil
}

func (d *deviceType) Name() string {
	return DeviceType
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

func getReqDomain(wrapper *openrtb_ext.RequestWrapper) string {
	reqDomain := ""
	if wrapper.Site != nil {
		reqDomain = wrapper.Site.Domain
	} else if wrapper.App != nil {
		reqDomain = wrapper.App.Domain
	} else if wrapper.DOOH != nil {
		reqDomain = wrapper.DOOH.Domain
	}
	return reqDomain
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

func checkNilArgs(params json.RawMessage, funcName string) error {
	var args []interface{}

	if err := jsonutil.Unmarshal(params, &args); err != nil {
		return err
	}
	if len(args) > 0 {
		return fmt.Errorf("%s expects 0 arguments", funcName)
	}
	return nil
}

func checkSingleArg(params json.RawMessage, funcName string) ([]interface{}, error) {
	var args []interface{}

	if err := jsonutil.Unmarshal(params, &args); err != nil {
		return nil, err
	}
	if len(args) != 1 {
		return nil, fmt.Errorf("%s expects one argument", funcName)
	}
	return args, nil
}

func checkArgsStringList(params json.RawMessage, funcName string) ([]string, error) {
	args, err := checkSingleArg(params, funcName)
	if err != nil {
		return nil, err
	}
	values, ok := args[0].([]string)
	if !ok {
		return nil, fmt.Errorf("%s arg 0 must be an array of strings", funcName)
	}
	return values, nil
}

func checkArgsInt8List(params json.RawMessage, funcName string) ([]int8, error) {
	args, err := checkSingleArg(params, funcName)
	if err != nil {
		return nil, err
	}
	values, ok := args[0].([]int8)
	if !ok {
		return nil, fmt.Errorf("%s arg 0 must be an array of ints", funcName)
	}
	return values, nil
}

func checkArgsInt(params json.RawMessage, funcName string) (int, error) {
	args, err := checkSingleArg(params, funcName)
	if err != nil {
		return 0, err
	}
	value, ok := args[0].(int)
	if !ok {
		return 0, fmt.Errorf("%s arg 0 must be an array of ints", funcName)
	}

	return value, nil
}

func checkArgsString(params json.RawMessage, funcName string) (string, error) {
	args, err := checkSingleArg(params, funcName)
	if err != nil {
		return "", err
	}
	value, ok := args[0].(string)
	if !ok {
		return "", fmt.Errorf("%s arg 0 must be an array of ints", funcName)
	}

	return value, nil
}
