package ortb

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"
)

func TestValidateAudio(t *testing.T) {
	tests := []struct {
		name      string
		audio     *openrtb2.Audio
		wantError bool
	}{
		{
			name:      "nil",
			audio:     nil,
			wantError: false,
		},
		{
			name: "well_formed",
			audio: &openrtb2.Audio{
				MIMEs:      []string{"MIME1"},
				Sequence:   0,
				MaxSeq:     0,
				MinBitrate: 0,
				MaxBitrate: 0,
			},
			wantError: false,
		},
		{
			name: "mimes_is_zero",
			audio: &openrtb2.Audio{
				MIMEs:      []string{},
				Sequence:   0,
				MaxSeq:     0,
				MinBitrate: 0,
				MaxBitrate: 0,
			},
			wantError: true,
		},
		{
			name: "negative_sequence",
			audio: &openrtb2.Audio{
				MIMEs:      []string{"MIME1"},
				Sequence:   -1,
				MaxSeq:     0,
				MinBitrate: 0,
				MaxBitrate: 0,
			},
			wantError: true,
		},
		{
			name: "negative_max_sequence",
			audio: &openrtb2.Audio{
				MIMEs:      []string{"MIME1"},
				Sequence:   0,
				MaxSeq:     -1,
				MinBitrate: 0,
				MaxBitrate: 0,
			},
			wantError: true,
		},
		{
			name: "negative_min_bit_rate",
			audio: &openrtb2.Audio{
				MIMEs:      []string{"MIME1"},
				Sequence:   0,
				MaxSeq:     0,
				MinBitrate: -1,
				MaxBitrate: 0,
			},
			wantError: true,
		},
		{
			name: "negative_max_bit_rate",
			audio: &openrtb2.Audio{
				MIMEs:      []string{"MIME1"},
				Sequence:   0,
				MaxSeq:     0,
				MinBitrate: 0,
				MaxBitrate: -1,
			},
			wantError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := validateAudio(test.audio, 1)
			if test.wantError {
				assert.Error(t, result)
			} else {
				assert.NoError(t, result)
			}
		})
	}
}
