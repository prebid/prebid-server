package openwrap

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/openrtb/v20/openrtb3"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/macros"
	mock_metrics "github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/metrics/mock"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models/adunitconfig"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models/nbr"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/profilemetadata"
	mock_profilemetadata "github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/profilemetadata/mock"
	mock_feature "github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/publisherfeature/mock"
	"github.com/prebid/prebid-server/v3/usersync"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

func TestRecordPublisherPartnerNoCookieStats(t *testing.T) {

	ctrl := gomock.NewController(t)
	mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)
	defer ctrl.Finish()

	type args struct {
		rctx models.RequestCtx
	}

	tests := []struct {
		name           string
		args           args
		getHttpRequest func() *http.Request
		setup          func(*mock_metrics.MockMetricsEngine)
	}{
		{
			name: "Empty cookies and empty partner config map",
			args: args{
				rctx: models.RequestCtx{},
			},
			setup: func(mme *mock_metrics.MockMetricsEngine) {},
			getHttpRequest: func() *http.Request {
				req, _ := http.NewRequest("GET", "http://anyurl.com", nil)
				return req
			},
		},
		{
			name: "Empty cookie and non-empty partner config map",
			args: args{
				rctx: models.RequestCtx{
					PartnerConfigMap: map[int]map[string]string{
						1: {
							models.SERVER_SIDE_FLAG:    "1",
							models.PREBID_PARTNER_NAME: "partner1",
							models.BidderCode:          "bidder1",
						},
					},
					PubIDStr: "5890",
				},
			},
			setup: func(mme *mock_metrics.MockMetricsEngine) {
				models.SyncerMap = make(map[string]usersync.Syncer)
				mme.EXPECT().RecordPublisherPartnerNoCookieStats("5890", "bidder1")
			},
			getHttpRequest: func() *http.Request {
				req, _ := http.NewRequest("GET", "http://anyurl.com", nil)
				return req
			},
		},
		{
			name: "only client side partner in config map",
			args: args{
				rctx: models.RequestCtx{
					PartnerConfigMap: map[int]map[string]string{
						1: {
							models.SERVER_SIDE_FLAG:    "0",
							models.PREBID_PARTNER_NAME: "partner1",
							models.BidderCode:          "bidder1",
						},
					},
					PubIDStr: "5890",
				},
			},
			setup: func(mme *mock_metrics.MockMetricsEngine) {
				models.SyncerMap = make(map[string]usersync.Syncer)
			},
			getHttpRequest: func() *http.Request {
				req, _ := http.NewRequest("GET", "http://anyurl.com", nil)
				return req
			},
		},
		{
			name: "GetUID returns empty uid",
			args: args{
				rctx: models.RequestCtx{
					PartnerConfigMap: map[int]map[string]string{
						1: {
							models.SERVER_SIDE_FLAG:    "1",
							models.PREBID_PARTNER_NAME: "pubmatic",
							models.BidderCode:          "pubmatic",
						},
					},
					PubIDStr: "5890",
				},
			},
			setup: func(mme *mock_metrics.MockMetricsEngine) {
				models.SyncerMap = map[string]usersync.Syncer{
					"pubmatic": fakeSyncer{
						key: "pubmatic",
					},
				}
				mme.EXPECT().RecordPublisherPartnerNoCookieStats("5890", "pubmatic")
			},
			getHttpRequest: func() *http.Request {
				req, _ := http.NewRequest("GET", "http://anyurl.com", nil)
				return req
			},
		},
		{
			name: "GetUID returns non empty uid",
			args: args{
				rctx: models.RequestCtx{
					PartnerConfigMap: map[int]map[string]string{
						1: {
							models.SERVER_SIDE_FLAG:    "1",
							models.PREBID_PARTNER_NAME: "pubmatic",
							models.BidderCode:          "pubmatic",
						},
					},
					PubIDStr: "5890",
				},
			},
			setup: func(mme *mock_metrics.MockMetricsEngine) {
				models.SyncerMap = map[string]usersync.Syncer{
					"pubmatic": fakeSyncer{
						key: "pubmatic",
					},
				}
			},
			getHttpRequest: func() *http.Request {
				req, _ := http.NewRequest("GET", "http://anyurl.com", nil)

				cookie := &http.Cookie{
					Name:  "uids",
					Value: "ewoJInRlbXBVSURzIjogewoJCSJwdWJtYXRpYyI6IHsKCQkJInVpZCI6ICI3RDc1RDI1Ri1GQUM5LTQ0M0QtQjJEMS1CMTdGRUUxMUUwMjciLAoJCQkiZXhwaXJlcyI6ICIyMDIyLTEwLTMxVDA5OjE0OjI1LjczNzI1Njg5OVoiCgkJfQoJfSwKCSJiZGF5IjogIjIwMjItMDUtMTdUMDY6NDg6MzguMDE3OTg4MjA2WiIKfQ==",
				}
				req.AddCookie(cookie)
				return req
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup(mockEngine)
			tc.args.rctx.MetricsEngine = mockEngine
			tc.args.rctx.ParsedUidCookie = usersync.ReadCookie(tc.getHttpRequest(), usersync.Base64Decoder{}, &config.HostCookie{})
			RecordPublisherPartnerNoCookieStats(tc.args.rctx)
		})
	}
}

// fakeSyncer implements syncer interface for unit test cases
type fakeSyncer struct {
	key string
}

func (s fakeSyncer) Key() string {
	return s.key
}

func (s fakeSyncer) DefaultSyncType() usersync.SyncType {
	return usersync.SyncType("")
}

func (s fakeSyncer) DefaultResponseFormat() usersync.SyncType {
	return usersync.SyncType("")
}

func (s fakeSyncer) SupportsType(syncTypes []usersync.SyncType) bool {
	return false
}

func (fakeSyncer) GetSync([]usersync.SyncType, macros.UserSyncPrivacy) (usersync.Sync, error) {
	return usersync.Sync{}, nil
}

