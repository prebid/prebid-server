package ortb

import (
	"fmt"

	"github.com/prebid/openrtb/v20/openrtb2"
)

func validateAudio(audio *openrtb2.Audio, impIndex int) error {
	if audio == nil {
		return nil
	}

	if len(audio.MIMEs) < 1 {
		return fmt.Errorf("request.imp[%d].audio.mimes must contain at least one supported MIME type", impIndex)
	}

	// The following fields were previously uints in the OpenRTB library we use, but have
	// since been changed to ints. We decided to maintain the non-negative check.
	if audio.Sequence < 0 {
		return fmt.Errorf("request.imp[%d].audio.sequence must be a positive number", impIndex)
	}
	if audio.MaxSeq < 0 {
		return fmt.Errorf("request.imp[%d].audio.maxseq must be a positive number", impIndex)
	}
	if audio.MinBitrate < 0 {
		return fmt.Errorf("request.imp[%d].audio.minbitrate must be a positive number", impIndex)
	}
	if audio.MaxBitrate < 0 {
		return fmt.Errorf("request.imp[%d].audio.maxbitrate must be a positive number", impIndex)
	}

	return nil
}
