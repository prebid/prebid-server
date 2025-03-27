package ortb

import (
	"fmt"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func isInterstitial(imp *openrtb_ext.ImpWrapper) bool {
	return imp.Instl == 1
}

func validateBanner(banner *openrtb2.Banner, impIndex int, isInterstitial bool) error {
	if banner == nil {
		return nil
	}

	// The following fields were previously uints in the OpenRTB library we use, but have
	// since been changed to ints. We decided to maintain the non-negative check.
	if banner.W != nil && *banner.W < 0 {
		return fmt.Errorf("request.imp[%d].banner.w must be a positive number", impIndex)
	}
	if banner.H != nil && *banner.H < 0 {
		return fmt.Errorf("request.imp[%d].banner.h must be a positive number", impIndex)
	}

	// The following fields are deprecated in the OpenRTB 2.5 spec but are still present
	// in the OpenRTB library we use. Enforce they are not specified.
	if banner.WMin != 0 {
		return fmt.Errorf("request.imp[%d].banner uses unsupported property: \"wmin\". Use the \"format\" array instead.", impIndex)
	}
	if banner.WMax != 0 {
		return fmt.Errorf("request.imp[%d].banner uses unsupported property: \"wmax\". Use the \"format\" array instead.", impIndex)
	}
	if banner.HMin != 0 {
		return fmt.Errorf("request.imp[%d].banner uses unsupported property: \"hmin\". Use the \"format\" array instead.", impIndex)
	}
	if banner.HMax != 0 {
		return fmt.Errorf("request.imp[%d].banner uses unsupported property: \"hmax\". Use the \"format\" array instead.", impIndex)
	}

	hasRootSize := banner.H != nil && banner.W != nil && *banner.H > 0 && *banner.W > 0
	if !hasRootSize && len(banner.Format) == 0 && !isInterstitial {
		return fmt.Errorf("request.imp[%d].banner has no sizes. Define \"w\" and \"h\", or include \"format\" elements.", impIndex)
	}

	for i, format := range banner.Format {
		if err := validateFormat(&format, impIndex, i); err != nil {
			return err
		}
	}

	return nil
}

func validateFormat(format *openrtb2.Format, impIndex, formatIndex int) error {
	if format == nil {
		return nil
	}
	usesHW := format.W != 0 || format.H != 0
	usesRatios := format.WMin != 0 || format.WRatio != 0 || format.HRatio != 0

	// The following fields were previously uints in the OpenRTB library we use, but have
	// since been changed to ints. We decided to maintain the non-negative check.
	if format.W < 0 {
		return fmt.Errorf("request.imp[%d].banner.format[%d].w must be a positive number", impIndex, formatIndex)
	}
	if format.H < 0 {
		return fmt.Errorf("request.imp[%d].banner.format[%d].h must be a positive number", impIndex, formatIndex)
	}
	if format.WRatio < 0 {
		return fmt.Errorf("request.imp[%d].banner.format[%d].wratio must be a positive number", impIndex, formatIndex)
	}
	if format.HRatio < 0 {
		return fmt.Errorf("request.imp[%d].banner.format[%d].hratio must be a positive number", impIndex, formatIndex)
	}
	if format.WMin < 0 {
		return fmt.Errorf("request.imp[%d].banner.format[%d].wmin must be a positive number", impIndex, formatIndex)
	}

	if usesHW && usesRatios {
		return fmt.Errorf("Request imp[%d].banner.format[%d] should define *either* {w, h} *or* {wmin, wratio, hratio}, but not both. If both are valid, send two \"format\" objects in the request.", impIndex, formatIndex)
	}
	if !usesHW && !usesRatios {
		return fmt.Errorf("Request imp[%d].banner.format[%d] should define *either* {w, h} (for static size requirements) *or* {wmin, wratio, hratio} (for flexible sizes) to be non-zero.", impIndex, formatIndex)
	}
	if usesHW && (format.W == 0 || format.H == 0) {
		return fmt.Errorf("Request imp[%d].banner.format[%d] must define non-zero \"h\" and \"w\" properties.", impIndex, formatIndex)
	}
	if usesRatios && (format.WMin == 0 || format.WRatio == 0 || format.HRatio == 0) {
		return fmt.Errorf("Request imp[%d].banner.format[%d] must define non-zero \"wmin\", \"wratio\", and \"hratio\" properties.", impIndex, formatIndex)
	}
	return nil
}