func TestGetDevicePlatform(t *testing.T) {
	type args struct {
		rCtx       models.RequestCtx
		bidRequest *openrtb2.BidRequest
	}
	tests := []struct {
		name string
		args args
		want models.DevicePlatform
	}{
		{
			name: "Test_empty_platform",
			args: args{
				rCtx: models.RequestCtx{
					Platform: "",
				},
				bidRequest: nil,
			},
			want: models.DevicePlatformNotDefined,
		},
		{
			name: "Test_platform_amp",
			args: args{
				rCtx: models.RequestCtx{
					Platform: "amp",
				},
				bidRequest: nil,
			},
			want: models.DevicePlatformMobileWeb,
		},
		{
			name: "Test_platform_in-app_with_iOS_UA",
			args: args{
				rCtx: models.RequestCtx{
					DeviceCtx: models.DeviceCtx{UA: "Mozilla/5.0 (iPhone; CPU iPhone OS 13_2_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.3 Mobile/15E148 Safari/604.1"},
					Platform:  "in-app",
				},
				bidRequest: nil,
			},
			want: models.DevicePlatformMobileAppIos,
		},
		{
			name: "Test_platform_in-app_with_Android_UA",
			args: args{
				rCtx: models.RequestCtx{
					DeviceCtx: models.DeviceCtx{UA: "Mozilla/5.0 (Linux; Android 7.0) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Focus/1.0 Chrome/59.0.3029.83 Mobile Safari/537.36"},
					Platform:  "in-app",
				},
				bidRequest: nil,
			},
			want: models.DevicePlatformMobileAppAndroid,
		},
		{
			name: "Test_platform_in-app_with_device.os_android",
			args: args{
				rCtx: models.RequestCtx{
					Platform: "in-app",
				},
				bidRequest: getORTBRequest("android", "", 0, false, true),
			},
			want: models.DevicePlatformMobileAppAndroid,
		},
		{
			name: "Test_platform_in-app_with_device.os_ios",
			args: args{
				rCtx: models.RequestCtx{
					DeviceCtx: models.DeviceCtx{UA: "Mozilla/5.0 (iPhone; CPU iPhone OS 13_2_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.3 Mobile/15E148 Safari/604.1"},
					Platform:  "in-app",
				},
				bidRequest: getORTBRequest("ios", "", 0, false, true),
			},
			want: models.DevicePlatformMobileAppIos,
		},
		{
			name: "Test_platform_in-app_with_device.ua_for_ios",
			args: args{
				rCtx: models.RequestCtx{
					DeviceCtx: models.DeviceCtx{UA: "Mozilla/5.0 (iPhone; CPU iPhone OS 13_2_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.3 Mobile/15E148 Safari/604.1"},
					Platform:  "in-app",
				},
				bidRequest: getORTBRequest("", "Mozilla/5.0 (iPhone; CPU iPhone OS 13_2_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.3 Mobile/15E148 Safari/604.1", 0, false, true),
			},
			want: models.DevicePlatformMobileAppIos,
		},
		{
			name: "Test_platform_display_with_device.deviceType_for_mobile",
			args: args{
				rCtx: models.RequestCtx{
					Platform: "display",
				},
				bidRequest: getORTBRequest("", "", adcom1.DeviceMobile, false, true),
			},
			want: models.DevicePlatformMobileWeb,
		},
		{
			name: "Test_platform_display_with_device.deviceType_for_tablet",
			args: args{
				rCtx: models.RequestCtx{
					Platform: "display",
				},
				bidRequest: getORTBRequest("", "", adcom1.DeviceMobile, false, true),
			},
			want: models.DevicePlatformMobileWeb,
		},
		{
			name: "Test_platform_display_with_device.deviceType_for_desktop",
			args: args{
				rCtx: models.RequestCtx{
					Platform: "display",
				},
				bidRequest: getORTBRequest("", "", adcom1.DevicePC, true, false),
			},
			want: models.DevicePlatformDesktop,
		},
		{
			name: "Test_platform_display_with_device.ua_for_mobile",
			args: args{
				rCtx: models.RequestCtx{
					DeviceCtx: models.DeviceCtx{UA: "Mozilla/5.0 (iPhone; CPU iPhone OS 13_2_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.3 Mobile/15E148 Safari/604.1"},
					Platform:  "display",
				},
				bidRequest: getORTBRequest("", "Mozilla/5.0 (iPhone; CPU iPhone OS 13_2_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.3 Mobile/15E148 Safari/604.1", 0, true, false),
			},
			want: models.DevicePlatformMobileWeb,
		},
		{
			name: "Test_platform_display_without_ua,_os_&_deviceType",
			args: args{
				rCtx: models.RequestCtx{
					Platform: "display",
				},
				bidRequest: getORTBRequest("", "", 0, false, true),
			},
			want: models.DevicePlatformDesktop,
		},
		{
			name: "Test_platform_video_with_deviceType_as_CTV",
			args: args{
				rCtx: models.RequestCtx{
					Platform: "video",
					PubIDStr: "5890",
				},
				bidRequest: getORTBRequest("", "", adcom1.DeviceTV, true, false),
			},
			want: models.DevicePlatformConnectedTv,
		},
		{
			name: "Test_platform_video_with_deviceType_as_connected_device",
			args: args{
				rCtx: models.RequestCtx{
					Platform: "video",
					PubIDStr: "5890",
				},
				bidRequest: getORTBRequest("", "", adcom1.DeviceConnected, true, false),
			},
			want: models.DevicePlatformConnectedTv,
		},
		{
			name: "Test_platform_video_with_deviceType_as_set_top_box",
			args: args{
				rCtx: models.RequestCtx{
					Platform: "video",
					PubIDStr: "5890",
				},
				bidRequest: getORTBRequest("", "", adcom1.DeviceSetTopBox, false, true),
			},
			want: models.DevicePlatformConnectedTv,
		},
		{
			name: "Test_platform_video_with_nil_values",
			args: args{
				rCtx: models.RequestCtx{
					Platform: "video",
				},
				bidRequest: getORTBRequest("", "", 0, true, false),
			},
			want: models.DevicePlatformDesktop,
		},
		{
			name: "Test_platform_video_with_site_entry",
			args: args{
				rCtx: models.RequestCtx{
					Platform: "video",
				},
				bidRequest: getORTBRequest("", "", 0, true, false),
			},
			want: models.DevicePlatformDesktop,
		},
		{
			name: "Test_platform_video_with_site_entry_and_mobile_UA",
			args: args{
				rCtx: models.RequestCtx{
					DeviceCtx: models.DeviceCtx{UA: "Mozilla/5.0 (iPhone; CPU iPhone OS 13_2_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.3 Mobile/15E148 Safari/604.1"},
					Platform:  "video",
				},
				bidRequest: getORTBRequest("", "Mozilla/5.0 (iPhone; CPU iPhone OS 13_2_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.3 Mobile/15E148 Safari/604.1", 0, true, false),
			},
			want: models.DevicePlatformMobileWeb,
		},
		{
			name: "Test_platform_video_with_app_entry_and_iOS_mobile_UA",
			args: args{
				rCtx: models.RequestCtx{
					DeviceCtx: models.DeviceCtx{UA: "Mozilla/5.0 (iPhone; CPU iPhone OS 13_2_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.3 Mobile/15E148 Safari/604.1"},
					Platform:  "video",
				},
				bidRequest: getORTBRequest("", "Mozilla/5.0 (iPhone; CPU iPhone OS 13_2_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.3 Mobile/15E148 Safari/604.1", 0, false, true),
			},
			want: models.DevicePlatformMobileAppIos,
		},
		{
			name: "Test_platform_video_with_app_entry_and_android_mobile_UA",
			args: args{
				rCtx: models.RequestCtx{
					DeviceCtx: models.DeviceCtx{UA: "Mozilla/5.0 (Linux; Android 7.0) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Focus/1.0 Chrome/59.0.3029.83 Mobile Safari/537.36"},
					Platform:  "video",
				},

				bidRequest: getORTBRequest("", "Mozilla/5.0 (Linux; Android 7.0) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Focus/1.0 Chrome/59.0.3029.83 Mobile Safari/537.36", 0, false, true),
			},
			want: models.DevicePlatformMobileAppAndroid,
		},
		{
			name: "Test_platform_video_with_app_entry_and_android_os",
			args: args{
				rCtx: models.RequestCtx{
					Platform: "video",
				},
				bidRequest: getORTBRequest("android", "", 0, false, true),
			},
			want: models.DevicePlatformMobileAppAndroid,
		},
		{
			name: "Test_platform_video_with_app_entry_and_ios_os",
			args: args{
				rCtx: models.RequestCtx{
					Platform: "video",
				},
				bidRequest: getORTBRequest("ios", "", 0, false, true),
			},
			want: models.DevicePlatformMobileAppIos,
		},
		{
			name: "Test_platform_video_with_CTV_and_device_type",
			args: args{
				rCtx: models.RequestCtx{
					DeviceCtx: models.DeviceCtx{UA: "Mozilla/5.0 (SMART-TV; Linux; Tizen 4.0) AppleWebKit/538.1 (KHTML, like Gecko) Version/4.0 TV Safari/538.1"},
					Platform:  "video",
					PubIDStr:  "5890",
				},
				bidRequest: getORTBRequest("", "Mozilla/5.0 (SMART-TV; Linux; Tizen 4.0) AppleWebKit/538.1 (KHTML, like Gecko) Version/4.0 TV Safari/538.1", 3, false, true),
			},
			want: models.DevicePlatformConnectedTv,
		},
		{
			name: "Test_platform_video_with_CTV_and_no_device_type",
			args: args{
				rCtx: models.RequestCtx{
					DeviceCtx: models.DeviceCtx{UA: "AppleCoreMedia/1.0.0.20L498 (Apple TV; U; CPU OS 16_4_1 like Mac OS X; en_us)"},
					Platform:  "video",
				},
				bidRequest: getORTBRequest("", "AppleCoreMedia/1.0.0.20L498 (Apple TV; U; CPU OS 16_4_1 like Mac OS X; en_us)", 0, true, false),
			},
			want: models.DevicePlatformConnectedTv,
		},
		{
			name: "Test_platform_video_for_non_CTV_User_agent_with_device_type_7",
			args: args{
				rCtx: models.RequestCtx{
					DeviceCtx: models.DeviceCtx{UA: "AppleCoreMedia/1.0.0.20L498 (iphone ; U; CPU OS 16_4_1 like Mac OS X; en_us)"},
					Platform:  "video",
					PubIDStr:  "5890",
				},
				bidRequest: getORTBRequest("", "AppleCoreMedia/1.0.0.20L498 (iphone ; U; CPU OS 16_4_1 like Mac OS X; en_us)", 7, false, true),
			},
			want: models.DevicePlatformConnectedTv,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getDevicePlatform(tt.args.rCtx, tt.args.bidRequest)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsMobile(t *testing.T) {
	type args struct {
		deviceType      adcom1.DeviceType
		userAgentString string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test_for_deviceType_1",
			args: args{
				deviceType:      adcom1.DeviceMobile,
				userAgentString: "",
			},
			want: true,
		},
		{
			name: "Test_for_deviceType_2",
			args: args{
				deviceType:      adcom1.DevicePC,
				userAgentString: "",
			},
			want: false,
		},
		{
			name: "Test_for_deviceType_4:_phone",
			args: args{
				deviceType:      adcom1.DevicePhone,
				userAgentString: "",
			},
			want: true,
		},
		{
			name: "Test_for_deviceType_5:_tablet",
			args: args{
				deviceType:      adcom1.DeviceTablet,
				userAgentString: "",
			},
			want: true,
		},
		{
			name: "Test_for_iPad_User-Agent",
			args: args{
				deviceType:      0,
				userAgentString: "Mozilla/5.0 (iPad; CPU OS 10_3_3 like Mac OS X) AppleWebKit/603.3.8 (KHTML, like Gecko) Mobile/14G60",
			},
			want: true,
		},
		{
			name: "Test_for_iPhone_User-Agent",
			args: args{
				deviceType:      0,
				userAgentString: "Mozilla/5.0 (iPhone; CPU iPhone OS 13_2_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.3 Mobile/15E148 Safari/604.1",
			},
			want: true,
		},
		{
			name: "Test_for_Safari_web-browser_on_mobile_User-Agent",
			args: args{
				deviceType:      0,
				userAgentString: "MobileSafari/602.1 CFNetwork/811.5.4 Darwin/16.7.0",
			},
			want: true,
		},
		{
			name: "Test_for_Outlook_3_application_on_mobile_phone_User-Agent",
			args: args{
				deviceType:      0,
				userAgentString: "Outlook-iOS/709.2144270.prod.iphone (3.23.0)",
			},
			want: true,
		},
		{
			name: "Test_for_firefox_11_tablet_User-Agent",
			args: args{
				deviceType:      0,
				userAgentString: "Mozilla/5.0 (Android 4.4; Tablet; rv:41.0) Gecko/41.0 Firefox/41.0",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isMobile(tt.args.deviceType, tt.args.userAgentString)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_getUserAgent(t *testing.T) {
	type args struct {
		request   *openrtb2.BidRequest
		defaultUA string
	}
	tests := []struct {
		name   string
		args   args
		wantUA string
	}{
		{
			name: "request_is_nil",
			args: args{
				request:   nil,
				defaultUA: "default-ua",
			},

			wantUA: "default-ua",
		},
		{
			name: "req.device_is_nil",
			args: args{
				request: &openrtb2.BidRequest{
					Device: nil,
				},
				defaultUA: "default-ua",
			},
			wantUA: "default-ua",
		},
		{
			name: "req.device.ua_empty",
			args: args{
				request: &openrtb2.BidRequest{
					Device: &openrtb2.Device{
						UA: "",
					},
				},
				defaultUA: "default-ua",
			},
			wantUA: "default-ua",
		},
		{
			name: "req.device.ua_valid",
			args: args{
				request: &openrtb2.BidRequest{
					Device: &openrtb2.Device{
						UA: "Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36",
					},
				},
				defaultUA: "default-ua",
			},
			wantUA: "Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ua := getUserAgent(tt.args.request, tt.args.defaultUA)
			assert.Equal(t, tt.wantUA, ua, "mismatched UA")
		})
	}
}

func TestIsIos(t *testing.T) {
	type args struct {
		os              string
		userAgentString string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "iOS_test_for_Web_Browser_Mobile-Tablet",
			args: args{
				os:              "",
				userAgentString: "Mozilla/5.0 (iPad; CPU OS 10_3_3 like Mac OS X) AppleWebKit/603.3.8 (KHTML, like Gecko) Mobile/14G60",
			},
			want: true,
		},
		{
			name: "iOS_test_for_Safari_13_Mobile-Phone",
			args: args{
				os:              "",
				userAgentString: "Mozilla/5.0 (iPhone; CPU iPhone OS 13_2_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.3 Mobile/15E148 Safari/604.1",
			},
			want: true,
		},
		{
			name: "Test_for_Safari_web-browser_on_mobile_User-Agent",
			args: args{
				os:              "",
				userAgentString: "MobileSafari/602.1 CFNetwork/811.5.4 Darwin/16.7.0",
			},
			want: true,
		},
		{
			name: "Test_for_iPhone_XR_simulator_User-Agent",
			args: args{
				os:              "",
				userAgentString: "Mozilla/5.0 (iPhone; CPU iPhone OS 12_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) FxiOS/7.0.4 Mobile/16B91 Safari/605.1.15",
			},
			want: true,
		},
		{
			name: "Test_for_Outlook_3_Application_User-Agent",
			args: args{
				os:              "",
				userAgentString: "Outlook-iOS/709.2144270.prod.iphone (3.23.0)",
			},
			want: true,
		},
		{
			name: "iOS_test_for_Safari_12_Mobile-Phone",
			args: args{
				os:              "",
				userAgentString: "Mozilla/5.0 (iPhone; CPU iPhone OS 12_1_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/12.0 Mobile/15E148 Safari/604.1",
			},
			want: true,
		},
		{
			name: "Android_HiPad_X_UA_should_not_match_iOS",
			args: args{
				os:              "",
				userAgentString: "Mozilla/5.0 (Linux; Android 10; HiPad X Build/QP1A.190711.020; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/145.0.7632.79 Safari/537.36",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isIos(tt.args.os, tt.args.userAgentString)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsAndroid(t *testing.T) {
	type args struct {
		os              string
		userAgentString string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test_android_with_correct_os_value",
			args: args{
				os:              "android",
				userAgentString: "",
			},
			want: true,
		},
		{
			name: "Test_android_with_invalid_os_value",
			args: args{
				os:              "ios",
				userAgentString: "",
			},
			want: false,
		},
		{
			name: "Test_android_with_invalid_osv_alue",
			args: args{
				os:              "",
				userAgentString: "",
			},
			want: false,
		},
		{
			name: "Test_android_with_UA_value",
			args: args{
				os:              "",
				userAgentString: "Mozilla/5.0 (Linux; Android 7.0) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Focus/1.0 Chrome/59.0.3029.83 Mobile Safari/537.36",
			},
			want: true,
		},
		{
			name: "Android_HiPad_X_UA_should_match_Android",
			args: args{
				os:              "",
				userAgentString: "Mozilla/5.0 (Linux; Android 10; HiPad X Build/QP1A.190711.020; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/145.0.7632.79 Safari/537.36",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAndroid(tt.args.os, tt.args.userAgentString)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetMobileAppPlatform(t *testing.T) {
	tests := []struct {
		name            string
		os              string
		userAgentString string
		want            models.DevicePlatform
	}{
		{
			name:            "os_ios_exact",
			os:              "ios",
			userAgentString: "",
			want:            models.DevicePlatformMobileAppIos,
		},
		{
			name:            "os_iOS_case_insensitive",
			os:              "iOS",
			userAgentString: "",
			want:            models.DevicePlatformMobileAppIos,
		},
		{
			name:            "os_ios_with_version_not_exact_match_uses_ua_fallback",
			os:              "ios 15.0",
			userAgentString: "",
			want:            models.DevicePlatformNotDefined,
		},
		{
			name:            "os_ios_with_version_ua_iphone_fallback_ios",
			os:              "ios 15.0",
			userAgentString: "Mozilla/5.0 (iPhone; CPU iPhone OS 15_0 like Mac OS X) AppleWebKit/605.1.15",
			want:            models.DevicePlatformMobileAppIos,
		},
		{
			name:            "os_padded_ios_not_equal_to_ios_without_ua",
			os:              "  ios  ",
			userAgentString: "",
			want:            models.DevicePlatformNotDefined,
		},
		{
			name:            "os_android_exact",
			os:              "android",
			userAgentString: "",
			want:            models.DevicePlatformMobileAppAndroid,
		},
		{
			name:            "os_Android_case_insensitive",
			os:              "Android",
			userAgentString: "",
			want:            models.DevicePlatformMobileAppAndroid,
		},
		{
			name:            "os_android_with_version_not_exact_match_without_ua",
			os:              "android 10",
			userAgentString: "",
			want:            models.DevicePlatformNotDefined,
		},
		{
			name:            "os_android_with_version_ua_android_fallback",
			os:              "android 10",
			userAgentString: "Mozilla/5.0 (Linux; Android 10; Pixel 3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.120 Mobile Safari/537.36",
			want:            models.DevicePlatformMobileAppAndroid,
		},
		{
			name:            "os_ios_without_space_is_invalid_without_ua",
			os:              "ios17",
			userAgentString: "",
			want:            models.DevicePlatformNotDefined,
		},
		{
			name:            "os_android_without_space_is_invalid_without_ua",
			os:              "android14",
			userAgentString: "",
			want:            models.DevicePlatformNotDefined,
		},
		{
			name:            "os_empty_ua_ios_fallback",
			os:              "",
			userAgentString: "Mozilla/5.0 (iPhone; CPU iPhone OS 13_2_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.3 Mobile/15E148 Safari/604.1",
			want:            models.DevicePlatformMobileAppIos,
		},
		{
			name:            "ua_ipad_should_be_ios",
			os:              "",
			userAgentString: "Mozilla/5.0 (iPad; CPU OS 10_3_3 like Mac OS X) AppleWebKit/603.3.8 (KHTML, like Gecko) Mobile/14G60",
			want:            models.DevicePlatformMobileAppIos,
		},
		{
			name:            "ua_mobile_safari_darwin_should_be_ios",
			os:              "",
			userAgentString: "MobileSafari/602.1 CFNetwork/811.5.4 Darwin/16.7.0",
			want:            models.DevicePlatformMobileAppIos,
		},
		{
			name:            "ua_outlook_ios_should_be_ios",
			os:              "",
			userAgentString: "Outlook-iOS/709.2144270.prod.iphone (3.23.0)",
			want:            models.DevicePlatformMobileAppIos,
		},
		{
			name:            "ua_iphone_fxios_should_be_ios",
			os:              "",
			userAgentString: "Mozilla/5.0 (iPhone; CPU iPhone OS 12_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) FxiOS/7.0.4 Mobile/16B91 Safari/605.1.15",
			want:            models.DevicePlatformMobileAppIos,
		},
		{
			name:            "os_empty_ua_android_fallback",
			os:              "",
			userAgentString: "Mozilla/5.0 (Linux; Android 7.0) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/59.0.3029.83 Mobile Safari/537.36",
			want:            models.DevicePlatformMobileAppAndroid,
		},
		{
			name:            "hipad_max_android_ua_should_be_android",
			os:              "",
			userAgentString: "Mozilla/5.0 (Linux; Android 12; HiPad Max Build/SKQ1.220119.001; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/145.0.7632.120 Safari/537.36",
			want:            models.DevicePlatformMobileAppAndroid,
		},
		{
			name:            "ua_partial_word_hipad_without_android_should_not_be_ios",
			os:              "Android",
			userAgentString: "Mozilla/5.0 (Linux; HiPad Max Build/SKQ1.220119.001; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/145.0.7632.120 Safari/537.36",
			want:            models.DevicePlatformMobileAppAndroid,
		},
		{
			name:            "ua_partial_word_myiphoneapp_should_not_be_ios",
			os:              "ios",
			userAgentString: "MyIphoneApp/1.0 (custom agent)",
			want:            models.DevicePlatformMobileAppIos,
		},
		{
			name:            "os_invalid_ua_empty_not_defined",
			os:              "windows",
			userAgentString: "",
			want:            models.DevicePlatformNotDefined,
		},
		{
			name:            "os_empty_ua_empty_not_defined",
			os:              "",
			userAgentString: "",
			want:            models.DevicePlatformNotDefined,
		},
		{
			name:            "os_ios_overrides_android_ua",
			os:              "ios",
			userAgentString: "Mozilla/5.0 (Linux; Android 10) AppleWebKit/537.36",
			want:            models.DevicePlatformMobileAppIos,
		},
		{
			name:            "os_android_overrides_ios_ua",
			os:              "android",
			userAgentString: "Mozilla/5.0 (iPhone; CPU iPhone OS 12_1 like Mac OS X) AppleWebKit/605.1.15",
			want:            models.DevicePlatformMobileAppAndroid,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getMobileAppPlatform(tt.os, tt.userAgentString)
			assert.Equal(t, tt.want, got)
		})
	}
}

func getORTBRequest(os, ua string, deviceType adcom1.DeviceType, withSite, withApp bool) *openrtb2.BidRequest {
	request := new(openrtb2.BidRequest)

	if withSite {
		request.Site = &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				ID: "1010",
			},
		}
	}

	if withApp {
		request.App = &openrtb2.App{
			Publisher: &openrtb2.Publisher{
				ID: "1010",
			},
		}
	}

	request.Device = new(openrtb2.Device)
	request.Device.UA = ua

	request.Device.OS = os

	request.Device.DeviceType = deviceType

	return request
}

func TestGetSourceAndOrigin(t *testing.T) {
	type args struct {
		bidRequest *openrtb2.BidRequest
	}
	type want struct {
		source string
		origin string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "bidRequest_site_conatins_Domain",
			args: args{
				bidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Page:   "http://www.test.com",
						Domain: "test.com",
					},
				},
			},
			want: want{
				source: "test.com",
				origin: "test.com",
			},
		},
		{
			name: "bidRequest_conatins_Site_Page",
			args: args{
				bidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Page: "http://www.test.com",
					},
				},
			},
			want: want{
				source: "www.test.com",
				origin: "www.test.com",
			},
		},
		{
			name: "bidRequest_conatins_App_Bundle",
			args: args{
				bidRequest: &openrtb2.BidRequest{
					App: &openrtb2.App{
						Bundle: "com.pub.test",
					},
				},
			},
			want: want{
				source: "com.pub.test",
				origin: "com.pub.test",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, origin := getSourceAndOrigin(tt.args.bidRequest)
			assert.Equal(t, tt.want.source, source)
			assert.Equal(t, tt.want.origin, origin)
		})
	}
}

func TestGetHostName(t *testing.T) {
	var (
		node string
		pod  string
	)

	saveEnvVarsForServerName := func() {
		node, _ = os.LookupEnv(models.ENV_VAR_NODE_NAME)
		pod, _ = os.LookupEnv(models.ENV_VAR_POD_NAME)
	}

	resetEnvVarsForServerName := func() {
		os.Setenv(models.ENV_VAR_NODE_NAME, node)
		os.Setenv(models.ENV_VAR_POD_NAME, pod)
	}
	type args struct {
		nodeName string
		podName  string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "default_value",
			args: args{},
			want: models.DEFAULT_NODENAME + ":" + models.DEFAULT_PODNAME,
		},
		{
			name: "valid_name",
			args: args{
				nodeName: "sfo2hyp084.sfo2.pubmatic.com",
				podName:  "ssheaderbidding-0-0-38-pr-26-2-k8s-5679748b7b-tqh42",
			},
			want: "sfo2hyp084:0-0-38-pr-26-2-k8s-5679748b7b-tqh42",
		},
		{
			name: "special_characters",
			args: args{
				nodeName: "sfo2hyp084.sfo2.pubmatic.com!!!@#$-_^%x090",
				podName:  "ssheaderbidding-0-0-38-pr-26-2-k8s-5679748b7b-tqh42",
			},
			want: "sfo2hyp084:0-0-38-pr-26-2-k8s-5679748b7b-tqh42",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saveEnvVarsForServerName()

			if len(tt.args.nodeName) > 0 {
				os.Setenv(models.ENV_VAR_NODE_NAME, tt.args.nodeName)
			}

			if len(tt.args.podName) > 0 {
				os.Setenv(models.ENV_VAR_POD_NAME, tt.args.podName)
			}

			got := GetHostName()
			assert.Equal(t, tt.want, got)

			resetEnvVarsForServerName()
		})
	}
}

func TestGetPubmaticErrorCode(t *testing.T) {
	type args struct {
		standardNBR openrtb3.NoBidReason
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "ErrMissingPublisherID",
			args: args{
				standardNBR: nbr.InvalidPublisherID,
			},
			want: 604,
		},
		{
			name: "ErrBadRequest",
			args: args{
				standardNBR: nbr.InvalidRequestExt,
			},
			want: 18,
		},
		{
			name: "ErrMissingProfileID",
			args: args{
				standardNBR: nbr.InvalidProfileID,
			},
			want: 700,
		},
		{
			name: "ErrAllPartnerThrottled",
			args: args{
				standardNBR: nbr.AllPartnerThrottled,
			},
			want: 11,
		},
		{
			name: "ErrPrebidInvalidCustomPriceGranularity",
			args: args{
				standardNBR: nbr.InvalidPriceGranularityConfig,
			},
			want: 26,
		},
		{
			name: "ErrMissingTagID",
			args: args{
				standardNBR: nbr.InvalidImpressionTagID,
			},
			want: 605,
		},
		{
			name: "ErrInvalidConfiguration",
			args: args{
				standardNBR: nbr.InvalidProfileConfiguration,
			},
			want: 6,
		},
		{
			name: "ErrInvalidConfiguration_platform",
			args: args{
				standardNBR: nbr.InvalidPlatform,
			},
			want: 6,
		},
		{
			name: "ErrInvalidConfiguration_AllSlotsDisabled",
			args: args{
				standardNBR: nbr.AllSlotsDisabled,
			},
			want: 6,
		},
		{
			name: "ErrInvalidConfiguration_ServerSidePartnerNotConfigured",
			args: args{
				standardNBR: nbr.ServerSidePartnerNotConfigured,
			},
			want: 6,
		},
		{
			name: "ErrInvalidImpression",
			args: args{
				standardNBR: nbr.InternalError,
			},
			want: 17,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getPubmaticErrorCode(tt.args.standardNBR)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_getIP(t *testing.T) {
	type args struct {
		bidRequest *openrtb2.BidRequest
		defaultIP  string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test_empty_Device_IP",
			args: args{
				bidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{
						IP: "",
					},
				},
				defaultIP: "10.20.30.40",
			},
			want: "10.20.30.40",
		},
		{
			name: "Test_valid_Device_IP",
			args: args{
				bidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{
						IP: "10.20.30.40",
					},
				},
				defaultIP: "",
			},
			want: "10.20.30.40",
		},
		{
			name: "Test_valid_Device_IP_with_default_IP",
			args: args{
				bidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{
						IP: "10.20.30.40",
					},
				},
				defaultIP: "20.30.40.50",
			},
			want: "10.20.30.40",
		},
		{
			name: "Test_empty_Device_IP_with_default_IP",
			args: args{
				bidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{
						IP: "",
					},
				},
				defaultIP: "",
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getIP(tt.args.bidRequest, tt.args.defaultIP); got != tt.want {
				t.Errorf("getIP() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckIsVideoEnabledForAMP(t *testing.T) {
	type args struct {
		adUnitConfig *adunitconfig.AdConfig
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "empty_adunitConfig",
			args: args{
				adUnitConfig: nil,
			},
			want: false,
		},
		{
			name: "adunitConfig_video_is_nil",
			args: args{
				adUnitConfig: &adunitconfig.AdConfig{
					Video: nil,
				},
			},
			want: false,
		},
		{
			name: "adunitConfig_video_is_disabled",
			args: args{
				adUnitConfig: &adunitconfig.AdConfig{
					Video: &adunitconfig.Video{
						Enabled: ptrutil.ToPtr(false),
					},
				},
			},
			want: false,
		},
		{
			name: "adunitConfig_video_is_enabled_but_empty_AmptrafficPercentage",
			args: args{
				adUnitConfig: &adunitconfig.AdConfig{
					Video: &adunitconfig.Video{
						Enabled:              ptrutil.ToPtr(true),
						AmpTrafficPercentage: nil,
					},
				},
			},
			want: true,
		},
		{
			name: "adunitConfig_video_is_enabled_but_and_AmptrafficPercentage_is_0",
			args: args{
				adUnitConfig: &adunitconfig.AdConfig{
					Video: &adunitconfig.Video{
						Enabled:              ptrutil.ToPtr(true),
						AmpTrafficPercentage: ptrutil.ToPtr(0),
					},
				},
			},
			want: false,
		},
		{
			name: "adunitConfig_video_is_enabled_but_and_AmptrafficPercentage_is_100",
			args: args{
				adUnitConfig: &adunitconfig.AdConfig{
					Video: &adunitconfig.Video{
						Enabled:              ptrutil.ToPtr(true),
						AmpTrafficPercentage: ptrutil.ToPtr(100),
					},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isVideoEnabledForAMP(tt.args.adUnitConfig)
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}

func TestGetRequestIP(t *testing.T) {

	tests := []struct {
		name    string
		request *http.Request
		body    []byte
		want    string
	}{
		{
			name: "Vaild IP present in device ip only",
			request: func() *http.Request {
				r, err := http.NewRequest("POST", "http://localhost/openrtb/2.5?sshb=1", nil)
				if err != nil {
					panic(err)
				}
				return r
			}(),
			body: []byte(`{"imp":[{"tagid":"/43743431/DMDemo","id":"div-gpt-ad-1460505748561-0","banner":{"format":[{"w":300,"h":250}]}}],"device":{"ip":"10.23.14.71","ua":"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36"},"id":"5bdd7da5-1166-40fe-a9cb-3bf3c3164cd3"}`),
			want: "10.23.14.71",
		},
		{
			name: "Vaild IP present in device ip and X-FORWARDED-FOR",
			request: func() *http.Request {
				r, err := http.NewRequest("POST", "http://localhost/openrtb/2.5?sshb=1", nil)
				if err != nil {
					panic(err)
				}
				r.Header.Add("X-FORWARDED-FOR", "10.12.13.14")
				return r
			}(),
			body: []byte(`{"imp":[{"tagid":"/43743431/DMDemo","id":"div-gpt-ad-1460505748561-0","banner":{"format":[{"w":300,"h":250}]}}],"device":{"ip":"10.23.14.71","ua":"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36"},"id":"5bdd7da5-1166-40fe-a9cb-3bf3c3164cd3"}`),
			want: "10.23.14.71",
		},
		{
			name: "Vaild IP present X-FORWARDED-FOR",
			request: func() *http.Request {
				r, err := http.NewRequest("POST", "http://localhost/openrtb/2.5?sshb=1", nil)
				if err != nil {
					panic(err)
				}
				r.Header.Add("X-FORWARDED-FOR", "10.12.13.14")
				return r
			}(),
			body: []byte(`{"imp":[{"tagid":"/43743431/DMDemo","id":"div-gpt-ad-1460505748561-0","banner":{"format":[{"w":300,"h":250}]}}],"device":{"ua":"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36"},"id":"5bdd7da5-1166-40fe-a9cb-3bf3c3164cd3"}`),
			want: "10.12.13.14",
		},
		{
			name: "No Vaild IP present",
			request: func() *http.Request {
				r, err := http.NewRequest("POST", "http://localhost/openrtb/2.5?sshb=1", nil)
				if err != nil {
					panic(err)
				}
				return r
			}(),
			body: []byte(`{"imp":[{"tagid":"/43743431/DMDemo","id":"div-gpt-ad-1460505748561-0","banner":{"format":[{"w":300,"h":250}]}}],"device":{"ua":"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36"},"id":"5bdd7da5-1166-40fe-a9cb-3bf3c3164cd3"}`),
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetRequestIP(tt.body, tt.request); got != tt.want {
				t.Errorf("GetRequestIP() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetRequestUserAgent(t *testing.T) {

	tests := []struct {
		name    string
		request *http.Request
		body    []byte
		want    string
	}{
		{
			name: "Vaild IP present in device UA only",
			request: func() *http.Request {
				r, err := http.NewRequest("POST", "http://localhost/openrtb/2.5?sshb=1", nil)
				if err != nil {
					panic(err)
				}
				return r
			}(),
			body: []byte(`{"imp":[{"tagid":"/43743431/DMDemo","id":"div-gpt-ad-1460505748561-0","banner":{"format":[{"w":300,"h":250}]}}],"device":{"ip":"10.23.14.71","ua":"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36"},"id":"5bdd7da5-1166-40fe-a9cb-3bf3c3164cd3"}`),
			want: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36",
		},
		{
			name: "Vaild IP present in device ua and header",
			request: func() *http.Request {
				r, err := http.NewRequest("POST", "http://localhost/openrtb/2.5?sshb=1", nil)
				if err != nil {
					panic(err)
				}
				r.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.169 Safari/537.36")
				return r
			}(),
			body: []byte(`{"imp":[{"tagid":"/43743431/DMDemo","id":"div-gpt-ad-1460505748561-0","banner":{"format":[{"w":300,"h":250}]}}],"device":{"ip":"10.23.14.71","ua":"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36"},"id":"5bdd7da5-1166-40fe-a9cb-3bf3c3164cd3"}`),
			want: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36",
		},
		{
			name: "Vaild IP present header only",
			request: func() *http.Request {
				r, err := http.NewRequest("POST", "http://localhost/openrtb/2.5?sshb=1", nil)
				if err != nil {
					panic(err)
				}
				r.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.169 Safari/537.36")
				return r
			}(),
			body: []byte(`{"imp":[{"tagid":"/43743431/DMDemo","id":"div-gpt-ad-1460505748561-0","banner":{"format":[{"w":300,"h":250}]}}],"device":{"ip":"10.20.12.45"},"id":"5bdd7da5-1166-40fe-a9cb-3bf3c3164cd3"}`),
			want: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.169 Safari/537.36",
		},
		{
			name: "No Vaild UA present",
			request: func() *http.Request {
				r, err := http.NewRequest("POST", "http://localhost/openrtb/2.5?sshb=1", nil)
				if err != nil {
					panic(err)
				}
				return r
			}(),
			body: []byte(`{"imp":[{"tagid":"/43743431/DMDemo","id":"div-gpt-ad-1460505748561-0","banner":{"format":[{"w":300,"h":250}]}}],"device":{"ip":"10.20.12.45"},"id":"5bdd7da5-1166-40fe-a9cb-3bf3c3164cd3"}`),
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetRequestUserAgent(tt.body, tt.request); got != tt.want {
				t.Errorf("GetRequestUserAgent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetProfileType(t *testing.T) {
	type args struct {
		partnerConfigMap map[int]map[string]string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "Empty partnerConfigMap",
			args: args{
				partnerConfigMap: map[int]map[string]string{},
			},
			want: 0,
		},
		{
			name: "partnerConfigMap with valid profile type",
			args: args{
				partnerConfigMap: map[int]map[string]string{
					-1: {
						models.ProfileTypeKey: "1",
					},
				},
			},
			want: 1,
		},
		{
			name: "partnerConfigMap with invalid profile type",
			args: args{
				partnerConfigMap: map[int]map[string]string{
					1: {
						models.ProfileTypeKey: "invalid",
					},
				},
			},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getProfileType(tt.args.partnerConfigMap)
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}

func TestGetProfileTypePlatform(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockProfileMetaData := mock_profilemetadata.NewMockProfileMetaData(ctrl)

	type args struct {
		partnerConfigMap map[int]map[string]string
		profileMetaData  profilemetadata.ProfileMetaData
	}
	tests := []struct {
		name  string
		args  args
		want  int
		setup func()
	}{
		{
			name: "Empty partnerConfigMap",
			args: args{
				partnerConfigMap: map[int]map[string]string{},
				profileMetaData:  mockProfileMetaData,
			},
			want: 0,
		},
		{
			name: "partnerConfigMap with valid platform",
			args: args{
				partnerConfigMap: map[int]map[string]string{
					-1: {
						models.PLATFORM_KEY: "in-app",
					},
				},
				profileMetaData: mockProfileMetaData,
			},
			setup: func() {
				mockProfileMetaData.EXPECT().GetProfileTypePlatform("in-app").Return(4, true)
			},
			want: 4,
		},
		{
			name: "partnerConfigMap with invalid platform",
			args: args{
				partnerConfigMap: map[int]map[string]string{
					-1: {
						models.PLATFORM_KEY: "invalid",
					},
				},
				profileMetaData: mockProfileMetaData,
			},
			setup: func() {
				mockProfileMetaData.EXPECT().GetProfileTypePlatform("invalid").Return(0, false)
			},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			got := getProfileTypePlatform(tt.args.partnerConfigMap, tt.args.profileMetaData)
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}

func TestGetAppPlatform(t *testing.T) {
	type args struct {
		partnerConfigMap map[int]map[string]string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "Empty partnerConfigMap",
			args: args{
				partnerConfigMap: map[int]map[string]string{},
			},
			want: 0,
		},
		{
			name: "partnerConfigMap with valid AppPlatform",
			args: args{
				partnerConfigMap: map[int]map[string]string{
					-1: {
						models.AppPlatformKey: "5",
					},
				},
			},
			want: 5,
		},
		{
			name: "partnerConfigMap with invalid platform",
			args: args{
				partnerConfigMap: map[int]map[string]string{
					-11: {
						models.AppPlatformKey: "invalid",
					},
				},
			},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getAppPlatform(tt.args.partnerConfigMap)
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}

func TestGetAppIntegrationPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockProfileMetaData := mock_profilemetadata.NewMockProfileMetaData(ctrl)

	type args struct {
		partnerConfigMap map[int]map[string]string
		profileMetaData  profilemetadata.ProfileMetaData
	}
	tests := []struct {
		name  string
		args  args
		want  int
		setup func()
	}{
		{
			name: "Empty partnerConfigMap",
			args: args{
				partnerConfigMap: map[int]map[string]string{},
				profileMetaData:  mockProfileMetaData,
			},
			want: -1,
		},
		{
			name: "partnerConfigMap with valid AppIntegrationPath",
			args: args{
				partnerConfigMap: map[int]map[string]string{
					-1: {
						models.IntegrationPathKey: "React Native Plugin",
					},
				},
				profileMetaData: mockProfileMetaData,
			},
			setup: func() {
				mockProfileMetaData.EXPECT().GetAppIntegrationPath("React Native Plugin").Return(3, true)

			},
			want: 3,
		},
		{
			name: "partnerConfigMap with invalid AppIntegrationPath",
			args: args{
				partnerConfigMap: map[int]map[string]string{
					-1: {
						models.IntegrationPathKey: "invalid",
					},
				},
				profileMetaData: mockProfileMetaData,
			},
			setup: func() {
				mockProfileMetaData.EXPECT().GetAppIntegrationPath("invalid").Return(0, false)

			},
			want: -1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			got := getAppIntegrationPath(tt.args.partnerConfigMap, tt.args.profileMetaData)
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}

func TestGetAppSubIntegrationPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockProfileMetaData := mock_profilemetadata.NewMockProfileMetaData(ctrl)

	type args struct {
		partnerConfigMap map[int]map[string]string
		profileMetaData  profilemetadata.ProfileMetaData
	}
	tests := []struct {
		name  string
		args  args
		want  int
		setup func()
	}{
		{
			name: "Empty partnerConfigMap",
			args: args{
				partnerConfigMap: map[int]map[string]string{},
				profileMetaData:  mockProfileMetaData,
			},
			want: -1,
		},
		{
			name: "partnerConfigMap with valid AppSubIntegrationPath",
			args: args{
				partnerConfigMap: map[int]map[string]string{
					-1: {
						models.SubIntegrationPathKey: "AppLovin Max SDK Bidding",
					},
				},
				profileMetaData: mockProfileMetaData,
			},
			setup: func() {
				mockProfileMetaData.EXPECT().GetAppSubIntegrationPath("AppLovin Max SDK Bidding").Return(8, true)
			},
			want: 8,
		},
		{
			name: "partnerConfigMap with invalid AppSubIntegrationPath",
			args: args{
				partnerConfigMap: map[int]map[string]string{
					-1: {
						models.SubIntegrationPathKey: "invalid",
					},
				},
				profileMetaData: mockProfileMetaData,
			},
			setup: func() {
				mockProfileMetaData.EXPECT().GetAppSubIntegrationPath("invalid").Return(0, false)
			},
			want: -1,
		},
		{
			name: "partnerConfigMap with inavalid AppSubIntegrationPath but valid adserver",
			args: args{
				partnerConfigMap: map[int]map[string]string{
					-1: {
						models.SubIntegrationPathKey: "invalid",
						models.AdserverKey:           "DFP",
					},
				},
				profileMetaData: mockProfileMetaData,
			},
			setup: func() {
				mockProfileMetaData.EXPECT().GetAppSubIntegrationPath("invalid").Return(0, false)
				mockProfileMetaData.EXPECT().GetAppSubIntegrationPath("DFP").Return(1, true)
			},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			got := getAppSubIntegrationPath(tt.args.partnerConfigMap, tt.args.profileMetaData)
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}

func TestSearchAccountID(t *testing.T) {
	// Correctness for lookup within Publisher object left to TestGetAccountID
	// This however tests the expected lookup paths in outer site, app and dooh
	testCases := []struct {
		description       string
		request           []byte
		expectedAccID     string
		expectedError     error
		expectedIsAppReq  bool
		expectedIsSiteReq bool
		expectedIsDOOHReq bool
	}{
		{
			description:       "No publisher available",
			request:           []byte(`{}`),
			expectedAccID:     "",
			expectedError:     nil,
			expectedIsAppReq:  false,
			expectedIsDOOHReq: false,
		},
		{
			description:       "Publisher.ID doesn't exist",
			request:           []byte(`{"site":{"publisher":{}}}`),
			expectedAccID:     "",
			expectedError:     nil,
			expectedIsAppReq:  false,
			expectedIsDOOHReq: false,
		},
		{
			description:       "Publisher.ID not a string",
			request:           []byte(`{"site":{"publisher":{"id":42}}}`),
			expectedAccID:     "",
			expectedError:     errors.New("site.publisher.id must be a string"),
			expectedIsAppReq:  false,
			expectedIsDOOHReq: false,
		},
		{
			description:       "Publisher available in request.site",
			request:           []byte(`{"site":{"publisher":{"id":"42"}}}`),
			expectedAccID:     "42",
			expectedError:     nil,
			expectedIsAppReq:  false,
			expectedIsDOOHReq: false,
		},
		{
			description:       "Publisher available in request.app",
			request:           []byte(`{"app":{"publisher":{"id":"42"}}}`),
			expectedAccID:     "42",
			expectedError:     nil,
			expectedIsAppReq:  true,
			expectedIsDOOHReq: false,
		},
		{
			description:       "Publisher available in request.dooh",
			request:           []byte(`{"dooh":{"publisher":{"id":"42"}}}`),
			expectedAccID:     "42",
			expectedError:     nil,
			expectedIsAppReq:  false,
			expectedIsDOOHReq: true,
		},
	}

	for _, test := range testCases {
		accountId, isAppReq, isDOOHReq, err := searchAccountId(test.request)
		assert.Equal(t, test.expectedAccID, accountId, "searchAccountID should return expected account ID for test case: %s", test.description)
		assert.Equal(t, test.expectedIsAppReq, isAppReq, "searchAccountID should return expected isAppReq for test case: %s", test.description)
		assert.Equal(t, test.expectedIsDOOHReq, isDOOHReq, "searchAccountID should return expected isDOOHReq for test case: %s", test.description)
		assert.Equal(t, test.expectedError, err, "searchAccountID should return expected error for test case: %s", test.description)
	}

}

func TestGetAccountIdFromRawRequest(t *testing.T) {
	testCases := []struct {
		description       string
		hasStoredRequest  bool
		storedRequest     json.RawMessage
		originalRequest   []byte
		expectedAccID     string
		expectedIsAppReq  bool
		expectedIsDOOHReq bool
		expectedError     []error
	}{
		{
			description:       "hasStoredRequest is false",
			hasStoredRequest:  false,
			storedRequest:     []byte(`{"app":{"publisher":{"id":"42"}}}`),
			originalRequest:   []byte(`{"app":{"publisher":{"id":"50"}}}`),
			expectedAccID:     "50",
			expectedError:     nil,
			expectedIsAppReq:  true,
			expectedIsDOOHReq: false,
		},
		{
			description:       "Publisher.ID doesn't exist in storedrequest",
			hasStoredRequest:  true,
			storedRequest:     []byte(`{"site":{"publisher":{}}}`),
			expectedAccID:     "unknown",
			expectedError:     nil,
			expectedIsAppReq:  false,
			expectedIsDOOHReq: false,
		},
		{
			description:       "Publisher.ID as string in original request",
			originalRequest:   []byte(`{"site":{"publisher":{"id":"42"}}}`),
			expectedAccID:     "42",
			expectedError:     nil,
			expectedIsAppReq:  false,
			expectedIsDOOHReq: false,
		},
	}
	for _, test := range testCases {
		accountId, isAppReq, isDOOHReq, err := getAccountIdFromRawRequest(test.hasStoredRequest, test.storedRequest, test.originalRequest)
		assert.Equal(t, test.expectedAccID, accountId, "getAccountIdFromRawRequest should return expected account ID for test case: %s", test.description)
		assert.Equal(t, test.expectedIsAppReq, isAppReq, "getAccountIdFromRawRequest should return expected isAppReq for test case: %s", test.description)
		assert.Equal(t, test.expectedIsDOOHReq, isDOOHReq, "getAccountIdFromRawRequest should return expected isDOOHReq for test case: %s", test.description)
		assert.Equal(t, test.expectedError, err, "getAccountIdFromRawRequest should return expected error for test case: %s", test.description)
	}

}

func TestGetStringValueFromRequest(t *testing.T) {
	testCases := []struct {
		description   string
		request       []byte
		key           []string
		expectedAccID string
		expectedError error
		expectedExist bool
	}{
		{
			description:   "Both input are nil",
			request:       nil,
			key:           nil,
			expectedAccID: "",
			expectedError: nil,
			expectedExist: false,
		},
		{
			description:   "key is nil",
			request:       []byte(`{}`),
			key:           nil,
			expectedAccID: "",
			expectedError: errors.New(" must be a string"),
			expectedExist: true,
		},
		{
			description:   "Invalid request",
			request:       []byte(`{"dooh":{"publisher":{"id":42}}}`),
			key:           []string{"dooh", "publisher", "id"},
			expectedAccID: "",
			expectedError: errors.New("dooh.publisher.id must be a string"),
			expectedExist: true,
		},
		{
			description:   "Correct input from request",
			request:       []byte(`{"dooh":{"publisher":{"id":"42"}}}`),
			key:           []string{"dooh", "publisher", "id"},
			expectedAccID: "42",
			expectedError: nil,
			expectedExist: true,
		},
	}
	for _, test := range testCases {
		accountId, exists, err := getStringValueFromRequest(test.request, test.key)
		assert.Equal(t, test.expectedAccID, accountId, "getStringValueFromRequest should return expected account ID for test case: %s", test.description)
		assert.Equal(t, test.expectedExist, exists, "getStringValueFromRequest should return expected exists for test case: %s", test.description)
		assert.Equal(t, test.expectedError, err, "getStringValueFromRequest should return expected error for test case: %s", test.description)
	}
}

func TestUpdateUserExtWithValidValues(t *testing.T) {
	type args struct {
		user *openrtb2.User
	}
	tests := []struct {
		name string
		args args
		want *openrtb2.User
	}{
		{
			name: "test_valid_user_eids",
			args: args{
				user: &openrtb2.User{
					EIDs: []openrtb2.EID{
						{
							Source: "uidapi.com",
							UIDs: []openrtb2.UID{
								{
									ID: "UID2:testUID",
								},
							},
						},
					},
				},
			},
			want: &openrtb2.User{
				EIDs: []openrtb2.EID{
					{
						Source: "uidapi.com",
						UIDs: []openrtb2.UID{
							{
								ID: "testUID",
							},
						},
					},
				},
			},
		},
		{
			name: "test_user_eids_and_user_ext_eids",
			args: args{
				user: &openrtb2.User{
					Ext: json.RawMessage(`{"eids":[{"source":"uidapi.com","uids":[{"id":"UID2:testUID"},{"id":"testUID2"}]},{"source":"euid.eu","uids":[{"id":"testeuid"}]},{"source":"liveramp.com","uids":[{"id":""}]}]}`),
					EIDs: []openrtb2.EID{
						{
							Source: "uidapi.com",
							UIDs: []openrtb2.UID{
								{
									ID: "UID2:testUID",
								},
							},
						},
					},
				},
			},
			want: &openrtb2.User{
				Ext: json.RawMessage(`{"eids":[{"source":"uidapi.com","uids":[{"id":"testUID"},{"id":"testUID2"}]},{"source":"euid.eu","uids":[{"id":"testeuid"}]}]}`),
				EIDs: []openrtb2.EID{
					{
						Source: "uidapi.com",
						UIDs: []openrtb2.UID{
							{
								ID: "testUID",
							},
						},
					},
				},
			},
		},
		{
			name: "test_user_ext_eids",
			args: args{
				user: &openrtb2.User{
					Ext: json.RawMessage(`{"eids":[{"source":"uidapi.com","uids":[{"id":"UID2:testUID"},{"id":"testUID2"}]},{"source":"euid.eu","uids":[{"id":"testeuid"}]},{"source":"liveramp.com","uids":[{"id":""}]}]}`),
				},
			},
			want: &openrtb2.User{
				Ext: json.RawMessage(`{"eids":[{"source":"uidapi.com","uids":[{"id":"testUID"},{"id":"testUID2"}]},{"source":"euid.eu","uids":[{"id":"testeuid"}]}]}`),
			},
		},
		{
			name: "test_user_ext_eids_invalid",
			args: args{
				user: &openrtb2.User{
					Ext: json.RawMessage(`{"eids":[{"source":"uidapi.com","uids":[{"id":"UID2:"},{"id":""}]},{"source":"euid.eu","uids":[{"id":"euid:"}]},{"source":"liveramp.com","uids":[{"id":""}]}]}`),
				},
			},
			want: &openrtb2.User{
				Ext: json.RawMessage(`{}`),
			},
		},
		{
			name: "test_valid_user_eids_invalid",
			args: args{
				user: &openrtb2.User{
					EIDs: []openrtb2.EID{
						{
							Source: "uidapi.com",
							UIDs: []openrtb2.UID{
								{
									ID: "UID2:",
								},
							},
						},
					},
				},
			},
			want: &openrtb2.User{},
		},
		{
			name: "test_valid_user_ext_sessionduration_impdepth",
			args: args{
				user: &openrtb2.User{
					Ext: json.RawMessage(`{"sessionduration":40,"impdepth":10}`),
				},
			},
			want: &openrtb2.User{
				Ext: json.RawMessage(`{"sessionduration":40,"impdepth":10}`),
			},
		},
		{
			name: "test_invalid_user_ext_sessionduration_impdepth",
			args: args{
				user: &openrtb2.User{
					Ext: json.RawMessage(`{
					"sessionduration": -20,
					"impdepth": -10
					}`),
				},
			},
			want: &openrtb2.User{
				Ext: json.RawMessage(`{}`),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			UpdateUserExtWithValidValues(tt.args.user)
			assert.Equal(t, tt.want, tt.args.user)
		})
	}
}

func TestUpdateImpProtocols(t *testing.T) {
	tests := []struct {
		name         string
		impProtocols []adcom1.MediaCreativeSubtype
		want         []adcom1.MediaCreativeSubtype
	}{
		{
			name:         "Empty_Protocols",
			impProtocols: []adcom1.MediaCreativeSubtype{},
			want: []adcom1.MediaCreativeSubtype{
				adcom1.CreativeVAST30,
				adcom1.CreativeVAST30Wrapper,
				adcom1.CreativeVAST40,
				adcom1.CreativeVAST40Wrapper,
			},
		},
		{
			name: "VAST20_Protocols_Present",
			impProtocols: []adcom1.MediaCreativeSubtype{
				adcom1.CreativeVAST20,
			},
			want: []adcom1.MediaCreativeSubtype{
				adcom1.CreativeVAST20,
				adcom1.CreativeVAST30,
				adcom1.CreativeVAST30Wrapper,
				adcom1.CreativeVAST40,
				adcom1.CreativeVAST40Wrapper,
			},
		},
		{
			name: "VAST30_Protocols_Present",
			impProtocols: []adcom1.MediaCreativeSubtype{
				adcom1.CreativeVAST30,
			},
			want: []adcom1.MediaCreativeSubtype{
				adcom1.CreativeVAST30,
				adcom1.CreativeVAST30Wrapper,
				adcom1.CreativeVAST40,
				adcom1.CreativeVAST40Wrapper,
			},
		},
		{
			name: "All_Protocols_Present",
			impProtocols: []adcom1.MediaCreativeSubtype{
				adcom1.CreativeVAST30,
				adcom1.CreativeVAST30Wrapper,
				adcom1.CreativeVAST40,
				adcom1.CreativeVAST40Wrapper,
			},
			want: []adcom1.MediaCreativeSubtype{
				adcom1.CreativeVAST30,
				adcom1.CreativeVAST30Wrapper,
				adcom1.CreativeVAST40,
				adcom1.CreativeVAST40Wrapper,
			},
		},
		{
			name: "Additional_Protocols_Present",
			impProtocols: []adcom1.MediaCreativeSubtype{
				adcom1.CreativeVAST30,
				adcom1.CreativeVAST30Wrapper,
				adcom1.CreativeVAST40,
				adcom1.CreativeVAST40Wrapper,
				adcom1.CreativeVAST20,
			},
			want: []adcom1.MediaCreativeSubtype{
				adcom1.CreativeVAST30,
				adcom1.CreativeVAST30Wrapper,
				adcom1.CreativeVAST40,
				adcom1.CreativeVAST40Wrapper,
				adcom1.CreativeVAST20,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UpdateImpProtocols(tt.impProtocols)
			assert.ElementsMatch(t, tt.want, got)
		})
	}
}

func TestGetDisplayManagerAndVer(t *testing.T) {
	type args struct {
		app *openrtb2.App
	}
	type want struct {
		displayManager    string
		displayManagerVer string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "app not present",
			args: args{
				app: nil,
			},
			want: want{
				displayManager:    "",
				displayManagerVer: "",
			},
		},
		{
			name: "request app object is not nil but app.ext has no source and version",
			args: args{

				app: &openrtb2.App{
					Name: "AutoScout24",
					Ext:  json.RawMessage(`{}`),
				},
			},
			want: want{
				displayManager:    "",
				displayManagerVer: "",
			},
		},
		{
			name: "request app object is not nil and app.ext has source and version",
			args: args{

				app: &openrtb2.App{
					Name: "AutoScout24",
					Ext:  json.RawMessage(`{"source":"prebid-mobile","version":"1.0.0"}`),
				},
			},
			want: want{
				displayManager:    "prebid-mobile",
				displayManagerVer: "1.0.0",
			},
		},
		{
			name: "request app object is not nil and app.ext.prebid has source and version",
			args: args{
				app: &openrtb2.App{
					Name: "AutoScout24",
					Ext:  json.RawMessage(`{"prebid":{"source":"prebid-mobile","version":"1.0.0"}}`),
				},
			},
			want: want{
				displayManager:    "prebid-mobile",
				displayManagerVer: "1.0.0",
			},
		},
		{
			name: "request app object is not nil and app.ext has only version",
			args: args{
				app: &openrtb2.App{
					Name: "AutoScout24",
					Ext:  json.RawMessage(`{"version":"1.0.0"}`),
				},
			},
			want: want{
				displayManager:    "",
				displayManagerVer: "",
			},
		},
		{
			name: "request app object is not nil and app.ext has only source",
			args: args{
				app: &openrtb2.App{
					Name: "AutoScout24",
					Ext:  json.RawMessage(`{"source":"prebid-mobile"}`),
				},
			},
			want: want{
				displayManager:    "",
				displayManagerVer: "",
			},
		},
		{
			name: "request app object is not nil and app.ext have empty source but version is present",
			args: args{
				app: &openrtb2.App{
					Name: "AutoScout24",
					Ext:  json.RawMessage(`{"source":"", "version":"1.0.0"}`),
				},
			},
			want: want{
				displayManager:    "",
				displayManagerVer: "",
			},
		},
		{
			name: "request app object is not nil and app.ext have empty version but source is present",
			args: args{
				app: &openrtb2.App{
					Name: "AutoScout24",
					Ext:  json.RawMessage(`{"source":"prebid-mobile", "version":""}`),
				},
			},
			want: want{
				displayManager:    "",
				displayManagerVer: "",
			},
		},
		{
			name: "request app object is not nil and both app.ext and app.ext.prebid have source and version",
			args: args{
				app: &openrtb2.App{
					Name: "AutoScout24",
					Ext:  json.RawMessage(`{"source":"prebid-mobile-android","version":"2.0.0","prebid":{"source":"prebid-mobile","version":"1.0.0"}}`),
				},
			},
			want: want{
				displayManager:    "prebid-mobile",
				displayManagerVer: "1.0.0",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			displayManager, displayManagerVer := getDisplayManagerAndVer(tt.args.app)
			assert.Equal(t, tt.want.displayManager, displayManager)
			assert.Equal(t, tt.want.displayManagerVer, displayManagerVer)
		})
	}
}

func TestGetAdunitFormat(t *testing.T) {
	tests := []struct {
		name   string
		imp    openrtb2.Imp
		reward *int8
		want   string
	}{
		{
			name: "reward is not nil and imp.video is not nil",
			imp: openrtb2.Imp{
				Video: &openrtb2.Video{},
			},
			reward: openrtb2.Int8Ptr(1),
			want:   models.AdUnitFormatRwddVideo,
		},
		{
			name: "reward is not nil and imp.video is nil",
			imp: openrtb2.Imp{
				Video: nil,
			},
			reward: openrtb2.Int8Ptr(1),
			want:   "",
		},
		{
			name: "reward is nil and imp.video is not nil",
			imp: openrtb2.Imp{
				Video: &openrtb2.Video{},
			},
			reward: nil,
			want:   "",
		},
		{
			name: "imp.instl is 1",
			imp: openrtb2.Imp{
				Instl: 1,
			},
			reward: nil,
			want:   models.AdUnitFormatInstl,
		},
		{
			name: "imp.banner is not nil",
			imp: openrtb2.Imp{
				Banner: &openrtb2.Banner{},
				Instl:  0,
			},
			reward: nil,
			want:   models.AdUnitFormatBanner,
		},
		{
			name: "imp.instl is not 1",
			imp: openrtb2.Imp{
				Instl: 0,
			},
			reward: nil,
			want:   "",
		},
		{
			name: "invalid adunitformat with banner and video",
			imp: openrtb2.Imp{
				Banner: &openrtb2.Banner{},
				Video:  &openrtb2.Video{},
			},
			want: "",
		},
		{
			name: "invalid adunitformat with banner and rewarded flag on",
			imp: openrtb2.Imp{
				Banner: &openrtb2.Banner{},
			},
			reward: openrtb2.Int8Ptr(1),
			want:   "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adunitFormat := getAdunitFormat(tt.reward, tt.imp)
			assert.Equal(t, tt.want, adunitFormat)
		})
	}
}

func TestOpenWrapGetMultiFloors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockFeature := mock_feature.NewMockFeature(ctrl)
	mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)

	type args struct {
		rctx   models.RequestCtx
		reward *int8
		imp    openrtb2.Imp
	}
	tests := []struct {
		name  string
		args  args
		want  *models.MultiFloors
		setup func()
	}{
		{
			name: "endpoint is not an SDK bidding path endpoint",
			args: args{
				rctx: models.RequestCtx{
					Endpoint: models.EndpointV25,
				},
			},
			want:  nil,
			setup: func() {},
		},
		{
			name: "GoogleSDK endpoint runs MBMF gating",
			args: args{
				rctx: models.RequestCtx{
					Endpoint:  models.EndpointGoogleSDK,
					PubID:     5890,
					PubIDStr:  "5890",
					ProfileID: 1234,
					DeviceCtx: models.DeviceCtx{
						DerivedCountryCode: "US",
					},
				},
				imp: openrtb2.Imp{
					TagID: "adunit",
					Instl: 1,
				},
			},
			want: &models.MultiFloors{IsActive: true, Tier1: 1, Tier2: 2, Tier3: 3},
			setup: func() {
				mockFeature.EXPECT().IsMBMFPublisherEnabled(5890).Return(true)
				mockFeature.EXPECT().IsMBMFCountryForPublisher("US", 5890).Return(true)
				mockFeature.EXPECT().IsMBMFEnabledForAdUnitFormat(5890, models.AdUnitFormatInstl).Return(true)
				mockFeature.EXPECT().GetProfileAdUnitMultiFloors(1234).Return(nil)
				mockFeature.EXPECT().GetMBMFFloorsForAdUnitFormat(5890, models.AdUnitFormatInstl).Return(&models.MultiFloors{
					IsActive: true, Tier1: 1, Tier2: 2, Tier3: 3,
				})
				mockEngine.EXPECT().RecordMBMFRequests(models.EndpointGoogleSDK, "5890", models.MBMFSuccess)
			},
		},
		{
			name: "ApplovinMAX endpoint runs MBMF gating",
			args: args{
				rctx: models.RequestCtx{
					Endpoint:  models.EndpointAppLovinMax,
					PubID:     5890,
					PubIDStr:  "5890",
					ProfileID: 1234,
					DeviceCtx: models.DeviceCtx{
						DerivedCountryCode: "US",
					},
				},
				imp: openrtb2.Imp{
					TagID: "adunit",
					Instl: 1,
				},
			},
			want: &models.MultiFloors{IsActive: true, Tier1: 1, Tier2: 2, Tier3: 3},
			setup: func() {
				mockFeature.EXPECT().IsMBMFPublisherEnabled(5890).Return(true)
				mockFeature.EXPECT().IsMBMFCountryForPublisher("US", 5890).Return(true)
				mockFeature.EXPECT().IsMBMFEnabledForAdUnitFormat(5890, models.AdUnitFormatInstl).Return(true)
				mockFeature.EXPECT().GetProfileAdUnitMultiFloors(1234).Return(nil)
				mockFeature.EXPECT().GetMBMFFloorsForAdUnitFormat(5890, models.AdUnitFormatInstl).Return(&models.MultiFloors{
					IsActive: true, Tier1: 1, Tier2: 2, Tier3: 3,
				})
				mockEngine.EXPECT().RecordMBMFRequests(models.EndpointAppLovinMax, "5890", models.MBMFSuccess)
			},
		},
		{
			name: "Unity levelplay endpoint runs MBMF gating",
			args: args{
				rctx: models.RequestCtx{
					Endpoint:  models.EndpointUnityLevelPlay,
					PubID:     5890,
					PubIDStr:  "5890",
					ProfileID: 1234,
					DeviceCtx: models.DeviceCtx{
						DerivedCountryCode: "US",
					},
				},
				imp: openrtb2.Imp{
					TagID: "adunit",
					Instl: 1,
				},
			},
			want: &models.MultiFloors{IsActive: true, Tier1: 1, Tier2: 2, Tier3: 3},
			setup: func() {
				mockFeature.EXPECT().IsMBMFPublisherEnabled(5890).Return(true)
				mockFeature.EXPECT().IsMBMFCountryForPublisher("US", 5890).Return(true)
				mockFeature.EXPECT().IsMBMFEnabledForAdUnitFormat(5890, models.AdUnitFormatInstl).Return(true)
				mockFeature.EXPECT().GetProfileAdUnitMultiFloors(1234).Return(nil)
				mockFeature.EXPECT().GetMBMFFloorsForAdUnitFormat(5890, models.AdUnitFormatInstl).Return(&models.MultiFloors{
					IsActive: true, Tier1: 1, Tier2: 2, Tier3: 3,
				})
				mockEngine.EXPECT().RecordMBMFRequests(models.EndpointUnityLevelPlay, "5890", models.MBMFSuccess)
			},
		},
		{
			name: "APS endpoint runs MBMF gating",
			args: args{
				rctx: models.RequestCtx{
					Endpoint:  models.EndpointAPS,
					PubID:     5890,
					PubIDStr:  "5890",
					ProfileID: 1234,
					DeviceCtx: models.DeviceCtx{
						DerivedCountryCode: "US",
					},
				},
				imp: openrtb2.Imp{
					TagID: "adunit",
					Instl: 1,
				},
			},
			want: &models.MultiFloors{IsActive: true, Tier1: 1, Tier2: 2, Tier3: 3},
			setup: func() {
				mockFeature.EXPECT().IsMBMFPublisherEnabled(5890).Return(true)
				mockFeature.EXPECT().IsMBMFCountryForPublisher("US", 5890).Return(true)
				mockFeature.EXPECT().IsMBMFEnabledForAdUnitFormat(5890, models.AdUnitFormatInstl).Return(true)
				mockFeature.EXPECT().GetProfileAdUnitMultiFloors(1234).Return(nil)
				mockFeature.EXPECT().GetMBMFFloorsForAdUnitFormat(5890, models.AdUnitFormatInstl).Return(&models.MultiFloors{
					IsActive: true, Tier1: 1, Tier2: 2, Tier3: 3,
				})
				mockEngine.EXPECT().RecordMBMFRequests(models.EndpointAPS, "5890", models.MBMFSuccess)
			},
		},
		{
			name: "publisher is not enabled for multi floors",
			args: args{
				rctx: models.RequestCtx{
					Endpoint:  models.EndpointAppLovinMax,
					PubID:     5890,
					PubIDStr:  "5890",
					ProfileID: 1234,
					DeviceCtx: models.DeviceCtx{
						DerivedCountryCode: "IN",
					},
				},
			},
			want: nil,
			setup: func() {
				mockFeature.EXPECT().IsMBMFPublisherEnabled(5890).Return(false)
				mockEngine.EXPECT().RecordMBMFRequests(models.EndpointAppLovinMax, "5890", models.MBMFPubDisabled)
			},
		},
		{
			name: "publisher is enabled but country disabled",
			args: args{
				rctx: models.RequestCtx{
					Endpoint:  models.EndpointAppLovinMax,
					PubID:     5890,
					PubIDStr:  "5890",
					ProfileID: 1234,
					DeviceCtx: models.DeviceCtx{
						DerivedCountryCode: "US",
					},
				},
			},
			want: nil,
			setup: func() {
				mockFeature.EXPECT().IsMBMFCountryForPublisher("US", 5890).Return(false)
				mockEngine.EXPECT().RecordMBMFRequests(models.EndpointAppLovinMax, "5890", models.MBMFCountryDisabled)
				mockFeature.EXPECT().IsMBMFPublisherEnabled(5890).Return(true)
			},
		},
		{
			name: "publisher enabled but adunitformat level disabled for instl",
			args: args{
				rctx: models.RequestCtx{
					Endpoint:  models.EndpointAppLovinMax,
					PubID:     5890,
					PubIDStr:  "5890",
					ProfileID: 1234,
					DeviceCtx: models.DeviceCtx{
						DerivedCountryCode: "US",
					},
				},
				imp: openrtb2.Imp{
					Instl: 1,
				},
			},
			want: nil,
			setup: func() {
				mockFeature.EXPECT().IsMBMFCountryForPublisher("US", 5890).Return(true)
				mockFeature.EXPECT().IsMBMFPublisherEnabled(5890).Return(true)
				mockFeature.EXPECT().IsMBMFEnabledForAdUnitFormat(5890, models.AdUnitFormatInstl).Return(false)
				mockEngine.EXPECT().RecordMBMFRequests(models.EndpointAppLovinMax, "5890", models.MBMFAdUnitFormatDisabled)
			},
		},
		{
			name: "publisher enabled for adunitformat instl and adunitlevel floors explicitly disabled",
			args: args{
				rctx: models.RequestCtx{
					Endpoint:  models.EndpointAppLovinMax,
					PubID:     5890,
					PubIDStr:  "5890",
					ProfileID: 1234,
					DeviceCtx: models.DeviceCtx{
						DerivedCountryCode: "US",
					},
				},
				imp: openrtb2.Imp{
					TagID: "adunit",
					Instl: 1,
				},
			},
			want: nil,
			setup: func() {
				mockFeature.EXPECT().IsMBMFCountryForPublisher("US", 5890).Return(true)
				mockFeature.EXPECT().IsMBMFPublisherEnabled(5890).Return(true)
				mockFeature.EXPECT().IsMBMFEnabledForAdUnitFormat(5890, models.AdUnitFormatInstl).Return(true)
				mockFeature.EXPECT().GetProfileAdUnitMultiFloors(1234).Return(map[string]*models.MultiFloors{
					"adunit": {
						IsActive: false,
					},
				})
				mockEngine.EXPECT().RecordMBMFRequests(models.EndpointAppLovinMax, "5890", models.MBMFAdUnitDisabled)
			},
		},
		{
			name: "publisher enabled for adunitformat instl and adunitlevel floors explicitly enabled",
			args: args{
				rctx: models.RequestCtx{
					Endpoint:  models.EndpointAppLovinMax,
					PubID:     5890,
					PubIDStr:  "5890",
					ProfileID: 1234,
					DeviceCtx: models.DeviceCtx{
						DerivedCountryCode: "US",
					},
				},
				imp: openrtb2.Imp{
					TagID: "adunit",
					Instl: 1,
				},
			},
			want: &models.MultiFloors{
				IsActive: true,
				Tier1:    1.1,
				Tier2:    2.1,
				Tier3:    3.1,
			},
			setup: func() {
				mockFeature.EXPECT().IsMBMFCountryForPublisher("US", 5890).Return(true)
				mockFeature.EXPECT().IsMBMFPublisherEnabled(5890).Return(true)
				mockFeature.EXPECT().IsMBMFEnabledForAdUnitFormat(5890, models.AdUnitFormatInstl).Return(true)
				mockFeature.EXPECT().GetProfileAdUnitMultiFloors(1234).Return(map[string]*models.MultiFloors{
					"adunit": {
						IsActive: true,
						Tier1:    1.1,
						Tier2:    2.1,
						Tier3:    3.1,
					},
				})
				mockEngine.EXPECT().RecordMBMFRequests(models.EndpointAppLovinMax, "5890", models.MBMFSuccess)
			},
		},
		{
			name: "publisher enabled for adunitformat instl and adunitlevel floors not found, go for pubid level adunitformat floors",
			args: args{
				rctx: models.RequestCtx{
					Endpoint:  models.EndpointAppLovinMax,
					PubID:     5890,
					PubIDStr:  "5890",
					ProfileID: 1234,
					DeviceCtx: models.DeviceCtx{
						DerivedCountryCode: "US",
					},
				},
				imp: openrtb2.Imp{
					TagID: "adunit1234",
					Instl: 1,
				},
			},
			want: &models.MultiFloors{
				IsActive: true,
				Tier1:    1.1,
				Tier2:    2.1,
				Tier3:    3.1,
			},
			setup: func() {
				mockFeature.EXPECT().IsMBMFCountryForPublisher("US", 5890).Return(true)
				mockFeature.EXPECT().IsMBMFPublisherEnabled(5890).Return(true)
				mockFeature.EXPECT().IsMBMFEnabledForAdUnitFormat(5890, models.AdUnitFormatInstl).Return(true)
				mockFeature.EXPECT().GetProfileAdUnitMultiFloors(1234).Return(nil)
				mockEngine.EXPECT().RecordMBMFRequests(models.EndpointAppLovinMax, "5890", models.MBMFSuccess)
				mockFeature.EXPECT().GetMBMFFloorsForAdUnitFormat(5890, models.AdUnitFormatInstl).Return(&models.MultiFloors{
					IsActive: true,
					Tier1:    1.1,
					Tier2:    2.1,
					Tier3:    3.1,
				})
			},
		},
		{
			name: "publisher enabled for adunitformat instl and adunitlevel floors not found, go for default adunitformat floors",
			args: args{
				rctx: models.RequestCtx{
					Endpoint:  models.EndpointAppLovinMax,
					PubID:     5890,
					PubIDStr:  "5890",
					ProfileID: 1234,
					DeviceCtx: models.DeviceCtx{
						DerivedCountryCode: "US",
					},
				},
				imp: openrtb2.Imp{
					TagID: "adunit1234",
					Instl: 1,
				},
			},
			want: &models.MultiFloors{
				IsActive: true,
				Tier1:    1,
				Tier2:    2,
				Tier3:    3,
			},
			setup: func() {
				mockFeature.EXPECT().IsMBMFCountryForPublisher("US", 5890).Return(true)
				mockFeature.EXPECT().IsMBMFPublisherEnabled(5890).Return(true)
				mockFeature.EXPECT().IsMBMFEnabledForAdUnitFormat(5890, models.AdUnitFormatInstl).Return(true)
				mockFeature.EXPECT().GetProfileAdUnitMultiFloors(1234).Return(map[string]*models.MultiFloors{})
				mockFeature.EXPECT().GetMBMFFloorsForAdUnitFormat(5890, models.AdUnitFormatInstl).Return(&models.MultiFloors{
					IsActive: true,
					Tier1:    1,
					Tier2:    2,
					Tier3:    3,
				})
				mockEngine.EXPECT().RecordMBMFRequests(models.EndpointAppLovinMax, "5890", models.MBMFSuccess)
			},
		},
		{
			name: "publisher enabled but adunitformat level disabled for rewarded video",
			args: args{
				rctx: models.RequestCtx{
					Endpoint:  models.EndpointAppLovinMax,
					PubID:     5890,
					PubIDStr:  "5890",
					ProfileID: 1234,
					DeviceCtx: models.DeviceCtx{
						DerivedCountryCode: "US",
					},
				},
				reward: openrtb2.Int8Ptr(1),
				imp: openrtb2.Imp{
					TagID: "adunit-rw",
					Video: &openrtb2.Video{},
				},
			},
			want: nil,
			setup: func() {
				mockFeature.EXPECT().IsMBMFCountryForPublisher("US", 5890).Return(true)
				mockFeature.EXPECT().IsMBMFPublisherEnabled(5890).Return(true)
				mockFeature.EXPECT().IsMBMFEnabledForAdUnitFormat(5890, models.AdUnitFormatRwddVideo).Return(false)
				mockEngine.EXPECT().RecordMBMFRequests(models.EndpointAppLovinMax, "5890", models.MBMFAdUnitFormatDisabled)
			},
		},
		{
			name: "publisher enabled for adunitformat rewarded video and adunitlevel floors explicitly disabled",
			args: args{
				rctx: models.RequestCtx{
					Endpoint:  models.EndpointAppLovinMax,
					PubID:     5890,
					PubIDStr:  "5890",
					ProfileID: 1234,
					DeviceCtx: models.DeviceCtx{
						DerivedCountryCode: "US",
					},
				},
				reward: openrtb2.Int8Ptr(1),
				imp: openrtb2.Imp{
					TagID: "adunit",
					Video: &openrtb2.Video{},
				},
			},
			want: nil,
			setup: func() {
				mockFeature.EXPECT().IsMBMFCountryForPublisher("US", 5890).Return(true)
				mockFeature.EXPECT().IsMBMFPublisherEnabled(5890).Return(true)
				mockFeature.EXPECT().IsMBMFEnabledForAdUnitFormat(5890, models.AdUnitFormatRwddVideo).Return(true)
				mockFeature.EXPECT().GetProfileAdUnitMultiFloors(1234).Return(map[string]*models.MultiFloors{
					"adunit": {
						IsActive: false,
					},
				})
				mockEngine.EXPECT().RecordMBMFRequests(models.EndpointAppLovinMax, "5890", models.MBMFAdUnitDisabled)
			},
		},
		{
			name: "publisher enabled for adunitformat rewarded video and adunitlevel floors explicitly enabled",
			args: args{
				rctx: models.RequestCtx{
					Endpoint:  models.EndpointAppLovinMax,
					PubID:     5890,
					PubIDStr:  "5890",
					ProfileID: 1234,
					DeviceCtx: models.DeviceCtx{
						DerivedCountryCode: "US",
					},
				},
				reward: openrtb2.Int8Ptr(1),
				imp: openrtb2.Imp{
					TagID: "adunit",
					Video: &openrtb2.Video{},
				},
			},
			want: &models.MultiFloors{
				IsActive: true,
				Tier1:    1.1,
				Tier2:    2.1,
				Tier3:    3.1,
			},
			setup: func() {
				mockFeature.EXPECT().IsMBMFCountryForPublisher("US", 5890).Return(true)
				mockFeature.EXPECT().IsMBMFPublisherEnabled(5890).Return(true)
				mockFeature.EXPECT().IsMBMFEnabledForAdUnitFormat(5890, models.AdUnitFormatRwddVideo).Return(true)
				mockFeature.EXPECT().GetProfileAdUnitMultiFloors(1234).Return(map[string]*models.MultiFloors{
					"adunit": {
						IsActive: true,
						Tier1:    1.1,
						Tier2:    2.1,
						Tier3:    3.1,
					},
				})
				mockEngine.EXPECT().RecordMBMFRequests(models.EndpointAppLovinMax, "5890", models.MBMFSuccess)
			},
		},
		{
			name: "publisher enabled for adunitformat rewarded video and adunitlevel floors not found, go for pubid level adunitformat floors",
			args: args{
				rctx: models.RequestCtx{
					Endpoint:  models.EndpointAppLovinMax,
					PubID:     5890,
					PubIDStr:  "5890",
					ProfileID: 1234,
					DeviceCtx: models.DeviceCtx{
						DerivedCountryCode: "US",
					},
				},
				reward: openrtb2.Int8Ptr(1),
				imp: openrtb2.Imp{
					TagID: "adunit1234",
					Video: &openrtb2.Video{},
				},
			},
			want: &models.MultiFloors{
				IsActive: true,
				Tier1:    1.1,
				Tier2:    2.1,
				Tier3:    3.1,
			},
			setup: func() {
				mockFeature.EXPECT().IsMBMFCountryForPublisher("US", 5890).Return(true)
				mockFeature.EXPECT().IsMBMFPublisherEnabled(5890).Return(true)
				mockFeature.EXPECT().IsMBMFEnabledForAdUnitFormat(5890, models.AdUnitFormatRwddVideo).Return(true)
				mockFeature.EXPECT().GetProfileAdUnitMultiFloors(1234).Return(nil)
				mockEngine.EXPECT().RecordMBMFRequests(models.EndpointAppLovinMax, "5890", models.MBMFSuccess)
				mockFeature.EXPECT().GetMBMFFloorsForAdUnitFormat(5890, models.AdUnitFormatRwddVideo).Return(&models.MultiFloors{
					IsActive: true,
					Tier1:    1.1,
					Tier2:    2.1,
					Tier3:    3.1,
				})
			},
		},
		{
			name: "publisher enabled for adunitformat rewarded video and adunitlevel floors not found, go for default adunitformat floors",
			args: args{
				rctx: models.RequestCtx{
					Endpoint:  models.EndpointAppLovinMax,
					PubID:     5890,
					PubIDStr:  "5890",
					ProfileID: 1234,
					DeviceCtx: models.DeviceCtx{
						DerivedCountryCode: "US",
					},
				},
				reward: openrtb2.Int8Ptr(1),
				imp: openrtb2.Imp{
					TagID: "adunit1234",
					Video: &openrtb2.Video{},
				},
			},
			want: &models.MultiFloors{
				IsActive: true,
				Tier1:    1,
				Tier2:    2,
				Tier3:    3,
			},
			setup: func() {
				mockFeature.EXPECT().IsMBMFCountryForPublisher("US", 5890).Return(true)
				mockFeature.EXPECT().IsMBMFPublisherEnabled(5890).Return(true)
				mockFeature.EXPECT().IsMBMFEnabledForAdUnitFormat(5890, models.AdUnitFormatRwddVideo).Return(true)
				mockFeature.EXPECT().GetProfileAdUnitMultiFloors(1234).Return(map[string]*models.MultiFloors{})
				mockFeature.EXPECT().GetMBMFFloorsForAdUnitFormat(5890, models.AdUnitFormatRwddVideo).Return(&models.MultiFloors{
					IsActive: true,
					Tier1:    1,
					Tier2:    2,
					Tier3:    3,
				})
				mockEngine.EXPECT().RecordMBMFRequests(models.EndpointAppLovinMax, "5890", models.MBMFSuccess)
			},
		},
		{
			name: "publisher enabled for adunitformat BANNER and adunitlevel floors not found, DON'T apply default adunitformat floors",
			args: args{
				rctx: models.RequestCtx{
					Endpoint:  models.EndpointAppLovinMax,
					PubID:     5890,
					PubIDStr:  "5890",
					ProfileID: 1234,
					DeviceCtx: models.DeviceCtx{
						DerivedCountryCode: "US",
					},
				},
				imp: openrtb2.Imp{
					TagID:  "adunit1234",
					Instl:  0,
					Banner: &openrtb2.Banner{},
				},
			},
			want: nil,
			setup: func() {
				mockFeature.EXPECT().IsMBMFCountryForPublisher("US", 5890).Return(true)
				mockFeature.EXPECT().IsMBMFPublisherEnabled(5890).Return(true)
				mockFeature.EXPECT().IsMBMFEnabledForAdUnitFormat(5890, models.AdUnitFormatBanner).Return(true)
				mockFeature.EXPECT().GetProfileAdUnitMultiFloors(1234).Return(map[string]*models.MultiFloors{})
				mockEngine.EXPECT().RecordMBMFRequests(models.EndpointAppLovinMax, "5890", models.MBMFAdUnitDisabled)
			},
		},
		{
			name: "banner profile adunit level floors disabled for adunit",
			args: args{
				rctx: models.RequestCtx{
					Endpoint:  models.EndpointAppLovinMax,
					PubID:     5890,
					PubIDStr:  "5890",
					ProfileID: 1234,
					DeviceCtx: models.DeviceCtx{
						DerivedCountryCode: "US",
					},
				},
				imp: openrtb2.Imp{
					TagID:  "adunit1234",
					Banner: &openrtb2.Banner{},
				},
			},
			want: nil,
			setup: func() {
				mockFeature.EXPECT().IsMBMFCountryForPublisher("US", 5890).Return(true)
				mockFeature.EXPECT().IsMBMFPublisherEnabled(5890).Return(true)
				mockFeature.EXPECT().IsMBMFEnabledForAdUnitFormat(5890, models.AdUnitFormatBanner).Return(true)
				mockFeature.EXPECT().GetProfileAdUnitMultiFloors(1234).Return(map[string]*models.MultiFloors{
					"adunit1234": {
						IsActive: false,
						Tier1:    1,
						Tier2:    2,
						Tier3:    3,
					},
				})
				mockEngine.EXPECT().RecordMBMFRequests(models.EndpointAppLovinMax, "5890", models.MBMFAdUnitDisabled)
			},
		},
		{
			name: "banner profile adunit level floors and adunitformat floors not present",
			args: args{
				rctx: models.RequestCtx{
					Endpoint:  models.EndpointAppLovinMax,
					PubID:     5890,
					PubIDStr:  "5890",
					ProfileID: 1234,
					DeviceCtx: models.DeviceCtx{
						DerivedCountryCode: "US",
					},
				},
				imp: openrtb2.Imp{
					TagID:  "adunit1234",
					Banner: &openrtb2.Banner{},
				},
			},
			want: nil,
			setup: func() {
				mockFeature.EXPECT().IsMBMFCountryForPublisher("US", 5890).Return(true)
				mockFeature.EXPECT().IsMBMFPublisherEnabled(5890).Return(true)
				mockFeature.EXPECT().IsMBMFEnabledForAdUnitFormat(5890, models.AdUnitFormatBanner).Return(true)
				mockFeature.EXPECT().GetProfileAdUnitMultiFloors(1234).Return(map[string]*models.MultiFloors{})
				mockEngine.EXPECT().RecordMBMFRequests(models.EndpointAppLovinMax, "5890", models.MBMFAdUnitDisabled)
			},
		},
		{
			name: "publisher enabled but invalid adformat",
			args: args{
				rctx: models.RequestCtx{
					Endpoint:  models.EndpointAppLovinMax,
					PubID:     5890,
					PubIDStr:  "5890",
					ProfileID: 1234,
					DeviceCtx: models.DeviceCtx{
						DerivedCountryCode: "US",
					},
				},
				imp: openrtb2.Imp{
					Instl:  0,
					Banner: &openrtb2.Banner{},
					Video:  &openrtb2.Video{},
				},
			},
			want: nil,
			setup: func() {
				mockFeature.EXPECT().IsMBMFCountryForPublisher("US", 5890).Return(true)
				mockFeature.EXPECT().IsMBMFPublisherEnabled(5890).Return(true)
				mockEngine.EXPECT().RecordMBMFRequests(models.EndpointAppLovinMax, "5890", models.MBMFInvalidAdFormat)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			m := OpenWrap{
				pubFeatures:  mockFeature,
				metricEngine: mockEngine,
			}
			got := m.getMultiFloors(tt.args.rctx, tt.args.reward, tt.args.imp)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsPrivacyEnforced(t *testing.T) {
	tests := []struct {
		name   string
		regs   *openrtb2.Regs
		device *openrtb2.Device
		want   bool
	}{
		{
			name: "nil regs and device lmt 1",
			regs: nil,
			device: &openrtb2.Device{
				Lmt: ptrutil.ToPtr[int8](1),
			},
			want: true,
		},
		{
			name: "empty regs and device lmt 0",
			regs: &openrtb2.Regs{},
			device: &openrtb2.Device{
				Lmt: ptrutil.ToPtr[int8](0),
			},
			want: false,
		},
		{
			name: "regs with coppa 1",
			regs: &openrtb2.Regs{COPPA: 1},
			want: true,
		},
		{
			name: "regs with gdpr 1",
			regs: &openrtb2.Regs{GDPR: ptrutil.ToPtr[int8](1)},
			want: true,
		},
		{
			name: "regs with empty gdpr",
			regs: &openrtb2.Regs{GDPR: ptrutil.ToPtr[int8](0)},
			want: false,
		},
		{
			name: "regs with ext gdpr 1",
			regs: &openrtb2.Regs{
				Ext: json.RawMessage(`{"gdpr":1}`),
			},
			want: true,
		},
		{
			name: "regs with invalid ext json",
			regs: &openrtb2.Regs{
				Ext: json.RawMessage(`{invalid json`),
			},
			want: false,
		},
		{
			name: "regs with both gdpr and ext gdpr 1",
			regs: &openrtb2.Regs{
				GDPR: ptrutil.ToPtr[int8](1),
				Ext:  json.RawMessage(`{"gdpr":1}`),
			},
			want: true,
		},
		{
			name: "regs with gdpr nil, GPPSID has tcf2",
			regs: &openrtb2.Regs{GDPR: nil, GPPSID: []int8{2}},
			want: true,
		},
		{
			name: "regs with gdpr 1, GPPSID has uspv1",
			regs: &openrtb2.Regs{GDPR: ptrutil.ToPtr[int8](1), GPPSID: []int8{6}},
			want: false,
		},
		{
			name: "regs with gdpr 0, GPPSID has tcf2",
			regs: &openrtb2.Regs{GDPR: ptrutil.ToPtr[int8](0), GPPSID: []int8{2}},
			want: true,
		},
		{
			name: "regs with GPPSID has tcf2",
			regs: &openrtb2.Regs{GPPSID: []int8{2}},
			want: true,
		},
		{
			name: "regs with GPPSID has uspv1",
			regs: &openrtb2.Regs{GPPSID: []int8{6}},
			want: false,
		},
		{
			name: "regs with usprivacy",
			regs: &openrtb2.Regs{
				USPrivacy: "1YNN",
			},
			want: true,
		},
		{
			name: "regs ext with usprivacy",
			regs: &openrtb2.Regs{
				Ext: json.RawMessage(`{"us_privacy":"1YNN"}`),
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPrivacyEnforced(tt.regs, tt.device)
			assert.Equal(t, tt.want, got, "isPrivacyEnforced() = %v, want %v", got, tt.want)
		})
	}
}
