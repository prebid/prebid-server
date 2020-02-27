package openrtb_ext

// ExtImpZeroclickfraud defines the contract for bidrequest.imp[i].ext.datablocks
type ExtImpZeroclickfraud struct {
	SourceId int    `json:"sourceId"`
	Host     string `json:"host"`
}
