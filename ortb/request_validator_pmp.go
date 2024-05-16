package ortb

import (
	"fmt"

	"github.com/prebid/openrtb/v20/openrtb2"
)

func validatePmp(pmp *openrtb2.PMP, impIndex int) error {
	if pmp == nil {
		return nil
	}

	for dealIndex, deal := range pmp.Deals {
		if deal.ID == "" {
			return fmt.Errorf("request.imp[%d].pmp.deals[%d] missing required field: \"id\"", impIndex, dealIndex)
		}
	}
	return nil
}
