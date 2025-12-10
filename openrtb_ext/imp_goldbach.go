package openrtb_ext

import (
	"fmt"

	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type ImpExtGoldbach struct {
	PublisherID     string                         `json:"publisherId"`
	SlotID          string                         `json:"slotId"`
	CustomTargeting map[string]stringOrStringArray `json:"customTargeting,omitempty"`
}

// stringOrStringArray is a custom type that can hold either a single value of type T or an array of values of type T.
type stringOrStringArray []string

func (t *stringOrStringArray) UnmarshalJSON(b []byte) error {
	// try to unmarshal as a single value of type T first
	var itemResult string
	if err := jsonutil.UnmarshalValid(b, &itemResult); err == nil {
		*t = []string{itemResult}
		return nil
	}

	// then try to unmarshal as an array of type T
	var arrayResult []string
	if err := jsonutil.UnmarshalValid(b, &arrayResult); err == nil {
		*t = arrayResult
		return nil
	}

	// if both attempts fail, return an error
	return fmt.Errorf("value should be of type %T or %T", itemResult, arrayResult)
}
