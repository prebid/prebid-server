package ccpa

import (
	"encoding/json"

	"github.com/buger/jsonparser"
	"github.com/mxmCherry/openrtb"
)

// Policy represents the CCPA regulation for an OpenRTB bid request.
type Policy struct {
	Value string
}

// Write mutates an OpenRTB bid request with the context of the CCPA policy.
func (p Policy) Write(req *openrtb.BidRequest) error {
	if p.Value == "" {
		return nil
	}

	if req.Regs == nil {
		req.Regs = &openrtb.Regs{}
	}

	if req.Regs.Ext == nil {
		req.Regs.Ext = json.RawMessage(`{"us_privacy":"` + p.Value + `"}`)
		return nil
	}

	var err error
	req.Regs.Ext, err = jsonparser.Set(req.Regs.Ext, []byte(`"`+p.Value+`"`), "us_privacy")
	return err
}
