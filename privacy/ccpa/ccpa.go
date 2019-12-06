package ccpa

import (
	"encoding/json"

	"github.com/buger/jsonparser"
	"github.com/mxmCherry/openrtb"
)

// Policy represents the CCPA regulation for an OpenRTB bid request.
type Policy struct {
	Signal string
}

// Write mutates an OpenRTB bid request with the context of the CCPA policy.
func (p Policy) Write(req *openrtb.BidRequest) error {
	if p.Signal == "" {
		return nil
	}

	if req.Regs == nil {
		req.Regs = &openrtb.Regs{}
	}

	if req.Regs.Ext == nil {
		req.Regs.Ext = json.RawMessage(`{"us_privacy":"` + p.Signal + `"}`)
		return nil
	}

	var err error
	req.Regs.Ext, err = jsonparser.Set(req.Regs.Ext, []byte(`"`+p.Signal+`"`), "us_privacy")
	return err
}
