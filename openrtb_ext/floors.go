package openrtb_ext

// Defines strings for FetchStatus
const (
	FetchSuccess    = "success"
	FetchTimeout    = "timeout"
	FetchError      = "error"
	FetchInprogress = "inprogress"
	FetchNone       = "none"
)

// Defines strings for PriceFloorLocation
const (
	NoDataLocation  = "noData"
	RequestLocation = "request"
	FetchLocation   = "fetch"
)

// PriceFloorRules defines the contract for bidrequest.ext.prebid.floors
type PriceFloorRules struct {
	FloorMin           float64                `json:"floormin,omitempty"`
	FloorMinCur        string                 `json:"floormincur,omitempty"`
	SkipRate           int                    `json:"skiprate,omitempty"`
	Location           *PriceFloorEndpoint    `json:"floorendpoint,omitempty"`
	Data               *PriceFloorData        `json:"data,omitempty"`
	Enforcement        *PriceFloorEnforcement `json:"enforcement,omitempty"`
	Enabled            *bool                  `json:"enabled,omitempty"`
	Skipped            *bool                  `json:"skipped,omitempty"`
	FloorProvider      string                 `json:"floorprovider,omitempty"`
	FetchStatus        string                 `json:"fetchstatus,omitempty"`
	PriceFloorLocation string                 `json:"location,omitempty"`
}

type PriceFloorEndpoint struct {
	URL string `json:"url,omitempty"`
}

type PriceFloorData struct {
	Currency            string                 `json:"currency,omitempty"`
	SkipRate            int                    `json:"skiprate,omitempty"`
	FloorsSchemaVersion string                 `json:"floorsschemaversion,omitempty"`
	ModelTimestamp      int                    `json:"modeltimestamp,omitempty"`
	ModelGroups         []PriceFloorModelGroup `json:"modelgroups,omitempty"`
	FloorProvider       string                 `json:"floorprovider,omitempty"`
}

type PriceFloorModelGroup struct {
	Currency     string             `json:"currency,omitempty"`
	ModelWeight  *int               `json:"modelweight,omitempty"`
	ModelVersion string             `json:"modelversion,omitempty"`
	SkipRate     int                `json:"skiprate,omitempty"`
	Schema       PriceFloorSchema   `json:"schema,omitempty"`
	Values       map[string]float64 `json:"values,omitempty"`
	Default      float64            `json:"default,omitempty"`
}
type PriceFloorSchema struct {
	Fields    []string `json:"fields,omitempty"`
	Delimiter string   `json:"delimiter,omitempty"`
}

type PriceFloorEnforcement struct {
	EnforceJS     *bool `json:"enforcejs,omitempty"`
	EnforcePBS    *bool `json:"enforcepbs,omitempty"`
	FloorDeals    *bool `json:"floordeals,omitempty"`
	BidAdjustment *bool `json:"bidadjustment,omitempty"`
	EnforceRate   int   `json:"enforcerate,omitempty"`
}

type ImpFloorExt struct {
	FloorRule      string  `json:"floorrule,omitempty"`
	FloorRuleValue float64 `json:"floorrulevalue,omitempty"`
	FloorValue     float64 `json:"floorvalue,omitempty"`
}
type Price struct {
	FloorMin    float64 `json:"floormin,omitempty"`
	FloorMinCur string  `json:"floormincur,omitempty"`
}

type ExtImp struct {
	Prebid *ImpExtPrebid `json:"prebid,omitempty"`
}

type ImpExtPrebid struct {
	Floors Price `json:"floors,omitempty"`
}

// GetEnabled will check if floors is enabled in request
func (Floors *PriceFloorRules) GetEnabled() bool {
	if Floors != nil && Floors.Enabled != nil {
		return *Floors.Enabled
	}
	return true
}

func (modelGroup PriceFloorModelGroup) Copy() PriceFloorModelGroup {
	newModelGroup := new(PriceFloorModelGroup)
	newModelGroup.Currency = modelGroup.Currency
	newModelGroup.ModelVersion = modelGroup.ModelVersion
	newModelGroup.SkipRate = modelGroup.SkipRate
	newModelGroup.Default = modelGroup.Default
	if modelGroup.ModelWeight != nil {
		newModelGroup.ModelWeight = new(int)
		*newModelGroup.ModelWeight = *modelGroup.ModelWeight
	}

	newModelGroup.Schema.Delimiter = modelGroup.Schema.Delimiter
	newModelGroup.Schema.Fields = make([]string, len(modelGroup.Schema.Fields))
	copy(newModelGroup.Schema.Fields, modelGroup.Schema.Fields)
	newModelGroup.Values = make(map[string]float64, len(modelGroup.Values))
	for key, val := range modelGroup.Values {
		newModelGroup.Values[key] = val
	}
	return *newModelGroup
}
