package ortb

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

func TestValidateVideo(t *testing.T) {
	tests := []struct {
		name      string
		video     *openrtb2.Video
		wantError bool
	}{
		{
			name:      "nil",
			video:     nil,
			wantError: false,
		},
		{
			name: "well_formed",
			video: &openrtb2.Video{
				MIMEs:      []string{"MIME1"},
				W:          ptrutil.ToPtr[int64](0),
				H:          ptrutil.ToPtr[int64](0),
				MinBitRate: 0,
				MaxBitRate: 0,
			},
			wantError: false,
		},
		{
			name: "well_formed_with_nil_dims",
			video: &openrtb2.Video{
				MIMEs:      []string{"MIME1"},
				W:          nil,
				H:          nil,
				MinBitRate: 0,
				MaxBitRate: 0,
			},
			wantError: false,
		},
		{
			name: "mimes_is_zero",
			video: &openrtb2.Video{
				MIMEs:      []string{},
				W:          ptrutil.ToPtr[int64](0),
				H:          ptrutil.ToPtr[int64](0),
				MinBitRate: 0,
				MaxBitRate: 0,
			},
			wantError: true,
		},
		{
			name: "negative_width",
			video: &openrtb2.Video{
				MIMEs:      []string{"MIME1"},
				W:          ptrutil.ToPtr[int64](-1),
				H:          ptrutil.ToPtr[int64](0),
				MinBitRate: 0,
				MaxBitRate: 0,
			},
			wantError: true,
		},
		{
			name: "negative_height",
			video: &openrtb2.Video{
				MIMEs:      []string{"MIME1"},
				W:          ptrutil.ToPtr[int64](0),
				H:          ptrutil.ToPtr[int64](-1),
				MinBitRate: 0,
				MaxBitRate: 0,
			},
			wantError: true,
		},
		{
			name: "negative_min_bit_rate",
			video: &openrtb2.Video{
				MIMEs:      []string{"MIME1"},
				W:          ptrutil.ToPtr[int64](0),
				H:          ptrutil.ToPtr[int64](0),
				MinBitRate: -1,
				MaxBitRate: 0,
			},
			wantError: true,
		},
		{
			name: "negative_max_bit_rate",
			video: &openrtb2.Video{
				MIMEs:      []string{"MIME1"},
				W:          ptrutil.ToPtr[int64](0),
				H:          ptrutil.ToPtr[int64](0),
				MinBitRate: 0,
				MaxBitRate: -1,
			},
			wantError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := validateVideo(test.video, 1)
			if test.wantError {
				assert.Error(t, result)
			} else {
				assert.NoError(t, result)
			}
		})
	}
}
