package openrtb_ext

import (
	"encoding/json"
	"fmt"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/util/maputil"
	"github.com/prebid/prebid-server/util/ptrutil"
	"github.com/prebid/prebid-server/util/sliceutil"
)

// FirstPartyDataExtKey defines a field name within request.ext and request.imp.ext reserved for first party data.
const FirstPartyDataExtKey = "data"

// FirstPartyDataContextExtKey defines a field name within request.ext and request.imp.ext reserved for first party data.
const FirstPartyDataContextExtKey = "context"

// SKAdNExtKey defines the field name within request.ext reserved for Apple's SKAdNetwork.
const SKAdNExtKey = "skadn"

// GPIDKey defines the field name within request.ext reserved for the Global Placement ID (GPID),
const GPIDKey = "gpid"

// TIDKey reserved for Per-Impression Transactions IDs for Multi-Impression Bid Requests.
const TIDKey = "tid"

// AuctionEnvironmentKey is the json key under imp[].ext for ExtImp.AuctionEnvironment
const AuctionEnvironmentKey = string(BidderReservedAE)

// NativeExchangeSpecificLowerBound defines the lower threshold of exchange specific types for native ads. There is no upper bound.
const NativeExchangeSpecificLowerBound = 500

const MaxDecimalFigures int = 15

// ExtRequest defines the contract for bidrequest.ext
type ExtRequest struct {
	Prebid ExtRequestPrebid      `json:"prebid"`
	SChain *openrtb2.SupplyChain `json:"schain,omitempty"`
}

// ExtRequestPrebid defines the contract for bidrequest.ext.prebid
type ExtRequestPrebid struct {
	Aliases              map[string]string         `json:"aliases,omitempty"`
	AliasGVLIDs          map[string]uint16         `json:"aliasgvlids,omitempty"`
	BidAdjustmentFactors map[string]float64        `json:"bidadjustmentfactors,omitempty"`
	BidderConfigs        []BidderConfig            `json:"bidderconfig,omitempty"`
	BidderParams         json.RawMessage           `json:"bidderparams,omitempty"`
	Cache                *ExtRequestPrebidCache    `json:"cache,omitempty"`
	Channel              *ExtRequestPrebidChannel  `json:"channel,omitempty"`
	CurrencyConversions  *ExtRequestCurrency       `json:"currency,omitempty"`
	Data                 *ExtRequestPrebidData     `json:"data,omitempty"`
	Debug                bool                      `json:"debug,omitempty"`
	Events               json.RawMessage           `json:"events,omitempty"`
	Experiment           *Experiment               `json:"experiment,omitempty"`
	Integration          string                    `json:"integration,omitempty"`
	MultiBid             []*ExtMultiBid            `json:"multibid,omitempty"`
	Passthrough          json.RawMessage           `json:"passthrough,omitempty"`
	SChains              []*ExtRequestPrebidSChain `json:"schains,omitempty"`
	Server               *ExtRequestPrebidServer   `json:"server,omitempty"`
	StoredRequest        *ExtStoredRequest         `json:"storedrequest,omitempty"`
	SupportDeals         bool                      `json:"supportdeals,omitempty"`
	Targeting            *ExtRequestTargeting      `json:"targeting,omitempty"`

	// NoSale specifies bidders with whom the publisher has a legal relationship where the
	// passing of personally identifiable information doesn't constitute a sale per CCPA law.
	// The array may contain a single sstar ('*') entry to represent all bidders.
	NoSale []string `json:"nosale,omitempty"`

	//AlternateBidderCodes is populated with host's AlternateBidderCodes config if not defined in request
	AlternateBidderCodes *ExtAlternateBidderCodes `json:"alternatebiddercodes,omitempty"`

	Floors      *PriceFloorRules       `json:"floors,omitempty"`
	MultiBidMap map[string]ExtMultiBid `json:"-"`
	// Trace controls the level of detail in the output information returned from executing hooks.
	// There are two options:
	// - verbose: sets maximum level of output information
	// - basic: excludes debugmessages and analytic_tags from output
	// any other value or an empty string disables trace output at all.
	Trace string `json:"trace,omitempty"`

	BidAdjustments    *ExtRequestPrebidBidAdjustments `json:"bidadjustments,omitempty"`
	AdServerTargeting []AdServerTarget                `json:"adservertargeting,omitempty"`
}

type AdServerTarget struct {
	Key    string `json:"key,omitempty"`
	Source string `json:"source,omitempty"`
	Value  string `json:"value,omitempty"`
}

// Experiment defines if experimental features are available for the request
type Experiment struct {
	AdsCert *AdsCert `json:"adscert,omitempty"`
}

