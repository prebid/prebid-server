package openwrap

import (
	"encoding/json"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	mock_metrics "github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/metrics/mock"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models/nbr"
	mock_feature "github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/publisherfeature/mock"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

func TestUpdateImpression(t *testing.T) {
	type args struct {
		signalImps []openrtb2.Imp
		maxImps    []openrtb2.Imp
	}
	tests := []struct {
		name string
		args args
		want []openrtb2.Imp
	}{
		{
			name: "maxImps nil",
			args: args{
				maxImps: nil,
			},
			want: nil,
		},
		{
			name: "signalImps nil",
			args: args{
				maxImps:    nil,
				signalImps: nil,
			},
			want: nil,
		},
		{
			name: "signalImps with no impressions",
			args: args{
				maxImps:    nil,
				signalImps: []openrtb2.Imp{},
			},
			want: nil,
		},
		{
			name: "maxImps with no impressions",
			args: args{
				maxImps:    []openrtb2.Imp{},
				signalImps: []openrtb2.Imp{},
			},
			want: []openrtb2.Imp{},
		},
		{
			name: "maxImps and signalImps with empty impressions",
			args: args{
				maxImps:    []openrtb2.Imp{{}},
				signalImps: []openrtb2.Imp{{}},
			},
			want: []openrtb2.Imp{{}},
		},
		{
			name: "maxImp video not present",
			args: args{
				signalImps: []openrtb2.Imp{{ClickBrowser: openrtb2.Int8Ptr(0), DisplayManager: "PubMatic_SDK", DisplayManagerVer: "1.2"}},
				maxImps:    []openrtb2.Imp{{ClickBrowser: openrtb2.Int8Ptr(1), DisplayManager: "Applovin_SDK"}},
			},
			want: []openrtb2.Imp{{ClickBrowser: openrtb2.Int8Ptr(0), DisplayManager: "PubMatic_SDK", DisplayManagerVer: "1.2"}},
		},
		{
			name: "only maxImp has video",
			args: args{
				signalImps: []openrtb2.Imp{{ClickBrowser: openrtb2.Int8Ptr(0), DisplayManager: "PubMatic_SDK", DisplayManagerVer: "1.2"}},
				maxImps:    []openrtb2.Imp{{ClickBrowser: openrtb2.Int8Ptr(1), DisplayManager: "Applovin_SDK", Video: &openrtb2.Video{W: openrtb2.Int64Ptr(300), H: openrtb2.Int64Ptr(250)}}},
			},
			want: []openrtb2.Imp{{ClickBrowser: openrtb2.Int8Ptr(0), DisplayManager: "PubMatic_SDK", DisplayManagerVer: "1.2", Video: &openrtb2.Video{W: openrtb2.Int64Ptr(300), H: openrtb2.Int64Ptr(250)}}},
		},
		{
			name: "maxImp and sdkImp has video",
			args: args{
				signalImps: []openrtb2.Imp{{ClickBrowser: openrtb2.Int8Ptr(0), DisplayManager: "PubMatic_SDK", DisplayManagerVer: "1.2", Video: &openrtb2.Video{W: openrtb2.Int64Ptr(300), H: openrtb2.Int64Ptr(250), BAttr: []adcom1.CreativeAttribute{1, 2}}}},
				maxImps:    []openrtb2.Imp{{ClickBrowser: openrtb2.Int8Ptr(1), DisplayManager: "Applovin_SDK", Video: &openrtb2.Video{W: openrtb2.Int64Ptr(750), H: openrtb2.Int64Ptr(500), BAttr: []adcom1.CreativeAttribute{6, 1, 8, 4}}}},
			},
			want: []openrtb2.Imp{{ClickBrowser: openrtb2.Int8Ptr(0), DisplayManager: "PubMatic_SDK", DisplayManagerVer: "1.2", Video: &openrtb2.Video{W: openrtb2.Int64Ptr(300), H: openrtb2.Int64Ptr(250), BAttr: []adcom1.CreativeAttribute{1, 2}}}},
		},
		{
			name: "maxImp has and sdkImp has banner with api",
			args: args{
				signalImps: []openrtb2.Imp{{Banner: &openrtb2.Banner{ID: "sdk_banner", API: []adcom1.APIFramework{1, 2, 3, 4}}}},
				maxImps:    []openrtb2.Imp{{Banner: &openrtb2.Banner{ID: "max_banner"}}},
			},
			want: []openrtb2.Imp{{Banner: &openrtb2.Banner{ID: "max_banner", API: []adcom1.APIFramework{1, 2, 3, 4}}}},
		},
		{
			name: "maxImp has bannertype rewarded",
			args: args{
				signalImps: []openrtb2.Imp{{Banner: &openrtb2.Banner{ID: "sdk_banner", API: []adcom1.APIFramework{1, 2, 3, 4}}}},
				maxImps:    []openrtb2.Imp{{Banner: &openrtb2.Banner{ID: "max_banner", Ext: json.RawMessage(`{"bannertype":"rewarded"}`)}}},
			},
			want: []openrtb2.Imp{{}},
		},
		{
			name: "Banner API not present in signalImp",
			args: args{
				signalImps: []openrtb2.Imp{{Banner: &openrtb2.Banner{ID: "sdk_banner"}}},
				maxImps:    []openrtb2.Imp{{Banner: &openrtb2.Banner{ID: "max_banner", API: []adcom1.APIFramework{1, 2}, Ext: json.RawMessage(`{"bannertype":"not-rewarded"}`)}}},
			},
			want: []openrtb2.Imp{{Banner: &openrtb2.Banner{ID: "max_banner", API: []adcom1.APIFramework{1, 2}, Ext: json.RawMessage(`{"bannertype":"not-rewarded"}`)}}},
		},
		{
			name: "maxImp has no ext, signalImp has reward in ext",
			args: args{
				signalImps: []openrtb2.Imp{{Ext: json.RawMessage(`{"reward":1}`)}},
				maxImps:    []openrtb2.Imp{{}},
			},
			want: []openrtb2.Imp{{Ext: json.RawMessage(`{"reward":1}`)}},
		},
		{
			name: "maxImp has no ext, signalImp has reward,skadn and gpid,owsdk in ext",
			args: args{
				signalImps: []openrtb2.Imp{{Ext: json.RawMessage(`{"owsdk":{"ctaoverlay":1},"reward":1,"skadn":{"versions":["2.0","2.1"],"sourceapp":"11111","skadnetids":["424m5254lk.skadnetwork","4fzdc2evr5.skadnetwork"]},"gpid":"/adunitname/234"}`)}},
				maxImps:    []openrtb2.Imp{{}},
			},
			want: []openrtb2.Imp{{Ext: json.RawMessage(`{"reward":1,"skadn":{"versions":["2.0","2.1"],"sourceapp":"11111","skadnetids":["424m5254lk.skadnetwork","4fzdc2evr5.skadnetwork"]},"gpid":"/adunitname/234","owsdk":{"ctaoverlay":1}}`)}},
		},
		{
			name: "signalImp has Native, maxImp doesn't have Native",
			args: args{
				signalImps: []openrtb2.Imp{{Native: &openrtb2.Native{Request: "native_request_data", Ver: "1.2"}}},
				maxImps:    []openrtb2.Imp{{}},
			},
			want: []openrtb2.Imp{{Native: &openrtb2.Native{Request: "native_request_data", Ver: "1.2"}}},
		},
		{
			name: "signalImp has Native, maxImp also has Native",
			args: args{
				signalImps: []openrtb2.Imp{{Native: &openrtb2.Native{Request: "signal_native_request", Ver: "1.2"}}},
				maxImps:    []openrtb2.Imp{{Native: &openrtb2.Native{Request: "max_native_request", Ver: "1.0"}}},
			},
			want: []openrtb2.Imp{{Native: &openrtb2.Native{Request: "signal_native_request", Ver: "1.2"}}},
		},
		{
			name: "maxImp has Native but signalImp doesn't have Native",
			args: args{
				signalImps: []openrtb2.Imp{{}},
				maxImps:    []openrtb2.Imp{{Native: &openrtb2.Native{Request: "max_native_request", Ver: "1.0"}}},
			},
			want: []openrtb2.Imp{{Native: nil}},
		},
		{
			name: "signalImp_exp_overwrites_maxImp_exp",
			args: args{
				signalImps: []openrtb2.Imp{{Exp: 3600}},
				maxImps:    []openrtb2.Imp{{Exp: 120}},
			},
			want: []openrtb2.Imp{{Exp: 3600}},
		},
		{
			name: "signalImp_exp_zero_keeps_maxImp_exp",
			args: args{
				signalImps: []openrtb2.Imp{{Exp: 0}},
				maxImps:    []openrtb2.Imp{{Exp: 120}},
			},
			want: []openrtb2.Imp{{Exp: 120}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateImpression(tt.args.signalImps, tt.args.maxImps)
			assert.Equal(t, tt.want, tt.args.maxImps, tt.name)
		})
	}
}

func TestUpdateDevice(t *testing.T) {
	type args struct {
		sdkDevice  *openrtb2.Device
		maxRequest *openrtb2.BidRequest
	}
	tests := []struct {
		name string
		args args
		want *openrtb2.Device
	}{
		{
			name: "sdkDevice_nil",
			args: args{
				sdkDevice:  nil,
				maxRequest: &openrtb2.BidRequest{Device: &openrtb2.Device{DeviceType: 5}},
			},
			want: &openrtb2.Device{DeviceType: 5},
		},
		{
			name: "sdkDevice_has_ip",
			args: args{
				sdkDevice:  &openrtb2.Device{IP: "127.0.0.1"},
				maxRequest: &openrtb2.BidRequest{Device: &openrtb2.Device{DeviceType: 5}},
			},
			want: &openrtb2.Device{DeviceType: 5, IP: "127.0.0.1"},
		},
		{
			name: "maxrequest_has_ip",
			args: args{
				sdkDevice:  nil,
				maxRequest: &openrtb2.BidRequest{Device: &openrtb2.Device{IP: "10.0.0.1"}},
			},
			want: &openrtb2.Device{IP: "10.0.0.1"},
		},
		{
			name: "sdkDevice_and_maxrequest_both_has_ip",
			args: args{
				sdkDevice:  &openrtb2.Device{IP: "127.0.0.1"},
				maxRequest: &openrtb2.BidRequest{Device: &openrtb2.Device{IP: "10.0.0.1"}},
			},
			want: &openrtb2.Device{IP: "127.0.0.1"},
		},
		{
			name: "sdkDevice_has_mccmnc_connectiontype",
			args: args{
				sdkDevice:  &openrtb2.Device{MCCMNC: "mccmnc", ConnectionType: adcom1.Connection2G.Ptr()},
				maxRequest: &openrtb2.BidRequest{Device: &openrtb2.Device{DeviceType: 5}},
			},
			want: &openrtb2.Device{DeviceType: 5, MCCMNC: "mccmnc", ConnectionType: adcom1.Connection2G.Ptr()},
		},
		{
			name: "sdkDevice_has_model",
			args: args{
				sdkDevice:  &openrtb2.Device{MCCMNC: "mccmnc", ConnectionType: adcom1.Connection2G.Ptr(), Model: "phone"},
				maxRequest: &openrtb2.BidRequest{Device: &openrtb2.Device{DeviceType: 5}},
			},
			want: &openrtb2.Device{DeviceType: 5, MCCMNC: "mccmnc", ConnectionType: adcom1.Connection2G.Ptr(), Model: "phone"},
		},
		{
			name: "sdkDevice_has_ext_atts",
			args: args{
				sdkDevice:  &openrtb2.Device{Ext: json.RawMessage(`{"atts":3}`)},
				maxRequest: &openrtb2.BidRequest{},
			},
			want: &openrtb2.Device{Ext: json.RawMessage(`{"atts":3}`)},
		},
		{
			name: "sdkDevice_has_geo_city_and_utcoffset",
			args: args{
				sdkDevice:  &openrtb2.Device{Geo: &openrtb2.Geo{City: "Delhi", UTCOffset: 3}, Ext: json.RawMessage(`{"atts":3}`)},
				maxRequest: &openrtb2.BidRequest{},
			},
			want: &openrtb2.Device{Geo: &openrtb2.Geo{City: "Delhi", UTCOffset: 3}, Ext: json.RawMessage(`{"atts":3}`)},
		},
		{
			name: "sdkDevice_geo_does_not_override_existing_lat_lon_but_merges_other_fields",
			args: func() args {
				reqLat := 12.34
				reqLon := 56.78
				signalLat := 11.11
				signalLon := 22.22
				return args{
					sdkDevice: &openrtb2.Device{Geo: &openrtb2.Geo{
						Lat:       &signalLat,
						Lon:       &signalLon,
						Country:   "IN",
						Region:    "DL",
						Metro:     "NCR",
						City:      "New Delhi",
						ZIP:       "110001",
						UTCOffset: 330,
					}},
					maxRequest: &openrtb2.BidRequest{Device: &openrtb2.Device{Geo: &openrtb2.Geo{
						Lat: &reqLat,
						Lon: &reqLon,
					}}},
				}
			}(),
			want: func() *openrtb2.Device {
				wantLat := 12.34
				wantLon := 56.78
				return &openrtb2.Device{Geo: &openrtb2.Geo{
					Lat:       &wantLat,
					Lon:       &wantLon,
					Country:   "IN",
					Region:    "DL",
					Metro:     "NCR",
					City:      "New Delhi",
					ZIP:       "110001",
					UTCOffset: 330,
				}}
			}(),
		},
		{
			name: "sdkDevice_has_ipv6",
			args: args{
				sdkDevice:  &openrtb2.Device{IPv6: "2001:db8::1"},
				maxRequest: &openrtb2.BidRequest{Device: &openrtb2.Device{DeviceType: 5}},
			},
			want: &openrtb2.Device{DeviceType: 5, IPv6: "2001:db8::1"},
		},
		{
			name: "sdkDevice_has_devicetype_make_os_osv_hwv_dims_pxr_js_lang_carrier_lmt",
			args: args{
				sdkDevice: &openrtb2.Device{
					DeviceType: 4,
					Make:       "Samsung",
					OS:         "android",
					OSV:        "13",
					HWV:        "SM-G991B",
					W:          1080,
					H:          2400,
					PxRatio:    2.75,
					JS:         ptrutil.ToPtr(int8(1)),
					Language:   "en_US",
					Carrier:    "MYTEL",
					Lmt:        ptrutil.ToPtr(int8(1)),
				},
				maxRequest: &openrtb2.BidRequest{Device: &openrtb2.Device{DeviceType: 5}},
			},
			want: &openrtb2.Device{
				DeviceType: 4,
				Make:       "Samsung",
				OS:         "android",
				OSV:        "13",
				HWV:        "SM-G991B",
				W:          1080,
				H:          2400,
				PxRatio:    2.75,
				JS:         ptrutil.ToPtr(int8(1)),
				Language:   "en_US",
				Carrier:    "MYTEL",
				Lmt:        ptrutil.ToPtr(int8(1)),
			},
		},
		{
			name: "signalDevice_has_ext_ifv",
			args: args{
				sdkDevice:  &openrtb2.Device{Ext: json.RawMessage(`{"ifv":"193DBF06-B1D8-4684-BE35-0FB0770C463C"}`)},
				maxRequest: &openrtb2.BidRequest{},
			},
			want: &openrtb2.Device{Ext: json.RawMessage(`{"ifv":"193DBF06-B1D8-4684-BE35-0FB0770C463C"}`)},
		},
		{
			name: "request_has_ifv_signal_does_not",
			args: args{
				sdkDevice:  &openrtb2.Device{Make: "Apple"},
				maxRequest: &openrtb2.BidRequest{Device: &openrtb2.Device{Ext: json.RawMessage(`{"ifv":"REQUEST-IFV-VALUE"}`)}},
			},
			want: &openrtb2.Device{Make: "Apple", Ext: json.RawMessage(`{"ifv":"REQUEST-IFV-VALUE"}`)},
		},
		{
			name: "both_request_and_signal_have_ifv_signal_wins",
			args: args{
				sdkDevice:  &openrtb2.Device{Ext: json.RawMessage(`{"ifv":"SIGNAL-IFV-VALUE"}`)},
				maxRequest: &openrtb2.BidRequest{Device: &openrtb2.Device{Ext: json.RawMessage(`{"ifv":"REQUEST-IFV-VALUE"}`)}},
			},
			want: &openrtb2.Device{Ext: json.RawMessage(`{"ifv":"SIGNAL-IFV-VALUE"}`)},
		},
		{
			name: "signalDevice_has_both_atts_and_ifv",
			args: args{
				sdkDevice:  &openrtb2.Device{Ext: json.RawMessage(`{"atts":3,"ifv":"193DBF06-B1D8-4684-BE35-0FB0770C463C"}`)},
				maxRequest: &openrtb2.BidRequest{},
			},
			want: &openrtb2.Device{Ext: json.RawMessage(`{"atts":3,"ifv":"193DBF06-B1D8-4684-BE35-0FB0770C463C"}`)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateDevice(tt.args.sdkDevice, tt.args.maxRequest)
			assert.Equal(t, tt.want, tt.args.maxRequest.Device, tt.name)
		})
	}
}

func TestUpdateApp(t *testing.T) {
	type args struct {
		signalApp  *openrtb2.App
		maxRequest *openrtb2.BidRequest
	}
	tests := []struct {
		name string
		args args
		want *openrtb2.App
	}{
		{
			name: "signalApp is nil",
			args: args{
				signalApp:  nil,
				maxRequest: &openrtb2.BidRequest{App: &openrtb2.App{ID: "signalApp"}},
			},
			want: &openrtb2.App{ID: "signalApp"},
		},
		{
			name: "maxDevice is nil",
			args: args{
				signalApp:  &openrtb2.App{},
				maxRequest: &openrtb2.BidRequest{},
			},
			want: &openrtb2.App{},
		},
		{
			name: "signalApp has Paid,Keywords,storelurl and Domain,",
			args: args{
				signalApp:  &openrtb2.App{Paid: openrtb2.Int8Ptr(1), Keywords: "k1=v1", Domain: "abc.com", StoreURL: "storeurl.com"},
				maxRequest: &openrtb2.BidRequest{},
			},
			want: &openrtb2.App{Paid: openrtb2.Int8Ptr(1), Keywords: "k1=v1", Domain: "abc.com", StoreURL: "storeurl.com"},
		},
		{
			name: "storeurl not presrnt in signal but present in max request",
			args: args{
				signalApp:  &openrtb2.App{Paid: openrtb2.Int8Ptr(1), Keywords: "k1=v1", Domain: "abc.com"},
				maxRequest: &openrtb2.BidRequest{App: &openrtb2.App{StoreURL: "storeurl.com"}},
			},
			want: &openrtb2.App{Paid: openrtb2.Int8Ptr(1), Keywords: "k1=v1", Domain: "abc.com", StoreURL: "storeurl.com"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateApp(tt.args.signalApp, tt.args.maxRequest)
			assert.Equal(t, tt.want, tt.args.maxRequest.App, tt.name)
		})
	}
}

func TestUpdateRegs(t *testing.T) {
	type args struct {
		signalRegs *openrtb2.Regs
		maxRequest *openrtb2.BidRequest
	}
	tests := []struct {
		name string
		args args
		want *openrtb2.Regs
	}{
		{
			name: "signalRegs is nil",
			args: args{
				signalRegs: nil,
				maxRequest: &openrtb2.BidRequest{},
			},
			want: nil,
		},
		{
			name: "signalRegsExt is nil",
			args: args{
				signalRegs: &openrtb2.Regs{},
				maxRequest: &openrtb2.BidRequest{},
			},
			want: &openrtb2.Regs{},
		},
		{
			name: "maxRegs is nil",
			args: args{
				signalRegs: &openrtb2.Regs{Ext: json.RawMessage(`{}`)},
				maxRequest: &openrtb2.BidRequest{},
			},
			want: &openrtb2.Regs{},
		},
		{
			name: "signalRegs has coppa, signalRegsExt has gdpr, gpp",
			args: args{
				signalRegs: &openrtb2.Regs{COPPA: 1, Ext: json.RawMessage(`{"gdpr":1,"gpp":"sdfewe3cer"}`)},
				maxRequest: &openrtb2.BidRequest{},
			},
			want: &openrtb2.Regs{COPPA: 1, Ext: json.RawMessage(`{"gdpr":1,"gpp":"sdfewe3cer"}`)},
		},
		{
			name: "signalRegs has coppa as 0, signalRegsExt has gdpr, gpp",
			args: args{
				signalRegs: &openrtb2.Regs{COPPA: 0, Ext: json.RawMessage(`{"gdpr":1,"gpp":"sdfewe3cer"}`)},
				maxRequest: &openrtb2.BidRequest{},
			},
			want: &openrtb2.Regs{Ext: json.RawMessage(`{"gdpr":1,"gpp":"sdfewe3cer"}`)},
		},
		{
			name: "signalRegsExt has gdpr, gpp, gpp_sid, us_privacy and maxRegsExt has gpp",
			args: args{
				signalRegs: &openrtb2.Regs{Ext: json.RawMessage(`{"gdpr":1,"gpp":"sdfewe3cer","gpp_sid":[6],"us_privacy":"uspConsentString"}`)},
				maxRequest: &openrtb2.BidRequest{Regs: &openrtb2.Regs{Ext: json.RawMessage(`{"gpp":"gpp_string"}`)}},
			},
			want: &openrtb2.Regs{Ext: json.RawMessage(`{"gpp":"sdfewe3cer","gdpr":1,"gpp_sid":[6],"us_privacy":"uspConsentString"}`)},
		},
		{
			name: "signalRegsExt has dsa",
			args: args{
				signalRegs: &openrtb2.Regs{Ext: json.RawMessage(`{"dsa":{"dsarequired":3,"pubrender":2,"datatopub":1,"transparency":[{"domain":"params.com","params":[1,2]},{"domain":"dsaicon2.com","params":[2,3]}]}}`)},
				maxRequest: &openrtb2.BidRequest{},
			},
			want: &openrtb2.Regs{Ext: json.RawMessage(`{"dsa":{"dsarequired":3,"pubrender":2,"datatopub":1,"transparency":[{"domain":"params.com","params":[1,2]},{"domain":"dsaicon2.com","params":[2,3]}]}}`)},
		},
		{
			name: "signalRegsExt has gdpr, gpp, gp_sid, us_privacy, dsa and maxRegsExt has gpp",
			args: args{
				signalRegs: &openrtb2.Regs{Ext: json.RawMessage(`{"gdpr":1,"gpp":"sdfewe3cer","gpp_sid":[6],"us_privacy":"uspConsentString","dsa":{"dsarequired":3,"pubrender":2,"datatopub":1,"transparency":[{"domain":"params.com","params":[1,2]},{"domain":"dsaicon2.com","params":[2,3]}]}`)},
				maxRequest: &openrtb2.BidRequest{Regs: &openrtb2.Regs{Ext: json.RawMessage(`{"gpp":"gpp_string"}`)}},
			},
			want: &openrtb2.Regs{Ext: json.RawMessage(`{"gpp":"sdfewe3cer","gdpr":1,"gpp_sid":[6],"us_privacy":"uspConsentString","dsa":{"dsarequired":3,"pubrender":2,"datatopub":1,"transparency":[{"domain":"params.com","params":[1,2]},{"domain":"dsaicon2.com","params":[2,3]}]}}`)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateRegs(tt.args.signalRegs, tt.args.maxRequest)
			assert.Equal(t, tt.want, tt.args.maxRequest.Regs, tt.name)
		})
	}
}

func TestUpdateSource(t *testing.T) {
	type args struct {
		signalSource *openrtb2.Source
		maxRequest   *openrtb2.BidRequest
	}
	tests := []struct {
		name string
		args args
		want *openrtb2.Source
	}{
		{
			name: "signalSource is nil",
			args: args{
				signalSource: nil,
				maxRequest:   &openrtb2.BidRequest{},
			},
			want: nil,
		},
		{
			name: "signalSourceExt is nil",
			args: args{
				signalSource: &openrtb2.Source{},
				maxRequest:   &openrtb2.BidRequest{},
			},
			want: nil,
		},
		{
			name: "maxSource is nil",
			args: args{
				signalSource: &openrtb2.Source{Ext: json.RawMessage(`{}`)},
				maxRequest:   &openrtb2.BidRequest{},
			},
			want: &openrtb2.Source{},
		},
		{
			name: "signalSourceExt has omidpn, omidpv",
			args: args{
				signalSource: &openrtb2.Source{Ext: json.RawMessage(`{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}`)},
				maxRequest:   &openrtb2.BidRequest{},
			},
			want: &openrtb2.Source{Ext: json.RawMessage(`{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}`)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateSource(tt.args.signalSource, tt.args.maxRequest)
			assert.Equal(t, tt.want, tt.args.maxRequest.Source, tt.name)
		})
	}
}

func TestUpdateUser(t *testing.T) {
	type args struct {
		signalUser *openrtb2.User
		maxRequest *openrtb2.BidRequest
	}
	tests := []struct {
		name string
		args args
		want *openrtb2.User
	}{
		{
			name: "signalUser is nil",
			args: args{
				signalUser: nil,
				maxRequest: &openrtb2.BidRequest{},
			},
			want: nil,
		},
		{
			name: "maxUser is nil",
			args: args{
				signalUser: &openrtb2.User{Ext: json.RawMessage(``)},
				maxRequest: &openrtb2.BidRequest{},
			},
			want: &openrtb2.User{},
		},
		{
			name: "signalUser has yob, gender, keywords",
			args: args{
				signalUser: &openrtb2.User{Yob: 1999, Gender: "M", Keywords: "k1=v2;k2=v2", Ext: json.RawMessage(``)},
				maxRequest: &openrtb2.BidRequest{},
			},
			want: &openrtb2.User{Yob: 1999, Gender: "M", Keywords: "k1=v2;k2=v2"},
		},
		{
			name: "signalUser and maxUser has yob, gender, keywords and data",
			args: args{
				signalUser: &openrtb2.User{Data: []openrtb2.Data{{ID: "123", Name: "PubMatic_SDK", Segment: []openrtb2.Segment{{ID: "seg_id", Name: "PubMatic_Seg", Value: "segment_value", Ext: json.RawMessage(`{"segtax":4}`)}}}}, Yob: 1999, Gender: "M", Keywords: "k1=v2;k2=v2", Ext: json.RawMessage(``)},
				maxRequest: &openrtb2.BidRequest{User: &openrtb2.User{Data: []openrtb2.Data{{ID: "max_id", Name: "Publisher Passed", Segment: []openrtb2.Segment{{ID: "max_seg_id", Name: "max_Seg", Value: "max_segment_value"}}}}, Yob: 2000, Gender: "F", Keywords: "k52=v43"}},
			},
			want: &openrtb2.User{Data: []openrtb2.Data{{ID: "123", Name: "PubMatic_SDK", Segment: []openrtb2.Segment{{ID: "seg_id", Name: "PubMatic_Seg", Value: "segment_value", Ext: json.RawMessage(`{"segtax":4}`)}}}}, Yob: 1999, Gender: "M", Keywords: "k1=v2;k2=v2"},
		},
		{
			name: "signalUserExt has consent",
			args: args{
				signalUser: &openrtb2.User{ID: "sdkID", Yob: 1999, Gender: "M", Keywords: "k1=v2;k2=v2", Ext: json.RawMessage(`{"consent":"consent_string"}`)},
				maxRequest: &openrtb2.BidRequest{User: &openrtb2.User{ID: "maxID", Yob: 2000, Gender: "F", Keywords: "k52=v43"}},
			},
			want: &openrtb2.User{ID: "maxID", Yob: 1999, Gender: "M", Keywords: "k1=v2;k2=v2", Ext: json.RawMessage(`{"consent":"consent_string"}`)},
		},
		{
			name: "signalUserExt has consent and eids",
			args: args{
				signalUser: &openrtb2.User{ID: "sdkID", Yob: 1999, Gender: "M", Keywords: "k1=v2;k2=v2", Ext: json.RawMessage(`{"consent":"consent_string","eids":[{"source":"amxid","uids":[{"atype":1,"id":"88de601e-3d98-48e7-81d7-00000000"}]},{"source":"adserver.org","uids":[{"id":"1234567","ext":{"rtiPartner":"TDID"}}]}]}`)},
				maxRequest: &openrtb2.BidRequest{User: &openrtb2.User{ID: "maxID", Yob: 2000, Gender: "F", Keywords: "k52=v43"}},
			},
			want: &openrtb2.User{ID: "maxID", Yob: 1999, Gender: "M", Keywords: "k1=v2;k2=v2", Ext: json.RawMessage(`{"consent":"consent_string","eids":[{"source":"amxid","uids":[{"atype":1,"id":"88de601e-3d98-48e7-81d7-00000000"}]},{"source":"adserver.org","uids":[{"id":"1234567","ext":{"rtiPartner":"TDID"}}]}]}`)},
		},
		{
			name: "signalUserExt_and_request_both_has_sessionduration_and_impdepth",
			args: args{
				signalUser: &openrtb2.User{ID: "sdkID", Yob: 1999, Gender: "M", Keywords: "k1=v2;k2=v2", Ext: json.RawMessage(`{"sessionduration":40,"impdepth":10}`)},
				maxRequest: &openrtb2.BidRequest{User: &openrtb2.User{ID: "maxID", Yob: 2000, Gender: "F", Keywords: "k52=v43", Ext: json.RawMessage(`{"sessionduration":50,"impdepth":15}`)}},
			},
			want: &openrtb2.User{ID: "maxID", Yob: 1999, Gender: "M", Keywords: "k1=v2;k2=v2", Ext: json.RawMessage(`{"sessionduration":40,"impdepth":10}`)},
		},
		{
			name: "signalUserExt_has_sessionduration_and_impdepth_with_consent",
			args: args{
				signalUser: &openrtb2.User{ID: "sdkID", Yob: 1999, Gender: "M", Keywords: "k1=v2;k2=v2", Ext: json.RawMessage(`{"sessionduration":40,"impdepth":10,"consent":"consent_string"}`)},
				maxRequest: &openrtb2.BidRequest{User: &openrtb2.User{ID: "maxID", Yob: 2000, Gender: "F", Keywords: "k52=v43"}},
			},
			want: &openrtb2.User{ID: "maxID", Yob: 1999, Gender: "M", Keywords: "k1=v2;k2=v2", Ext: json.RawMessage(`{"consent":"consent_string","sessionduration":40,"impdepth":10}`)},
		},
		{
			name: "signalUserExt_has_invalid_sessionduration_and_impdepth_with_consent",
			args: args{
				signalUser: &openrtb2.User{ID: "sdkID", Yob: 1999, Gender: "M", Keywords: "k1=v2;k2=v2", Ext: json.RawMessage(`{"sessionduration":-40,"impdepth":-10,"consent":"consent_string"}`)},
				maxRequest: &openrtb2.BidRequest{User: &openrtb2.User{ID: "maxID", Yob: 2000, Gender: "F", Keywords: "k52=v43"}},
			},
			want: &openrtb2.User{ID: "maxID", Yob: 1999, Gender: "M", Keywords: "k1=v2;k2=v2", Ext: json.RawMessage(`{"consent":"consent_string","sessionduration":-40,"impdepth":-10}`)},
		},
		{
			name: "request_has_sessionduration_and_impdepth_with_consent",
			args: args{
				signalUser: &openrtb2.User{ID: "sdkID", Yob: 1999, Gender: "M", Keywords: "k1=v2;k2=v2", Ext: json.RawMessage(`{"consent":"consent_string"}`)},
				maxRequest: &openrtb2.BidRequest{User: &openrtb2.User{ID: "maxID", Yob: 2000, Gender: "F", Keywords: "k52=v43"}, Ext: json.RawMessage(`{"sessionduration":40,"impdepth":10,"consent":"consent_string1"}`)},
			},
			want: &openrtb2.User{ID: "maxID", Yob: 1999, Gender: "M", Keywords: "k1=v2;k2=v2", Ext: json.RawMessage(`{"consent":"consent_string"}`)},
		},
		{
			name: "request_has_invalid_sessionduration_and_impdepth_with_consent",
			args: args{
				signalUser: &openrtb2.User{ID: "sdkID", Yob: 1999, Gender: "M", Keywords: "k1=v2;k2=v2", Ext: json.RawMessage(`{"consent":"consent_string"}`)},
				maxRequest: &openrtb2.BidRequest{User: &openrtb2.User{ID: "maxID", Yob: 2000, Gender: "F", Keywords: "k52=v43"}, Ext: json.RawMessage(`{"sessionduration":-40,"impdepth":-10,"consent":"consent_string1"}`)},
			},
			want: &openrtb2.User{ID: "maxID", Yob: 1999, Gender: "M", Keywords: "k1=v2;k2=v2", Ext: json.RawMessage(`{"consent":"consent_string"}`)},
		},
		{
			name: "signalUserExt_has_invalid_and_request_has_valid_sessionduration_and_impdepth",
			args: args{
				signalUser: &openrtb2.User{ID: "sdkID", Yob: 1999, Gender: "M", Keywords: "k1=v2;k2=v2", Ext: json.RawMessage(`{"sessionduration":-40,"impdepth":"impdepth"}`)},
				maxRequest: &openrtb2.BidRequest{User: &openrtb2.User{ID: "maxID", Yob: 2000, Gender: "F", Keywords: "k52=v43", Ext: json.RawMessage(`{"sessionduration":50,"impdepth":15}`)}},
			},
			want: &openrtb2.User{ID: "maxID", Yob: 1999, Gender: "M", Keywords: "k1=v2;k2=v2", Ext: json.RawMessage(`{"sessionduration":-40,"impdepth":"impdepth"}`)},
		},
		{
			name: "signalUserExt_has_invalid_sessionduration_and_impdepth",
			args: args{
				signalUser: &openrtb2.User{ID: "sdkID", Yob: 1999, Gender: "M", Keywords: "k1=v2;k2=v2", Ext: json.RawMessage(`{"sessionduration":-40,"impdepth":"impdepth"}`)},
				maxRequest: &openrtb2.BidRequest{User: &openrtb2.User{ID: "maxID", Yob: 2000, Gender: "F", Keywords: "k52=v43"}},
			},
			want: &openrtb2.User{ID: "maxID", Yob: 1999, Gender: "M", Keywords: "k1=v2;k2=v2", Ext: json.RawMessage(`{"sessionduration":-40,"impdepth":"impdepth"}`)},
		},
		{
			name: "request_has_sessionduration_and_impdepth",
			args: args{
				signalUser: &openrtb2.User{ID: "sdkID", Yob: 1999, Gender: "M", Keywords: "k1=v2;k2=v2"},
				maxRequest: &openrtb2.BidRequest{User: &openrtb2.User{ID: "maxID", Yob: 2000, Gender: "F", Keywords: "k52=v43", Ext: json.RawMessage(`{"sessionduration":50,"impdepth":15}`)}},
			},
			want: &openrtb2.User{ID: "maxID", Yob: 1999, Gender: "M", Keywords: "k1=v2;k2=v2", Ext: json.RawMessage(`{}`)},
		},
		{
			name: "signalUserExt_has_sessionduration_and_impdepth",
			args: args{
				signalUser: &openrtb2.User{ID: "sdkID", Yob: 1999, Gender: "M", Keywords: "k1=v2;k2=v2", Ext: json.RawMessage(`{"sessionduration":40,"impdepth":10}`)},
				maxRequest: &openrtb2.BidRequest{User: &openrtb2.User{ID: "maxID", Yob: 2000, Gender: "F", Keywords: "k52=v43"}},
			},
			want: &openrtb2.User{ID: "maxID", Yob: 1999, Gender: "M", Keywords: "k1=v2;k2=v2", Ext: json.RawMessage(`{"sessionduration":40,"impdepth":10}`)},
		},
		{
			name: "signalUserExt_has_string_type_sessionduration_and_impdepth",
			args: args{
				signalUser: &openrtb2.User{ID: "sdkID", Yob: 1999, Gender: "M", Keywords: "k1=v2;k2=v2", Ext: json.RawMessage(`{"sessionduration":"sessionduration","impdepth":"impdepth"}`)},
				maxRequest: &openrtb2.BidRequest{User: &openrtb2.User{ID: "maxID", Yob: 2000, Gender: "F", Keywords: "k52=v43"}},
			},
			want: &openrtb2.User{ID: "maxID", Yob: 1999, Gender: "M", Keywords: "k1=v2;k2=v2", Ext: json.RawMessage(`{"sessionduration":"sessionduration","impdepth":"impdepth"}`)},
		},
		{
			name: "signalUserExt_has_zero_sessionduration_and_impdepth",
			args: args{
				signalUser: &openrtb2.User{ID: "sdkID", Yob: 1999, Gender: "M", Keywords: "k1=v2;k2=v2", Ext: json.RawMessage(`{"sessionduration":0,"impdepth":0}`)},
				maxRequest: &openrtb2.BidRequest{User: &openrtb2.User{ID: "maxID", Yob: 2000, Gender: "F", Keywords: "k52=v43"}},
			},
			want: &openrtb2.User{ID: "maxID", Yob: 1999, Gender: "M", Keywords: "k1=v2;k2=v2", Ext: json.RawMessage(`{"sessionduration":0,"impdepth":0}`)},
		},
		{
			name: "signalUserExt_has_sessionduration",
			args: args{
				signalUser: &openrtb2.User{ID: "sdkID", Yob: 1999, Gender: "M", Keywords: "k1=v2;k2=v2", Ext: json.RawMessage(`{"sessionduration":40}`)},
				maxRequest: &openrtb2.BidRequest{User: &openrtb2.User{ID: "maxID", Yob: 2000, Gender: "F", Keywords: "k52=v43"}},
			},
			want: &openrtb2.User{ID: "maxID", Yob: 1999, Gender: "M", Keywords: "k1=v2;k2=v2", Ext: json.RawMessage(`{"sessionduration":40}`)},
		},
		{
			name: "signalUserExt_has_impdepth",
			args: args{
				signalUser: &openrtb2.User{ID: "sdkID", Yob: 1999, Gender: "M", Keywords: "k1=v2;k2=v2", Ext: json.RawMessage(`{"impdepth":10}`)},
				maxRequest: &openrtb2.BidRequest{User: &openrtb2.User{ID: "maxID", Yob: 2000, Gender: "F", Keywords: "k52=v43"}},
			},
			want: &openrtb2.User{ID: "maxID", Yob: 1999, Gender: "M", Keywords: "k1=v2;k2=v2", Ext: json.RawMessage(`{"impdepth":10}`)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateUser(tt.args.signalUser, tt.args.maxRequest)
			assert.Equal(t, tt.want, tt.args.maxRequest.User, tt.name)
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
			got := setIfKeysExists(tt.args.source, tt.args.target, tt.args.keys...)
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}

func TestAddSignalDataInRequest(t *testing.T) {
	type args struct {
		signal     string
		maxRequest json.RawMessage
	}
	tests := []struct {
		name           string
		args           args
		wantMaxRequest json.RawMessage
	}{
		{
			name: "replace or add from signal",
			args: args{
				signal:     `{"device":{"devicetype":4,"w":393,"h":852,"ifa":"F5BA1637-7156-4369-BA7E-3C45033D9F61","mccmnc":"311-480","js":1,"osv":"17.3.1","connectiontype":5,"os":"iOS","pxratio":3,"geo":{"lastfix":8,"lat":37.48773508935608,"utcoffset":-480,"lon":-122.22855027909678,"type":1},"language":"en","make":"Apple","ext":{"atts":3},"ua":"signal_UA","model":"iPhone15,2","carrier":"Verizon"},"source":{"ext":{"omidpn":"Pubmatic","omidpv":"3.1.0"}},"id":"CE204A0E-31C3-4D7F-A1A0-D34AF5ED1A7F","app":{"id":"406719683","paid":1,"keywords":"k1=v1","domain":"abc.com","bundle":"406719683","storeurl":"https://apps.apple.com/us/app/gasbuddy-find-pay-for-gas/id406719683","name":"GasBuddy","publisher":{"id":"160361"},"ver":"700.89.22927"},"ext":{"wrapper":{"sumry_disable":1,"profileid":3422}},"imp":[{"secure":1,"tagid":"Mobile_iPhone_List_Screen_Bottom","banner":{"pos":0,"format":[{"w":300,"h":250}],"api":[5,6,7]},"id":"98D9318E-5276-402F-BAA4-CDBD8A364957","ext":{"skadn":{"sourceapp":"406719683","versions":["2.0","2.1","2.2","3.0","4.0"],"skadnetids":["cstr6suwn9.skadnetwork","7ug5zh24hu.skadnetwork","uw77j35x4d.skadnetwork","c6k4g5qg8m.skadnetwork","hs6bdukanm.skadnetwork","yclnxrl5pm.skadnetwork","3sh42y64q3.skadnetwork","cj5566h2ga.skadnetwork","klf5c3l5u5.skadnetwork","8s468mfl3y.skadnetwork","2u9pt9hc89.skadnetwork","7rz58n8ntl.skadnetwork","ppxm28t8ap.skadnetwork","mtkv5xtk9e.skadnetwork","cg4yq2srnc.skadnetwork","wzmmz9fp6w.skadnetwork","k674qkevps.skadnetwork","v72qych5uu.skadnetwork","578prtvx9j.skadnetwork","3rd42ekr43.skadnetwork","g28c52eehv.skadnetwork","2fnua5tdw4.skadnetwork","9nlqeag3gk.skadnetwork","5lm9lj6jb7.skadnetwork","97r2b46745.skadnetwork","e5fvkxwrpn.skadnetwork","4pfyvq9l8r.skadnetwork","tl55sbb4fm.skadnetwork","t38b2kh725.skadnetwork","prcb7njmu6.skadnetwork","mlmmfzh3r3.skadnetwork","9t245vhmpl.skadnetwork","9rd848q2bz.skadnetwork","4fzdc2evr5.skadnetwork","4468km3ulz.skadnetwork","m8dbw4sv7c.skadnetwork","ejvt5qm6ak.skadnetwork","5lm9lj6jb7.skadnetwork","44jx6755aq.skadnetwork","6g9af3uyq4.skadnetwork","u679fj5vs4.skadnetwork","rx5hdcabgc.skadnetwork","275upjj5gd.skadnetwork","p78axxw29g.skadnetwork"],"productpage":1,"version":"2.0"}},"displaymanagerver":"3.1.0","clickbrowser":1,"video":{"companionad":[{"pos":0,"format":[{"w":300,"h":250}],"vcm":1}],"protocols":[2,3,5,6,7,8,11,12,13,14],"h":250,"w":300,"linearity":1,"pos":0,"boxingallowed":1,"placement":2,"mimes":["video/3gpp2","video/quicktime","video/mp4","video/x-m4v","video/3gpp"],"companiontype":[1,2,3],"delivery":[2],"startdelay":0,"playbackend":1,"api":[7]},"displaymanager":"PubMatic_OpenWrap_SDK","instl":0}],"at":1,"cur":["USD"],"regs":{"coppa":1,"ext":{"ccpa":0,"gdpr":1,"gpp":"gpp_string","gpp_sid":[7],"us_privacy":"uspConsentString","consent":"0"}}}`,
				maxRequest: json.RawMessage(`{"id":"{BID_ID}","at":1,"bcat":["IAB26-4","IAB26-2","IAB25-6","IAB25-5","IAB25-4","IAB25-3","IAB25-1","IAB25-7","IAB8-18","IAB26-3","IAB26-1","IAB8-5","IAB25-2","IAB11-4"],"tmax":3000,"app":{"name":"DrawHappyAngel","ver":"0.5.4","bundle":"com.newstory.DrawHappyAngel","cat":["IAB9-30"],"id":"{NETWORK_APP_ID}","publisher":{"name":"New Story Inc.","ext":{"installed_sdk":{"id":"MOLOCO_BIDDING","sdk_version":{"major":1,"minor":0,"micro":0},"adapter_version":{"major":1,"minor":0,"micro":0}}}},"ext":{"orientation":1}},"device":{"ifa":"497a10d6-c4dd-4e04-a986-c32b7180d462","ip":"38.158.207.171","carrier":"MYTEL","language":"en_US","hwv":"ruby","ppi":440,"pxratio":2.75,"devicetype":4,"connectiontype":2,"js":1,"h":2400,"w":1080,"geo":{"type":2,"ipservice":3,"lat":40.7429,"lon":-73.9392,"long":-73.9392,"city":"Queens","country":"USA","region":"ny","dma":"501","metro":"501","zip":"11101","ext":{"org":"Myanmar Broadband Telecom Co.","isp":"Myanmar Broadband Telecom Co."}},"ext":{},"osv":"13.0.0","ua":"Mozilla/5.0 (Linux; Android 13; 22101316C Build/TP1A.220624.014; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/120.0.6099.230 Mobile Safari/537.36","make":"xiaomi","model":"22101316c","os":"android"},"imp":[{"id":"1","displaymanager":"applovin_mediation","displaymanagerver":"11.8.2","instl":0,"secure":0,"tagid":"{NETWORK_PLACEMENT_ID}","bidfloor":0.01,"bidfloorcur":"USD","exp":14400,"banner":{"id":"1","w":320,"h":50,"btype":[],"battr":[1,2,5,8,9,14,17],"pos":1,"format":[{"w":320,"h":50}]},"rwdd":0}],"user":{"data":[{"id":"1","name":"Publisher Passed","segment":[{"signal":"{BIDDING_SIGNAL}"}]}],"ext":{"gdpr":0}},"regs":{"ext":{"ccpa":0,"gdpr":1,"consent":"0","tcf_consent_string":"{TCF_STRING}"}},"source":{"ext":{"schain":{"ver":"1.0","complete":1,"nodes":[{"asi":"applovin.com","sid":"53bf468f18c5a0e2b7d4e3f748c677c1","rid":"494dbe15a3ce08c54f4e456363f35a022247f997","hp":1}]}}},"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"profileid":1234}}}}}}`),
			},
			wantMaxRequest: json.RawMessage(`{"id":"{BID_ID}","at":1,"bcat":["IAB26-4","IAB26-2","IAB25-6","IAB25-5","IAB25-4","IAB25-3","IAB25-1","IAB25-7","IAB8-18","IAB26-3","IAB26-1","IAB8-5","IAB25-2","IAB11-4"],"tmax":3000,"app":{"storeurl":"https://apps.apple.com/us/app/gasbuddy-find-pay-for-gas/id406719683","paid":1,"keywords":"k1=v1","domain":"abc.com","name":"DrawHappyAngel","ver":"0.5.4","bundle":"com.newstory.DrawHappyAngel","cat":["IAB9-30"],"id":"{NETWORK_APP_ID}","publisher":{"name":"New Story Inc.","ext":{"installed_sdk":{"id":"MOLOCO_BIDDING","sdk_version":{"major":1,"minor":0,"micro":0},"adapter_version":{"major":1,"minor":0,"micro":0}}}},"ext":{"orientation":1}},"device":{"ifa":"F5BA1637-7156-4369-BA7E-3C45033D9F61","ip":"38.158.207.171","carrier":"Verizon","language":"en","hwv":"ruby","ppi":440,"pxratio":3,"devicetype":4,"mccmnc":"311-480","connectiontype":5,"js":1,"h":852,"w":393,"geo":{"city":"Queens","type":2,"ipservice":3,"lat":40.7429,"lon":-73.9392,"long":-73.9392,"country":"USA","region":"ny","dma":"501","metro":"501","zip":"11101","utcoffset":-480,"ext":{"org":"Myanmar Broadband Telecom Co.","isp":"Myanmar Broadband Telecom Co."}},"ext":{"atts":3},"osv":"17.3.1","ua":"signal_UA","make":"Apple","model":"iPhone15,2","os":"iOS"},"imp":[{"id":"1","displaymanagerver":"3.1.0","clickbrowser":1,"displaymanager":"PubMatic_OpenWrap_SDK","instl":0,"secure":0,"tagid":"{NETWORK_PLACEMENT_ID}","bidfloor":0.01,"bidfloorcur":"USD","exp":14400,"banner":{"id":"1","w":320,"h":50,"btype":[],"api":[5,6,7],"battr":[1,2,5,8,9,14,17],"pos":1,"format":[{"w":320,"h":50}]},"video":{"companionad":[{"pos":0,"format":[{"w":300,"h":250}],"vcm":1}],"protocols":[2,3,5,6,7,8,11,12,13,14],"h":250,"w":300,"linearity":1,"pos":0,"boxingallowed":1,"placement":2,"mimes":["video/3gpp2","video/quicktime","video/mp4","video/x-m4v","video/3gpp"],"companiontype":[1,2,3],"delivery":[2],"startdelay":0,"playbackend":1,"api":[7]},"rwdd":0,"ext":{"skadn":{"sourceapp":"406719683","versions":["2.0","2.1","2.2","3.0","4.0"],"skadnetids":["cstr6suwn9.skadnetwork","7ug5zh24hu.skadnetwork","uw77j35x4d.skadnetwork","c6k4g5qg8m.skadnetwork","hs6bdukanm.skadnetwork","yclnxrl5pm.skadnetwork","3sh42y64q3.skadnetwork","cj5566h2ga.skadnetwork","klf5c3l5u5.skadnetwork","8s468mfl3y.skadnetwork","2u9pt9hc89.skadnetwork","7rz58n8ntl.skadnetwork","ppxm28t8ap.skadnetwork","mtkv5xtk9e.skadnetwork","cg4yq2srnc.skadnetwork","wzmmz9fp6w.skadnetwork","k674qkevps.skadnetwork","v72qych5uu.skadnetwork","578prtvx9j.skadnetwork","3rd42ekr43.skadnetwork","g28c52eehv.skadnetwork","2fnua5tdw4.skadnetwork","9nlqeag3gk.skadnetwork","5lm9lj6jb7.skadnetwork","97r2b46745.skadnetwork","e5fvkxwrpn.skadnetwork","4pfyvq9l8r.skadnetwork","tl55sbb4fm.skadnetwork","t38b2kh725.skadnetwork","prcb7njmu6.skadnetwork","mlmmfzh3r3.skadnetwork","9t245vhmpl.skadnetwork","9rd848q2bz.skadnetwork","4fzdc2evr5.skadnetwork","4468km3ulz.skadnetwork","m8dbw4sv7c.skadnetwork","ejvt5qm6ak.skadnetwork","5lm9lj6jb7.skadnetwork","44jx6755aq.skadnetwork","6g9af3uyq4.skadnetwork","u679fj5vs4.skadnetwork","rx5hdcabgc.skadnetwork","275upjj5gd.skadnetwork","p78axxw29g.skadnetwork"],"productpage":1,"version":"2.0"}}}],"user":{"data":[{"id":"1","name":"Publisher Passed","segment":[{"signal":"{BIDDING_SIGNAL}"}]}],"ext":{"gdpr":0}},"regs":{"coppa":1,"ext":{"ccpa":0,"gdpr":1,"consent":"0","tcf_consent_string":"{TCF_STRING}","gpp":"gpp_string","gpp_sid":[7],"us_privacy":"uspConsentString"}},"source":{"ext":{"schain":{"ver":"1.0","complete":1,"nodes":[{"asi":"applovin.com","sid":"53bf468f18c5a0e2b7d4e3f748c677c1","rid":"494dbe15a3ce08c54f4e456363f35a022247f997","hp":1}]},"omidpn":"Pubmatic","omidpv":"3.1.0"}},"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"profileid":1234,"clientconfig":0}}}}}}`),
		},
		{
			name: "replace or add from signal,and remove banner as bannertype rewarded",
			args: args{
				signal:     `{"device":{"devicetype":4,"w":393,"h":852,"ifa":"F5BA1637-7156-4369-BA7E-3C45033D9F61","mccmnc":"311-480","js":1,"osv":"17.3.1","connectiontype":5,"os":"iOS","pxratio":3,"geo":{"lastfix":8,"lat":37.48773508935608,"utcoffset":-480,"lon":-122.22855027909678,"type":1},"language":"en","make":"Apple","ext":{"atts":3},"ua":"signal_UA","model":"iPhone15,2","carrier":"Verizon"},"source":{"ext":{"omidpn":"Pubmatic","omidpv":"3.1.0"}},"id":"CE204A0E-31C3-4D7F-A1A0-D34AF5ED1A7F","app":{"id":"406719683","paid":1,"keywords":"k1=v1","domain":"abc.com","bundle":"406719683","storeurl":"https://apps.apple.com/us/app/gasbuddy-find-pay-for-gas/id406719683","name":"GasBuddy","publisher":{"id":"160361"},"ver":"700.89.22927"},"ext":{"wrapper":{"sumry_disable":1,"clientconfig":1,"profileid":3422}},"imp":[{"secure":1,"tagid":"Mobile_iPhone_List_Screen_Bottom","banner":{"pos":0,"format":[{"w":300,"h":250}],"api":[5,6,7]},"id":"98D9318E-5276-402F-BAA4-CDBD8A364957","ext":{"skadn":{"sourceapp":"406719683","versions":["2.0","2.1","2.2","3.0","4.0"],"skadnetids":["cstr6suwn9.skadnetwork","7ug5zh24hu.skadnetwork","uw77j35x4d.skadnetwork","c6k4g5qg8m.skadnetwork","hs6bdukanm.skadnetwork","yclnxrl5pm.skadnetwork","3sh42y64q3.skadnetwork","cj5566h2ga.skadnetwork","klf5c3l5u5.skadnetwork","8s468mfl3y.skadnetwork","2u9pt9hc89.skadnetwork","7rz58n8ntl.skadnetwork","ppxm28t8ap.skadnetwork","mtkv5xtk9e.skadnetwork","cg4yq2srnc.skadnetwork","wzmmz9fp6w.skadnetwork","k674qkevps.skadnetwork","v72qych5uu.skadnetwork","578prtvx9j.skadnetwork","3rd42ekr43.skadnetwork","g28c52eehv.skadnetwork","2fnua5tdw4.skadnetwork","9nlqeag3gk.skadnetwork","5lm9lj6jb7.skadnetwork","97r2b46745.skadnetwork","e5fvkxwrpn.skadnetwork","4pfyvq9l8r.skadnetwork","tl55sbb4fm.skadnetwork","t38b2kh725.skadnetwork","prcb7njmu6.skadnetwork","mlmmfzh3r3.skadnetwork","9t245vhmpl.skadnetwork","9rd848q2bz.skadnetwork","4fzdc2evr5.skadnetwork","4468km3ulz.skadnetwork","m8dbw4sv7c.skadnetwork","ejvt5qm6ak.skadnetwork","5lm9lj6jb7.skadnetwork","44jx6755aq.skadnetwork","6g9af3uyq4.skadnetwork","u679fj5vs4.skadnetwork","rx5hdcabgc.skadnetwork","275upjj5gd.skadnetwork","p78axxw29g.skadnetwork"],"productpage":1,"version":"2.0"}},"displaymanagerver":"3.1.0","clickbrowser":1,"video":{"companionad":[{"pos":0,"format":[{"w":300,"h":250}],"vcm":1}],"protocols":[2,3,5,6,7,8,11,12,13,14],"h":250,"w":300,"linearity":1,"pos":0,"boxingallowed":1,"placement":2,"mimes":["video/3gpp2","video/quicktime","video/mp4","video/x-m4v","video/3gpp"],"companiontype":[1,2,3],"delivery":[2],"startdelay":0,"playbackend":1,"api":[7]},"displaymanager":"PubMatic_OpenWrap_SDK","instl":0}],"at":1,"cur":["USD"],"regs":{"coppa":1,"ext":{"ccpa":0,"gdpr":1,"gpp":"gpp_string","gpp_sid":[7],"us_privacy":"uspConsentString","consent":"0"}}}`,
				maxRequest: json.RawMessage(`{"id":"{BID_ID}","at":1,"bcat":["IAB26-4","IAB26-2","IAB25-6","IAB25-5","IAB25-4","IAB25-3","IAB25-1","IAB25-7","IAB8-18","IAB26-3","IAB26-1","IAB8-5","IAB25-2","IAB11-4"],"tmax":3000,"app":{"name":"DrawHappyAngel","ver":"0.5.4","bundle":"com.newstory.DrawHappyAngel","cat":["IAB9-30"],"id":"{NETWORK_APP_ID}","publisher":{"name":"New Story Inc.","ext":{"installed_sdk":{"id":"MOLOCO_BIDDING","sdk_version":{"major":1,"minor":0,"micro":0},"adapter_version":{"major":1,"minor":0,"micro":0}}}},"ext":{"orientation":1}},"device":{"ifa":"F5BA1637-7156-4369-BA7E-3C45033D9F61","ip":"38.158.207.171","carrier":"MYTEL","language":"en_US","hwv":"ruby","ppi":440,"pxratio":2.75,"devicetype":4,"connectiontype":2,"js":1,"h":2400,"w":1080,"geo":{"type":2,"ipservice":3,"lat":40.7429,"lon":-73.9392,"long":-73.9392,"city":"Queens","country":"USA","region":"ny","dma":"501","metro":"501","zip":"11101","ext":{"org":"Myanmar Broadband Telecom Co.","isp":"Myanmar Broadband Telecom Co."}},"ext":{},"osv":"13.0.0","ua":"Mozilla/5.0 (Linux; Android 13; 22101316C Build/TP1A.220624.014; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/120.0.6099.230 Mobile Safari/537.36","make":"xiaomi","model":"22101316c","os":"android"},"imp":[{"id":"1","displaymanager":"applovin_mediation","displaymanagerver":"11.8.2","instl":0,"secure":0,"tagid":"{NETWORK_PLACEMENT_ID}","bidfloor":0.01,"bidfloorcur":"USD","exp":14400,"banner":{"id":"1","w":320,"h":50,"btype":[],"battr":[1,2,5,8,9,14,17],"pos":1,"format":[{"w":320,"h":50}],"ext":{"bannertype":"rewarded"}},"rwdd":0}],"user":{"data":[{"id":"1","name":"Publisher Passed","segment":[{"signal":"{BIDDING_SIGNAL}"}]}],"ext":{"gdpr":0}},"regs":{"ext":{"ccpa":0,"gdpr":1,"consent":"0","tcf_consent_string":"{TCF_STRING}"}},"source":{"ext":{"schain":{"ver":"1.0","complete":1,"nodes":[{"asi":"applovin.com","sid":"53bf468f18c5a0e2b7d4e3f748c677c1","rid":"494dbe15a3ce08c54f4e456363f35a022247f997","hp":1}]}}}}`),
			},
			wantMaxRequest: json.RawMessage(`{"id":"{BID_ID}","at":1,"bcat":["IAB26-4","IAB26-2","IAB25-6","IAB25-5","IAB25-4","IAB25-3","IAB25-1","IAB25-7","IAB8-18","IAB26-3","IAB26-1","IAB8-5","IAB25-2","IAB11-4"],"tmax":3000,"app":{"storeurl":"https://apps.apple.com/us/app/gasbuddy-find-pay-for-gas/id406719683","paid":1,"keywords":"k1=v1","domain":"abc.com","name":"DrawHappyAngel","ver":"0.5.4","bundle":"com.newstory.DrawHappyAngel","cat":["IAB9-30"],"id":"{NETWORK_APP_ID}","publisher":{"name":"New Story Inc.","ext":{"installed_sdk":{"id":"MOLOCO_BIDDING","sdk_version":{"major":1,"minor":0,"micro":0},"adapter_version":{"major":1,"minor":0,"micro":0}}}},"ext":{"orientation":1}},"device":{"ifa":"F5BA1637-7156-4369-BA7E-3C45033D9F61","ip":"38.158.207.171","carrier":"Verizon","language":"en","hwv":"ruby","ppi":440,"pxratio":3,"devicetype":4,"mccmnc":"311-480","connectiontype":5,"js":1,"h":852,"w":393,"geo":{"city":"Queens","type":2,"ipservice":3,"lat":40.7429,"lon":-73.9392,"long":-73.9392,"country":"USA","region":"ny","dma":"501","metro":"501","zip":"11101","utcoffset":-480,"ext":{"org":"Myanmar Broadband Telecom Co.","isp":"Myanmar Broadband Telecom Co."}},"ext":{"atts":3},"osv":"17.3.1","ua":"signal_UA","make":"Apple","model":"iPhone15,2","os":"iOS"},"imp":[{"id":"1","displaymanagerver":"3.1.0","clickbrowser":1,"displaymanager":"PubMatic_OpenWrap_SDK","instl":0,"secure":0,"tagid":"{NETWORK_PLACEMENT_ID}","bidfloor":0.01,"bidfloorcur":"USD","exp":14400,"rwdd":0,"video":{"companionad":[{"pos":0,"format":[{"w":300,"h":250}],"vcm":1}],"protocols":[2,3,5,6,7,8,11,12,13,14],"h":250,"w":300,"linearity":1,"pos":0,"boxingallowed":1,"placement":2,"mimes":["video/3gpp2","video/quicktime","video/mp4","video/x-m4v","video/3gpp"],"companiontype":[1,2,3],"delivery":[2],"startdelay":0,"playbackend":1,"api":[7]},"ext":{"skadn":{"sourceapp":"406719683","versions":["2.0","2.1","2.2","3.0","4.0"],"skadnetids":["cstr6suwn9.skadnetwork","7ug5zh24hu.skadnetwork","uw77j35x4d.skadnetwork","c6k4g5qg8m.skadnetwork","hs6bdukanm.skadnetwork","yclnxrl5pm.skadnetwork","3sh42y64q3.skadnetwork","cj5566h2ga.skadnetwork","klf5c3l5u5.skadnetwork","8s468mfl3y.skadnetwork","2u9pt9hc89.skadnetwork","7rz58n8ntl.skadnetwork","ppxm28t8ap.skadnetwork","mtkv5xtk9e.skadnetwork","cg4yq2srnc.skadnetwork","wzmmz9fp6w.skadnetwork","k674qkevps.skadnetwork","v72qych5uu.skadnetwork","578prtvx9j.skadnetwork","3rd42ekr43.skadnetwork","g28c52eehv.skadnetwork","2fnua5tdw4.skadnetwork","9nlqeag3gk.skadnetwork","5lm9lj6jb7.skadnetwork","97r2b46745.skadnetwork","e5fvkxwrpn.skadnetwork","4pfyvq9l8r.skadnetwork","tl55sbb4fm.skadnetwork","t38b2kh725.skadnetwork","prcb7njmu6.skadnetwork","mlmmfzh3r3.skadnetwork","9t245vhmpl.skadnetwork","9rd848q2bz.skadnetwork","4fzdc2evr5.skadnetwork","4468km3ulz.skadnetwork","m8dbw4sv7c.skadnetwork","ejvt5qm6ak.skadnetwork","5lm9lj6jb7.skadnetwork","44jx6755aq.skadnetwork","6g9af3uyq4.skadnetwork","u679fj5vs4.skadnetwork","rx5hdcabgc.skadnetwork","275upjj5gd.skadnetwork","p78axxw29g.skadnetwork"],"productpage":1,"version":"2.0"}}}],"user":{"data":[{"id":"1","name":"Publisher Passed","segment":[{"signal":"{BIDDING_SIGNAL}"}]}],"ext":{"gdpr":0}},"regs":{"coppa":1,"ext":{"ccpa":0,"gdpr":1,"consent":"0","tcf_consent_string":"{TCF_STRING}","gpp":"gpp_string","gpp_sid":[7],"us_privacy":"uspConsentString"}},"source":{"ext":{"schain":{"ver":"1.0","complete":1,"nodes":[{"asi":"applovin.com","sid":"53bf468f18c5a0e2b7d4e3f748c677c1","rid":"494dbe15a3ce08c54f4e456363f35a022247f997","hp":1}]},"omidpn":"Pubmatic","omidpv":"3.1.0"}},"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"clientconfig":1}}}}}}`),
		},
		{
			name: "replace imp.video from signal",
			args: args{
				signal:     `{"device":{"devicetype":4,"w":393,"h":852,"ifa":"F5BA1637-7156-4369-BA7E-3C45033D9F61","mccmnc":"311-480","js":1,"osv":"17.3.1","connectiontype":5,"os":"iOS","pxratio":3,"geo":{"lastfix":8,"lat":37.48773508935608,"utcoffset":-480,"lon":-122.22855027909678,"type":1},"language":"en","make":"Apple","ext":{"atts":3},"ua":"signal_UA","model":"iPhone15,2","carrier":"Verizon"},"source":{"ext":{"omidpn":"Pubmatic","omidpv":"3.1.0"}},"id":"CE204A0E-31C3-4D7F-A1A0-D34AF5ED1A7F","app":{"id":"406719683","paid":1,"keywords":"k1=v1","domain":"abc.com","bundle":"406719683","storeurl":"https://apps.apple.com/us/app/gasbuddy-find-pay-for-gas/id406719683","name":"GasBuddy","publisher":{"id":"160361"},"ver":"700.89.22927"},"ext":{"wrapper":{"sumry_disable":1,"clientconfig":1,"profileid":3422}},"imp":[{"secure":1,"tagid":"Mobile_iPhone_List_Screen_Bottom","banner":{"pos":0,"format":[{"w":300,"h":250}],"api":[5,6,7]},"id":"98D9318E-5276-402F-BAA4-CDBD8A364957","ext":{"skadn":{"sourceapp":"406719683","versions":["2.0","2.1","2.2","3.0","4.0"],"skadnetids":["cstr6suwn9.skadnetwork","7ug5zh24hu.skadnetwork","uw77j35x4d.skadnetwork","c6k4g5qg8m.skadnetwork","hs6bdukanm.skadnetwork","yclnxrl5pm.skadnetwork","3sh42y64q3.skadnetwork","cj5566h2ga.skadnetwork","klf5c3l5u5.skadnetwork","8s468mfl3y.skadnetwork","2u9pt9hc89.skadnetwork","7rz58n8ntl.skadnetwork","ppxm28t8ap.skadnetwork","mtkv5xtk9e.skadnetwork","cg4yq2srnc.skadnetwork","wzmmz9fp6w.skadnetwork","k674qkevps.skadnetwork","v72qych5uu.skadnetwork","578prtvx9j.skadnetwork","3rd42ekr43.skadnetwork","g28c52eehv.skadnetwork","2fnua5tdw4.skadnetwork","9nlqeag3gk.skadnetwork","5lm9lj6jb7.skadnetwork","97r2b46745.skadnetwork","e5fvkxwrpn.skadnetwork","4pfyvq9l8r.skadnetwork","tl55sbb4fm.skadnetwork","t38b2kh725.skadnetwork","prcb7njmu6.skadnetwork","mlmmfzh3r3.skadnetwork","9t245vhmpl.skadnetwork","9rd848q2bz.skadnetwork","4fzdc2evr5.skadnetwork","4468km3ulz.skadnetwork","m8dbw4sv7c.skadnetwork","ejvt5qm6ak.skadnetwork","5lm9lj6jb7.skadnetwork","44jx6755aq.skadnetwork","6g9af3uyq4.skadnetwork","u679fj5vs4.skadnetwork","rx5hdcabgc.skadnetwork","275upjj5gd.skadnetwork","p78axxw29g.skadnetwork"],"productpage":1,"version":"2.0"}},"displaymanagerver":"3.1.0","clickbrowser":1,"video":{"companionad":[{"pos":0,"format":[{"w":300,"h":250}],"vcm":1}],"protocols":[2,3,5,6,7,8,11,12,13,14],"h":250,"w":300,"linearity":1,"pos":0,"boxingallowed":1,"placement":2,"mimes":["video/3gpp2","video/quicktime","video/mp4","video/x-m4v","video/3gpp"],"companiontype":[1,2,3],"delivery":[2],"startdelay":0,"playbackend":1,"api":[7]},"displaymanager":"PubMatic_OpenWrap_SDK","instl":0}],"at":1,"cur":["USD"],"regs":{"coppa":1,"ext":{"ccpa":0,"gdpr":1,"gpp":"gpp_string","gpp_sid":[7],"us_privacy":"uspConsentString","consent":"0"}}}`,
				maxRequest: json.RawMessage(`{"id":"{BID_ID}","at":1,"bcat":["IAB26-4","IAB26-2","IAB25-6","IAB25-5","IAB25-4","IAB25-3","IAB25-1","IAB25-7","IAB8-18","IAB26-3","IAB26-1","IAB8-5","IAB25-2","IAB11-4"],"tmax":3000,"app":{"name":"DrawHappyAngel","ver":"0.5.4","bundle":"com.newstory.DrawHappyAngel","cat":["IAB9-30"],"id":"{NETWORK_APP_ID}","publisher":{"name":"New Story Inc.","ext":{"installed_sdk":{"id":"MOLOCO_BIDDING","sdk_version":{"major":1,"minor":0,"micro":0},"adapter_version":{"major":1,"minor":0,"micro":0}}}},"ext":{"orientation":1}},"device":{"ifa":"F5BA1637-7156-4369-BA7E-3C45033D9F61","ip":"38.158.207.171","carrier":"MYTEL","language":"en_US","hwv":"ruby","ppi":440,"pxratio":2.75,"devicetype":4,"connectiontype":2,"js":1,"h":2400,"w":1080,"geo":{"type":2,"ipservice":3,"lat":40.7429,"lon":-73.9392,"long":-73.9392,"city":"Queens","country":"USA","region":"ny","dma":"501","metro":"501","zip":"11101","ext":{"org":"Myanmar Broadband Telecom Co.","isp":"Myanmar Broadband Telecom Co."}},"ext":{},"osv":"13.0.0","ua":"Mozilla/5.0 (Linux; Android 13; 22101316C Build/TP1A.220624.014; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/120.0.6099.230 Mobile Safari/537.36","make":"xiaomi","model":"22101316c","os":"android"},"imp":[{"id":"1","displaymanager":"applovin_mediation","displaymanagerver":"11.8.2","instl":1,"secure":0,"tagid":"{NETWORK_PLACEMENT_ID}","exp":14400,"banner":{"id":"1","w":320,"h":480,"btype":[],"battr":[1,2,5,8,9,14,17],"pos":7,"format":[{"w":320,"h":480}]},"video":{"w":320,"h":480,"battr":[1,2,5,8,9,14,17],"mimes":["video/mp4","video/3gpp","video/3gpp2","video/x-m4v"],"placement":5,"pos":7,"minduration":5,"maxduration":60,"skipafter":5,"skipmin":0,"startdelay":0,"playbackmethod":[1],"linearity":1},"rwdd":0}],"user":{"data":[{"id":"1","name":"Publisher Passed","segment":[{"signal":"{BIDDING_SIGNAL}"}]}],"ext":{"gdpr":0}},"regs":{"ext":{"ccpa":0,"gdpr":1,"consent":"0","tcf_consent_string":"{TCF_STRING}"}},"source":{"ext":{"schain":{"ver":"1.0","complete":1,"nodes":[{"asi":"applovin.com","sid":"53bf468f18c5a0e2b7d4e3f748c677c1","rid":"494dbe15a3ce08c54f4e456363f35a022247f997","hp":1}]}}}}`),
			},
			wantMaxRequest: json.RawMessage(`{"id":"{BID_ID}","at":1,"bcat":["IAB26-4","IAB26-2","IAB25-6","IAB25-5","IAB25-4","IAB25-3","IAB25-1","IAB25-7","IAB8-18","IAB26-3","IAB26-1","IAB8-5","IAB25-2","IAB11-4"],"tmax":3000,"app":{"storeurl":"https://apps.apple.com/us/app/gasbuddy-find-pay-for-gas/id406719683","paid":1,"keywords":"k1=v1","domain":"abc.com","name":"DrawHappyAngel","ver":"0.5.4","bundle":"com.newstory.DrawHappyAngel","cat":["IAB9-30"],"id":"{NETWORK_APP_ID}","publisher":{"name":"New Story Inc.","ext":{"installed_sdk":{"id":"MOLOCO_BIDDING","sdk_version":{"major":1,"minor":0,"micro":0},"adapter_version":{"major":1,"minor":0,"micro":0}}}},"ext":{"orientation":1}},"device":{"ifa":"F5BA1637-7156-4369-BA7E-3C45033D9F61","ip":"38.158.207.171","carrier":"Verizon","language":"en","hwv":"ruby","ppi":440,"pxratio":3,"devicetype":4,"mccmnc":"311-480","connectiontype":5,"js":1,"h":852,"w":393,"geo":{"city":"Queens","type":2,"ipservice":3,"lat":40.7429,"lon":-73.9392,"long":-73.9392,"country":"USA","region":"ny","dma":"501","metro":"501","zip":"11101","utcoffset":-480,"ext":{"org":"Myanmar Broadband Telecom Co.","isp":"Myanmar Broadband Telecom Co."}},"ext":{"atts":3},"osv":"17.3.1","ua":"signal_UA","make":"Apple","model":"iPhone15,2","os":"iOS"},"imp":[{"id":"1","displaymanagerver":"3.1.0","clickbrowser":1,"displaymanager":"PubMatic_OpenWrap_SDK","instl":1,"secure":0,"tagid":"{NETWORK_PLACEMENT_ID}","exp":14400,"banner":{"id":"1","w":320,"h":480,"btype":[],"api":[5,6,7],"battr":[1,2,5,8,9,14,17],"pos":7,"format":[{"w":320,"h":480}]},"video":{"companionad":[{"pos":0,"format":[{"w":300,"h":250}],"vcm":1}],"protocols":[2,3,5,6,7,8,11,12,13,14],"h":250,"w":300,"linearity":1,"pos":0,"boxingallowed":1,"placement":2,"mimes":["video/3gpp2","video/quicktime","video/mp4","video/x-m4v","video/3gpp"],"companiontype":[1,2,3],"delivery":[2],"startdelay":0,"playbackend":1,"api":[7]},"rwdd":0,"ext":{"skadn":{"sourceapp":"406719683","versions":["2.0","2.1","2.2","3.0","4.0"],"skadnetids":["cstr6suwn9.skadnetwork","7ug5zh24hu.skadnetwork","uw77j35x4d.skadnetwork","c6k4g5qg8m.skadnetwork","hs6bdukanm.skadnetwork","yclnxrl5pm.skadnetwork","3sh42y64q3.skadnetwork","cj5566h2ga.skadnetwork","klf5c3l5u5.skadnetwork","8s468mfl3y.skadnetwork","2u9pt9hc89.skadnetwork","7rz58n8ntl.skadnetwork","ppxm28t8ap.skadnetwork","mtkv5xtk9e.skadnetwork","cg4yq2srnc.skadnetwork","wzmmz9fp6w.skadnetwork","k674qkevps.skadnetwork","v72qych5uu.skadnetwork","578prtvx9j.skadnetwork","3rd42ekr43.skadnetwork","g28c52eehv.skadnetwork","2fnua5tdw4.skadnetwork","9nlqeag3gk.skadnetwork","5lm9lj6jb7.skadnetwork","97r2b46745.skadnetwork","e5fvkxwrpn.skadnetwork","4pfyvq9l8r.skadnetwork","tl55sbb4fm.skadnetwork","t38b2kh725.skadnetwork","prcb7njmu6.skadnetwork","mlmmfzh3r3.skadnetwork","9t245vhmpl.skadnetwork","9rd848q2bz.skadnetwork","4fzdc2evr5.skadnetwork","4468km3ulz.skadnetwork","m8dbw4sv7c.skadnetwork","ejvt5qm6ak.skadnetwork","5lm9lj6jb7.skadnetwork","44jx6755aq.skadnetwork","6g9af3uyq4.skadnetwork","u679fj5vs4.skadnetwork","rx5hdcabgc.skadnetwork","275upjj5gd.skadnetwork","p78axxw29g.skadnetwork"],"productpage":1,"version":"2.0"}}}],"user":{"data":[{"id":"1","name":"Publisher Passed","segment":[{"signal":"{BIDDING_SIGNAL}"}]}],"ext":{"gdpr":0}},"regs":{"coppa":1,"ext":{"ccpa":0,"gdpr":1,"consent":"0","tcf_consent_string":"{TCF_STRING}","gpp":"gpp_string","gpp_sid":[7],"us_privacy":"uspConsentString"}},"source":{"ext":{"schain":{"ver":"1.0","complete":1,"nodes":[{"asi":"applovin.com","sid":"53bf468f18c5a0e2b7d4e3f748c677c1","rid":"494dbe15a3ce08c54f4e456363f35a022247f997","hp":1}]},"omidpn":"Pubmatic","omidpv":"3.1.0"}},"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"clientconfig":1}}}}}}`),
		},
		{
			name: "strictly replace app.storeurl from signal",
			args: args{
				signal:     `{"device":{"devicetype":4,"w":393,"h":852,"ifa":"F5BA1637-7156-4369-BA7E-3C45033D9F61","mccmnc":"311-480","js":1,"osv":"17.3.1","connectiontype":5,"os":"iOS","pxratio":3,"geo":{"lastfix":8,"lat":37.48773508935608,"utcoffset":-480,"lon":-122.22855027909678,"type":1},"language":"en","make":"Apple","ext":{"atts":3},"ua":"signal_UA","model":"iPhone15,2","carrier":"Verizon"},"source":{"ext":{"omidpn":"Pubmatic","omidpv":"3.1.0"}},"id":"CE204A0E-31C3-4D7F-A1A0-D34AF5ED1A7F","app":{"id":"406719683","paid":1,"keywords":"k1=v1","domain":"abc.com","bundle":"406719683","name":"GasBuddy","publisher":{"id":"160361"},"ver":"700.89.22927"},"ext":{"wrapper":{"sumry_disable":1,"clientconfig":1,"profileid":3422}},"imp":[{"secure":1,"tagid":"Mobile_iPhone_List_Screen_Bottom","banner":{"pos":0,"format":[{"w":300,"h":250}],"api":[5,6,7]},"id":"98D9318E-5276-402F-BAA4-CDBD8A364957","ext":{"skadn":{"sourceapp":"406719683","versions":["2.0","2.1","2.2","3.0","4.0"],"skadnetids":["cstr6suwn9.skadnetwork","7ug5zh24hu.skadnetwork","uw77j35x4d.skadnetwork","c6k4g5qg8m.skadnetwork","hs6bdukanm.skadnetwork","yclnxrl5pm.skadnetwork","3sh42y64q3.skadnetwork","cj5566h2ga.skadnetwork","klf5c3l5u5.skadnetwork","8s468mfl3y.skadnetwork","2u9pt9hc89.skadnetwork","7rz58n8ntl.skadnetwork","ppxm28t8ap.skadnetwork","mtkv5xtk9e.skadnetwork","cg4yq2srnc.skadnetwork","wzmmz9fp6w.skadnetwork","k674qkevps.skadnetwork","v72qych5uu.skadnetwork","578prtvx9j.skadnetwork","3rd42ekr43.skadnetwork","g28c52eehv.skadnetwork","2fnua5tdw4.skadnetwork","9nlqeag3gk.skadnetwork","5lm9lj6jb7.skadnetwork","97r2b46745.skadnetwork","e5fvkxwrpn.skadnetwork","4pfyvq9l8r.skadnetwork","tl55sbb4fm.skadnetwork","t38b2kh725.skadnetwork","prcb7njmu6.skadnetwork","mlmmfzh3r3.skadnetwork","9t245vhmpl.skadnetwork","9rd848q2bz.skadnetwork","4fzdc2evr5.skadnetwork","4468km3ulz.skadnetwork","m8dbw4sv7c.skadnetwork","ejvt5qm6ak.skadnetwork","5lm9lj6jb7.skadnetwork","44jx6755aq.skadnetwork","6g9af3uyq4.skadnetwork","u679fj5vs4.skadnetwork","rx5hdcabgc.skadnetwork","275upjj5gd.skadnetwork","p78axxw29g.skadnetwork"],"productpage":1,"version":"2.0"}},"displaymanagerver":"3.1.0","clickbrowser":1,"video":{"companionad":[{"pos":0,"format":[{"w":300,"h":250}],"vcm":1}],"protocols":[2,3,5,6,7,8,11,12,13,14],"h":250,"w":300,"linearity":1,"pos":0,"boxingallowed":1,"placement":2,"mimes":["video/3gpp2","video/quicktime","video/mp4","video/x-m4v","video/3gpp"],"companiontype":[1,2,3],"delivery":[2],"startdelay":0,"playbackend":1,"api":[7]},"displaymanager":"PubMatic_OpenWrap_SDK","instl":0}],"at":1,"cur":["USD"],"regs":{"coppa":1,"ext":{"ccpa":0,"gdpr":1,"gpp":"gpp_string","gpp_sid":[7],"us_privacy":"uspConsentString","consent":"0"}}}`,
				maxRequest: json.RawMessage(`{"id":"{BID_ID}","at":1,"bcat":["IAB26-4","IAB26-2","IAB25-6","IAB25-5","IAB25-4","IAB25-3","IAB25-1","IAB25-7","IAB8-18","IAB26-3","IAB26-1","IAB8-5","IAB25-2","IAB11-4"],"tmax":3000,"app":{"name":"DrawHappyAngel","ver":"0.5.4","bundle":"com.newstory.DrawHappyAngel","cat":["IAB9-30"],"id":"{NETWORK_APP_ID}","publisher":{"name":"New Story Inc.","ext":{"installed_sdk":{"id":"MOLOCO_BIDDING","sdk_version":{"major":1,"minor":0,"micro":0},"adapter_version":{"major":1,"minor":0,"micro":0}}}},"ext":{"orientation":1}},"device":{"ifa":"497a10d6-c4dd-4e04-a986-c32b7180d462","ip":"38.158.207.171","carrier":"MYTEL","language":"en_US","hwv":"ruby","ppi":440,"pxratio":2.75,"devicetype":4,"connectiontype":2,"js":1,"h":2400,"w":1080,"geo":{"type":2,"ipservice":3,"lat":40.7429,"lon":-73.9392,"long":-73.9392,"city":"Queens","country":"USA","region":"ny","dma":"501","metro":"501","zip":"11101","ext":{"org":"Myanmar Broadband Telecom Co.","isp":"Myanmar Broadband Telecom Co."}},"ext":{},"osv":"13.0.0","ua":"Mozilla/5.0 (Linux; Android 13; 22101316C Build/TP1A.220624.014; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/120.0.6099.230 Mobile Safari/537.36","make":"xiaomi","model":"22101316c","os":"android"},"imp":[{"id":"1","displaymanager":"applovin_mediation","displaymanagerver":"11.8.2","instl":1,"secure":0,"tagid":"{NETWORK_PLACEMENT_ID}","exp":14400,"banner":{"id":"1","w":320,"h":480,"btype":[],"battr":[1,2,5,8,9,14,17],"pos":7,"format":[{"w":320,"h":480}]},"video":{"w":320,"h":480,"battr":[1,2,5,8,9,14,17],"mimes":["video/mp4","video/3gpp","video/3gpp2","video/x-m4v"],"placement":5,"pos":7,"minduration":5,"maxduration":60,"skipafter":5,"skipmin":0,"startdelay":0,"playbackmethod":[1],"linearity":1},"rwdd":0}],"user":{"data":[{"id":"1","name":"Publisher Passed","segment":[{"signal":"{BIDDING_SIGNAL}"}]}],"ext":{"gdpr":0}},"regs":{"ext":{"ccpa":0,"gdpr":1,"consent":"0","tcf_consent_string":"{TCF_STRING}"}},"source":{"ext":{"schain":{"ver":"1.0","complete":1,"nodes":[{"asi":"applovin.com","sid":"53bf468f18c5a0e2b7d4e3f748c677c1","rid":"494dbe15a3ce08c54f4e456363f35a022247f997","hp":1}]}}}}`),
			},
			wantMaxRequest: json.RawMessage(`{"id":"{BID_ID}","at":1,"bcat":["IAB26-4","IAB26-2","IAB25-6","IAB25-5","IAB25-4","IAB25-3","IAB25-1","IAB25-7","IAB8-18","IAB26-3","IAB26-1","IAB8-5","IAB25-2","IAB11-4"],"tmax":3000,"app":{"paid":1,"keywords":"k1=v1","domain":"abc.com","name":"DrawHappyAngel","ver":"0.5.4","bundle":"com.newstory.DrawHappyAngel","cat":["IAB9-30"],"id":"{NETWORK_APP_ID}","publisher":{"name":"New Story Inc.","ext":{"installed_sdk":{"id":"MOLOCO_BIDDING","sdk_version":{"major":1,"minor":0,"micro":0},"adapter_version":{"major":1,"minor":0,"micro":0}}}},"ext":{"orientation":1}},"device":{"ifa":"F5BA1637-7156-4369-BA7E-3C45033D9F61","ip":"38.158.207.171","carrier":"Verizon","language":"en","hwv":"ruby","ppi":440,"pxratio":3,"devicetype":4,"mccmnc":"311-480","connectiontype":5,"js":1,"h":852,"w":393,"geo":{"city":"Queens","type":2,"ipservice":3,"lat":40.7429,"lon":-73.9392,"long":-73.9392,"country":"USA","region":"ny","dma":"501","metro":"501","zip":"11101","utcoffset":-480,"ext":{"org":"Myanmar Broadband Telecom Co.","isp":"Myanmar Broadband Telecom Co."}},"ext":{"atts":3},"osv":"17.3.1","ua":"signal_UA","make":"Apple","model":"iPhone15,2","os":"iOS"},"imp":[{"id":"1","displaymanagerver":"3.1.0","clickbrowser":1,"displaymanager":"PubMatic_OpenWrap_SDK","instl":1,"secure":0,"tagid":"{NETWORK_PLACEMENT_ID}","exp":14400,"banner":{"id":"1","w":320,"h":480,"btype":[],"api":[5,6,7],"battr":[1,2,5,8,9,14,17],"pos":7,"format":[{"w":320,"h":480}]},"video":{"companionad":[{"pos":0,"format":[{"w":300,"h":250}],"vcm":1}],"protocols":[2,3,5,6,7,8,11,12,13,14],"h":250,"w":300,"linearity":1,"pos":0,"boxingallowed":1,"placement":2,"mimes":["video/3gpp2","video/quicktime","video/mp4","video/x-m4v","video/3gpp"],"companiontype":[1,2,3],"delivery":[2],"startdelay":0,"playbackend":1,"api":[7]},"rwdd":0,"ext":{"skadn":{"sourceapp":"406719683","versions":["2.0","2.1","2.2","3.0","4.0"],"skadnetids":["cstr6suwn9.skadnetwork","7ug5zh24hu.skadnetwork","uw77j35x4d.skadnetwork","c6k4g5qg8m.skadnetwork","hs6bdukanm.skadnetwork","yclnxrl5pm.skadnetwork","3sh42y64q3.skadnetwork","cj5566h2ga.skadnetwork","klf5c3l5u5.skadnetwork","8s468mfl3y.skadnetwork","2u9pt9hc89.skadnetwork","7rz58n8ntl.skadnetwork","ppxm28t8ap.skadnetwork","mtkv5xtk9e.skadnetwork","cg4yq2srnc.skadnetwork","wzmmz9fp6w.skadnetwork","k674qkevps.skadnetwork","v72qych5uu.skadnetwork","578prtvx9j.skadnetwork","3rd42ekr43.skadnetwork","g28c52eehv.skadnetwork","2fnua5tdw4.skadnetwork","9nlqeag3gk.skadnetwork","5lm9lj6jb7.skadnetwork","97r2b46745.skadnetwork","e5fvkxwrpn.skadnetwork","4pfyvq9l8r.skadnetwork","tl55sbb4fm.skadnetwork","t38b2kh725.skadnetwork","prcb7njmu6.skadnetwork","mlmmfzh3r3.skadnetwork","9t245vhmpl.skadnetwork","9rd848q2bz.skadnetwork","4fzdc2evr5.skadnetwork","4468km3ulz.skadnetwork","m8dbw4sv7c.skadnetwork","ejvt5qm6ak.skadnetwork","5lm9lj6jb7.skadnetwork","44jx6755aq.skadnetwork","6g9af3uyq4.skadnetwork","u679fj5vs4.skadnetwork","rx5hdcabgc.skadnetwork","275upjj5gd.skadnetwork","p78axxw29g.skadnetwork"],"productpage":1,"version":"2.0"}}}],"user":{"data":[{"id":"1","name":"Publisher Passed","segment":[{"signal":"{BIDDING_SIGNAL}"}]}],"ext":{"gdpr":0}},"regs":{"coppa":1,"ext":{"ccpa":0,"gdpr":1,"consent":"0","tcf_consent_string":"{TCF_STRING}","gpp":"gpp_string","gpp_sid":[7],"us_privacy":"uspConsentString"}},"source":{"ext":{"schain":{"ver":"1.0","complete":1,"nodes":[{"asi":"applovin.com","sid":"53bf468f18c5a0e2b7d4e3f748c677c1","rid":"494dbe15a3ce08c54f4e456363f35a022247f997","hp":1}]},"omidpn":"Pubmatic","omidpv":"3.1.0"}},"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"clientconfig":1}}}}}}`),
		},
		{
			name: "prefer app.storeurl from request over signal",
			args: args{
				signal:     `{"device":{"devicetype":4,"w":393,"h":852,"ifa":"F5BA1637-7156-4369-BA7E-3C45033D9F61","mccmnc":"311-480","js":1,"osv":"17.3.1","connectiontype":5,"os":"iOS","pxratio":3,"geo":{"lastfix":8,"lat":37.48773508935608,"utcoffset":-480,"lon":-122.22855027909678,"type":1},"language":"en","make":"Apple","ext":{"atts":3},"ua":"signal_UA","model":"iPhone15,2","carrier":"Verizon"},"source":{"ext":{"omidpn":"Pubmatic","omidpv":"3.1.0"}},"id":"CE204A0E-31C3-4D7F-A1A0-D34AF5ED1A7F","app":{"id":"406719683","paid":1,"keywords":"k1=v1","domain":"abc.com","bundle":"406719683","name":"GasBuddy","publisher":{"id":"160361"},"ver":"700.89.22927"},"ext":{"wrapper":{"sumry_disable":1,"clientconfig":1,"profileid":3422}},"imp":[{"secure":1,"tagid":"Mobile_iPhone_List_Screen_Bottom","banner":{"pos":0,"format":[{"w":300,"h":250}],"api":[5,6,7]},"id":"98D9318E-5276-402F-BAA4-CDBD8A364957","ext":{"skadn":{"sourceapp":"406719683","versions":["2.0","2.1","2.2","3.0","4.0"],"skadnetids":["cstr6suwn9.skadnetwork","7ug5zh24hu.skadnetwork","uw77j35x4d.skadnetwork","c6k4g5qg8m.skadnetwork","hs6bdukanm.skadnetwork","yclnxrl5pm.skadnetwork","3sh42y64q3.skadnetwork","cj5566h2ga.skadnetwork","klf5c3l5u5.skadnetwork","8s468mfl3y.skadnetwork","2u9pt9hc89.skadnetwork","7rz58n8ntl.skadnetwork","ppxm28t8ap.skadnetwork","mtkv5xtk9e.skadnetwork","cg4yq2srnc.skadnetwork","wzmmz9fp6w.skadnetwork","k674qkevps.skadnetwork","v72qych5uu.skadnetwork","578prtvx9j.skadnetwork","3rd42ekr43.skadnetwork","g28c52eehv.skadnetwork","2fnua5tdw4.skadnetwork","9nlqeag3gk.skadnetwork","5lm9lj6jb7.skadnetwork","97r2b46745.skadnetwork","e5fvkxwrpn.skadnetwork","4pfyvq9l8r.skadnetwork","tl55sbb4fm.skadnetwork","t38b2kh725.skadnetwork","prcb7njmu6.skadnetwork","mlmmfzh3r3.skadnetwork","9t245vhmpl.skadnetwork","9rd848q2bz.skadnetwork","4fzdc2evr5.skadnetwork","4468km3ulz.skadnetwork","m8dbw4sv7c.skadnetwork","ejvt5qm6ak.skadnetwork","5lm9lj6jb7.skadnetwork","44jx6755aq.skadnetwork","6g9af3uyq4.skadnetwork","u679fj5vs4.skadnetwork","rx5hdcabgc.skadnetwork","275upjj5gd.skadnetwork","p78axxw29g.skadnetwork"],"productpage":1,"version":"2.0"}},"displaymanagerver":"3.1.0","clickbrowser":1,"video":{"companionad":[{"pos":0,"format":[{"w":300,"h":250}],"vcm":1}],"protocols":[2,3,5,6,7,8,11,12,13,14],"h":250,"w":300,"linearity":1,"pos":0,"boxingallowed":1,"placement":2,"mimes":["video/3gpp2","video/quicktime","video/mp4","video/x-m4v","video/3gpp"],"companiontype":[1,2,3],"delivery":[2],"startdelay":0,"playbackend":1,"api":[7]},"displaymanager":"PubMatic_OpenWrap_SDK","instl":0}],"at":1,"cur":["USD"],"regs":{"coppa":1,"ext":{"ccpa":0,"gdpr":1,"gpp":"gpp_string","gpp_sid":[7],"us_privacy":"uspConsentString","consent":"0"}}}`,
				maxRequest: json.RawMessage(`{"id":"{BID_ID}","at":1,"bcat":["IAB26-4","IAB26-2","IAB25-6","IAB25-5","IAB25-4","IAB25-3","IAB25-1","IAB25-7","IAB8-18","IAB26-3","IAB26-1","IAB8-5","IAB25-2","IAB11-4"],"tmax":3000,"app":{"storeurl":"https://apps.apple.com/us/app/gasbuddy-find-pay-for-gas/id406719683","name":"DrawHappyAngel","ver":"0.5.4","bundle":"com.newstory.DrawHappyAngel","cat":["IAB9-30"],"id":"{NETWORK_APP_ID}","publisher":{"name":"New Story Inc.","ext":{"installed_sdk":{"id":"MOLOCO_BIDDING","sdk_version":{"major":1,"minor":0,"micro":0},"adapter_version":{"major":1,"minor":0,"micro":0}}}},"ext":{"orientation":1}},"device":{"ifa":"497a10d6-c4dd-4e04-a986-c32b7180d462","ip":"38.158.207.171","carrier":"MYTEL","language":"en_US","hwv":"ruby","ppi":440,"pxratio":2.75,"devicetype":4,"connectiontype":2,"js":1,"h":2400,"w":1080,"geo":{"type":2,"ipservice":3,"lat":40.7429,"lon":-73.9392,"long":-73.9392,"city":"Queens","country":"USA","region":"ny","dma":"501","metro":"501","zip":"11101","ext":{"org":"Myanmar Broadband Telecom Co.","isp":"Myanmar Broadband Telecom Co."}},"ext":{},"osv":"13.0.0","ua":"Mozilla/5.0 (Linux; Android 13; 22101316C Build/TP1A.220624.014; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/120.0.6099.230 Mobile Safari/537.36","make":"xiaomi","model":"22101316c","os":"android"},"imp":[{"id":"1","displaymanager":"applovin_mediation","displaymanagerver":"11.8.2","instl":1,"secure":0,"tagid":"{NETWORK_PLACEMENT_ID}","exp":14400,"banner":{"id":"1","w":320,"h":480,"btype":[],"battr":[1,2,5,8,9,14,17],"pos":7,"format":[{"w":320,"h":480}]},"video":{"w":320,"h":480,"battr":[1,2,5,8,9,14,17],"mimes":["video/mp4","video/3gpp","video/3gpp2","video/x-m4v"],"placement":5,"pos":7,"minduration":5,"maxduration":60,"skipafter":5,"skipmin":0,"startdelay":0,"playbackmethod":[1],"linearity":1},"rwdd":0}],"user":{"data":[{"id":"1","name":"Publisher Passed","segment":[{"signal":"{BIDDING_SIGNAL}"}]}],"ext":{"gdpr":0}},"regs":{"ext":{"ccpa":0,"gdpr":1,"consent":"0","tcf_consent_string":"{TCF_STRING}"}},"source":{"ext":{"schain":{"ver":"1.0","complete":1,"nodes":[{"asi":"applovin.com","sid":"53bf468f18c5a0e2b7d4e3f748c677c1","rid":"494dbe15a3ce08c54f4e456363f35a022247f997","hp":1}]}}}}`),
			},
			wantMaxRequest: json.RawMessage(`{"id":"{BID_ID}","at":1,"bcat":["IAB26-4","IAB26-2","IAB25-6","IAB25-5","IAB25-4","IAB25-3","IAB25-1","IAB25-7","IAB8-18","IAB26-3","IAB26-1","IAB8-5","IAB25-2","IAB11-4"],"tmax":3000,"app":{"storeurl":"https://apps.apple.com/us/app/gasbuddy-find-pay-for-gas/id406719683","paid":1,"keywords":"k1=v1","domain":"abc.com","name":"DrawHappyAngel","ver":"0.5.4","bundle":"com.newstory.DrawHappyAngel","cat":["IAB9-30"],"id":"{NETWORK_APP_ID}","publisher":{"name":"New Story Inc.","ext":{"installed_sdk":{"id":"MOLOCO_BIDDING","sdk_version":{"major":1,"minor":0,"micro":0},"adapter_version":{"major":1,"minor":0,"micro":0}}}},"ext":{"orientation":1}},"device":{"ifa":"F5BA1637-7156-4369-BA7E-3C45033D9F61","ip":"38.158.207.171","carrier":"Verizon","language":"en","hwv":"ruby","ppi":440,"pxratio":3,"devicetype":4,"mccmnc":"311-480","connectiontype":5,"js":1,"h":852,"w":393,"geo":{"city":"Queens","type":2,"ipservice":3,"lat":40.7429,"lon":-73.9392,"long":-73.9392,"country":"USA","region":"ny","dma":"501","metro":"501","zip":"11101","utcoffset":-480,"ext":{"org":"Myanmar Broadband Telecom Co.","isp":"Myanmar Broadband Telecom Co."}},"ext":{"atts":3},"osv":"17.3.1","ua":"signal_UA","make":"Apple","model":"iPhone15,2","os":"iOS"},"imp":[{"id":"1","displaymanagerver":"3.1.0","clickbrowser":1,"displaymanager":"PubMatic_OpenWrap_SDK","instl":1,"secure":0,"tagid":"{NETWORK_PLACEMENT_ID}","exp":14400,"banner":{"id":"1","w":320,"h":480,"btype":[],"api":[5,6,7],"battr":[1,2,5,8,9,14,17],"pos":7,"format":[{"w":320,"h":480}]},"video":{"companionad":[{"pos":0,"format":[{"w":300,"h":250}],"vcm":1}],"protocols":[2,3,5,6,7,8,11,12,13,14],"h":250,"w":300,"linearity":1,"pos":0,"boxingallowed":1,"placement":2,"mimes":["video/3gpp2","video/quicktime","video/mp4","video/x-m4v","video/3gpp"],"companiontype":[1,2,3],"delivery":[2],"startdelay":0,"playbackend":1,"api":[7]},"rwdd":0,"ext":{"skadn":{"sourceapp":"406719683","versions":["2.0","2.1","2.2","3.0","4.0"],"skadnetids":["cstr6suwn9.skadnetwork","7ug5zh24hu.skadnetwork","uw77j35x4d.skadnetwork","c6k4g5qg8m.skadnetwork","hs6bdukanm.skadnetwork","yclnxrl5pm.skadnetwork","3sh42y64q3.skadnetwork","cj5566h2ga.skadnetwork","klf5c3l5u5.skadnetwork","8s468mfl3y.skadnetwork","2u9pt9hc89.skadnetwork","7rz58n8ntl.skadnetwork","ppxm28t8ap.skadnetwork","mtkv5xtk9e.skadnetwork","cg4yq2srnc.skadnetwork","wzmmz9fp6w.skadnetwork","k674qkevps.skadnetwork","v72qych5uu.skadnetwork","578prtvx9j.skadnetwork","3rd42ekr43.skadnetwork","g28c52eehv.skadnetwork","2fnua5tdw4.skadnetwork","9nlqeag3gk.skadnetwork","5lm9lj6jb7.skadnetwork","97r2b46745.skadnetwork","e5fvkxwrpn.skadnetwork","4pfyvq9l8r.skadnetwork","tl55sbb4fm.skadnetwork","t38b2kh725.skadnetwork","prcb7njmu6.skadnetwork","mlmmfzh3r3.skadnetwork","9t245vhmpl.skadnetwork","9rd848q2bz.skadnetwork","4fzdc2evr5.skadnetwork","4468km3ulz.skadnetwork","m8dbw4sv7c.skadnetwork","ejvt5qm6ak.skadnetwork","5lm9lj6jb7.skadnetwork","44jx6755aq.skadnetwork","6g9af3uyq4.skadnetwork","u679fj5vs4.skadnetwork","rx5hdcabgc.skadnetwork","275upjj5gd.skadnetwork","p78axxw29g.skadnetwork"],"productpage":1,"version":"2.0"}}}],"user":{"data":[{"id":"1","name":"Publisher Passed","segment":[{"signal":"{BIDDING_SIGNAL}"}]}],"ext":{"gdpr":0}},"regs":{"coppa":1,"ext":{"ccpa":0,"gdpr":1,"consent":"0","tcf_consent_string":"{TCF_STRING}","gpp":"gpp_string","gpp_sid":[7],"us_privacy":"uspConsentString"}},"source":{"ext":{"schain":{"ver":"1.0","complete":1,"nodes":[{"asi":"applovin.com","sid":"53bf468f18c5a0e2b7d4e3f748c677c1","rid":"494dbe15a3ce08c54f4e456363f35a022247f997","hp":1}]},"omidpn":"Pubmatic","omidpv":"3.1.0"}},"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"clientconfig":1}}}}}}`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var maxRequest openrtb2.BidRequest
			if err := json.Unmarshal(tt.args.maxRequest, &maxRequest); err != nil {
				t.Errorf("Unmarshal Faild for Incoming MaxRequest, Error: %s", err)
			}

			signalData := &openrtb2.BidRequest{}
			if err := json.Unmarshal([]byte(tt.args.signal), &signalData); err != nil {
				t.Errorf("Unmarshal Faild for Incoming MaxRequest, Error: %s", err)
			}

			var expectedMaxRequest openrtb2.BidRequest
			addSignalDataInRequest(signalData, &maxRequest)
			if err := json.Unmarshal(tt.wantMaxRequest, &expectedMaxRequest); err != nil {
				t.Errorf("Unmarshal Faild for Expected MaxRequest, Error: %s", err)
			}
			assert.Equal(t, expectedMaxRequest, maxRequest, tt.name)
		})
	}
}

func TestGetSignalData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)
	type args struct {
		requestBody []byte
		rctx        models.RequestCtx
	}
	tests := []struct {
		name  string
		args  args
		setup func()
		want  *openrtb2.BidRequest
	}{
		{
			name: "incorrect json body",
			args: args{
				requestBody: []byte(`{"id":"123","user":Passed","segment":[{"signal":{BIDDING_SIGNA}]}],"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"profileid":1234}}}}}}}`),
				rctx: models.RequestCtx{
					MetricsEngine: mockEngine,
				},
			},
			setup: func() {
				mockEngine.EXPECT().RecordSignalDataStatus("", "", models.MissingSignal)
			},
			want: nil,
		},
		{
			name: "signal parsing fail",
			args: args{
				requestBody: []byte(`{"id":"123","app":{"publisher":{"id":"5890"}},"user":{"data":[{"id":"1","name":"Publisher Passed","segment":[{"signal":"{BIDDING_SIGNAL}"}]}]},"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"profileid":1234}}}}}}`),
				rctx: models.RequestCtx{
					MetricsEngine: mockEngine,
					ProfileIDStr:  "1234",
				},
			},
			setup: func() {
				mockEngine.EXPECT().RecordSignalDataStatus("5890", "1234", models.InvalidSignal)
			},
			want: nil,
		},
		{
			name: "single user.data with signal with incorrect signal",
			args: args{
				requestBody: []byte(`{"id":"123","user":{"data":[{"id":"1","name":"Publisher Passed","segment":[{"signal":{}}]}]},"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"profileid":1234}}}}}}`),
				rctx: models.RequestCtx{
					MetricsEngine: mockEngine,
					ProfileIDStr:  "1234",
				},
			},
			setup: func() {
				mockEngine.EXPECT().RecordSignalDataStatus("", "1234", models.InvalidSignal)
			},
			want: nil,
		},
		{
			name: "single user.data with signal",
			args: args{
				requestBody: []byte(`{"id":"123","user":{"data":[{"id":"1","name":"Publisher Passed","segment":[{"signal":"{\"device\":{\"devicetype\":4,\"w\":393,\"h\":852}}"}]}],"ext":{"gdpr":0}}}`),
				rctx: models.RequestCtx{
					MetricsEngine: mockEngine,
				},
			},
			want: &openrtb2.BidRequest{
				Device: &openrtb2.Device{
					DeviceType: 4,
					W:          393,
					H:          852,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			got := getSignalData(tt.args.requestBody, tt.args.rctx)
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}

func TestUpdateAppLovinMaxRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)
	type args struct {
		requestBody []byte
		rctx        models.RequestCtx
	}
	tests := []struct {
		name  string
		args  args
		setup func()
		want  []byte
	}{
		{
			name: "signal not present",
			args: args{
				requestBody: []byte(`{"id":"1","app":{"publisher":{"id":"5890"}},"user":{"data":[{"segment":[{}]}]},"imp":[{"displaymanager":"applovin_mediation","displaymanagerver":"2.3"}],"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"profileid":1234}}}}}}`),
				rctx: models.RequestCtx{
					MetricsEngine: mockEngine,
				},
			},
			setup: func() {
				mockEngine.EXPECT().RecordSignalDataStatus("5890", "1234", models.MissingSignal)
			},
			want: []byte(`{"id":"1","app":{"publisher":{"id":"5890"}},"user":{"data":[{"segment":[{}]}]},"imp":[{"displaymanager":"PubMatic_OpenWrap_SDK"}],"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"profileid":1234}}}}}}`),
		},
		{
			name: "invalid request body",
			args: args{
				requestBody: []byte(`{"id","user":{"data":[{"segment":[{"signal":"{}"}]}]}}`),
			},
			want: []byte(`{"id","user":{"data":[{"segment":[{"signal":"{}"}]}]}}`),
		},
		{
			name: "update maxrequest body from signal",
			args: args{
				requestBody: []byte(`{"id":"test-case-1","at":1,"bcat":["IAB26-4","IAB26-2","IAB25-6","IAB25-5","IAB25-4","IAB25-3","IAB25-1","IAB25-7","IAB8-18","IAB26-3","IAB26-1","IAB8-5","IAB25-2","IAB11-4"],"tmax":1000,"app":{"publisher":{"name":"New Story Inc.","id":"5890","ext":{"installed_sdk":{"id":"MOLOCO_BIDDING","sdk_version":{"major":1,"minor":0,"micro":0},"adapter_version":{"major":1,"minor":0,"micro":0}}}},"paid":0,"name":"DrawHappyAngel","ver":"0.5.4","bundle":"com.newstory.DrawHappyAngel","cat":["IAB9-30"],"id":"1234567","ext":{"orientation":1}},"device":{"ifa":"497a10d6-c4dd-4e04-a986-c32b7180d462","ip":"38.158.207.171","carrier":"MYTEL","language":"en_US","hwv":"ruby","ppi":440,"pxratio":2.75,"devicetype":4,"connectiontype":2,"js":1,"h":2400,"w":1080,"geo":{"type":2,"ipservice":3,"lat":40.7429,"lon":-73.9392,"long":-73.9392,"city":"Queens","country":"USA","region":"ny","dma":"501","metro":"501","zip":"11101","ext":{"org":"Myanmar Broadband Telecom Co.","isp":"Myanmar Broadband Telecom Co."}},"ext":{},"osv":"13.0.0","ua":"Mozilla/5.0 (Linux; Android 13; 22101316C Build/TP1A.220624.014; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/120.0.6099.230 Mobile Safari/537.36","make":"xiaomi","model":"22101316c","os":"android"},"imp":[{"id":"1","displaymanager":"applovin_mediation","displaymanagerver":"11.8.2","instl":0,"secure":0,"tagid":"/43743431/DMDemo","bidfloor":0.01,"bidfloorcur":"USD","exp":14400,"banner":{"format":[{"w":728,"h":90},{"w":300,"h":250}],"w":700,"h":900},"video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"native":{"request":"native_request","ver":"1.2"},"rwdd":0}],"user":{"data":[{"id":"1","name":"Publisher Passed","segment":[{"signal":"{\"id\":\"95d6643c-3da6-40a2-b9ca-12279393ffbf\",\"at\":1,\"tmax\":500,\"cur\":[\"USD\"],\"imp\":[{\"id\":\"imp176227948\",\"clickbrowser\":0,\"displaymanager\":\"PubMatic_OpenBid_SDK\",\"displaymanagerver\":\"1.4.0\",\"tagid\":\"/43743431/DMDemo\",\"secure\":0,\"banner\":{\"pos\":7,\"format\":[{\"w\":300,\"h\":250}],\"api\":[5,6,7]},\"instl\":1}],\"app\":{\"paid\":4,\"name\":\"OpenWrapperSample\",\"bundle\":\"com.pubmatic.openbid.app\",\"storeurl\":\"https://itunes.apple.com/us/app/pubmatic-sdk-app/id1175273098\",\"ver\":\"1.0\",\"publisher\":{\"id\":\"5890\"}},\"device\":{\"geo\":{\"type\":1,\"lat\":37.421998333333335,\"lon\":-122.08400000000002},\"pxratio\":2.625,\"mccmnc\":\"310-260\",\"lmt\":0,\"ifa\":\"07c387f2-e030-428f-8336-42f682150759\",\"connectiontype\":5,\"carrier\":\"Android\",\"js\":1,\"ua\":\"signal_UA\",\"make\":\"Google\",\"model\":\"AndroidSDKbuiltforx86\",\"os\":\"Android\",\"osv\":\"9\",\"h\":1794,\"w\":1080,\"language\":\"en-US\",\"devicetype\":4,\"ext\":{\"atts\":3}},\"source\":{\"ext\":{\"omidpn\":\"PubMatic\",\"omidpv\":\"1.2.11-Pubmatic\"}},\"user\":{\"data\":[{\"id\":\"1234\"}]},\"ext\":{\"wrapper\":{\"ssauction\":1,\"sumry_disable\":0,\"profileid\":58135,\"versionid\":1,\"clientconfig\":1}}}"}]}],"ext":{"gdpr":0}},"regs":{"coppa":0,"ext":{"gdpr":0}},"source":{"ext":{"schain":{"ver":"1.0","complete":1,"nodes":[{"asi":"applovin.com","sid":"53bf468f18c5a0e2b7d4e3f748c677c1","rid":"494dbe15a3ce08c54f4e456363f35a022247f997","hp":1}]}}},"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"profileid":12929,"versionid":1,"clientconfig":1}}}}}}`),
			},
			want: []byte(`{"id":"test-case-1","imp":[{"id":"1","banner":{"format":[{"w":728,"h":90},{"w":300,"h":250}],"w":700,"h":900,"api":[5,6,7]},"video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"displaymanager":"PubMatic_OpenBid_SDK","displaymanagerver":"1.4.0","tagid":"/43743431/DMDemo","bidfloor":0.01,"bidfloorcur":"USD","clickbrowser":0,"secure":0,"exp":14400}],"app":{"id":"1234567","name":"DrawHappyAngel","bundle":"com.newstory.DrawHappyAngel","storeurl":"https://itunes.apple.com/us/app/pubmatic-sdk-app/id1175273098","cat":["IAB9-30"],"ver":"0.5.4","paid":4,"publisher":{"id":"5890","name":"New Story Inc.","ext":{"installed_sdk":{"id":"MOLOCO_BIDDING","sdk_version":{"major":1,"minor":0,"micro":0},"adapter_version":{"major":1,"minor":0,"micro":0}}}},"ext":{"orientation":1}},"device":{"geo":{"lat":40.7429,"lon":-73.9392,"type":2,"ipservice":3,"country":"USA","region":"ny","metro":"501","city":"Queens","zip":"11101","ext":{"org":"Myanmar Broadband Telecom Co.","isp":"Myanmar Broadband Telecom Co."}},"ua":"signal_UA","ip":"38.158.207.171","devicetype":4,"lmt":0,"make":"Google","model":"AndroidSDKbuiltforx86","os":"Android","osv":"9","hwv":"ruby","h":1794,"w":1080,"ppi":440,"pxratio":2.625,"js":1,"language":"en-US","carrier":"Android","mccmnc":"310-260","connectiontype":5,"ifa":"07c387f2-e030-428f-8336-42f682150759","ext":{"atts":3}},"user":{"data":[{"id":"1234"}],"ext":{"gdpr":0}},"at":1,"tmax":1000,"bcat":["IAB26-4","IAB26-2","IAB25-6","IAB25-5","IAB25-4","IAB25-3","IAB25-1","IAB25-7","IAB8-18","IAB26-3","IAB26-1","IAB8-5","IAB25-2","IAB11-4"],"source":{"ext":{"schain":{"ver":"1.0","complete":1,"nodes":[{"asi":"applovin.com","sid":"53bf468f18c5a0e2b7d4e3f748c677c1","rid":"494dbe15a3ce08c54f4e456363f35a022247f997","hp":1}]},"omidpn":"PubMatic","omidpv":"1.2.11-Pubmatic"}},"regs":{"ext":{"gdpr":0}},"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"profileid":12929,"versionid":1,"clientconfig":1}}}}}}`),
		},
		{
			name: "signal not present, native format removed from impression",
			args: args{
				requestBody: []byte(`{"id":"test","app":{"publisher":{"id":"5890"}},"imp":[{"id":"1","native":{"request":"{\"ver\":\"1.2\",\"assets\":[{\"id\":1,\"title\":{\"text\":\"Test\"}}]}","ver":"1.2"}}],"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"profileid":1234}}}}}}`),
				rctx: models.RequestCtx{
					MetricsEngine: mockEngine,
				},
			},
			setup: func() {
				mockEngine.EXPECT().RecordSignalDataStatus("5890", "1234", models.MissingSignal)
			},
			want: []byte(`{"id":"test","app":{"publisher":{"id":"5890"}},"imp":[{"id":"1","displaymanager":"PubMatic_OpenWrap_SDK"}],"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"profileid":1234}}}}}}`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			got := updateAppLovinMaxRequest(tt.args.requestBody, tt.args.rctx)
			if json.Valid(tt.want) && json.Valid(got) {
				assert.JSONEq(t, string(tt.want), string(got), tt.name)
			} else {
				assert.Equal(t, tt.want, got, tt.name)
			}
		})
	}
}

func TestUpdateRequestWrapper(t *testing.T) {
	type args struct {
		signalExt  json.RawMessage
		maxRequest *openrtb2.BidRequest
	}
	tests := []struct {
		name string
		args args
		want json.RawMessage
	}{
		{
			name: "clientconfig not present",
			args: args{
				signalExt:  json.RawMessage(`{"ssauction":1}`),
				maxRequest: &openrtb2.BidRequest{Ext: json.RawMessage(``)},
			},
			want: json.RawMessage(`{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"clientconfig":0}}}}}`),
		},
		{
			name: "clientconfig is 0",
			args: args{
				signalExt:  json.RawMessage(`{"wrapper":{"ssauction":1,"clientconfig":0}}`),
				maxRequest: &openrtb2.BidRequest{Ext: json.RawMessage(`{}`)},
			},
			want: json.RawMessage(`{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"clientconfig":0}}}}}`),
		},
		{
			name: "clientconfig is 1",
			args: args{
				signalExt:  json.RawMessage(`{"wrapper":{"ssauction":1,"clientconfig":1}}`),
				maxRequest: &openrtb2.BidRequest{Ext: json.RawMessage(`{}`)},
			},
			want: json.RawMessage(`{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"clientconfig":1}}}}}`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateRequestWrapper(tt.args.signalExt, tt.args.maxRequest)
			assert.Equal(t, tt.want, tt.args.maxRequest.Ext)
		})
	}
}

func TestUpdateMaxApplovinResponse(t *testing.T) {
	type args struct {
		rctx        models.RequestCtx
		bidResponse *openrtb2.BidResponse
	}
	tests := []struct {
		name string
		args args
		want models.AppLovinMax
	}{
		{
			name: "bidresponse contains NBR and debug is disabled",
			args: args{
				rctx: models.RequestCtx{
					Debug:    false,
					Endpoint: models.EndpointAppLovinMax,
				},
				bidResponse: &openrtb2.BidResponse{
					ID:  "123",
					NBR: ptrutil.ToPtr(nbr.InvalidPlatform),
				},
			},
			want: models.AppLovinMax{
				Reject: true,
			},
		},
		{
			name: "bidresponse contains NBR and debug is enabled",
			args: args{
				rctx: models.RequestCtx{
					Debug:    true,
					Endpoint: models.EndpointAppLovinMax,
				},
				bidResponse: &openrtb2.BidResponse{
					ID:  "123",
					NBR: ptrutil.ToPtr(nbr.InvalidPlatform),
				},
			},
			want: models.AppLovinMax{
				Reject: false,
			},
		},
		{
			name: "bidresponse seatbid is empty",
			args: args{
				rctx: models.RequestCtx{
					Debug:    false,
					Endpoint: models.EndpointAppLovinMax,
				},
				bidResponse: &openrtb2.BidResponse{
					ID:      "123",
					SeatBid: []openrtb2.SeatBid{},
				},
			},
			want: models.AppLovinMax{
				Reject: true,
			},
		},
		{
			name: "bidresponse seatbid.bid is empty",
			args: args{
				rctx: models.RequestCtx{
					Debug:    false,
					Endpoint: models.EndpointAppLovinMax,
				},
				bidResponse: &openrtb2.BidResponse{
					ID: "123",
					SeatBid: []openrtb2.SeatBid{
						{
							Bid: []openrtb2.Bid{},
						},
					},
				},
			},
			want: models.AppLovinMax{
				Reject: true,
			},
		},
		{
			name: "No NBR and valid bidresponse",
			args: args{
				rctx: models.RequestCtx{
					Debug:    false,
					Endpoint: models.EndpointAppLovinMax,
				},
				bidResponse: &openrtb2.BidResponse{
					ID: "123",
					SeatBid: []openrtb2.SeatBid{
						{
							Bid: []openrtb2.Bid{
								{
									ID:    "456",
									ImpID: "789",
									Price: 1.0,
									AdM:   "<img src=\"http://example.com\"></img>",
									BURL:  "http://example.com",
									Ext:   json.RawMessage(`{"key":"value"}`),
								},
							},
							Seat: "pubmatic",
						},
					},
				},
			},
			want: models.AppLovinMax{
				Reject: false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := updateAppLovinMaxResponse(tt.args.rctx, tt.args.bidResponse)
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}

func TestApplyMaxAppLovinResponse(t *testing.T) {
	type args struct {
		rctx        models.RequestCtx
		bidResponse *openrtb2.BidResponse
	}
	tests := []struct {
		name string
		args args
		want *openrtb2.BidResponse
	}{
		{
			name: "AppLovinMax.Reject is true",
			args: args{
				rctx: models.RequestCtx{
					Debug: true,
					AppLovinMax: models.AppLovinMax{
						Reject: true,
					},
				},
				bidResponse: &openrtb2.BidResponse{
					ID: "123",
					SeatBid: []openrtb2.SeatBid{
						{
							Bid: []openrtb2.Bid{
								{
									ID:    "456",
									ImpID: "789",
								},
							},
						},
					},
				},
			},
			want: &openrtb2.BidResponse{
				ID: "123",
				SeatBid: []openrtb2.SeatBid{
					{
						Bid: []openrtb2.Bid{
							{
								ID:    "456",
								ImpID: "789",
							},
						},
					},
				},
			},
		},
		{
			name: "bidresponse contains NBR and AppLovinMax.Reject is false",
			args: args{
				rctx: models.RequestCtx{
					Debug: false,
					AppLovinMax: models.AppLovinMax{
						Reject: false,
					},
				},
				bidResponse: &openrtb2.BidResponse{
					ID:  "123",
					NBR: ptrutil.ToPtr(nbr.InvalidPlatform),
				},
			},
			want: &openrtb2.BidResponse{
				ID:  "123",
				NBR: ptrutil.ToPtr(nbr.InvalidPlatform),
			},
		},
		{
			name: "failed to marshal bidresponse",
			args: args{
				rctx: models.RequestCtx{
					Debug: true,
					AppLovinMax: models.AppLovinMax{
						Reject: false,
					},
				},
				bidResponse: &openrtb2.BidResponse{
					ID: "123",
					SeatBid: []openrtb2.SeatBid{
						{
							Bid: []openrtb2.Bid{
								{
									ID:    "123",
									ImpID: "789",
								},
							},
						},
					},
					Ext: json.RawMessage(`{`),
				},
			},
			want: &openrtb2.BidResponse{
				ID: "123",
				SeatBid: []openrtb2.SeatBid{
					{
						Bid: []openrtb2.Bid{
							{
								ID:    "123",
								ImpID: "789",
							},
						},
					},
				},
				Ext: json.RawMessage(`{`),
			},
		},
		{
			name: "valid bidresponse",
			args: args{
				rctx: models.RequestCtx{
					AppLovinMax: models.AppLovinMax{
						Reject: false,
					},
				},
				bidResponse: &openrtb2.BidResponse{
					ID:    "123",
					BidID: "",
					Cur:   "USD",
					SeatBid: []openrtb2.SeatBid{
						{
							Bid: []openrtb2.Bid{
								{
									ID:    "456",
									ImpID: "789",
									Price: 1.0,
									AdM:   "<img src=\"http://example.com\"></img>",
									BURL:  "http://example.com",
									NURL:  "https://win.example/nurl",
									LURL:  "https://loss.example/lurl",
									Ext:   json.RawMessage(`{"key":"value"}`),
								},
							},
							Seat: "pubmatic",
						},
					},
					Ext: json.RawMessage(`{"key":"value"}`),
				},
			},
			want: &openrtb2.BidResponse{
				ID:    "123",
				BidID: "456",
				Cur:   "USD",
				SeatBid: []openrtb2.SeatBid{
					{
						Bid: []openrtb2.Bid{
							{
								ID:    "456",
								ImpID: "789",
								Price: 1.0,
								BURL:  "http://example.com",
								NURL:  "https://win.example/nurl",
								LURL:  "https://loss.example/lurl",
								Ext:   json.RawMessage(`{"signaldata":"{\"id\":\"123\",\"seatbid\":[{\"bid\":[{\"id\":\"456\",\"impid\":\"789\",\"price\":1,\"nurl\":\"https://win.example/nurl\",\"burl\":\"http://example.com\",\"lurl\":\"https://loss.example/lurl\",\"adm\":\"\\u003cimg src=\\\"http://example.com\\\"\\u003e\\u003c/img\\u003e\",\"ext\":{\"key\":\"value\"}}],\"seat\":\"pubmatic\"}],\"cur\":\"USD\",\"ext\":{\"key\":\"value\"}}"}`),
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyAppLovinMaxResponse(tt.args.rctx, tt.args.bidResponse)
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}

func TestModifyRequestBody(t *testing.T) {
	type args struct {
		requestBody []byte
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "empty requestbody",
			args: args{
				requestBody: []byte(``),
			},
			want: []byte(``),
		},
		{
			name: "applovinmax displaymanager",
			args: args{
				requestBody: []byte(`{"imp":[{"displaymanager":"applovin_mediation","displaymanagerver":"91.1"}]}`),
			},
			want: []byte(`{"imp":[{"displaymanager":"PubMatic_OpenWrap_SDK"}]}`),
		},
		{
			name: "applovinmax displaymanager and bannertype rewarded",
			args: args{
				requestBody: []byte(`{"imp":[{"displaymanager":"applovin_mediation","displaymanagerver":"91.1","banner":{"ext":{"bannertype":"rewarded"},"format":[{"w":728,"h":90},{"w":300,"h":250}],"w":700,"h":900,"api":[5,6,7]}}]}`),
			},
			want: []byte(`{"imp":[{"displaymanager":"PubMatic_OpenWrap_SDK"}]}`),
		},
		{
			name: "native format removed from impression",
			args: args{
				requestBody: []byte(`{
					"id": "test",
					"imp": [{
						"id": "1",
						"displaymanager": "applovin_mediation",
						"displaymanagerver": "1.0",
						"native": {
							"request": "{\"ver\":\"1.2\",\"assets\":[{\"id\":1,\"title\":{\"text\":\"Test\"}}]}",
							"ver": "1.2",
							"battr": [1, 2, 3],
							"api": [1, 2, 3],
							"ext": {}
						}
					}]
				}`),
			},
			want: []byte(`{
					"id": "test",
					"imp": [{
						"id": "1",
						"displaymanager": "PubMatic_OpenWrap_SDK"
					}]
				}`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := modifyRequestBody(tt.args.requestBody)
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}

func TestSetProfileID(t *testing.T) {
	type args struct {
		requestBody []byte
	}
	tests := []struct {
		name        string
		args        args
		wantProfile string
		wantBody    []byte
	}{
		{
			name: "profileID present in request.ext.prebid.bidderparams.pubmatic.wrapper",
			args: args{
				requestBody: []byte(`{"app":{"bundle":"com.pubmatic.openbid.app","name":"Sample","publisher":{"id":"156276"},"storeurl":"https://itunes.apple.com/us/app/pubmatic-sdk-app/id1175273098","ver":"1.0"},"at":1,"device":{"carrier":"MYTEL","connectiontype":2,"devicetype":4,"ext":{"atts":2},"geo":{"city":"Queens","country":"USA","dma":"501","ipservice":3,"lat":40.7429,"lon":-73.9392,"long":-73.9392,"metro":"501","region":"ny","type":2,"zip":"11101"},"h":2400,"hwv":"ruby","ifa":"497a10d6-c4dd-4e04-a986-c32b7180d462","ip":"38.158.207.171","js":1,"language":"en_US","make":"xiaomi","model":"22101316c","os":"android","osv":"13.0.0","ppi":440,"pxratio":2.75,"ua":"Mozilla/5.0 (Linux; Android 13; 22101316C Build/TP1A.220624.014; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/120.0.6099.230 Mobile Safari/537.36","w":1080},"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"profileid":1234}}}}},"id":"95d6643c-3da6-40a2-b9ca-12279393ffbf","imp":[{"banner":{"format":[{"w":728,"h":90},{"w":320,"h":50}],"api":[5,2],"w":700,"h":900},"clickbrowser":0,"displaymanager":"OpenBid_SDK","displaymanagerver":"1.4.0","ext":{"reward":1},"id":"imp176227948","secure":0,"tagid":"OpenWrapBidderBannerAdUnit"}],"regs":{"ext":{"gdpr":0}},"source":{"ext":{"omidpn":"PubMatic","omidpv":"1.2.11-Pubmatic"}},"user":{"data":[{"id":"1","name":"Publisher Passed","segment":[{"signal":"{\"id\":\"95d6643c-3da6-40a2-b9ca-12279393ffbf\",\"at\":1,\"tmax\":500,\"cur\":[\"USD\"],\"imp\":[{\"id\":\"imp176227948\",\"clickbrowser\":0,\"displaymanager\":\"PubMatic_OpenBid_SDK\",\"displaymanagerver\":\"1.4.0\",\"tagid\":\"/43743431/DMDemo\",\"secure\":0,\"banner\":{\"pos\":7,\"format\":[{\"w\":300,\"h\":250}],\"api\":[5,6,7]},\"instl\":1}],\"app\":{\"name\":\"OpenWrapperSample\",\"bundle\":\"com.pubmatic.openbid.app\",\"storeurl\":\"https://itunes.apple.com/us/app/pubmatic-sdk-app/id1175273098?appnexus_banner_fixedbid=1&fixedbid=1\",\"ver\":\"1.0\",\"publisher\":{\"id\":\"5890\"}},\"device\":{\"ext\":{\"atts\":0},\"geo\":{\"type\":1,\"lat\":37.421998333333335,\"lon\":-122.08400000000002},\"pxratio\":2.625,\"mccmnc\":\"310-260\",\"lmt\":0,\"ifa\":\"07c387f2-e030-428f-8336-42f682150759\",\"connectiontype\":6,\"carrier\":\"Android\",\"js\":1,\"ua\":\"Mozilla/5.0(Linux;Android9;AndroidSDKbuiltforx86Build/PSR1.180720.075;wv)AppleWebKit/537.36(KHTML,likeGecko)Version/4.0Chrome/69.0.3497.100MobileSafari/537.36\",\"make\":\"Google\",\"model\":\"AndroidSDKbuiltforx86\",\"os\":\"Android\",\"osv\":\"9\",\"h\":1794,\"w\":1080,\"language\":\"en-US\",\"devicetype\":4},\"source\":{\"ext\":{\"omidpn\":\"PubMatic\",\"omidpv\":\"1.2.11-Pubmatic\"}},\"user\":{},\"ext\":{\"wrapper\":{\"ssauction\":0,\"sumry_disable\":0,\"profileid\":58135,\"versionid\":1,\"clientconfig\":1}}}"}]}],"ext":{"consent":"CP2KIMAP2KIgAEPgABBYJGNX_H__bX9j-Xr3"}}}`),
			},
			wantProfile: "1234",
			wantBody:    []byte(`{"app":{"bundle":"com.pubmatic.openbid.app","name":"Sample","publisher":{"id":"156276"},"storeurl":"https://itunes.apple.com/us/app/pubmatic-sdk-app/id1175273098","ver":"1.0"},"at":1,"device":{"carrier":"MYTEL","connectiontype":2,"devicetype":4,"ext":{"atts":2},"geo":{"city":"Queens","country":"USA","dma":"501","ipservice":3,"lat":40.7429,"lon":-73.9392,"long":-73.9392,"metro":"501","region":"ny","type":2,"zip":"11101"},"h":2400,"hwv":"ruby","ifa":"497a10d6-c4dd-4e04-a986-c32b7180d462","ip":"38.158.207.171","js":1,"language":"en_US","make":"xiaomi","model":"22101316c","os":"android","osv":"13.0.0","ppi":440,"pxratio":2.75,"ua":"Mozilla/5.0 (Linux; Android 13; 22101316C Build/TP1A.220624.014; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/120.0.6099.230 Mobile Safari/537.36","w":1080},"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"profileid":1234}}}}},"id":"95d6643c-3da6-40a2-b9ca-12279393ffbf","imp":[{"banner":{"format":[{"w":728,"h":90},{"w":320,"h":50}],"api":[5,2],"w":700,"h":900},"clickbrowser":0,"displaymanager":"OpenBid_SDK","displaymanagerver":"1.4.0","ext":{"reward":1},"id":"imp176227948","secure":0,"tagid":"OpenWrapBidderBannerAdUnit"}],"regs":{"ext":{"gdpr":0}},"source":{"ext":{"omidpn":"PubMatic","omidpv":"1.2.11-Pubmatic"}},"user":{"data":[{"id":"1","name":"Publisher Passed","segment":[{"signal":"{\"id\":\"95d6643c-3da6-40a2-b9ca-12279393ffbf\",\"at\":1,\"tmax\":500,\"cur\":[\"USD\"],\"imp\":[{\"id\":\"imp176227948\",\"clickbrowser\":0,\"displaymanager\":\"PubMatic_OpenBid_SDK\",\"displaymanagerver\":\"1.4.0\",\"tagid\":\"/43743431/DMDemo\",\"secure\":0,\"banner\":{\"pos\":7,\"format\":[{\"w\":300,\"h\":250}],\"api\":[5,6,7]},\"instl\":1}],\"app\":{\"name\":\"OpenWrapperSample\",\"bundle\":\"com.pubmatic.openbid.app\",\"storeurl\":\"https://itunes.apple.com/us/app/pubmatic-sdk-app/id1175273098?appnexus_banner_fixedbid=1&fixedbid=1\",\"ver\":\"1.0\",\"publisher\":{\"id\":\"5890\"}},\"device\":{\"ext\":{\"atts\":0},\"geo\":{\"type\":1,\"lat\":37.421998333333335,\"lon\":-122.08400000000002},\"pxratio\":2.625,\"mccmnc\":\"310-260\",\"lmt\":0,\"ifa\":\"07c387f2-e030-428f-8336-42f682150759\",\"connectiontype\":6,\"carrier\":\"Android\",\"js\":1,\"ua\":\"Mozilla/5.0(Linux;Android9;AndroidSDKbuiltforx86Build/PSR1.180720.075;wv)AppleWebKit/537.36(KHTML,likeGecko)Version/4.0Chrome/69.0.3497.100MobileSafari/537.36\",\"make\":\"Google\",\"model\":\"AndroidSDKbuiltforx86\",\"os\":\"Android\",\"osv\":\"9\",\"h\":1794,\"w\":1080,\"language\":\"en-US\",\"devicetype\":4},\"source\":{\"ext\":{\"omidpn\":\"PubMatic\",\"omidpv\":\"1.2.11-Pubmatic\"}},\"user\":{},\"ext\":{\"wrapper\":{\"ssauction\":0,\"sumry_disable\":0,\"profileid\":58135,\"versionid\":1,\"clientconfig\":1}}}"}]}],"ext":{"consent":"CP2KIMAP2KIgAEPgABBYJGNX_H__bX9j-Xr3"}}}`),
		},
		{
			name: "profileID not present in request.ext.prebid.bidderparams.pubmatic.wrapper but present in request.app.id, remove app.id",
			args: args{
				requestBody: []byte(`{"app":{"bundle":"com.pubmatic.openbid.app","name":"Sample","publisher":{"id":"156276"},"id":"13137","storeurl":"https://itunes.apple.com/us/app/pubmatic-sdk-app/id1175273098","ver":"1.0"},"at":1,"device":{"carrier":"MYTEL","connectiontype":2,"devicetype":4,"ext":{"atts":2},"geo":{"city":"Queens","country":"USA","dma":"501","ipservice":3,"lat":40.7429,"lon":-73.9392,"long":-73.9392,"metro":"501","region":"ny","type":2,"zip":"11101"},"h":2400,"hwv":"ruby","ifa":"497a10d6-c4dd-4e04-a986-c32b7180d462","ip":"38.158.207.171","js":1,"language":"en_US","make":"xiaomi","model":"22101316c","os":"android","osv":"13.0.0","ppi":440,"pxratio":2.75,"ua":"Mozilla/5.0 (Linux; Android 13; 22101316C Build/TP1A.220624.014; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/120.0.6099.230 Mobile Safari/537.36","w":1080},"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{}}}}},"id":"95d6643c-3da6-40a2-b9ca-12279393ffbf","imp":[{"banner":{"format":[{"w":728,"h":90},{"w":320,"h":50}],"api":[5,2],"w":700,"h":900},"clickbrowser":0,"displaymanager":"OpenBid_SDK","displaymanagerver":"1.4.0","ext":{"reward":1},"id":"imp176227948","secure":0,"tagid":"OpenWrapBidderBannerAdUnit"}],"regs":{"ext":{"gdpr":0}},"source":{"ext":{"omidpn":"PubMatic","omidpv":"1.2.11-Pubmatic"}},"user":{"data":[{"id":"1","name":"Publisher Passed","segment":[{"signal":"{\"id\":\"95d6643c-3da6-40a2-b9ca-12279393ffbf\",\"at\":1,\"tmax\":500,\"cur\":[\"USD\"],\"imp\":[{\"id\":\"imp176227948\",\"clickbrowser\":0,\"displaymanager\":\"PubMatic_OpenBid_SDK\",\"displaymanagerver\":\"1.4.0\",\"tagid\":\"/43743431/DMDemo\",\"secure\":0,\"banner\":{\"pos\":7,\"format\":[{\"w\":300,\"h\":250}],\"api\":[5,6,7]},\"instl\":1}],\"app\":{\"name\":\"OpenWrapperSample\",\"bundle\":\"com.pubmatic.openbid.app\",\"storeurl\":\"https://itunes.apple.com/us/app/pubmatic-sdk-app/id1175273098?appnexus_banner_fixedbid=1&fixedbid=1\",\"ver\":\"1.0\",\"publisher\":{\"id\":\"5890\"}},\"device\":{\"ext\":{\"atts\":0},\"geo\":{\"type\":1,\"lat\":37.421998333333335,\"lon\":-122.08400000000002},\"pxratio\":2.625,\"mccmnc\":\"310-260\",\"lmt\":0,\"ifa\":\"07c387f2-e030-428f-8336-42f682150759\",\"connectiontype\":6,\"carrier\":\"Android\",\"js\":1,\"ua\":\"Mozilla/5.0(Linux;Android9;AndroidSDKbuiltforx86Build/PSR1.180720.075;wv)AppleWebKit/537.36(KHTML,likeGecko)Version/4.0Chrome/69.0.3497.100MobileSafari/537.36\",\"make\":\"Google\",\"model\":\"AndroidSDKbuiltforx86\",\"os\":\"Android\",\"osv\":\"9\",\"h\":1794,\"w\":1080,\"language\":\"en-US\",\"devicetype\":4},\"source\":{\"ext\":{\"omidpn\":\"PubMatic\",\"omidpv\":\"1.2.11-Pubmatic\"}},\"user\":{},\"ext\":{\"wrapper\":{\"ssauction\":0,\"sumry_disable\":0,\"profileid\":58135,\"versionid\":1,\"clientconfig\":1}}}"}]}],"ext":{"consent":"CP2KIMAP2KIgAEPgABBYJGNX_H__bX9j-Xr3"}}}`),
			},
			wantProfile: "13137",
			wantBody:    []byte(`{"app":{"bundle":"com.pubmatic.openbid.app","name":"Sample","publisher":{"id":"156276"},"storeurl":"https://itunes.apple.com/us/app/pubmatic-sdk-app/id1175273098","ver":"1.0"},"at":1,"device":{"carrier":"MYTEL","connectiontype":2,"devicetype":4,"ext":{"atts":2},"geo":{"city":"Queens","country":"USA","dma":"501","ipservice":3,"lat":40.7429,"lon":-73.9392,"long":-73.9392,"metro":"501","region":"ny","type":2,"zip":"11101"},"h":2400,"hwv":"ruby","ifa":"497a10d6-c4dd-4e04-a986-c32b7180d462","ip":"38.158.207.171","js":1,"language":"en_US","make":"xiaomi","model":"22101316c","os":"android","osv":"13.0.0","ppi":440,"pxratio":2.75,"ua":"Mozilla/5.0 (Linux; Android 13; 22101316C Build/TP1A.220624.014; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/120.0.6099.230 Mobile Safari/537.36","w":1080},"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"profileid":13137}}}}},"id":"95d6643c-3da6-40a2-b9ca-12279393ffbf","imp":[{"banner":{"format":[{"w":728,"h":90},{"w":320,"h":50}],"api":[5,2],"w":700,"h":900},"clickbrowser":0,"displaymanager":"OpenBid_SDK","displaymanagerver":"1.4.0","ext":{"reward":1},"id":"imp176227948","secure":0,"tagid":"OpenWrapBidderBannerAdUnit"}],"regs":{"ext":{"gdpr":0}},"source":{"ext":{"omidpn":"PubMatic","omidpv":"1.2.11-Pubmatic"}},"user":{"data":[{"id":"1","name":"Publisher Passed","segment":[{"signal":"{\"id\":\"95d6643c-3da6-40a2-b9ca-12279393ffbf\",\"at\":1,\"tmax\":500,\"cur\":[\"USD\"],\"imp\":[{\"id\":\"imp176227948\",\"clickbrowser\":0,\"displaymanager\":\"PubMatic_OpenBid_SDK\",\"displaymanagerver\":\"1.4.0\",\"tagid\":\"/43743431/DMDemo\",\"secure\":0,\"banner\":{\"pos\":7,\"format\":[{\"w\":300,\"h\":250}],\"api\":[5,6,7]},\"instl\":1}],\"app\":{\"name\":\"OpenWrapperSample\",\"bundle\":\"com.pubmatic.openbid.app\",\"storeurl\":\"https://itunes.apple.com/us/app/pubmatic-sdk-app/id1175273098?appnexus_banner_fixedbid=1&fixedbid=1\",\"ver\":\"1.0\",\"publisher\":{\"id\":\"5890\"}},\"device\":{\"ext\":{\"atts\":0},\"geo\":{\"type\":1,\"lat\":37.421998333333335,\"lon\":-122.08400000000002},\"pxratio\":2.625,\"mccmnc\":\"310-260\",\"lmt\":0,\"ifa\":\"07c387f2-e030-428f-8336-42f682150759\",\"connectiontype\":6,\"carrier\":\"Android\",\"js\":1,\"ua\":\"Mozilla/5.0(Linux;Android9;AndroidSDKbuiltforx86Build/PSR1.180720.075;wv)AppleWebKit/537.36(KHTML,likeGecko)Version/4.0Chrome/69.0.3497.100MobileSafari/537.36\",\"make\":\"Google\",\"model\":\"AndroidSDKbuiltforx86\",\"os\":\"Android\",\"osv\":\"9\",\"h\":1794,\"w\":1080,\"language\":\"en-US\",\"devicetype\":4},\"source\":{\"ext\":{\"omidpn\":\"PubMatic\",\"omidpv\":\"1.2.11-Pubmatic\"}},\"user\":{},\"ext\":{\"wrapper\":{\"ssauction\":0,\"sumry_disable\":0,\"profileid\":58135,\"versionid\":1,\"clientconfig\":1}}}"}]}],"ext":{"consent":"CP2KIMAP2KIgAEPgABBYJGNX_H__bX9j-Xr3"}}}`),
		},
		{
			name: "profileID present in both request.ext.prebid.bidderparams.pubmatic.wrapper and request.app.id give priority to wrapper",
			args: args{
				requestBody: []byte(`{"app":{"bundle":"com.pubmatic.openbid.app","name":"Sample","publisher":{"id":"156276"},"id":"13137","storeurl":"https://itunes.apple.com/us/app/pubmatic-sdk-app/id1175273098","ver":"1.0"},"at":1,"device":{"carrier":"MYTEL","connectiontype":2,"devicetype":4,"ext":{"atts":2},"geo":{"city":"Queens","country":"USA","dma":"501","ipservice":3,"lat":40.7429,"lon":-73.9392,"long":-73.9392,"metro":"501","region":"ny","type":2,"zip":"11101"},"h":2400,"hwv":"ruby","ifa":"497a10d6-c4dd-4e04-a986-c32b7180d462","ip":"38.158.207.171","js":1,"language":"en_US","make":"xiaomi","model":"22101316c","os":"android","osv":"13.0.0","ppi":440,"pxratio":2.75,"ua":"Mozilla/5.0 (Linux; Android 13; 22101316C Build/TP1A.220624.014; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/120.0.6099.230 Mobile Safari/537.36","w":1080},"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"profileid":1234}}}}},"id":"95d6643c-3da6-40a2-b9ca-12279393ffbf","imp":[{"banner":{"format":[{"w":728,"h":90},{"w":320,"h":50}],"api":[5,2],"w":700,"h":900},"clickbrowser":0,"displaymanager":"OpenBid_SDK","displaymanagerver":"1.4.0","ext":{"reward":1},"id":"imp176227948","secure":0,"tagid":"OpenWrapBidderBannerAdUnit"}],"regs":{"ext":{"gdpr":0}},"source":{"ext":{"omidpn":"PubMatic","omidpv":"1.2.11-Pubmatic"}},"user":{"data":[{"id":"1","name":"Publisher Passed","segment":[{"signal":"{\"id\":\"95d6643c-3da6-40a2-b9ca-12279393ffbf\",\"at\":1,\"tmax\":500,\"cur\":[\"USD\"],\"imp\":[{\"id\":\"imp176227948\",\"clickbrowser\":0,\"displaymanager\":\"PubMatic_OpenBid_SDK\",\"displaymanagerver\":\"1.4.0\",\"tagid\":\"/43743431/DMDemo\",\"secure\":0,\"banner\":{\"pos\":7,\"format\":[{\"w\":300,\"h\":250}],\"api\":[5,6,7]},\"instl\":1}],\"app\":{\"name\":\"OpenWrapperSample\",\"bundle\":\"com.pubmatic.openbid.app\",\"storeurl\":\"https://itunes.apple.com/us/app/pubmatic-sdk-app/id1175273098?appnexus_banner_fixedbid=1&fixedbid=1\",\"ver\":\"1.0\",\"publisher\":{\"id\":\"5890\"}},\"device\":{\"ext\":{\"atts\":0},\"geo\":{\"type\":1,\"lat\":37.421998333333335,\"lon\":-122.08400000000002},\"pxratio\":2.625,\"mccmnc\":\"310-260\",\"lmt\":0,\"ifa\":\"07c387f2-e030-428f-8336-42f682150759\",\"connectiontype\":6,\"carrier\":\"Android\",\"js\":1,\"ua\":\"Mozilla/5.0(Linux;Android9;AndroidSDKbuiltforx86Build/PSR1.180720.075;wv)AppleWebKit/537.36(KHTML,likeGecko)Version/4.0Chrome/69.0.3497.100MobileSafari/537.36\",\"make\":\"Google\",\"model\":\"AndroidSDKbuiltforx86\",\"os\":\"Android\",\"osv\":\"9\",\"h\":1794,\"w\":1080,\"language\":\"en-US\",\"devicetype\":4},\"source\":{\"ext\":{\"omidpn\":\"PubMatic\",\"omidpv\":\"1.2.11-Pubmatic\"}},\"user\":{},\"ext\":{\"wrapper\":{\"ssauction\":0,\"sumry_disable\":0,\"profileid\":58135,\"versionid\":1,\"clientconfig\":1}}}"}]}],"ext":{"consent":"CP2KIMAP2KIgAEPgABBYJGNX_H__bX9j-Xr3"}}}`),
			},
			wantProfile: "1234",
			wantBody:    []byte(`{"app":{"bundle":"com.pubmatic.openbid.app","name":"Sample","publisher":{"id":"156276"},"id":"13137","storeurl":"https://itunes.apple.com/us/app/pubmatic-sdk-app/id1175273098","ver":"1.0"},"at":1,"device":{"carrier":"MYTEL","connectiontype":2,"devicetype":4,"ext":{"atts":2},"geo":{"city":"Queens","country":"USA","dma":"501","ipservice":3,"lat":40.7429,"lon":-73.9392,"long":-73.9392,"metro":"501","region":"ny","type":2,"zip":"11101"},"h":2400,"hwv":"ruby","ifa":"497a10d6-c4dd-4e04-a986-c32b7180d462","ip":"38.158.207.171","js":1,"language":"en_US","make":"xiaomi","model":"22101316c","os":"android","osv":"13.0.0","ppi":440,"pxratio":2.75,"ua":"Mozilla/5.0 (Linux; Android 13; 22101316C Build/TP1A.220624.014; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/120.0.6099.230 Mobile Safari/537.36","w":1080},"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"profileid":1234}}}}},"id":"95d6643c-3da6-40a2-b9ca-12279393ffbf","imp":[{"banner":{"format":[{"w":728,"h":90},{"w":320,"h":50}],"api":[5,2],"w":700,"h":900},"clickbrowser":0,"displaymanager":"OpenBid_SDK","displaymanagerver":"1.4.0","ext":{"reward":1},"id":"imp176227948","secure":0,"tagid":"OpenWrapBidderBannerAdUnit"}],"regs":{"ext":{"gdpr":0}},"source":{"ext":{"omidpn":"PubMatic","omidpv":"1.2.11-Pubmatic"}},"user":{"data":[{"id":"1","name":"Publisher Passed","segment":[{"signal":"{\"id\":\"95d6643c-3da6-40a2-b9ca-12279393ffbf\",\"at\":1,\"tmax\":500,\"cur\":[\"USD\"],\"imp\":[{\"id\":\"imp176227948\",\"clickbrowser\":0,\"displaymanager\":\"PubMatic_OpenBid_SDK\",\"displaymanagerver\":\"1.4.0\",\"tagid\":\"/43743431/DMDemo\",\"secure\":0,\"banner\":{\"pos\":7,\"format\":[{\"w\":300,\"h\":250}],\"api\":[5,6,7]},\"instl\":1}],\"app\":{\"name\":\"OpenWrapperSample\",\"bundle\":\"com.pubmatic.openbid.app\",\"storeurl\":\"https://itunes.apple.com/us/app/pubmatic-sdk-app/id1175273098?appnexus_banner_fixedbid=1&fixedbid=1\",\"ver\":\"1.0\",\"publisher\":{\"id\":\"5890\"}},\"device\":{\"ext\":{\"atts\":0},\"geo\":{\"type\":1,\"lat\":37.421998333333335,\"lon\":-122.08400000000002},\"pxratio\":2.625,\"mccmnc\":\"310-260\",\"lmt\":0,\"ifa\":\"07c387f2-e030-428f-8336-42f682150759\",\"connectiontype\":6,\"carrier\":\"Android\",\"js\":1,\"ua\":\"Mozilla/5.0(Linux;Android9;AndroidSDKbuiltforx86Build/PSR1.180720.075;wv)AppleWebKit/537.36(KHTML,likeGecko)Version/4.0Chrome/69.0.3497.100MobileSafari/537.36\",\"make\":\"Google\",\"model\":\"AndroidSDKbuiltforx86\",\"os\":\"Android\",\"osv\":\"9\",\"h\":1794,\"w\":1080,\"language\":\"en-US\",\"devicetype\":4},\"source\":{\"ext\":{\"omidpn\":\"PubMatic\",\"omidpv\":\"1.2.11-Pubmatic\"}},\"user\":{},\"ext\":{\"wrapper\":{\"ssauction\":0,\"sumry_disable\":0,\"profileid\":58135,\"versionid\":1,\"clientconfig\":1}}}"}]}],"ext":{"consent":"CP2KIMAP2KIgAEPgABBYJGNX_H__bX9j-Xr3"}}}`),
		},
		{
			name: "profileID not present in both request.ext.prebid.bidderparams.pubmatic.wrapper and request.app.id",
			args: args{
				requestBody: []byte(`{"app":{"bundle":"com.pubmatic.openbid.app","name":"Sample","publisher":{"id":"156276"},"storeurl":"https://itunes.apple.com/us/app/pubmatic-sdk-app/id1175273098","ver":"1.0"},"at":1,"device":{"carrier":"MYTEL","connectiontype":2,"devicetype":4,"ext":{"atts":2},"geo":{"city":"Queens","country":"USA","dma":"501","ipservice":3,"lat":40.7429,"lon":-73.9392,"long":-73.9392,"metro":"501","region":"ny","type":2,"zip":"11101"},"h":2400,"hwv":"ruby","ifa":"497a10d6-c4dd-4e04-a986-c32b7180d462","ip":"38.158.207.171","js":1,"language":"en_US","make":"xiaomi","model":"22101316c","os":"android","osv":"13.0.0","ppi":440,"pxratio":2.75,"ua":"Mozilla/5.0 (Linux; Android 13; 22101316C Build/TP1A.220624.014; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/120.0.6099.230 Mobile Safari/537.36","w":1080},"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{}}}}},"id":"95d6643c-3da6-40a2-b9ca-12279393ffbf","imp":[{"banner":{"format":[{"w":728,"h":90},{"w":320,"h":50}],"api":[5,2],"w":700,"h":900},"clickbrowser":0,"displaymanager":"OpenBid_SDK","displaymanagerver":"1.4.0","ext":{"reward":1},"id":"imp176227948","secure":0,"tagid":"OpenWrapBidderBannerAdUnit"}],"regs":{"ext":{"gdpr":0}},"source":{"ext":{"omidpn":"PubMatic","omidpv":"1.2.11-Pubmatic"}},"user":{"data":[{"id":"1","name":"Publisher Passed","segment":[{"signal":"{\"id\":\"95d6643c-3da6-40a2-b9ca-12279393ffbf\",\"at\":1,\"tmax\":500,\"cur\":[\"USD\"],\"imp\":[{\"id\":\"imp176227948\",\"clickbrowser\":0,\"displaymanager\":\"PubMatic_OpenBid_SDK\",\"displaymanagerver\":\"1.4.0\",\"tagid\":\"/43743431/DMDemo\",\"secure\":0,\"banner\":{\"pos\":7,\"format\":[{\"w\":300,\"h\":250}],\"api\":[5,6,7]},\"instl\":1}],\"app\":{\"name\":\"OpenWrapperSample\",\"bundle\":\"com.pubmatic.openbid.app\",\"storeurl\":\"https://itunes.apple.com/us/app/pubmatic-sdk-app/id1175273098?appnexus_banner_fixedbid=1&fixedbid=1\",\"ver\":\"1.0\",\"publisher\":{\"id\":\"5890\"}},\"device\":{\"ext\":{\"atts\":0},\"geo\":{\"type\":1,\"lat\":37.421998333333335,\"lon\":-122.08400000000002},\"pxratio\":2.625,\"mccmnc\":\"310-260\",\"lmt\":0,\"ifa\":\"07c387f2-e030-428f-8336-42f682150759\",\"connectiontype\":6,\"carrier\":\"Android\",\"js\":1,\"ua\":\"Mozilla/5.0(Linux;Android9;AndroidSDKbuiltforx86Build/PSR1.180720.075;wv)AppleWebKit/537.36(KHTML,likeGecko)Version/4.0Chrome/69.0.3497.100MobileSafari/537.36\",\"make\":\"Google\",\"model\":\"AndroidSDKbuiltforx86\",\"os\":\"Android\",\"osv\":\"9\",\"h\":1794,\"w\":1080,\"language\":\"en-US\",\"devicetype\":4},\"source\":{\"ext\":{\"omidpn\":\"PubMatic\",\"omidpv\":\"1.2.11-Pubmatic\"}},\"user\":{},\"ext\":{\"wrapper\":{\"ssauction\":0,\"sumry_disable\":0,\"profileid\":58135,\"versionid\":1,\"clientconfig\":1}}}"}]}],"ext":{"consent":"CP2KIMAP2KIgAEPgABBYJGNX_H__bX9j-Xr3"}}}`),
			},
			wantProfile: "",
			wantBody:    []byte(`{"app":{"bundle":"com.pubmatic.openbid.app","name":"Sample","publisher":{"id":"156276"},"storeurl":"https://itunes.apple.com/us/app/pubmatic-sdk-app/id1175273098","ver":"1.0"},"at":1,"device":{"carrier":"MYTEL","connectiontype":2,"devicetype":4,"ext":{"atts":2},"geo":{"city":"Queens","country":"USA","dma":"501","ipservice":3,"lat":40.7429,"lon":-73.9392,"long":-73.9392,"metro":"501","region":"ny","type":2,"zip":"11101"},"h":2400,"hwv":"ruby","ifa":"497a10d6-c4dd-4e04-a986-c32b7180d462","ip":"38.158.207.171","js":1,"language":"en_US","make":"xiaomi","model":"22101316c","os":"android","osv":"13.0.0","ppi":440,"pxratio":2.75,"ua":"Mozilla/5.0 (Linux; Android 13; 22101316C Build/TP1A.220624.014; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/120.0.6099.230 Mobile Safari/537.36","w":1080},"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{}}}}},"id":"95d6643c-3da6-40a2-b9ca-12279393ffbf","imp":[{"banner":{"format":[{"w":728,"h":90},{"w":320,"h":50}],"api":[5,2],"w":700,"h":900},"clickbrowser":0,"displaymanager":"OpenBid_SDK","displaymanagerver":"1.4.0","ext":{"reward":1},"id":"imp176227948","secure":0,"tagid":"OpenWrapBidderBannerAdUnit"}],"regs":{"ext":{"gdpr":0}},"source":{"ext":{"omidpn":"PubMatic","omidpv":"1.2.11-Pubmatic"}},"user":{"data":[{"id":"1","name":"Publisher Passed","segment":[{"signal":"{\"id\":\"95d6643c-3da6-40a2-b9ca-12279393ffbf\",\"at\":1,\"tmax\":500,\"cur\":[\"USD\"],\"imp\":[{\"id\":\"imp176227948\",\"clickbrowser\":0,\"displaymanager\":\"PubMatic_OpenBid_SDK\",\"displaymanagerver\":\"1.4.0\",\"tagid\":\"/43743431/DMDemo\",\"secure\":0,\"banner\":{\"pos\":7,\"format\":[{\"w\":300,\"h\":250}],\"api\":[5,6,7]},\"instl\":1}],\"app\":{\"name\":\"OpenWrapperSample\",\"bundle\":\"com.pubmatic.openbid.app\",\"storeurl\":\"https://itunes.apple.com/us/app/pubmatic-sdk-app/id1175273098?appnexus_banner_fixedbid=1&fixedbid=1\",\"ver\":\"1.0\",\"publisher\":{\"id\":\"5890\"}},\"device\":{\"ext\":{\"atts\":0},\"geo\":{\"type\":1,\"lat\":37.421998333333335,\"lon\":-122.08400000000002},\"pxratio\":2.625,\"mccmnc\":\"310-260\",\"lmt\":0,\"ifa\":\"07c387f2-e030-428f-8336-42f682150759\",\"connectiontype\":6,\"carrier\":\"Android\",\"js\":1,\"ua\":\"Mozilla/5.0(Linux;Android9;AndroidSDKbuiltforx86Build/PSR1.180720.075;wv)AppleWebKit/537.36(KHTML,likeGecko)Version/4.0Chrome/69.0.3497.100MobileSafari/537.36\",\"make\":\"Google\",\"model\":\"AndroidSDKbuiltforx86\",\"os\":\"Android\",\"osv\":\"9\",\"h\":1794,\"w\":1080,\"language\":\"en-US\",\"devicetype\":4},\"source\":{\"ext\":{\"omidpn\":\"PubMatic\",\"omidpv\":\"1.2.11-Pubmatic\"}},\"user\":{},\"ext\":{\"wrapper\":{\"ssauction\":0,\"sumry_disable\":0,\"profileid\":58135,\"versionid\":1,\"clientconfig\":1}}}"}]}],"ext":{"consent":"CP2KIMAP2KIgAEPgABBYJGNX_H__bX9j-Xr3"}}}`),
		},
		{
			name: "profileID not present in request.ext.prebid.bidderparams.pubmatic.wrapper but invalid present in request.app.id",
			args: args{
				requestBody: []byte(`{"app":{"bundle":"com.pubmatic.openbid.app","name":"Sample","publisher":{"id":"156276"},"id":"abcde","storeurl":"https://itunes.apple.com/us/app/pubmatic-sdk-app/id1175273098","ver":"1.0"},"at":1,"device":{"carrier":"MYTEL","connectiontype":2,"devicetype":4,"ext":{"atts":2},"geo":{"city":"Queens","country":"USA","dma":"501","ipservice":3,"lat":40.7429,"lon":-73.9392,"long":-73.9392,"metro":"501","region":"ny","type":2,"zip":"11101"},"h":2400,"hwv":"ruby","ifa":"497a10d6-c4dd-4e04-a986-c32b7180d462","ip":"38.158.207.171","js":1,"language":"en_US","make":"xiaomi","model":"22101316c","os":"android","osv":"13.0.0","ppi":440,"pxratio":2.75,"ua":"Mozilla/5.0 (Linux; Android 13; 22101316C Build/TP1A.220624.014; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/120.0.6099.230 Mobile Safari/537.36","w":1080},"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{}}}}},"id":"95d6643c-3da6-40a2-b9ca-12279393ffbf","imp":[{"banner":{"format":[{"w":728,"h":90},{"w":320,"h":50}],"api":[5,2],"w":700,"h":900},"clickbrowser":0,"displaymanager":"OpenBid_SDK","displaymanagerver":"1.4.0","ext":{"reward":1},"id":"imp176227948","secure":0,"tagid":"OpenWrapBidderBannerAdUnit"}],"regs":{"ext":{"gdpr":0}},"source":{"ext":{"omidpn":"PubMatic","omidpv":"1.2.11-Pubmatic"}},"user":{"data":[{"id":"1","name":"Publisher Passed","segment":[{"signal":"{\"id\":\"95d6643c-3da6-40a2-b9ca-12279393ffbf\",\"at\":1,\"tmax\":500,\"cur\":[\"USD\"],\"imp\":[{\"id\":\"imp176227948\",\"clickbrowser\":0,\"displaymanager\":\"PubMatic_OpenBid_SDK\",\"displaymanagerver\":\"1.4.0\",\"tagid\":\"/43743431/DMDemo\",\"secure\":0,\"banner\":{\"pos\":7,\"format\":[{\"w\":300,\"h\":250}],\"api\":[5,6,7]},\"instl\":1}],\"app\":{\"name\":\"OpenWrapperSample\",\"bundle\":\"com.pubmatic.openbid.app\",\"storeurl\":\"https://itunes.apple.com/us/app/pubmatic-sdk-app/id1175273098?appnexus_banner_fixedbid=1&fixedbid=1\",\"ver\":\"1.0\",\"publisher\":{\"id\":\"5890\"}},\"device\":{\"ext\":{\"atts\":0},\"geo\":{\"type\":1,\"lat\":37.421998333333335,\"lon\":-122.08400000000002},\"pxratio\":2.625,\"mccmnc\":\"310-260\",\"lmt\":0,\"ifa\":\"07c387f2-e030-428f-8336-42f682150759\",\"connectiontype\":6,\"carrier\":\"Android\",\"js\":1,\"ua\":\"Mozilla/5.0(Linux;Android9;AndroidSDKbuiltforx86Build/PSR1.180720.075;wv)AppleWebKit/537.36(KHTML,likeGecko)Version/4.0Chrome/69.0.3497.100MobileSafari/537.36\",\"make\":\"Google\",\"model\":\"AndroidSDKbuiltforx86\",\"os\":\"Android\",\"osv\":\"9\",\"h\":1794,\"w\":1080,\"language\":\"en-US\",\"devicetype\":4},\"source\":{\"ext\":{\"omidpn\":\"PubMatic\",\"omidpv\":\"1.2.11-Pubmatic\"}},\"user\":{},\"ext\":{\"wrapper\":{\"ssauction\":0,\"sumry_disable\":0,\"profileid\":58135,\"versionid\":1,\"clientconfig\":1}}}"}]}],"ext":{"consent":"CP2KIMAP2KIgAEPgABBYJGNX_H__bX9j-Xr3"}}}`),
			},
			wantProfile: "",
			wantBody:    []byte(`{"app":{"bundle":"com.pubmatic.openbid.app","name":"Sample","publisher":{"id":"156276"},"id":"abcde","storeurl":"https://itunes.apple.com/us/app/pubmatic-sdk-app/id1175273098","ver":"1.0"},"at":1,"device":{"carrier":"MYTEL","connectiontype":2,"devicetype":4,"ext":{"atts":2},"geo":{"city":"Queens","country":"USA","dma":"501","ipservice":3,"lat":40.7429,"lon":-73.9392,"long":-73.9392,"metro":"501","region":"ny","type":2,"zip":"11101"},"h":2400,"hwv":"ruby","ifa":"497a10d6-c4dd-4e04-a986-c32b7180d462","ip":"38.158.207.171","js":1,"language":"en_US","make":"xiaomi","model":"22101316c","os":"android","osv":"13.0.0","ppi":440,"pxratio":2.75,"ua":"Mozilla/5.0 (Linux; Android 13; 22101316C Build/TP1A.220624.014; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/120.0.6099.230 Mobile Safari/537.36","w":1080},"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{}}}}},"id":"95d6643c-3da6-40a2-b9ca-12279393ffbf","imp":[{"banner":{"format":[{"w":728,"h":90},{"w":320,"h":50}],"api":[5,2],"w":700,"h":900},"clickbrowser":0,"displaymanager":"OpenBid_SDK","displaymanagerver":"1.4.0","ext":{"reward":1},"id":"imp176227948","secure":0,"tagid":"OpenWrapBidderBannerAdUnit"}],"regs":{"ext":{"gdpr":0}},"source":{"ext":{"omidpn":"PubMatic","omidpv":"1.2.11-Pubmatic"}},"user":{"data":[{"id":"1","name":"Publisher Passed","segment":[{"signal":"{\"id\":\"95d6643c-3da6-40a2-b9ca-12279393ffbf\",\"at\":1,\"tmax\":500,\"cur\":[\"USD\"],\"imp\":[{\"id\":\"imp176227948\",\"clickbrowser\":0,\"displaymanager\":\"PubMatic_OpenBid_SDK\",\"displaymanagerver\":\"1.4.0\",\"tagid\":\"/43743431/DMDemo\",\"secure\":0,\"banner\":{\"pos\":7,\"format\":[{\"w\":300,\"h\":250}],\"api\":[5,6,7]},\"instl\":1}],\"app\":{\"name\":\"OpenWrapperSample\",\"bundle\":\"com.pubmatic.openbid.app\",\"storeurl\":\"https://itunes.apple.com/us/app/pubmatic-sdk-app/id1175273098?appnexus_banner_fixedbid=1&fixedbid=1\",\"ver\":\"1.0\",\"publisher\":{\"id\":\"5890\"}},\"device\":{\"ext\":{\"atts\":0},\"geo\":{\"type\":1,\"lat\":37.421998333333335,\"lon\":-122.08400000000002},\"pxratio\":2.625,\"mccmnc\":\"310-260\",\"lmt\":0,\"ifa\":\"07c387f2-e030-428f-8336-42f682150759\",\"connectiontype\":6,\"carrier\":\"Android\",\"js\":1,\"ua\":\"Mozilla/5.0(Linux;Android9;AndroidSDKbuiltforx86Build/PSR1.180720.075;wv)AppleWebKit/537.36(KHTML,likeGecko)Version/4.0Chrome/69.0.3497.100MobileSafari/537.36\",\"make\":\"Google\",\"model\":\"AndroidSDKbuiltforx86\",\"os\":\"Android\",\"osv\":\"9\",\"h\":1794,\"w\":1080,\"language\":\"en-US\",\"devicetype\":4},\"source\":{\"ext\":{\"omidpn\":\"PubMatic\",\"omidpv\":\"1.2.11-Pubmatic\"}},\"user\":{},\"ext\":{\"wrapper\":{\"ssauction\":0,\"sumry_disable\":0,\"profileid\":58135,\"versionid\":1,\"clientconfig\":1}}}"}]}],"ext":{"consent":"CP2KIMAP2KIgAEPgABBYJGNX_H__bX9j-Xr3"}}}`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := setProfileID(tt.args.requestBody)
			assert.Equal(t, tt.wantBody, got)
			assert.Equal(t, tt.wantProfile, got1)
		})
	}
}

func TestOpenWrap_getApplovinMultiFloors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockFeature := mock_feature.NewMockFeature(ctrl)

	type args struct {
		rctx models.RequestCtx
	}
	tests := []struct {
		name  string
		args  args
		want  models.MultiFloorsConfig
		setup func()
	}{
		{
			name: "endpoint is not of applovinmax",
			args: args{
				rctx: models.RequestCtx{
					Endpoint: models.EndpointV25,
				},
			},
			want: models.MultiFloorsConfig{
				Enabled: false,
			},
			setup: func() {},
		},
		{
			name: "AB test disabled",
			args: args{
				rctx: models.RequestCtx{
					Endpoint:     models.EndpointAppLovinMax,
					PubID:        5890,
					ProfileIDStr: "1234",
				},
			},
			want: models.MultiFloorsConfig{
				Enabled: false,
			},
			setup: func() {
				mockFeature.EXPECT().IsApplovinMultiFloorsEnabled(5890, "1234").Return(false)
			},
		},
		{
			name: "AB test enabled",
			args: args{
				rctx: models.RequestCtx{
					Endpoint:     models.EndpointAppLovinMax,
					PubID:        5890,
					ProfileIDStr: "1234",
				},
			},
			want: models.MultiFloorsConfig{
				Enabled: true,
				Config: models.ApplovinAdUnitFloors{
					"adunit_name": {1.5, 1.2, 2.2},
				},
			},
			setup: func() {
				mockFeature.EXPECT().IsApplovinMultiFloorsEnabled(5890, "1234").Return(true)
				mockFeature.EXPECT().GetApplovinMultiFloors(5890, "1234").Return(models.ApplovinAdUnitFloors{
					"adunit_name": {1.5, 1.2, 2.2},
				})
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			m := OpenWrap{
				pubFeatures: mockFeature,
			}
			got := m.getApplovinMultiFloors(tt.args.rctx)
			assert.Equal(t, tt.want, got)
		})
	}
}
