package ortb2blocking

import (
	"encoding/json"
	"fmt"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

func newConfig(data json.RawMessage) (config, error) {
	var cfg config
	if err := jsonutil.UnmarshalValid(data, &cfg); err != nil {
		return cfg, fmt.Errorf("failed to parse config: %s", err)
	}
	return cfg, nil
}

type config struct {
	Attributes Attributes `json:"attributes"`
}

type Attributes struct {
	Badv  Badv  `json:"badv"`
	Bcat  Bcat  `json:"bcat"`
	Bapp  Bapp  `json:"bapp"`
	Btype Btype `json:"btype"`
	Battr Battr `json:"battr"`
}

type Badv struct {
	ActionOverrides        BadvActionOverride `json:"action_overrides"`
	AllowedAdomainForDeals []string           `json:"allowed_adomain_for_deals"`
	BlockedAdomain         []string           `json:"blocked_adomain"`
	BlockUnknownAdomain    bool               `json:"block_unknown_adomain"`
	EnforceBlocks          bool               `json:"enforce_blocks"`
}

type BadvActionOverride struct {
	AllowedAdomainForDeals []ActionOverride `json:"allowed_adomain_for_deals"`
	BlockedAdomain         []ActionOverride `json:"blocked_adomain"`
	BlockUnknownAdomain    []ActionOverride `json:"block_unknown_adomain"`
	EnforceBlocks          []ActionOverride `json:"enforce_blocks"`
}

type Bcat struct {
	ActionOverrides       BcatActionOverride      `json:"action_overrides"`
	AllowedAdvCatForDeals []string                `json:"allowed_adv_cat_for_deals"`
	BlockedAdvCat         []string                `json:"blocked_adv_cat"`
	BlockUnknownAdvCat    bool                    `json:"block_unknown_adv_cat"`
	CategoryTaxonomy      adcom1.CategoryTaxonomy `json:"category_taxonomy"`
	EnforceBlocks         bool                    `json:"enforce_blocks"`
}

type BcatActionOverride struct {
	AllowedAdvCatForDeals []ActionOverride `json:"allowed_adv_cat_for_deals"`
	BlockedAdvCat         []ActionOverride `json:"blocked_adv_cat"`
	BlockUnknownAdvCat    []ActionOverride `json:"block_unknown_adv_cat"`
	EnforceBlocks         []ActionOverride `json:"enforce_blocks"`
}

type Bapp struct {
	ActionOverrides    BappActionOverride `json:"action_overrides"`
	AllowedAppForDeals []string           `json:"allowed_app_for_deals"`
	BlockedApp         []string           `json:"blocked_app"`
	EnforceBlocks      bool               `json:"enforce_blocks"`
}

type BappActionOverride struct {
	AllowedAppForDeals []ActionOverride `json:"allowed_app_for_deals"`
	BlockedApp         []ActionOverride `json:"blocked_app"`
	EnforceBlocks      []ActionOverride `json:"enforce_blocks"`
}

type Btype struct {
	ActionOverrides   BtypeActionOverride `json:"action_overrides"`
	BlockedBannerType []int               `json:"blocked_banner_type"`
}

type BtypeActionOverride struct {
	BlockedBannerType []ActionOverride `json:"blocked_banner_type"`
}

type Battr struct {
	ActionOverrides           BattrActionOverride `json:"action_overrides"`
	AllowedBannerAttrForDeals []int               `json:"allowed_banner_attr_for_deals"`
	BlockedBannerAttr         []int               `json:"blocked_banner_attr"`
	EnforceBlocks             bool                `json:"enforce_blocks"`
}

type BattrActionOverride struct {
	AllowedBannerAttrForDeals []ActionOverride `json:"allowed_banner_attr_for_deals"`
	BlockedBannerAttr         []ActionOverride `json:"blocked_banner_attr"`
	EnforceBlocks             []ActionOverride `json:"enforce_blocks"`
}

type ActionOverride struct {
	Conditions Conditions `json:"conditions"`
	Override   Override   `json:"override"`
}

type Conditions struct {
	Bidders    []string `json:"bidders"`
	MediaTypes []string `json:"media_types"`
	DealIds    []string `json:"deal_ids"`
}

type Override struct {
	IsActive bool
	Ids      []int
	Names    []string
}

func (o *Override) UnmarshalJSON(bytes []byte) error {
	var overrideData interface{}
	if err := jsonutil.UnmarshalValid(bytes, &overrideData); err != nil {
		return err
	}

	switch overrideValue := overrideData.(type) {
	case bool:
		o.IsActive = overrideValue
	case []interface{}:
		for _, value := range overrideValue {
			switch override := value.(type) {
			case string:
				o.Names = append(o.Names, override)
			case float64:
				o.Ids = append(o.Ids, int(override))
			}
		}
	}

	return nil
}
