package openrtb_ext

// ExtImpDatablocks defines the contract for bidrequest.imp[i].ext.datablocks
type ExtImpDatablocks struct {
	SourceId int    `json:"sourceId"`
	Host     string `json:"host"`
}
