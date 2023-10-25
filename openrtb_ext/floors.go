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

// GetEnforcePBS will check if floors enforcement is enabled in request
func (Floors *PriceFloorRules) GetEnforcePBS() bool {
	if Floors != nil && Floors.Enforcement != nil && Floors.Enforcement.EnforcePBS != nil {
		return *Floors.Enforcement.EnforcePBS
	}
	return true
}

// GetFloorsSkippedFlag will return  floors skipped flag
func (Floors *PriceFloorRules) GetFloorsSkippedFlag() bool {
	if Floors != nil && Floors.Skipped != nil {
		return *Floors.Skipped
	}
	return false
}

// GetEnforceRate will return enforcement rate in request
func (Floors *PriceFloorRules) GetEnforceRate() int {
	if Floors != nil && Floors.Enforcement != nil {
		return Floors.Enforcement.EnforceRate
	}
	return 0
}

// GetEnforceDealsFlag will return FloorDeals flag in request
func (Floors *PriceFloorRules) GetEnforceDealsFlag() bool {
	if Floors != nil && Floors.Enforcement != nil && Floors.Enforcement.FloorDeals != nil {
		return *Floors.Enforcement.FloorDeals
	}
	return false
}

// GetEnabled will check if floors is enabled in request
func (Floors *PriceFloorRules) GetEnabled() bool {
	if Floors != nil && Floors.Enabled != nil {
		return *Floors.Enabled
	}
	return true
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

func (mg PriceFloorModelGroup) Copy() PriceFloorModelGroup {
	newMg := new(PriceFloorModelGroup)
	newMg.Currency = mg.Currency
	newMg.ModelVersion = mg.ModelVersion
	newMg.SkipRate = mg.SkipRate
	newMg.Default = mg.Default
	if mg.ModelWeight != nil {
		newMg.ModelWeight = new(int)
		*newMg.ModelWeight = *mg.ModelWeight
	}

	newMg.Schema.Delimiter = mg.Schema.Delimiter
	newMg.Schema.Fields = make([]string, len(mg.Schema.Fields))
	copy(newMg.Schema.Fields, mg.Schema.Fields)
	newMg.Values = make(map[string]float64, len(mg.Values))
	for key, val := range mg.Values {
		newMg.Values[key] = val
	}
	return *newMg
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