// AdsCert defines if Call Sign feature is enabled for request
type AdsCert struct {
	Enabled bool `json:"enabled,omitempty"`
}

type BidderConfig struct {
	Bidders []string `json:"bidders,omitempty"`
	Config  *Config  `json:"config,omitempty"`
}

type Config struct {
	ORTB2 *ORTB2 `json:"ortb2,omitempty"`
}

type ORTB2 struct { //First party data
	Site map[string]json.RawMessage `json:"site,omitempty"`
	App  map[string]json.RawMessage `json:"app,omitempty"`
	User map[string]json.RawMessage `json:"user,omitempty"`
}

type ExtRequestCurrency struct {
	ConversionRates map[string]map[string]float64 `json:"rates"`
	UsePBSRates     *bool                         `json:"usepbsrates"`
}

// ExtRequestPrebid defines the contract for bidrequest.ext.prebid.schains
type ExtRequestPrebidSChain struct {
	Bidders []string             `json:"bidders,omitempty"`
	SChain  openrtb2.SupplyChain `json:"schain"`
}

// ExtRequestPrebidChannel defines the contract for bidrequest.ext.prebid.channel
type ExtRequestPrebidChannel struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ExtRequestPrebidCache defines the contract for bidrequest.ext.prebid.cache
type ExtRequestPrebidCache struct {
	Bids    *ExtRequestPrebidCacheBids `json:"bids,omitempty"`
	VastXML *ExtRequestPrebidCacheVAST `json:"vastxml,omitempty"`
}

type ExtRequestPrebidServer struct {
	ExternalUrl string `json:"externalurl"`
	GvlID       int    `json:"gvlid"`
	DataCenter  string `json:"datacenter"`
}

// ExtRequestPrebidCacheBids defines the contract for bidrequest.ext.prebid.cache.bids
type ExtRequestPrebidCacheBids struct {
	ReturnCreative *bool `json:"returnCreative,omitempty"`
}

// ExtRequestPrebidCacheVAST defines the contract for bidrequest.ext.prebid.cache.vastxml
type ExtRequestPrebidCacheVAST struct {
	ReturnCreative *bool `json:"returnCreative,omitempty"`
}

// ExtRequestPrebidBidAdjustments defines the contract for bidrequest.ext.prebid.bidadjustments
type ExtRequestPrebidBidAdjustments struct {
	MediaType MediaType `json:"mediatype,omitempty"`
}

// AdjustmentsByDealID maps a dealID to a slice of bid adjustments
type AdjustmentsByDealID map[string][]Adjustment

// MediaType defines contract for bidrequest.ext.prebid.bidadjustments.mediatype
// BidderName will map to a DealID that will map to a slice of bid adjustments
type MediaType struct {
	Banner         map[BidderName]AdjustmentsByDealID `json:"banner,omitempty"`
	VideoInstream  map[BidderName]AdjustmentsByDealID `json:"video-instream,omitempty"`
	VideoOutstream map[BidderName]AdjustmentsByDealID `json:"video-outstream,omitempty"`
	Audio          map[BidderName]AdjustmentsByDealID `json:"audio,omitempty"`
	Native         map[BidderName]AdjustmentsByDealID `json:"native,omitempty"`
	WildCard       map[BidderName]AdjustmentsByDealID `json:"*,omitempty"`
}

// Adjustment defines the object that will be present in the slice of bid adjustments found from MediaType map
type Adjustment struct {
	Type     string  `json:"adjtype,omitempty"`
	Value    float64 `json:"value,omitempty"`
	Currency string  `json:"currency,omitempty"`
}

// ExtRequestTargeting defines the contract for bidrequest.ext.prebid.targeting
type ExtRequestTargeting struct {
	PriceGranularity     *PriceGranularity        `json:"pricegranularity,omitempty"`
	IncludeWinners       *bool                    `json:"includewinners,omitempty"`
	IncludeBidderKeys    *bool                    `json:"includebidderkeys,omitempty"`
	IncludeBrandCategory *ExtIncludeBrandCategory `json:"includebrandcategory,omitempty"`
	IncludeFormat        bool                     `json:"includeformat,omitempty"`
	DurationRangeSec     []int                    `json:"durationrangesec,omitempty"`
	PreferDeals          bool                     `json:"preferdeals,omitempty"`
	AppendBidderNames    bool                     `json:"appendbiddernames,omitempty"`
}

