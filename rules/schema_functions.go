package rules

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/buger/jsonparser"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/prebid/prebid-server/v3/util/randomutil"
)

const (
	Channel          = "channel"
	DataCenter       = "dataCenter"
	DataCenterIn     = "dataCenterIn"
	DeviceCountry    = "deviceCountry"
	DeviceCountryIn  = "deviceCountryIn"
	EidAvailable     = "eidAvailable"
	EidIn            = "eidIn"
	FpdAvailable     = "fpdAvailable"
	GppSidAvailable  = "gppSidAvailable"
	GppSidIn         = "gppSidIn"
	Percent          = "percent"
	TcfInScope       = "tcfInScope"
	UserFpdAvailable = "userFpdAvailable"
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
	case FpdAvailable:
		return NewFpdAvailable(params)
	case GppSidAvailable:
		return NewGppSidAvailable(params)
	case GppSidIn:
		return NewGppSidIn(params)
	case TcfInScope:
		return NewTcfInScope(params)
	case Percent:
		return NewPercent(params)
	default:
		return nil, fmt.Errorf("Schema function %s was not created", name)
	}
}

// ------------deviceCountryIn--------------
type deviceCountryIn struct {
	Countries        []string `json:"countries"`
	CountryDirectory map[string]struct{}
}

func NewDeviceCountryIn(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	schemaFunc := &deviceCountryIn{}
	if err := jsonutil.Unmarshal(params, schemaFunc); err != nil {
		return nil, err
	}

	if len(schemaFunc.Countries) == 0 {
		return nil, errors.New("Missing countries argument for deviceCountryIn schema function")
	}

	schemaFunc.CountryDirectory = make(map[string]struct{})
	for i := 0; i < len(schemaFunc.Countries); i++ {
		schemaFunc.CountryDirectory[schemaFunc.Countries[i]] = struct{}{}
	}

	return schemaFunc, nil
}

func (dci *deviceCountryIn) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	deviceGeo := getDeviceGeo(wrapper)
	if deviceGeo == nil || len(deviceGeo.Country) == 0 {
		return "false", nil
	}

	_, found := dci.CountryDirectory[deviceGeo.Country]
	return fmt.Sprintf("%t", found), nil
}

func (dci *deviceCountryIn) Name() string {
	return DeviceCountryIn
}

// ------------deviceCountry----------------
type deviceCountry struct{}

func NewDeviceCountry(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	if err := checkNilArgs(params, DeviceCountry); err != nil {
		return nil, err
	}
	return &deviceCountry{}, nil
}

func (dc *deviceCountry) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	if deviceGeo := getDeviceGeo(wrapper); deviceGeo != nil && len(deviceGeo.Country) > 0 {
		return deviceGeo.Country, nil
	}
	return "", nil

}

func (dci *deviceCountry) Name() string {
	return DeviceCountry
}

// ------------datacenter-------------------

type dataCenter struct{}

func NewDataCenter(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	if err := checkNilArgs(params, DataCenter); err != nil {
		return nil, err
	}
	return &dataCenter{}, nil
}

func (dc *dataCenter) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	if deviceGeo := getDeviceGeo(wrapper); deviceGeo != nil {
		return deviceGeo.Region, nil
	}
	return "", nil

}

func (dci *dataCenter) Name() string {
	return DataCenter
}

// ------------dataCenterIn-----------------
type dataCenterIn struct {
	DataCenters   []string `json:"datacenters"`
	DataCenterDir map[string]struct{}
}

func NewDataCenterIn(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	schemaFunc := &dataCenterIn{}
	if err := jsonutil.Unmarshal(params, schemaFunc); err != nil {
		return nil, err
	}

	if len(schemaFunc.DataCenters) == 0 {
		return nil, errors.New("Empty datacenter argument in dataCenterIn schema function")
	}

	schemaFunc.DataCenterDir = make(map[string]struct{})
	for i := 0; i < len(schemaFunc.DataCenters); i++ {
		schemaFunc.DataCenterDir[schemaFunc.DataCenters[i]] = struct{}{}
	}

	return schemaFunc, nil
}

func (dc *dataCenterIn) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	deviceGeo := getDeviceGeo(wrapper)
	if deviceGeo == nil || len(deviceGeo.Region) == 0 {
		return "false", nil
	}

	_, found := dc.DataCenterDir[deviceGeo.Region]
	return fmt.Sprintf("%t", found), nil
}

func (dci *dataCenterIn) Name() string {
	return DataCenterIn
}

// ------------channel------------------
type channel struct{}

