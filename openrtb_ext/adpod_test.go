package openrtb_ext

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVideoAdPod_Validate(t *testing.T) {
	type fields struct {
		MinAds                      *int
		MaxAds                      *int
		MinDuration                 *int
		MaxDuration                 *int
		AdvertiserExclusionPercent  *int
		IABCategoryExclusionPercent *int
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr []error
	}{
		{
			name: "ErrInvalidMinAds",
			fields: fields{
				MinAds: getIntPtr(-1),
			},
			wantErr: []error{errInvalidMinAds},
		},
		{
			name: "ZeroMinAds",
			fields: fields{
				MinAds: getIntPtr(0),
			},
			wantErr: []error{errInvalidMinAds},
		},
		{
			name: "ErrInvalidMaxAds",
			fields: fields{
				MaxAds: getIntPtr(-1),
			},
			wantErr: []error{errInvalidMaxAds},
		},
		{
			name: "ZeroMaxAds",
			fields: fields{
				MaxAds: getIntPtr(0),
			},
			wantErr: []error{errInvalidMaxAds},
		},
		{
			name: "ErrInvalidMinDuration",
			fields: fields{
				MinDuration: getIntPtr(-1),
			},
			wantErr: []error{errInvalidMinDuration},
		},
		{
			name: "ZeroMinDuration",
			fields: fields{
				MinDuration: getIntPtr(0),
			},
			wantErr: []error{errInvalidMinDuration},
		},
		{
			name: "ErrInvalidMaxDuration",
			fields: fields{
				MaxDuration: getIntPtr(-1),
			},
			wantErr: []error{errInvalidMaxDuration},
		},
		{
			name: "ZeroMaxDuration",
			fields: fields{
				MaxDuration: getIntPtr(0),
			},
			wantErr: []error{errInvalidMaxDuration},
		},
		{
			name: "ErrInvalidAdvertiserExclusionPercent_NegativeValue",
			fields: fields{
				AdvertiserExclusionPercent: getIntPtr(-1),
			},
			wantErr: []error{errInvalidAdvertiserExclusionPercent},
		},
		{
			name: "ErrInvalidAdvertiserExclusionPercent_InvalidRange",
			fields: fields{
				AdvertiserExclusionPercent: getIntPtr(-1),
			},
			wantErr: []error{errInvalidAdvertiserExclusionPercent},
		},
		{
			name: "ErrInvalidIABCategoryExclusionPercent_Negative",
			fields: fields{
				IABCategoryExclusionPercent: getIntPtr(-1),
			},
			wantErr: []error{errInvalidIABCategoryExclusionPercent},
		},
		{
			name: "ErrInvalidIABCategoryExclusionPercent_InvalidRange",
			fields: fields{
				IABCategoryExclusionPercent: getIntPtr(101),
			},
			wantErr: []error{errInvalidIABCategoryExclusionPercent},
		},
		{
			name: "ErrInvalidMinMaxAds",
			fields: fields{
				MinAds: getIntPtr(5),
				MaxAds: getIntPtr(2),
			},
			wantErr: []error{errInvalidMinMaxAds},
		},
		{
			name: "ErrInvalidMinMaxDuration",
			fields: fields{
				MinDuration: getIntPtr(5),
				MaxDuration: getIntPtr(2),
			},
			wantErr: []error{errInvalidMinMaxDuration},
		},
		{
			name: "Valid",
			fields: fields{
				MinAds:                      getIntPtr(3),
				MaxAds:                      getIntPtr(4),
				MinDuration:                 getIntPtr(20),
				MaxDuration:                 getIntPtr(30),
				AdvertiserExclusionPercent:  getIntPtr(100),
				IABCategoryExclusionPercent: getIntPtr(100),
			},
			wantErr: nil,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := &VideoAdPod{
				MinAds:                      tt.fields.MinAds,
				MaxAds:                      tt.fields.MaxAds,
				MinDuration:                 tt.fields.MinDuration,
				MaxDuration:                 tt.fields.MaxDuration,
				AdvertiserExclusionPercent:  tt.fields.AdvertiserExclusionPercent,
				IABCategoryExclusionPercent: tt.fields.IABCategoryExclusionPercent,
			}

			actualErr := pod.Validate()
			assert.Equal(t, tt.wantErr, actualErr)
		})
	}
}

