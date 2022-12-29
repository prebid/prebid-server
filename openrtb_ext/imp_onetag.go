package openrtb_ext

import (
	"encoding/json"
)

type ExtImpOnetag struct {
	PubId string          `json:"pubId"`
	Ext   json.RawMessage `json:"ext"`
}