func NewChannel(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	if err := checkNilArgs(params, Channel); err != nil {
		return nil, err
	}
	return &channel{}, nil
}

func (c *channel) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	prebid, err := getExtRequestPrebid(wrapper)
	if err != nil {
		return "", err
	}

	if prebid == nil || prebid.Channel == nil || len(prebid.Channel.Name) == 0 {
		return "", nil
	}
	chName := prebid.Channel.Name
	if chName == "pbjs" {
		return "web", nil
	}
	return chName, nil
}

func (c *channel) Name() string {
	return Channel
}

// ------------eidAvailable------------------
type eidAvailable struct{}

func NewEidAvailable(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	if err := checkNilArgs(params, EidAvailable); err != nil {
		return nil, err
	}
	return &eidAvailable{}, nil
}

func (ea *eidAvailable) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	if len(getUserEIDS(wrapper)) > 0 {
		return "true", nil
	}
	return "false", nil
}
func (ea *eidAvailable) Name() string {
	return EidAvailable
}

// ------------eidIn-------------------------
type eidIn struct {
	EidSources []string `json:"sources"`
	Eids       map[string]struct{}
}

func NewEidIn(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	schemaFunc := &eidIn{}
	if err := jsonutil.Unmarshal(params, schemaFunc); err != nil {
		return nil, err
	}

	if len(schemaFunc.EidSources) == 0 {
		return nil, errors.New("Empty sources argument in eidIn schema function")
	}

	schemaFunc.Eids = make(map[string]struct{})
	for i := range schemaFunc.EidSources {
		schemaFunc.Eids[schemaFunc.EidSources[i]] = struct{}{}
	}

	return schemaFunc, nil
}

func (ei *eidIn) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	if len(ei.Eids) == 0 {
		return "false", nil
	}

	eids := getUserEIDS(wrapper)

	for i := range eids {
		if _, found := ei.Eids[eids[i].Source]; found {
			return "true", nil
		}
	}
	return "false", nil
}

func (ei *eidIn) Name() string {
	return EidIn
}

// ------------userFpdAvailable------------------
type userFpdAvailable struct{}

func NewUserFpdAvailable(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	if err := checkNilArgs(params, UserFpdAvailable); err != nil {
		return nil, err
	}
	return &userFpdAvailable{}, nil
}

func (ufpd *userFpdAvailable) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	return checkUserDataAndUserExtData(wrapper)
}

func (ufpd *userFpdAvailable) Name() string {
	return UserFpdAvailable
}

// ------------fpdAvail------------------
type fpdAvailable struct{}

func NewFpdAvailable(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	if err := checkNilArgs(params, FpdAvailable); err != nil {
		return nil, err
	}
	return &fpdAvailable{}, nil
}

func (fpd *fpdAvailable) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	if wrapper == nil || wrapper.BidRequest == nil {
		return "false", nil
	}

	if found, _ := hasSiteContentDataOrSiteExtData(wrapper); found == "true" {
		return "true", nil
	}

	if found, _ := hasAppContentDataOrAppExtData(wrapper); found == "true" {
		return "true", nil
	}

	return checkUserDataAndUserExtData(wrapper)
}

func (fpd *fpdAvailable) Name() string {
	return FpdAvailable
}

// ------------gppSid------------------
type gppSidIn struct {
	SidList []int8 `json:"sids"`
	GppSids map[int8]struct{}
}

func NewGppSidIn(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	schemaFunc := &gppSidIn{}
	if err := jsonutil.Unmarshal(params, schemaFunc); err != nil {
		return nil, err
	}

	if len(schemaFunc.SidList) == 0 {
		return nil, errors.New("Empty GPPSIDs list argument in gppSidIn schema function")
	}

	schemaFunc.GppSids = make(map[int8]struct{})
	for i := range schemaFunc.SidList {
		schemaFunc.GppSids[schemaFunc.SidList[i]] = struct{}{}
	}

	return schemaFunc, nil
}

func (sid *gppSidIn) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	if len(sid.GppSids) == 0 {
		return "false", nil
	}

	if !hasGPPSIDs(wrapper) {
		return "false", nil
	}

	for i := range wrapper.Regs.GPPSID {
		if _, found := sid.GppSids[wrapper.Regs.GPPSID[i]]; found {
			return "true", nil
		}
	}

	return "false", nil
}

func (sid *gppSidIn) Name() string {
	return GppSidIn
}

// ------------gppSidAvailable-------------
type gppSidAvailable struct{}

func NewGppSidAvailable(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	if err := checkNilArgs(params, GppSidAvailable); err != nil {
		return nil, err
	}
	return &gppSidAvailable{}, nil
}

