package taboola

import (
	"encoding/json"
)

type TBLASiteExt struct {
	PageType   string          `json:"pageType,omitempty"`
	RTBSiteExt json.RawMessage `json:"rtbSiteExt,omitempty"`
}
