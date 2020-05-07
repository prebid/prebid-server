package gdpr

import (
	"encoding/json"

	"github.com/PubMatic-OpenWrap/openrtb"
	"github.com/buger/jsonparser"
)

// Policy represents the GDPR regulation for an OpenRTB bid request.
type Policy struct {
	Signal  string
	Consent string
}

// Write mutates an OpenRTB bid request with the context of the GDPR policy.
func (p Policy) Write(req *openrtb.BidRequest) error {
	if p.Consent == "" {
		return nil
	}

	if req.User == nil {
		req.User = &openrtb.User{}
	}

	if req.User.Ext == nil {
		req.User.Ext = json.RawMessage(`{"consent":"` + p.Consent + `"}`)
		return nil
	}

	var err error
	req.User.Ext, err = jsonparser.Set(req.User.Ext, []byte(`"`+p.Consent+`"`), "consent")
	return err
}
