package ortb2blocking

import (
	"encoding/json"
	"fmt"
)

func newConfig(data json.RawMessage) (Config, error) {
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return c, fmt.Errorf("failed to parse config: %s", err)
	}
	return c, nil
}

type Config struct {
	DryRun     bool       `json:"dry_run"`
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
	EnforceBlocks          bool     `json:"enforce_blocks"`
	BlockUnknownAdomain    bool     `json:"block_unknown_adomain"`
	BlockedAdomain         []string `json:"blocked_adomain"`
	AllowedAdomainForDeals []string `json:"allowed_adomain_for_deals"`
	// todo: maybe change all action_overrides to []map[string][]ActionOverride
	ActionOverrides []BadvActionOverride `json:"action_overrides"`
}

type BadvActionOverride struct {
	BlockedAdomain         []ActionOverride `json:"blocked_adomain"`
	BlockUnknownAdomain    []ActionOverride `json:"block_unknown_adomain"`
	AllowedAdomainForDeals []ActionOverride `json:"allowed_adomain_for_deals"`
}

type Bcat struct {
	EnforceBlocks         bool                 `json:"enforce_blocks"`
	BlockUnknownAdvCat    bool                 `json:"block_unknown_adv_cat"`
	CategoryTaxonomy      int                  `json:"category_taxonomy"`
	BlockedAdvCat         []string             `json:"blocked_adv_cat"`
	AllowedAdvCatForDeals []string             `json:"allowed_adv_cat_for_deals"`
	ActionOverrides       []BcatActionOverride `json:"action_overrides"`
}

type BcatActionOverride struct {
	BlockedAdvCat         []ActionOverride `json:"blocked_adv_cat"`
	EnforceBlocks         []ActionOverride `json:"enforce_blocks"`
	BlockUnknownAdvCat    []ActionOverride `json:"block_unknown_adv_cat"`
	AllowedAdvCatForDeals []ActionOverride `json:"allowed_adv_cat_for_deals"`
}

type Bapp struct {
	EnforceBlocks   bool                 `json:"enforce_blocks"`
	BlockedApp      []string             `json:"blocked_app"`
	ActionOverrides []BappActionOverride `json:"action_overrides"`
}

type BappActionOverride struct {
	BlockedApp []ActionOverride `json:"blocked_app"`
}

type Btype struct {
	BlockedBannerType []int                 `json:"blocked_banner_type"`
	ActionOverrides   []BtypeActionOverride `json:"action_overrides"`
}

type BtypeActionOverride struct {
	BlockedBannerType []ActionOverride `json:"blocked_banner_type"`
}

type Battr struct {
	EnforceBlocks     bool                  `json:"enforce_blocks"`
	BlockedBannerAttr []int                 `json:"blocked_banner_attr"`
	ActionOverrides   []BattrActionOverride `json:"action_overrides"`
}

type BattrActionOverride struct {
	EnforceBlocks []ActionOverride `json:"enforce_blocks"`
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
	IsOn  bool
	Ids   []int
	Names []string
}

func (o *Override) UnmarshalJSON(bytes []byte) error {
	var d interface{}
	if err := json.Unmarshal(bytes, &d); err != nil {
		return err
	}

	switch v := d.(type) {
	case []interface{}:
		for _, val := range v {
			switch d := val.(type) {
			case string:
				o.Names = append(o.Names, d)
			case float64:
				o.Ids = append(o.Ids, int(d))
			}
		}
	case []int:
		o.Ids = v
	case bool:
		o.IsOn = v
	}

	return nil
}
