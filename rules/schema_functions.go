package rules

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"

	"github.com/buger/jsonparser"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

const (
	AdUnitCode       = "adUnitCode"
	Channel          = "channel"
	DataCenter       = "dataCenter"
	DataCenterIn     = "dataCenterIn"
	DeviceCountry    = "deviceCountry"
	DeviceCountryIn  = "deviceCountryIn"
	DeviceTypeIn     = "deviceTypeIn"
	EidAvailable     = "eidAvailable"
	EidIn            = "eidIn"
	FpdAvailable     = "fpdAvailable"
	GppSidAvailable  = "gppSidAvailable"
	GppSidIn         = "gppSidIn"
	MediaTypes       = "mediaTypes"
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

// ------------deviceCountry----------------

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

type deviceCountry struct{}

func NewDeviceCountry(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	return &deviceCountry{}, checkNilArgs(params, DeviceCountry)
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

// ------------datacenters------------------

type dataCenter struct{}

func NewDataCenter(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	return &dataCenter{}, checkNilArgs(params, DataCenter)
}

func (dc *dataCenter) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	if deviceGeo := getDeviceGeo(wrapper); deviceGeo != nil && len(deviceGeo.Region) > 0 {
		return wrapper.Device.Geo.Region, nil
	}
	return "", nil

}

func (dci *dataCenter) Name() string {
	return DataCenter
}

type dataCenterIn struct {
	DataCenterDir map[string]struct{}
}

func NewDataCenterIn(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	dataCenters, err := checkArgsStringList(params, DataCenterIn)
	if err != nil {
		return nil, err
	}

	schemaFunc := &dataCenterIn{
		DataCenterDir: make(map[string]struct{}),
	}

	for i := 0; i < len(dataCenters); i++ {
		schemaFunc.DataCenterDir[dataCenters[i]] = struct{}{}
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
type channel struct {
}

func NewChannel(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	return &channel{}, checkNilArgs(params, DataCenter)
}

func (c *channel) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	prebid, err := getExtRequestPrebid(wrapper)
	if err != nil {
		return "", err
	}

	if prebid == nil || prebid.Channel == nil {
		return "", nil
	}

	if len(prebid.Channel.Name) == 0 {
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

type eidIn struct {
	eidList []string
	eidDir  map[string]struct{}
}

func NewEidIn(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	eidsParam, err := checkArgsStringList(params, EidIn)
	if err != nil {
		return nil, err
	}

	schemaFunc := &eidIn{
		eidList: eidsParam,
		eidDir:  make(map[string]struct{}),
	}

	for i := 0; i < len(eidsParam); i++ {
		schemaFunc.eidDir[eidsParam[i]] = struct{}{}
	}

	return schemaFunc, nil
}

func (ei *eidIn) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	if len(ei.eidDir) == 0 {
		return "false", nil
	}

	eids := getUserEIDS(wrapper)

	for i := 0; i < len(eids); i++ {
		if _, found := ei.eidDir[eids[i].Source]; found {
			return "true", nil
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
	if len(getUserEIDS(wrapper)) == 0 {
		return "true", nil
	}
	return "false", nil
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
	return &fpdAvailable{}, checkNilArgs(params, FpdAvailable)
}

func (fpd *fpdAvailable) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	if wrapper == nil || wrapper.BidRequest == nil {
		return "false", nil
	}

	if found, _ := checkSiteContentDataAndSiteExtData(wrapper); found == "true" {
		return "true", nil
	}

	if found, _ := checkAppContentDataAndAppExtData(wrapper); found == "true" {
		return "true", nil
	}

	return checkUserDataAndUserExtData(wrapper)
}

func (fpd *fpdAvailable) Name() string {
	return FpdAvailable
}

// ------------gppSid------------------
type gppSidIn struct {
	gppSids map[int8]struct{}
}

func NewGppSidIn(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	gppSids, err := checkArgsInt8List(params, GppSidIn)
	if err != nil {
		return nil, err
	}

	schemaFunc := &gppSidIn{
		gppSids: make(map[int8]struct{}),
	}

	for i := 0; i < len(gppSids); i++ {
		schemaFunc.gppSids[gppSids[i]] = struct{}{}
	}

	return schemaFunc, nil
}

func (sid *gppSidIn) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	if len(sid.gppSids) == 0 {
		return "false", nil
	}

	if !hasGPPIDs(wrapper) {
		return "false", errors.New("request.regs.gppsid not found")
	}

	for i := 0; i < len(wrapper.Regs.GPPSID); i++ {
		if _, found := sid.gppSids[wrapper.Regs.GPPSID[i]]; found {
			return "true", nil
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
	return &gppSidAvailable{}, checkNilArgs(params, GppSidAvailable)
}

func (sid *gppSidAvailable) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	if hasGPPIDs(wrapper) {
		return "true", nil
	}
	return "false", nil
}

func (sid *gppSidAvailable) Name() string {
	return GppSidAvailable
}

// ------------tcfInScope------------------
type tcfInScope struct {
}

func NewTcfInScope(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	return &tcfInScope{}, checkNilArgs(params, TcfInScope)
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
	value int
}

func NewPercent(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error) {
	percentValue, err := checkArgsInt(params, Percent)
	return &percent{value: percentValue}, err
}

func (p *percent) Call(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	pValue := p.value
	if pValue <= 0 {
		return "false", nil
	}
	if pValue >= 100 {
		return "true", nil
	}
	randNum := randRange(0, 100)
	if randNum < pValue {
		return "true", nil
	}
	return "false", nil
}

func (p *percent) Name() string {
	return Percent
}

// ------------prebidKey------------------
// ------------domain------------------
// ------------bundle--------------------
// ------------bundleIn------------------
// ------------mediaTypes------------------
// ------------adUnitCode------------------
// ------------deviceType------------------
// ------------deviceTypeIn----------------
// ----------helper functions---------
func getRequestRegs(wrapper *openrtb_ext.RequestWrapper) *openrtb2.Regs {
	if wrapper != nil && wrapper.BidRequest != nil && wrapper.Regs != nil {
		return wrapper.Regs
	}
	return nil
}

func hasGPPIDs(wrapper *openrtb_ext.RequestWrapper) bool {
	regs := getRequestRegs(wrapper)
	if regs != nil {
		for i := 0; i < len(regs.GPPSID); i++ {
			if regs.GPPSID[i] > int8(0) {
				return true
			}
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

	_, err := jsonparser.GetString(val, "[0]", "id")

	return err == nil
}

func checkSiteContentDataAndSiteExtData(wrapper *openrtb_ext.RequestWrapper) (string, error) {
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

func checkAppContentDataAndAppExtData(wrapper *openrtb_ext.RequestWrapper) (string, error) {
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

func randRange(min, max int) int {
	return rand.Intn(max-min) + min
}

func checkNilArgs(params json.RawMessage, funcName string) error {
	if params == nil {
		// no params handling
		// { "function": "channel"}
		return nil
	}

	var args [][]interface{}

	if err := jsonutil.Unmarshal(params, &args); err != nil {
		return err
	}
	if len(args) > 0 {
		return fmt.Errorf("%s expects 0 arguments", funcName)
	}
	return nil
}

func checkSingleArgList(params json.RawMessage, funcName string) ([]interface{}, error) {
	var args [][]interface{}

	if err := jsonutil.Unmarshal(params, &args); err != nil {
		return nil, err
	}
	if len(args) != 1 {
		return nil, fmt.Errorf("%s expects one argument", funcName)
	}
	return args[0], nil
}

func checkArgsStringList(params json.RawMessage, funcName string) ([]string, error) {
	args, err := checkSingleArgList(params, funcName)
	if err != nil {
		return nil, err
	}

	values := make([]string, len(args))
	for i, v := range args {
		stringValue, ok := v.(string)
		if !ok {
			return nil, errors.New("error converting value to string")
		}
		values[i] = stringValue
	}

	return values, nil
}

func checkArgsInt8List(params json.RawMessage, funcName string) ([]int8, error) {
	args, err := checkSingleArgList(params, funcName)
	if err != nil {
		return nil, err
	}
	values := make([]int8, len(args))
	for i, v := range args {
		intValue, ok := v.(int8)
		if !ok {
			return nil, errors.New("error converting value to int8")
		}
		values[i] = intValue
	}
	return values, nil
}

func checkArgsInt(params json.RawMessage, funcName string) (int, error) {
	//stub
	return 0, nil
}

func checkArgsString(params json.RawMessage, funcName string) (string, error) {
	//stub
	return "", nil
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