func (sid *gppSidAvailable) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	return fmt.Sprintf("%t", hasGPPSIDs(wrapper)), nil
}

func (sid *gppSidAvailable) Name() string {
	return GppSidAvailable
}

// ------------tcfInScope------------------
type tcfInScope struct{}

func NewTcfInScope(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	if err := checkNilArgs(params, TcfInScope); err != nil {
		return nil, err
	}
	return &tcfInScope{}, nil
}

func (tcf *tcfInScope) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	if regs := getRequestRegs(wrapper); regs != nil && regs.GDPR != nil && *regs.GDPR == int8(1) {
		return "true", nil
	}
	return "false", nil
}

func (tcf *tcfInScope) Name() string {
	return TcfInScope
}

// ------------percent------------------
type percent struct {
	Percent *int `json:"pct"`
	rand    randomutil.RandomGenerator
}

func NewPercent(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	schemaFunc := &percent{
		rand: randomutil.RandomNumberGenerator{},
	}
	if err := jsonutil.Unmarshal(params, schemaFunc); err != nil {
		return nil, err
	}
	if schemaFunc.Percent == nil {
		schemaFunc.Percent = ptrutil.ToPtr(5)
	}
	if *schemaFunc.Percent < 0 {
		*schemaFunc.Percent = 0
	}
	if *schemaFunc.Percent > 100 {
		*schemaFunc.Percent = 100
	}
	return schemaFunc, nil
}

func (p *percent) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	pValue := *p.Percent
	if pValue <= 0 {
		return "false", nil
	}
	if pValue >= 100 {
		return "true", nil
	}

	randNum := p.rand.Intn(100)
	if randNum <= pValue {
		return "true", nil
	}
	return "false", nil
}

func (p *percent) Name() string {
	return Percent
}

func checkUserDataAndUserExtData(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	if wrapper == nil {
		return "false", nil
	}

	if hasUserData(wrapper) {
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
	val, found := ext["data"]
	if !found {
		return false
	}

	_, dataType, _, err := jsonparser.Get(val)
	if err != nil || dataType != jsonparser.Array {
		return false
	}

	hasElements := false
	jsonparser.ArrayEach(val, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		hasElements = true
		return
	})

	return hasElements
}

func hasSiteContentDataOrSiteExtData(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	if wrapper != nil && wrapper.BidRequest != nil && wrapper.Site != nil {
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
	return "false", nil
}

func hasAppContentDataOrAppExtData(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	if wrapper != nil && wrapper.BidRequest != nil && wrapper.App != nil {
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
	return "false", nil
}

func checkNilArgs(params json.RawMessage, funcName string) error {
	if len(params) == 0 {
		return nil
	}

	if string(params) == "null" || string(params) == "{}" {
		return nil
	}

	return fmt.Errorf("%s expects 0 arguments", funcName)
}

func getDeviceGeo(wrapper *openrtb_ext.RequestWrapper) *openrtb2.Geo {
	if wrapper != nil && wrapper.BidRequest != nil && wrapper.Device != nil && wrapper.Device.Geo != nil {
		return wrapper.Device.Geo
	}
	return nil
}

func getExtRequestPrebid(wrapper *openrtb_ext.RequestWrapper) (*openrtb_ext.ExtRequestPrebid, error) {
	reqExt, err := wrapper.GetRequestExt()
	if err != nil {
		return nil, err
	}

	return reqExt.GetPrebid(), nil
}

func getUserEIDS(wrapper *openrtb_ext.RequestWrapper) []openrtb2.EID {
	if wrapper != nil && wrapper.User != nil && len(wrapper.User.EIDs) > 0 {
		return wrapper.User.EIDs
	}
	return nil
}

func hasGPPSIDs(wrapper *openrtb_ext.RequestWrapper) bool {
	regs := getRequestRegs(wrapper)
	if regs == nil {
		return false
	}
	for i := range regs.GPPSID {
		if regs.GPPSID[i] > int8(0) {
			return true
		}
	}
	return false
}

func hasUserData(wrapper *openrtb_ext.RequestWrapper) bool {
	if wrapper == nil || wrapper.BidRequest == nil || wrapper.User == nil {
		return false
	}
	return len(wrapper.User.Data) > 0
}

func getRequestRegs(wrapper *openrtb_ext.RequestWrapper) *openrtb2.Regs {
	if wrapper != nil && wrapper.BidRequest != nil && wrapper.Regs != nil {
		return wrapper.Regs
	}
	return nil
}
