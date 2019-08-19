package openrtb_ext

import (
	"errors"

	"github.com/buger/jsonparser"
)

// ExtSite defines the contract for bidrequest.site.ext
type ExtSite struct {
	// AMP should be 1 if the request comes from an AMP page, and 0 if not.
	AMP int8 `json:"amp"`
}

func (es *ExtSite) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		return errors.New("request.site.ext must have some data in it")
	}
	if value, dataType, _, _ := jsonparser.Get(b, "amp"); (dataType != jsonparser.NotExist && dataType != jsonparser.Number) || (len(value) != 1) {
		return errors.New(`request.site.ext.amp must be either 1, 0, or undefined`)
	} else {
		switch value[0] {
		case byte(48): // 0
			es.AMP = 0
		case byte(49): // 1
			es.AMP = 1
		default:
			return errors.New(`request.site.ext.amp must be either 1, 0, or undefined`)
		}
	}
	return nil
}
