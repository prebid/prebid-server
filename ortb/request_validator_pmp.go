package ortb

import (
	"fmt"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/errortypes"
)

func validatePmp(pmp *openrtb2.PMP, impIndex int) error {
	if pmp == nil {
		return nil
	}

	for dealIndex, deal := range pmp.Deals {
		if deal.ID == "" {
			return &errortypes.BadInput{Message: fmt.Sprintf("request.imp[%d].pmp.deals[%d] missing required field: \"id\"", impIndex, dealIndex)}
		}
	}
	return nil
}
