package openrtb_ext

import (
	"fmt"

	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type ImpExtGoldbach struct {
	PublisherID     string                                  `json:"publisherId"`
	SlotID          string                                  `json:"slotId"`
	CustomTargeting map[string]ImpExtGoldbachTargetingValue `json:"customTargeting,omitempty"`
}

// custom targeting values can be a single string or an array of strings
type ImpExtGoldbachTargetingValue []string

func (t *ImpExtGoldbachTargetingValue) UnmarshalJSON(b []byte) error {
	// try to unmarshal as a single string first
	var stringResult string
	if err := jsonutil.UnmarshalValid(b, &stringResult); err == nil {
		*t = []string{stringResult}
		return nil
	}

	// then try to unmarshal as an array of strings
	var arrayResult []string
	if err := jsonutil.UnmarshalValid(b, &arrayResult); err == nil {
		*t = arrayResult
		return nil
	}

	// if both attempts fail, return an error
	return fmt.Errorf("targeting values must be a string or an array of strings, got %s", b)
}