func TestExtRequestAdPod_Validate(t *testing.T) {
	type fields struct {
		VideoAdPod                          VideoAdPod
		CrossPodAdvertiserExclusionPercent  *int
		CrossPodIABCategoryExclusionPercent *int
		IABCategoryExclusionWindow          *int
		AdvertiserExclusionWindow           *int
		VideoLengthMatching                 string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr []error
	}{
		{
			name: "ErrInvalidCrossPodAdvertiserExclusionPercent_Negative",
			fields: fields{
				CrossPodAdvertiserExclusionPercent: getIntPtr(-1),
			},
			wantErr: []error{errInvalidCrossPodAdvertiserExclusionPercent},
		},
		{
			name: "ErrInvalidCrossPodAdvertiserExclusionPercent_InvalidRange",
			fields: fields{
				CrossPodAdvertiserExclusionPercent: getIntPtr(101),
			},
			wantErr: []error{errInvalidCrossPodAdvertiserExclusionPercent},
		},
		{
			name: "ErrInvalidCrossPodIABCategoryExclusionPercent_Negative",
			fields: fields{
				CrossPodIABCategoryExclusionPercent: getIntPtr(-1),
			},
			wantErr: []error{errInvalidCrossPodIABCategoryExclusionPercent},
		},
		{
			name: "ErrInvalidCrossPodIABCategoryExclusionPercent_InvalidRange",
			fields: fields{
				CrossPodIABCategoryExclusionPercent: getIntPtr(101),
			},
			wantErr: []error{errInvalidCrossPodIABCategoryExclusionPercent},
		},
		{
			name: "ErrInvalidIABCategoryExclusionWindow",
			fields: fields{
				IABCategoryExclusionWindow: getIntPtr(-1),
			},
			wantErr: []error{errInvalidIABCategoryExclusionWindow},
		},
		{
			name: "ErrInvalidAdvertiserExclusionWindow",
			fields: fields{
				AdvertiserExclusionWindow: getIntPtr(-1),
			},
			wantErr: []error{errInvalidAdvertiserExclusionWindow},
		},
		{
			name: "ErrInvalidVideoLengthMatching",
			fields: fields{
				VideoLengthMatching: "invalid",
			},
			wantErr: []error{errInvalidVideoLengthMatching},
		},
		{
			name: "InvalidAdPod",
			fields: fields{
				VideoAdPod: VideoAdPod{
					MinAds: getIntPtr(-1),
				},
			},
			wantErr: []error{getRequestAdPodError(errInvalidMinAds)},
		},
		{
			name: "Valid",
			fields: fields{
				CrossPodAdvertiserExclusionPercent:  getIntPtr(100),
				CrossPodIABCategoryExclusionPercent: getIntPtr(0),
				IABCategoryExclusionWindow:          getIntPtr(10),
				AdvertiserExclusionWindow:           getIntPtr(10),
				VideoAdPod: VideoAdPod{
					MinAds:                      getIntPtr(3),
					MaxAds:                      getIntPtr(4),
					MinDuration:                 getIntPtr(20),
					MaxDuration:                 getIntPtr(30),
					AdvertiserExclusionPercent:  getIntPtr(100),
					IABCategoryExclusionPercent: getIntPtr(100),
				},
			},
			wantErr: nil,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ext := &ExtRequestAdPod{
				VideoAdPod:                          tt.fields.VideoAdPod,
				CrossPodAdvertiserExclusionPercent:  tt.fields.CrossPodAdvertiserExclusionPercent,
				CrossPodIABCategoryExclusionPercent: tt.fields.CrossPodIABCategoryExclusionPercent,
				IABCategoryExclusionWindow:          tt.fields.IABCategoryExclusionWindow,
				AdvertiserExclusionWindow:           tt.fields.AdvertiserExclusionWindow,
				VideoLengthMatching:                 tt.fields.VideoLengthMatching,
			}
			actualErr := ext.Validate()
			assert.Equal(t, tt.wantErr, actualErr)
		})
	}
}

func TestExtVideoAdPod_Validate(t *testing.T) {
	type fields struct {
		Offset *int
		AdPod  *VideoAdPod
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr []error
	}{
		{
			name: "ErrInvalidAdPodOffset",
			fields: fields{
				Offset: getIntPtr(-1),
			},
			wantErr: []error{errInvalidAdPodOffset},
		},
		{
			name: "InvalidAdPod",
			fields: fields{
				AdPod: &VideoAdPod{
					MinAds: getIntPtr(-1),
				},
			},
			wantErr: []error{getRequestAdPodError(errInvalidMinAds)},
		},
		{
			name: "Valid",
			fields: fields{
				Offset: getIntPtr(10),
				AdPod: &VideoAdPod{
					MinAds:                      getIntPtr(3),
					MaxAds:                      getIntPtr(4),
					MinDuration:                 getIntPtr(20),
					MaxDuration:                 getIntPtr(30),
					AdvertiserExclusionPercent:  getIntPtr(100),
					IABCategoryExclusionPercent: getIntPtr(100),
				},
			},
			wantErr: nil,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ext := &ExtVideoAdPod{
				Offset: tt.fields.Offset,
				AdPod:  tt.fields.AdPod,
			}
			actualErr := ext.Validate()
			assert.Equal(t, tt.wantErr, actualErr)
		})
	}
}