type ExtIncludeBrandCategory struct {
	PrimaryAdServer     int    `json:"primaryadserver"`
	Publisher           string `json:"publisher"`
	WithCategory        bool   `json:"withcategory"`
	TranslateCategories *bool  `json:"translatecategories,omitempty"`
}

// PriceGranularity defines the allowed values for bidrequest.ext.prebid.targeting.pricegranularity
type PriceGranularity struct {
	Precision *int               `json:"precision,omitempty"`
	Ranges    []GranularityRange `json:"ranges,omitempty"`
}

type PriceGranularityRaw PriceGranularity

// GranularityRange struct defines a range of prices used by PriceGranularity
type GranularityRange struct {
	Min       float64 `json:"min"`
	Max       float64 `json:"max"`
	Increment float64 `json:"increment"`
}

func (pg *PriceGranularity) UnmarshalJSON(b []byte) error {
	// price granularity used to be a string referencing a predefined value, try to parse
	// and map the legacy string before falling back to the modern custom model.
	legacyID := ""
	if err := json.Unmarshal(b, &legacyID); err == nil {
		if legacyValue, ok := NewPriceGranularityFromLegacyID(legacyID); ok {
			*pg = legacyValue
			return nil
		}
	}

	// use a type-alias to avoid calling back into this UnmarshalJSON implementation
	modernValue := PriceGranularityRaw{}
	err := json.Unmarshal(b, &modernValue)
	if err == nil {
		*pg = (PriceGranularity)(modernValue)
	}
	return err
}

func NewPriceGranularityDefault() PriceGranularity {
	pg, _ := NewPriceGranularityFromLegacyID("medium")
	return pg
}

// NewPriceGranularityFromLegacyID converts a legacy string into the new PriceGranularity structure.
func NewPriceGranularityFromLegacyID(v string) (PriceGranularity, bool) {
	precision2 := 2

	switch v {
	case "low":
		return PriceGranularity{
			Precision: &precision2,
			Ranges: []GranularityRange{{
				Min:       0,
				Max:       5,
				Increment: 0.5}},
		}, true

	case "med", "medium":
		return PriceGranularity{
			Precision: &precision2,
			Ranges: []GranularityRange{{
				Min:       0,
				Max:       20,
				Increment: 0.1}},
		}, true

	case "high":
		return PriceGranularity{
			Precision: &precision2,
			Ranges: []GranularityRange{{
				Min:       0,
				Max:       20,
				Increment: 0.01}},
		}, true

	case "auto":
		return PriceGranularity{
			Precision: &precision2,
			Ranges: []GranularityRange{
				{
					Min:       0,
					Max:       5,
					Increment: 0.05,
				},
				{
					Min:       5,
					Max:       10,
					Increment: 0.1,
				},
				{
					Min:       10,
					Max:       20,
					Increment: 0.5,
				},
			},
		}, true

	case "dense":
		return PriceGranularity{
			Precision: &precision2,
			Ranges: []GranularityRange{
				{
					Min:       0,
					Max:       3,
					Increment: 0.01,
				},
				{
					Min:       3,
					Max:       8,
					Increment: 0.05,
				},
				{
					Min:       8,
					Max:       20,
					Increment: 0.5,
				},
			},
		}, true
	}

	return PriceGranularity{}, false
}

// ExtRequestPrebidData defines Prebid's First Party Data (FPD) and related bid request options.
type ExtRequestPrebidData struct {
	EidPermissions []ExtRequestPrebidDataEidPermission `json:"eidpermissions"`
	Bidders        []string                            `json:"bidders,omitempty"`
}

// ExtRequestPrebidDataEidPermission defines a filter rule for filter user.ext.eids
type ExtRequestPrebidDataEidPermission struct {
	Source  string   `json:"source"`
	Bidders []string `json:"bidders"`
}

type ExtMultiBid struct {
	Bidder                 string   `json:"bidder,omitempty"`
	Bidders                []string `json:"bidders,omitempty"`
	MaxBids                *int     `json:"maxbids,omitempty"`
	TargetBidderCodePrefix string   `json:"targetbiddercodeprefix,omitempty"`
}

func (m ExtMultiBid) String() string {
	maxBid := "<nil>"
	if m.MaxBids != nil {
		maxBid = fmt.Sprintf("%d", *m.MaxBids)
	}
	return fmt.Sprintf("{Bidder:%s, Bidders:%v, MaxBids:%s, TargetBidderCodePrefix:%s}", m.Bidder, m.Bidders, maxBid, m.TargetBidderCodePrefix)
}

