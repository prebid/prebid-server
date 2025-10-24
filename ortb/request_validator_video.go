package ortb

import (
	"fmt"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/errortypes"
)

func validateVideo(video *openrtb2.Video, impIndex int) error {
	if video == nil {
		return nil
	}

	if len(video.MIMEs) < 1 {
		return &errortypes.BadInput{Message: fmt.Sprintf("request.imp[%d].video.mimes must contain at least one supported MIME type", impIndex)}
	}

	// The following fields were previously uints in the OpenRTB library we use, but have
	// since been changed to ints. We decided to maintain the non-negative check.
	if video.W != nil && *video.W < 0 {
		return &errortypes.BadInput{Message: fmt.Sprintf("request.imp[%d].video.w must be a positive number", impIndex)}
	}
	if video.H != nil && *video.H < 0 {
		return &errortypes.BadInput{Message: fmt.Sprintf("request.imp[%d].video.h must be a positive number", impIndex)}
	}
	if video.MinBitRate < 0 {
		return &errortypes.BadInput{Message: fmt.Sprintf("request.imp[%d].video.minbitrate must be a positive number", impIndex)}
	}
	if video.MaxBitRate < 0 {
		return &errortypes.BadInput{Message: fmt.Sprintf("request.imp[%d].video.maxbitrate must be a positive number", impIndex)}
	}

	return nil
}
