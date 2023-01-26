package openrtb_ext

import "encoding/json"

type ImpExtLimelightDigital struct {
	Host        string      `json:"host"`
	PublisherID json.Number `json:"publisherId"`
}