func (erp *ExtRequestPrebid) Clone() *ExtRequestPrebid {
	if erp == nil {
		return nil
	}

	clone := *erp
	clone.Aliases = maputil.Clone(erp.Aliases)
	clone.AliasGVLIDs = maputil.Clone(erp.AliasGVLIDs)
	clone.BidAdjustmentFactors = maputil.Clone(erp.BidAdjustmentFactors)

	if erp.BidderConfigs != nil {
		clone.BidderConfigs = make([]BidderConfig, len(erp.BidderConfigs))
		for i, bc := range erp.BidderConfigs {
			clonedBidderConfig := BidderConfig{Bidders: sliceutil.Clone(bc.Bidders)}
			if bc.Config != nil {
				config := &Config{}
				if bc.Config.ORTB2 != nil {
					config.ORTB2 = &ORTB2{
						Site: maputil.Clone(bc.Config.ORTB2.Site),
						App:  maputil.Clone(bc.Config.ORTB2.App),
						User: maputil.Clone(bc.Config.ORTB2.User),
					}
				}
				clonedBidderConfig.Config = config
			}
			clone.BidderConfigs[i] = clonedBidderConfig
		}
	}

	if erp.Cache != nil {
		clone.Cache = &ExtRequestPrebidCache{}
		if erp.Cache.Bids != nil {
			clone.Cache.Bids = &ExtRequestPrebidCacheBids{}
			clone.Cache.Bids.ReturnCreative = ptrutil.Clone(erp.Cache.Bids.ReturnCreative)
		}
		if erp.Cache.VastXML != nil {
			clone.Cache.VastXML = &ExtRequestPrebidCacheVAST{}
			clone.Cache.VastXML.ReturnCreative = ptrutil.Clone(erp.Cache.VastXML.ReturnCreative)
		}
	}

	if erp.Channel != nil {
		channel := *erp.Channel
		clone.Channel = &channel
	}

	if erp.CurrencyConversions != nil {
		newConvRates := make(map[string]map[string]float64, len(erp.CurrencyConversions.ConversionRates))
		for key, val := range erp.CurrencyConversions.ConversionRates {
			newConvRates[key] = maputil.Clone(val)
		}
		clone.CurrencyConversions = &ExtRequestCurrency{ConversionRates: newConvRates}
		if erp.CurrencyConversions.UsePBSRates != nil {
			clone.CurrencyConversions.UsePBSRates = ptrutil.ToPtr(*erp.CurrencyConversions.UsePBSRates)
		}
	}

	if erp.Data != nil {
		clone.Data = &ExtRequestPrebidData{Bidders: sliceutil.Clone(erp.Data.Bidders)}
		if erp.Data.EidPermissions != nil {
			newEidPermissions := make([]ExtRequestPrebidDataEidPermission, len(erp.Data.EidPermissions))
			for i, eidp := range erp.Data.EidPermissions {
				newEidPermissions[i] = ExtRequestPrebidDataEidPermission{
					Source:  eidp.Source,
					Bidders: sliceutil.Clone(eidp.Bidders),
				}
			}
			clone.Data.EidPermissions = newEidPermissions
		}
	}

	if erp.Experiment != nil {
		clone.Experiment = &Experiment{}
		if erp.Experiment.AdsCert != nil {
			clone.Experiment.AdsCert = ptrutil.ToPtr(*erp.Experiment.AdsCert)
		}
	}

	if erp.MultiBid != nil {
		clone.MultiBid = make([]*ExtMultiBid, len(erp.MultiBid))
		for i, mulBid := range erp.MultiBid {
			newMulBid := &ExtMultiBid{
				Bidder:                 mulBid.Bidder,
				Bidders:                sliceutil.Clone(mulBid.Bidders),
				TargetBidderCodePrefix: mulBid.TargetBidderCodePrefix,
			}
			if mulBid.MaxBids != nil {
				newMulBid.MaxBids = ptrutil.ToPtr(*mulBid.MaxBids)
			}
			clone.MultiBid[i] = newMulBid
		}
	}

	if erp.SChains != nil {
		clone.SChains = make([]*ExtRequestPrebidSChain, len(erp.SChains))
		for i, schain := range erp.SChains {
			newChain := *schain
			newNodes := sliceutil.Clone(schain.SChain.Nodes)
			for j, node := range newNodes {
				if node.HP != nil {
					newNodes[j].HP = ptrutil.ToPtr(*newNodes[j].HP)
				}
			}
			newChain.SChain.Nodes = newNodes
			clone.SChains[i] = &newChain
		}
	}

	clone.Server = ptrutil.Clone(erp.Server)

	clone.StoredRequest = ptrutil.Clone(erp.StoredRequest)

	if erp.Targeting != nil {
		newTargeting := &ExtRequestTargeting{
			IncludeFormat:     erp.Targeting.IncludeFormat,
			DurationRangeSec:  sliceutil.Clone(erp.Targeting.DurationRangeSec),
			PreferDeals:       erp.Targeting.PreferDeals,
			AppendBidderNames: erp.Targeting.AppendBidderNames,
		}
		if erp.Targeting.PriceGranularity != nil {
			newPriceGranularity := &PriceGranularity{
				Ranges: sliceutil.Clone(erp.Targeting.PriceGranularity.Ranges),
			}
			newPriceGranularity.Precision = ptrutil.Clone(erp.Targeting.PriceGranularity.Precision)
			newTargeting.PriceGranularity = newPriceGranularity
		}
		newTargeting.IncludeWinners = ptrutil.Clone(erp.Targeting.IncludeWinners)
		newTargeting.IncludeBidderKeys = ptrutil.Clone(erp.Targeting.IncludeBidderKeys)
		if erp.Targeting.IncludeBrandCategory != nil {
			newIncludeBrandCategory := *erp.Targeting.IncludeBrandCategory
			newIncludeBrandCategory.TranslateCategories = ptrutil.Clone(erp.Targeting.IncludeBrandCategory.TranslateCategories)
			newTargeting.IncludeBrandCategory = &newIncludeBrandCategory
		}
		clone.Targeting = newTargeting
	}

	clone.NoSale = sliceutil.Clone(erp.NoSale)

	if erp.AlternateBidderCodes != nil {
		newAlternateBidderCodes := ExtAlternateBidderCodes{Enabled: erp.AlternateBidderCodes.Enabled}
		if erp.AlternateBidderCodes.Bidders != nil {
			newBidders := make(map[string]ExtAdapterAlternateBidderCodes, len(erp.AlternateBidderCodes.Bidders))
			for key, val := range erp.AlternateBidderCodes.Bidders {
				newBidders[key] = ExtAdapterAlternateBidderCodes{
					Enabled:            val.Enabled,
					AllowedBidderCodes: sliceutil.Clone(val.AllowedBidderCodes),
				}
			}
			newAlternateBidderCodes.Bidders = newBidders
		}
		clone.AlternateBidderCodes = &newAlternateBidderCodes
	}

	if erp.Floors != nil {
		clonedFloors := *erp.Floors
		clonedFloors.Location = ptrutil.Clone(erp.Floors.Location)
		if erp.Floors.Data != nil {
			clonedData := *erp.Floors.Data
			if erp.Floors.Data.ModelGroups != nil {
				clonedData.ModelGroups = make([]PriceFloorModelGroup, len(erp.Floors.Data.ModelGroups))
				for i, pfmg := range erp.Floors.Data.ModelGroups {
					clonedData.ModelGroups[i] = pfmg
					clonedData.ModelGroups[i].ModelWeight = ptrutil.Clone(pfmg.ModelWeight)
					clonedData.ModelGroups[i].Schema.Fields = sliceutil.Clone(pfmg.Schema.Fields)
					clonedData.ModelGroups[i].Values = maputil.Clone(pfmg.Values)
				}
			}
			clonedFloors.Data = &clonedData
		}
		if erp.Floors.Enforcement != nil {
			clonedFloors.Enforcement = &PriceFloorEnforcement{
				EnforceJS:     ptrutil.Clone(erp.Floors.Enforcement.EnforceJS),
				EnforcePBS:    ptrutil.Clone(erp.Floors.Enforcement.EnforcePBS),
				FloorDeals:    ptrutil.Clone(erp.Floors.Enforcement.FloorDeals),
				BidAdjustment: ptrutil.Clone(erp.Floors.Enforcement.BidAdjustment),
				EnforceRate:   erp.Floors.Enforcement.EnforceRate,
			}
		}
		clonedFloors.Enabled = ptrutil.Clone(erp.Floors.Enabled)
		clonedFloors.Skipped = ptrutil.Clone(erp.Floors.Skipped)
		clone.Floors = &clonedFloors
	}
	if erp.MultiBidMap != nil {
		clone.MultiBidMap = make(map[string]ExtMultiBid, len(erp.MultiBidMap))
		for k, v := range erp.MultiBidMap {
			// Make v a deep copy of the ExtMultiBid struct
			v.Bidders = sliceutil.Clone(v.Bidders)
			v.MaxBids = ptrutil.Clone(v.MaxBids)
			clone.MultiBidMap[k] = v
		}
	}
	clone.AdServerTargeting = sliceutil.Clone(erp.AdServerTargeting)

	return &clone
}
