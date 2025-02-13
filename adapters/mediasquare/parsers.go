package mediasquare

import (
	"fmt"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

// parserDSA: Struct used to extracts dsa content of a jsonutil.
type parserDSA struct {
	DSA interface{} `json:"dsa,omitempty"`
}

// setContent: Unmarshal a []byte into the parserDSA struct.
func (parser *parserDSA) setContent(extJsonBytes []byte) error {
	if len(extJsonBytes) > 0 {
		if err := jsonutil.Unmarshal(extJsonBytes, parser); err != nil {
			return errorWriter("<setContent(*parserDSA)> extJsonBytes", err, false)
		}
		return nil
	}
	return errorWriter("<setContent(*parserDSA)> extJsonBytes", nil, true)
}

// getValue: Returns the DSA value as a string, defaultly returns empty-string.
func (parser parserDSA) getValue(request *openrtb2.BidRequest) (dsa string) {
	if request == nil || request.Regs == nil {
		return
	}
	parser.setContent(request.Regs.Ext)
	if parser.DSA != nil {
		dsa = fmt.Sprint(parser.DSA)
	}
	return
}

// parserGDPR: Struct used to extract pair of GDPR/Consent of a jsonutil.
type parserGDPR struct {
	GDPR    interface{} `json:"gdpr,omitempty"`
	Consent interface{} `json:"consent,omitempty"`
}

// setContent: Unmarshal a []byte into the parserGDPR struct.
func (parser *parserGDPR) setContent(extJsonBytes []byte) error {
	if len(extJsonBytes) > 0 {
		if err := jsonutil.Unmarshal(extJsonBytes, parser); err != nil {
			return errorWriter("<setContent(*parserGDPR)> extJsonBytes", err, false)
		}
		return nil
	}
	return errorWriter("<setContent(*parserGDPR)> extJsonBytes", nil, true)
}

// value: Returns the consent or GDPR-string depending of the parserGDPR content, defaulty return empty-string.
func (parser *parserGDPR) value() (gdpr string) {
	switch {
	case parser.Consent != nil:
		gdpr = fmt.Sprint(parser.Consent)
	case parser.GDPR != nil:
		gdpr = fmt.Sprint(parser.GDPR)
	}
	return
}

// getValue: Returns the consent or GDPR-string depending on the openrtb2.User content, defaultly returns empty-string.
func (parser parserGDPR) getValue(field string, request *openrtb2.BidRequest) (gdpr string) {
	if request != nil {
		switch {
		case field == "consent_requirement" && request.Regs != nil:
			gdpr = "false"
			if ptrInt8ToBool(request.Regs.GDPR) {
				gdpr = "true"
			}
		case field == "consent_string" && request.User != nil:
			gdpr = request.User.Consent
			if len(gdpr) <= 0 {
				parser.setContent(request.User.Ext)
				gdpr = parser.value()
			}
		}
	}
	return
}
