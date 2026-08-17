package sdkutils

import (
	"reflect"
	"testing"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/openrtb/v20/openrtb3"
	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

func TestMergeDevice(t *testing.T) {
	lmt0 := int8(0)
	lmt1 := int8(1)
	js1 := int8(1)
	ct5 := adcom1.ConnectionType(5)
	lat := float64(40.74)
	lon := float64(-73.93)

	tests := []struct {
		name     string
		dst      *openrtb2.Device
		src      *openrtb2.Device
		expected *openrtb2.Device
	}{
		{
			name:     "both_nil",
			dst:      nil,
			src:      nil,
			expected: nil,
		},
		{
			name: "src_nil_returns_dst_unchanged",
			dst: &openrtb2.Device{
				UA: "existing_ua",
			},
			src: nil,
			expected: &openrtb2.Device{
				UA: "existing_ua",
			},
		},
		{
			name: "dst_nil_src_non-nil_allocates_new_device",
			dst:  nil,
			src: &openrtb2.Device{
				UA:   "signal_ua",
				Make: "Google",
			},
			expected: &openrtb2.Device{
				UA:   "signal_ua",
				Make: "Google",
			},
		},
		{
			name: "src_fields_overwrite_empty_dst_fields",
			dst:  &openrtb2.Device{},
			src: &openrtb2.Device{
				UA:             "signal_ua",
				Make:           "Google",
				Model:          "Pixel",
				IP:             "1.2.3.4",
				IPv6:           "::1",
				DeviceType:     adcom1.DeviceType(4),
				IFA:            "test-ifa",
				HWV:            "ruby",
				OS:             "Android",
				OSV:            "13",
				W:              1080,
				H:              1920,
				PxRatio:        2.75,
				Language:       "en",
				Carrier:        "T-Mobile",
				MCCMNC:         "310-260",
				JS:             &js1,
				Lmt:            &lmt1,
				ConnectionType: &ct5,
			},
			expected: &openrtb2.Device{
				UA:             "signal_ua",
				Make:           "Google",
				Model:          "Pixel",
				IP:             "1.2.3.4",
				IPv6:           "::1",
				DeviceType:     adcom1.DeviceType(4),
				IFA:            "test-ifa",
				HWV:            "ruby",
				OS:             "Android",
				OSV:            "13",
				W:              1080,
				H:              1920,
				PxRatio:        2.75,
				Language:       "en",
				Carrier:        "T-Mobile",
				MCCMNC:         "310-260",
				JS:             &js1,
				Lmt:            &lmt1,
				ConnectionType: &ct5,
			},
		},
		{
			name: "dst_fields_preserved_when_src_fields_are_zero/empty",
			dst: &openrtb2.Device{
				UA:   "existing_ua",
				Make: "Apple",
				OS:   "iOS",
			},
			src: &openrtb2.Device{
				UA:   "",
				Make: "",
				OS:   "",
			},
			expected: &openrtb2.Device{
				UA:   "existing_ua",
				Make: "Apple",
				OS:   "iOS",
			},
		},
		{
			name: "partial_merge:_only_non-zero_src_fields_overwrite_dst",
			dst: &openrtb2.Device{
				UA:   "existing_ua",
				Make: "Apple",
				OS:   "iOS",
				OSV:  "16",
			},
			src: &openrtb2.Device{
				Make: "Google",
				OS:   "Android",
			},
			expected: &openrtb2.Device{
				UA:   "existing_ua",
				Make: "Google",
				OS:   "Android",
				OSV:  "16",
			},
		},
		{
			name: "lmt_zero_value_in_src_does_not_overwrite_dst_lmt",
			dst: &openrtb2.Device{
				Lmt: &lmt1,
			},
			src: &openrtb2.Device{
				Lmt: &lmt0,
			},
			expected: &openrtb2.Device{
				Lmt: &lmt0,
			},
		},
		{
			name: "geo_nil_in_src_leaves_dst_geo_unchanged",
			dst: &openrtb2.Device{
				Geo: &openrtb2.Geo{Country: "USA"},
			},
			src: &openrtb2.Device{
				UA:  "signal_ua",
				Geo: nil,
			},
			expected: &openrtb2.Device{
				UA:  "signal_ua",
				Geo: &openrtb2.Geo{Country: "USA"},
			},
		},
		{
			name: "geo_nil_in_dst_gets_allocated_from_src",
			dst:  &openrtb2.Device{},
			src: &openrtb2.Device{
				Geo: &openrtb2.Geo{
					Country: "USA",
					Region:  "ny",
					City:    "Queens",
					ZIP:     "11101",
					Lat:     &lat,
					Lon:     &lon,
				},
			},
			expected: &openrtb2.Device{
				Geo: &openrtb2.Geo{
					Country: "USA",
					Region:  "ny",
					City:    "Queens",
					ZIP:     "11101",
					Lat:     &lat,
					Lon:     &lon,
				},
			},
		},
		{
			name: "geo_lat/lon_preserved_in_dst_when_already_set",
			dst: &openrtb2.Device{
				Geo: &openrtb2.Geo{
					Lat: &lat,
					Lon: &lon,
				},
			},
			src: &openrtb2.Device{
				Geo: &openrtb2.Geo{
					Lat:     func() *float64 { v := 0.0; return &v }(),
					Lon:     func() *float64 { v := 0.0; return &v }(),
					Country: "USA",
				},
			},
			expected: &openrtb2.Device{
				Geo: &openrtb2.Geo{
					Lat:     &lat,
					Lon:     &lon,
					Country: "USA",
				},
			},
		},
		{
			name: "geo_partial_merge:_only_non-empty_src_geo_fields_overwrite_dst",
			dst: &openrtb2.Device{
				Geo: &openrtb2.Geo{
					Country: "IND",
					City:    "Mumbai",
				},
			},
			src: &openrtb2.Device{
				Geo: &openrtb2.Geo{
					Country: "USA",
					Region:  "ny",
					City:    "",
				},
			},
			expected: &openrtb2.Device{
				Geo: &openrtb2.Geo{
					Country: "USA",
					Region:  "ny",
					City:    "Mumbai",
				},
			},
		},
		{
			name:     "signal_ppi_copied",
			dst:      &openrtb2.Device{},
			src:      &openrtb2.Device{PPI: 440},
			expected: &openrtb2.Device{PPI: 440},
		},
		{
			name:     "signal_ppi_overwrites_dst_ppi",
			dst:      &openrtb2.Device{PPI: 320},
			src:      &openrtb2.Device{PPI: 440},
			expected: &openrtb2.Device{PPI: 440},
		},
		{
			name:     "zero_signal_ppi_does_not_overwrite_dst_ppi",
			dst:      &openrtb2.Device{PPI: 320},
			src:      &openrtb2.Device{},
			expected: &openrtb2.Device{PPI: 320},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeDevice(tt.dst, tt.src)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMergeBanner(t *testing.T) {
	tests := []struct {
		name     string
		dst      *openrtb2.Banner
		src      *openrtb2.Banner
		expected *openrtb2.Banner
	}{
		{
			name:     "nil_dst",
			dst:      nil,
			src:      &openrtb2.Banner{MIMEs: []string{"image/jpeg"}},
			expected: nil,
		},
		{
			name:     "nil_src",
			dst:      &openrtb2.Banner{},
			src:      nil,
			expected: &openrtb2.Banner{},
		},
		{
			name:     "signal_mimes_copied",
			dst:      &openrtb2.Banner{},
			src:      &openrtb2.Banner{MIMEs: []string{"image/jpeg", "image/png"}},
			expected: &openrtb2.Banner{MIMEs: []string{"image/jpeg", "image/png"}},
		},
		{
			name:     "signal_mimes_overwrite_dst_mimes",
			dst:      &openrtb2.Banner{MIMEs: []string{"image/gif"}},
			src:      &openrtb2.Banner{MIMEs: []string{"image/jpeg"}},
			expected: &openrtb2.Banner{MIMEs: []string{"image/jpeg"}},
		},
		{
			name:     "empty_signal_mimes_do_not_overwrite_dst",
			dst:      &openrtb2.Banner{MIMEs: []string{"image/gif"}},
			src:      &openrtb2.Banner{},
			expected: &openrtb2.Banner{MIMEs: []string{"image/gif"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			MergeBanner(tt.dst, tt.src)
			assert.Equal(t, tt.expected, tt.dst)
		})
	}
}

func TestCopyPath(t *testing.T) {
	tests := []struct {
		name      string
		source    []byte
		target    []byte
		path      []string
		expected  []byte
		expectErr bool
	}{
		{
			name:      "Nil source",
			source:    nil,
			target:    []byte(`{"key":"value"}`),
			path:      []string{"key"},
			expected:  []byte(`{"key":"value"}`),
			expectErr: false,
		},
		{
			name:      "Nil target",
			source:    []byte(`{"key":"value"}`),
			target:    nil,
			path:      []string{"key"},
			expected:  []byte(`{"key":"value"}`),
			expectErr: false,
		},
		{
			name:      "Copy string value",
			source:    []byte(`{"key":"value"}`),
			target:    []byte(`{"other_key":"other_value"}`),
			path:      []string{"key"},
			expected:  []byte(`{"other_key":"other_value","key":"value"}`),
			expectErr: false,
		},
		{
			name:      "Copy number value",
			source:    []byte(`{"key":123}`),
			target:    []byte(`{}`),
			path:      []string{"key"},
			expected:  []byte(`{"key":123}`),
			expectErr: false,
		},
		{
			name:      "Copy boolean value",
			source:    []byte(`{"key":true}`),
			target:    []byte(`{}`),
			path:      []string{"key"},
			expected:  []byte(`{"key":true}`),
			expectErr: false,
		},
		{
			name:      "Skip empty string",
			source:    []byte(`{"key":""}`),
			target:    []byte(`{}`),
			path:      []string{"key"},
			expected:  []byte(`{}`),
			expectErr: false,
		},
		{
			name:      "Skip empty array",
			source:    []byte(`{"key":[]}`),
			target:    []byte(`{}`),
			path:      []string{"key"},
			expected:  []byte(`{}`),
			expectErr: false,
		},
		{
			name:      "Skip empty object",
			source:    []byte(`{"key":{}}`),
			target:    []byte(`{}`),
			path:      []string{"key"},
			expected:  []byte(`{}`),
			expectErr: false,
		},
		{
			name:      "Copy non-empty array",
			source:    []byte(`{"key":[1,2,3]}`),
			target:    []byte(`{}`),
			path:      []string{"key"},
			expected:  []byte(`{"key":[1,2,3]}`),
			expectErr: false,
		},
		{
			name:      "Copy non-empty object",
			source:    []byte(`{"key":{"nested":"value"}}`),
			target:    []byte(`{}`),
			path:      []string{"key"},
			expected:  []byte(`{"key":{"nested":"value"}}`),
			expectErr: false,
		},
		{
			name:      "Invalid path",
			source:    []byte(`{"key":"value"}`),
			target:    []byte(`{}`),
			path:      []string{"invalid"},
			expected:  []byte(`{}`),
			expectErr: true,
		},
		{
			name:      "Empty value in source but valid value in target",
			source:    []byte(`{"key":""}`),
			target:    []byte(`{"key":"existing"}`),
			path:      []string{"key"},
			expected:  []byte(`{"key":"existing"}`),
			expectErr: false,
		},
		{
			name:      "Empty value in source but valid object in target",
			source:    []byte(`{"key":{}}`),
			target:    []byte(`{"key":{"nested":{"nested_key":"nested_value"}}}`),
			path:      []string{"key"},
			expected:  []byte(`{"key":{"nested":{"nested_key":"nested_value"}}}`),
			expectErr: false,
		},
		{
			name:      "Invalid path with target non empty",
			source:    []byte(`{"key":"value"}`),
			target:    []byte(`{"key":"existing"}`),
			path:      []string{"invalid"},
			expected:  []byte(`{"key":"existing"}`),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CopyPath(tt.source, tt.target, tt.path...)
			if tt.expectErr {
				assert.Error(t, err)
			}
			if !reflect.DeepEqual(tt.expected, result) {
				t.Errorf("Expected %v, but got %v", tt.expected, result)
			}
		})
	}
}

func TestSetIfKeysExists(t *testing.T) {
	type args struct {
		source []byte
		target []byte
		keys   []string
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "keys not found in source",
			args: args{
				source: nil,
				target: nil,
				keys:   []string{"key1", "key2"},
			},
			want: nil,
		},
		{
			name: "int value key found out of all keys",
			args: args{
				source: []byte(`{"key1":23,"key40":"v40"}`),
				target: nil,
				keys:   []string{"key1", "key2"},
			},
			want: []byte(`{"key1":23}`),
		},
		{
			name: "copies numeric zero",
			args: args{
				source: []byte(`{"impdepth":0,"lastadomain":"example.com"}`),
				target: nil,
				keys:   []string{"impdepth", "lastadomain"},
			},
			want: []byte(`{"impdepth":0,"lastadomain":"example.com"}`),
		},
		{
			name: "string value key found out of all keys",
			args: args{
				source: []byte(`{"key1":23,"key40":"v40"}`),
				target: nil,
				keys:   []string{"key40", "key2"},
			},
			want: []byte(`{"key40":"v40"}`),
		},
		{
			name: "overwrite string value key in target",
			args: args{
				source: []byte(`{"key1":55555,"key40":"v40"}`),
				target: []byte(`{"key1":23,"key40":"will_overwrite"}`),
				keys:   []string{"key40", "key2"},
			},
			want: []byte(`{"key1":23,"key40":"v40"}`),
		},
		{
			name: "error while setting key, return oldTarget",
			args: args{
				source: []byte(`{"key1":555555,"key40":"v40"}`),
				target: []byte(`"key1":23,"key40":"value40"}`),
				keys:   []string{"key40", "key2"},
			},
			want: []byte(`"key1":23,"key40":"value40"}`),
		},
		{
			name: "overwrite key in target with object",
			args: args{
				source: []byte(`{"key1":55555,"key40":{"user":{"id":"1kjh3429kjh295jkl","ext":{"consent":"CONSENT_STRING"}},"regs":{"ext":{"gdpr":1}}}}`),
				target: []byte(`{"key1":23,"key40":[]}`),
				keys:   []string{"key40", "key2"},
			},
			want: []byte(`{"key1":23,"key40":{"user":{"id":"1kjh3429kjh295jkl","ext":{"consent":"CONSENT_STRING"}},"regs":{"ext":{"gdpr":1}}}}`),
		},
		{
			name: "set slice in key",
			args: args{
				source: []byte(`{"key1":555555,"key40":[1,2,3,4,5]}`),
				target: []byte(`{"key1":23,"key40":"value40"}`),
				keys:   []string{"key40", "key2"},
			},
			want: []byte(`{"key1":23,"key40":[1,2,3,4,5]}`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SetIfKeysExists(tt.args.source, tt.args.target, tt.args.keys...)
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}

func TestAddSize300x600ForInterstitialBanner(t *testing.T) {
	tests := []struct {
		name     string
		imp      openrtb2.Imp
		expected openrtb2.Imp
	}{
		{
			name: "nil Banner",
			imp: openrtb2.Imp{
				ID: "test_imp",
			},
			expected: openrtb2.Imp{
				ID: "test_imp",
			},
		},
		{
			name: "Banner with W/H fields set to 320x480 (phone portrait), no 300x600",
			imp: openrtb2.Imp{
				ID: "test_imp",
				Banner: &openrtb2.Banner{
					W: ptrutil.ToPtr[int64](320),
					H: ptrutil.ToPtr[int64](480),
				},
			},
			expected: openrtb2.Imp{
				ID: "test_imp",
				Banner: &openrtb2.Banner{
					W: ptrutil.ToPtr[int64](320),
					H: ptrutil.ToPtr[int64](480),
					Format: []openrtb2.Format{
						{W: 300, H: 600},
					},
				},
			},
		},
		{
			name: "Banner with W/H fields set to 768x1024 (tablet portrait), no 300x600",
			imp: openrtb2.Imp{
				ID: "test_imp",
				Banner: &openrtb2.Banner{
					W: ptrutil.ToPtr[int64](768),
					H: ptrutil.ToPtr[int64](1024),
				},
			},
			expected: openrtb2.Imp{
				ID: "test_imp",
				Banner: &openrtb2.Banner{
					W: ptrutil.ToPtr[int64](768),
					H: ptrutil.ToPtr[int64](1024),
					Format: []openrtb2.Format{
						{W: 300, H: 600},
					},
				},
			},
		},
		{
			name: "Banner with W/H fields set to 1024x768 (tablet landscape), no 300x600",
			imp: openrtb2.Imp{
				ID: "test_imp",
				Banner: &openrtb2.Banner{
					W: ptrutil.ToPtr[int64](1024),
					H: ptrutil.ToPtr[int64](768),
				},
			},
			expected: openrtb2.Imp{
				ID: "test_imp",
				Banner: &openrtb2.Banner{
					W: ptrutil.ToPtr[int64](1024),
					H: ptrutil.ToPtr[int64](768),
					Format: []openrtb2.Format{
						{W: 300, H: 600},
					},
				},
			},
		},
		{
			name: "Banner with W/H fields set to 300x600 already",
			imp: openrtb2.Imp{
				ID: "test_imp",
				Banner: &openrtb2.Banner{
					W: ptrutil.ToPtr[int64](300),
					H: ptrutil.ToPtr[int64](600),
				},
			},
			expected: openrtb2.Imp{
				ID: "test_imp",
				Banner: &openrtb2.Banner{
					W: ptrutil.ToPtr[int64](300),
					H: ptrutil.ToPtr[int64](600),
				},
			},
		},
		{
			name: "Banner with Format containing 320x480 (phone portrait), no 300x600",
			imp: openrtb2.Imp{
				ID: "test_imp",
				Banner: &openrtb2.Banner{
					Format: []openrtb2.Format{
						{W: 320, H: 480},
						{W: 320, H: 50},
					},
				},
			},
			expected: openrtb2.Imp{
				ID: "test_imp",
				Banner: &openrtb2.Banner{
					Format: []openrtb2.Format{
						{W: 320, H: 480},
						{W: 320, H: 50},
						{W: 300, H: 600},
					},
				},
			},
		},
		{
			name: "Banner with Format containing 768x1024 (tablet portrait), no 300x600",
			imp: openrtb2.Imp{
				ID: "test_imp",
				Banner: &openrtb2.Banner{
					Format: []openrtb2.Format{
						{W: 768, H: 1024},
						{W: 320, H: 50},
					},
				},
			},
			expected: openrtb2.Imp{
				ID: "test_imp",
				Banner: &openrtb2.Banner{
					Format: []openrtb2.Format{
						{W: 768, H: 1024},
						{W: 320, H: 50},
						{W: 300, H: 600},
					},
				},
			},
		},
		{
			name: "Banner with Format containing 1024x768 (tablet landscape), no 300x600",
			imp: openrtb2.Imp{
				ID: "test_imp",
				Banner: &openrtb2.Banner{
					Format: []openrtb2.Format{
						{W: 1024, H: 768},
						{W: 320, H: 50},
					},
				},
			},
			expected: openrtb2.Imp{
				ID: "test_imp",
				Banner: &openrtb2.Banner{
					Format: []openrtb2.Format{
						{W: 1024, H: 768},
						{W: 320, H: 50},
						{W: 300, H: 600},
					},
				},
			},
		},
		{
			name: "Banner with Format containing both 320x480 and 300x600",
			imp: openrtb2.Imp{
				ID: "test_imp",
				Banner: &openrtb2.Banner{
					Format: []openrtb2.Format{
						{W: 320, H: 480},
						{W: 300, H: 600},
						{W: 320, H: 50},
					},
				},
			},
			expected: openrtb2.Imp{
				ID: "test_imp",
				Banner: &openrtb2.Banner{
					Format: []openrtb2.Format{
						{W: 320, H: 480},
						{W: 300, H: 600},
						{W: 320, H: 50},
					},
				},
			},
		},
		{
			name: "Banner with neither supported sizes nor 300x600",
			imp: openrtb2.Imp{
				ID: "test_imp",
				Banner: &openrtb2.Banner{
					Format: []openrtb2.Format{
						{W: 320, H: 50},
						{W: 728, H: 90},
					},
				},
			},
			expected: openrtb2.Imp{
				ID: "test_imp",
				Banner: &openrtb2.Banner{
					Format: []openrtb2.Format{
						{W: 320, H: 50},
						{W: 728, H: 90},
					},
				},
			},
		},
		{
			name: "Banner with W/H set to different size and Format containing 320x480",
			imp: openrtb2.Imp{
				ID: "test_imp",
				Banner: &openrtb2.Banner{
					W: ptrutil.ToPtr[int64](728),
					H: ptrutil.ToPtr[int64](90),
					Format: []openrtb2.Format{
						{W: 320, H: 480},
					},
				},
			},
			expected: openrtb2.Imp{
				ID: "test_imp",
				Banner: &openrtb2.Banner{
					W: ptrutil.ToPtr[int64](728),
					H: ptrutil.ToPtr[int64](90),
					Format: []openrtb2.Format{
						{W: 320, H: 480},
						{W: 300, H: 600},
					},
				},
			},
		},
		{
			name: "Banner with nil W/H but Format containing 320x480",
			imp: openrtb2.Imp{
				ID: "test_imp",
				Banner: &openrtb2.Banner{
					Format: []openrtb2.Format{
						{W: 320, H: 480},
					},
				},
			},
			expected: openrtb2.Imp{
				ID: "test_imp",
				Banner: &openrtb2.Banner{
					Format: []openrtb2.Format{
						{W: 320, H: 480},
						{W: 300, H: 600},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AddSize300x600ForInterstitialBanner(&tt.imp)
			assert.Equal(t, tt.expected, tt.imp, "Banner formats should match expected")
		})
	}
}

func TestIsGoogleSDKResponseRejected(t *testing.T) {
	tests := []struct {
		name   string
		rCtx   *models.RequestCtx
		ao     analytics.AuctionObject
		expect bool
	}{
		{
			name: "Both responses and request context are nil",
			ao: analytics.AuctionObject{
				HookExecutionOutcome: nil,
			},
			expect: false,
		},
		{
			name: "Response is nil but request context is not",
			rCtx: &models.RequestCtx{
				Endpoint: models.EndpointGoogleSDK,
			},
			ao: analytics.AuctionObject{
				HookExecutionOutcome: nil,
			},
			expect: false,
		},
		{
			name: "response with nbr",
			ao: analytics.AuctionObject{
				Response: &openrtb2.BidResponse{
					NBR: openrtb3.NoBidReason(6).Ptr(),
				},
			},
			rCtx: &models.RequestCtx{
				Endpoint: models.EndpointGoogleSDK,
			},
			expect: true,
		},
		{
			name: "Request context is not Google SDK endpoint",
			rCtx: &models.RequestCtx{
				Endpoint: models.EndpointAppLovinMax,
			},
			ao: analytics.AuctionObject{
				Response: &openrtb2.BidResponse{},
			},
			expect: false,
		},
		{
			name: "Reject is false and NBR is nil",
			rCtx: &models.RequestCtx{
				Endpoint: models.EndpointGoogleSDK,
			},
			ao: analytics.AuctionObject{
				Response: &openrtb2.BidResponse{
					NBR: nil,
				},
			},
			expect: false,
		},
		{
			name: "Reject is true",
			rCtx: &models.RequestCtx{
				Endpoint: models.EndpointGoogleSDK,
				GoogleSDK: models.GoogleSDK{
					Reject: true,
				},
			},
			ao: analytics.AuctionObject{
				Response: &openrtb2.BidResponse{},
			},
			expect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := IsGoogleSDKResponseRejected(tt.rCtx, tt.ao)
			assert.Equal(t, tt.expect, actual, "Expected reject status should match actual reject status")
		})
	}
}

func TestIsSdkBiddingEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		expected bool
	}{
		// SDK Integration endpoints - should return true
		{
			name:     "AppLovinMax endpoint should be SDK integration",
			endpoint: models.EndpointAppLovinMax,
			expected: true,
		},
		{
			name:     "UnityLevelPlay endpoint should be SDK integration",
			endpoint: models.EndpointUnityLevelPlay,
			expected: true,
		},
		{
			name:     "GoogleSDK endpoint should be SDK integration",
			endpoint: models.EndpointGoogleSDK,
			expected: true,
		},
		{
			name:     "APS endpoint should be SDK integration",
			endpoint: models.EndpointAPS,
			expected: true,
		},

		// Non-SDK endpoints - should return false
		{
			name:     "WebS2S endpoint should not be SDK integration",
			endpoint: models.EndpointWebS2S,
			expected: false,
		},
		{
			name:     "Empty string should not be SDK integration",
			endpoint: "",
			expected: false,
		},
		{
			name:     "Unknown endpoint should not be SDK integration",
			endpoint: "unknown-endpoint",
			expected: false,
		},
		{
			name:     "Case-sensitive endpoint check - lowercase APS should match",
			endpoint: "aps",
			expected: true,
		},
		{
			name:     "Case-sensitive endpoint check - mixed case should not match",
			endpoint: "AppLovinMax",
			expected: false,
		},
		{
			name:     "Partial endpoint match should not be considered SDK integration",
			endpoint: models.EndpointAppLovinMax + "-extra",
			expected: false,
		},
		{
			name:     "Endpoint with whitespace should not be SDK integration",
			endpoint: " " + models.EndpointGoogleSDK + " ",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSdkBiddingEndpoint(tt.endpoint)
			assert.Equal(t, tt.expected, result, "IsSdkBiddingEndpoint(%q) should return %v", tt.endpoint, tt.expected)
		})
	}
}

func TestIsSdkEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		expected bool
	}{
		{name: "v25 endpoint", endpoint: models.EndpointV25, expected: true},
		{name: "google sdk endpoint", endpoint: models.EndpointGoogleSDK, expected: true},
		{name: "applovin max endpoint", endpoint: models.EndpointAppLovinMax, expected: true},
		{name: "amp endpoint", endpoint: models.EndpointAMP, expected: false},
		{name: "web s2s endpoint", endpoint: models.EndpointWebS2S, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsSdkEndpoint(tt.endpoint))
		})
	}
}
