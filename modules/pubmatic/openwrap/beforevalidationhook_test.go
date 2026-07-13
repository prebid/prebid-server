package openwrap

import (
	"context"
	"encoding/json"
	"errors"

	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/prebid/openrtb/v20/adcom1"
	nativeRequests "github.com/prebid/openrtb/v20/native1/request"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/openrtb/v20/openrtb3"
	"github.com/prebid/prebid-server/v3/currency"
	"github.com/prebid/prebid-server/v3/hooks/hookanalytics"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	adapters "github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/adapters"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/cache"
	mock_cache "github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/cache/mock"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/config"
	metrics "github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/metrics"
	mock_metrics "github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/metrics/mock"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models/adpodconfig"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models/adunitconfig"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models/nbr"
	mock_profilemetadata "github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/profilemetadata/mock"
	mock_feature "github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/publisherfeature/mock"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/wakanda"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

var rctx = models.RequestCtx{
	PubIDStr:                  "5890",
	PubID:                     5890,
	ProfileID:                 1234,
	DisplayID:                 1,
	SSAuction:                 -1,
	Platform:                  "in-app",
	Debug:                     true,
	DeviceCtx:                 models.DeviceCtx{UA: "go-test", IP: "127.0.0.1", Country: "IND", DerivedCountryCode: "IN"},
	IsCTVRequest:              false,
	TrackerEndpoint:           "t.pubmatic.com",
	VideoErrorTrackerEndpoint: "t.pubmatic.com/error",
	UidCookie: &http.Cookie{
		Name:  "uids",
		Value: `eyJ0ZW1wVUlEcyI6eyIzM2Fjcm9zcyI6eyJ1aWQiOiIxMTkxNzkxMDk5Nzc2NjEiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OTo0My4zODg4Nzk5NVoifSwiYWRmIjp7InVpZCI6IjgwNDQ2MDgzMzM3Nzg4MzkwNzgiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxMS4wMzMwNTQ3MjdaIn0sImFka2VybmVsIjp7InVpZCI6IkE5MTYzNTAwNzE0OTkyOTMyOTkwIiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MDkuMzczMzg1NjYyWiJ9LCJhZGtlcm5lbEFkbiI6eyJ1aWQiOiJBOTE2MzUwMDcxNDk5MjkzMjk5MCIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjEzLjQzNDkyNTg5NloifSwiYWRtaXhlciI6eyJ1aWQiOiIzNjZhMTdiMTJmMjI0ZDMwOGYzZTNiOGRhOGMzYzhhMCIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjA5LjU5MjkxNDgwMVoifSwiYWRueHMiOnsidWlkIjoiNDE5Mjg5ODUzMDE0NTExOTMiLCJleHBpcmVzIjoiMjAyMy0wMS0xOFQwOTo1MzowOC44MjU0NDI2NzZaIn0sImFqYSI6eyJ1aWQiOiJzMnN1aWQ2RGVmMFl0bjJveGQ1aG9zS1AxVmV3IiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MTMuMjM5MTc2MDU0WiJ9LCJlcGxhbm5pbmciOnsidWlkIjoiQUoxRjBTOE5qdTdTQ0xWOSIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjEwLjkyOTk2MDQ3M1oifSwiZ2Ftb3NoaSI6eyJ1aWQiOiJndXNyXzM1NmFmOWIxZDhjNjQyYjQ4MmNiYWQyYjdhMjg4MTYxIiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MTAuNTI0MTU3MjI1WiJ9LCJncmlkIjp7InVpZCI6IjRmYzM2MjUwLWQ4NTItNDU5Yy04NzcyLTczNTZkZTE3YWI5NyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjE0LjY5NjMxNjIyN1oifSwiZ3JvdXBtIjp7InVpZCI6IjdENzVEMjVGLUZBQzktNDQzRC1CMkQxLUIxN0ZFRTExRTAyNyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjM5LjIyNjIxMjUzMloifSwiaXgiOnsidWlkIjoiWW9ORlNENlc5QkphOEh6eEdtcXlCUUFBXHUwMDI2Mjk3IiwiZXhwaXJlcyI6IjIwMjMtMDUtMzFUMDc6NTM6MzguNTU1ODI3MzU0WiJ9LCJqaXhpZSI6eyJ1aWQiOiI3MzY3MTI1MC1lODgyLTExZWMtYjUzOC0xM2FjYjdhZjBkZTQiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxMi4xOTEwOTk3MzJaIn0sImxvZ2ljYWQiOnsidWlkIjoiQVZ4OVROQS11c25pa3M4QURzTHpWa3JvaDg4QUFBR0JUREh0UUEiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OTowOS40NTUxNDk2MTZaIn0sIm1lZGlhbmV0Ijp7InVpZCI6IjI5Nzg0MjM0OTI4OTU0MTAwMDBWMTAiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxMy42NzIyMTUxMjhaIn0sIm1naWQiOnsidWlkIjoibTU5Z1hyN0xlX1htIiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MTcuMDk3MDAxNDcxWiJ9LCJuYW5vaW50ZXJhY3RpdmUiOnsidWlkIjoiNmFlYzhjMTAzNzlkY2I3ODQxMmJjODBiNmRkOWM5NzMxNzNhYjdkNzEyZTQzMWE1YTVlYTcwMzRlNTZhNThhMCIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjE2LjcxNDgwNzUwNVoifSwib25ldGFnIjp7InVpZCI6IjdPelZoVzFOeC1LOGFVak1HMG52NXVNYm5YNEFHUXZQbnVHcHFrZ3k0ckEiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OTowOS4xNDE3NDEyNjJaIn0sIm9wZW54Ijp7InVpZCI6IjVkZWNlNjIyLTBhMjMtMGRhYi0zYTI0LTVhNzcwMTBlNDU4MiIsImV4cGlyZXMiOiIyMDIzLTA1LTMxVDA3OjUyOjQ3LjE0MDQxNzM2M1oifSwicHVibWF0aWMiOnsidWlkIjoiN0Q3NUQyNUYtRkFDOS00NDNELUIyRDEtQjE3RkVFMTFFMDI3IiwiZXhwaXJlcyI6IjIwMjItMTAtMzFUMDk6MTQ6MjUuNzM3MjU2ODk5WiJ9LCJyaWNoYXVkaWVuY2UiOnsidWlkIjoiY2I2YzYzMjAtMzNlMi00Nzc0LWIxNjAtMXp6MTY1NDg0MDc0OSIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjA5LjUyNTA3NDE4WiJ9LCJzbWFydHlhZHMiOnsidWlkIjoiMTJhZjE1ZTQ0ZjAwZDA3NjMwZTc0YzQ5MTU0Y2JmYmE0Zjg0N2U4ZDRhMTU0YzhjM2Q1MWY1OGNmNzJhNDYyNyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjEwLjgyNTAzMTg4NFoifSwic21pbGV3YW50ZWQiOnsidWlkIjoiZGQ5YzNmZTE4N2VmOWIwOWNhYTViNzExNDA0YzI4MzAiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxNC4yNTU2MDkzNjNaIn0sInN5bmFjb3JtZWRpYSI6eyJ1aWQiOiJHRFBSIiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MDkuOTc5NTgzNDM4WiJ9LCJ0cmlwbGVsaWZ0Ijp7InVpZCI6IjcwMjE5NzUwNTQ4MDg4NjUxOTQ2MyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjA4Ljk4OTY3MzU3NFoifSwidmFsdWVpbXByZXNzaW9uIjp7InVpZCI6IjlkMDgxNTVmLWQ5ZmUtNGI1OC04OThlLWUyYzU2MjgyYWIzZSIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjA5LjA2NzgzOTE2NFoifSwidmlzeCI6eyJ1aWQiOiIyN2UwYWMzYy1iNDZlLTQxYjMtOTkyYy1mOGQyNzE0OTQ5NWUiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxMi45ODk1MjM1NzNaIn0sInlpZWxkbGFiIjp7InVpZCI6IjY5NzE0ZDlmLWZiMDAtNGE1Zi04MTljLTRiZTE5MTM2YTMyNSIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjExLjMwMzAyNjYxNVoifSwieWllbGRtbyI6eyJ1aWQiOiJnOTZjMmY3MTlmMTU1MWIzMWY2MyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjEwLjExMDUyODYwOVoifSwieWllbGRvbmUiOnsidWlkIjoiMmE0MmZiZDMtMmM3MC00ZWI5LWIxYmQtMDQ2OTY2NTBkOTQ4IiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MTAuMzE4MzMzOTM5WiJ9LCJ6ZXJvY2xpY2tmcmF1ZCI6eyJ1aWQiOiJiOTk5NThmZS0yYTg3LTJkYTQtOWNjNC05NjFmZDExM2JlY2UiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxNS43MTk1OTQ1NjZaIn19LCJiZGF5IjoiMjAyMi0wNS0xN1QwNjo0ODozOC4wMTc5ODgyMDZaIn0=`,
	},
	KADUSERCookie: &http.Cookie{
		Name:  "KADUSERCOOKIE",
		Value: `7D75D25F-FAC9-443D-B2D1-B17FEE11E027`,
	},
	OriginCookie:             "go-test",
	Aliases:                  make(map[string]string),
	ImpBidCtx:                make(map[string]models.ImpCtx),
	PrebidBidderCode:         make(map[string]string),
	BidderResponseTimeMillis: make(map[string]int),
	ProfileIDStr:             "1234",
	Endpoint:                 models.EndpointV25,
	SeatNonBids:              make(map[string][]openrtb_ext.NonBid),
	WakandaDebug:             &wakanda.Debug{},
}

func getTestBidRequest(isSite bool) *openrtb2.BidRequest {

	testReq := &openrtb2.BidRequest{}

	testReq.ID = "testID"

	testReq.Imp = []openrtb2.Imp{
		{
			ID: "testImp1",
			Banner: &openrtb2.Banner{
				W: ptrutil.ToPtr[int64](200),
				H: ptrutil.ToPtr[int64](300),
			},
			Video: &openrtb2.Video{
				W:     ptrutil.ToPtr[int64](200),
				H:     ptrutil.ToPtr[int64](300),
				Plcmt: 1,
			},
		},
	}
	if !isSite {
		testReq.App = &openrtb2.App{
			Publisher: &openrtb2.Publisher{
				ID: "1010",
			},
			Content: &openrtb2.Content{
				Language: "english",
			},
		}
	} else {
		testReq.Site = &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				ID: "1010",
			},
			Content: &openrtb2.Content{
				Language: "english",
			},
		}
	}
	testReq.Cur = []string{"EUR"}
	testReq.WLang = []string{"english", "hindi"}
	testReq.Device = &openrtb2.Device{
		DeviceType: 1,
		Language:   "english",
	}
	return testReq
}

func TestGetPageURL(t *testing.T) {
	type args struct {
		bidRequest *openrtb2.BidRequest
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "App_storeurl_is_not_empty",
			args: args{
				bidRequest: &openrtb2.BidRequest{
					App: &openrtb2.App{
						StoreURL: "testurlApp",
					},
				},
			},
			want: "testurlApp",
		},
		{
			name: "Site_page_is_not_empty",
			args: args{
				bidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Page: "testurlSite",
					},
				},
			},
			want: "testurlSite",
		},
		{
			name: "both_app_and_site_are_nil",
			args: args{
				bidRequest: &openrtb2.BidRequest{},
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getPageURL(tt.args.bidRequest)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetVASTEventMacros(t *testing.T) {
	type args struct {
		rctx models.RequestCtx
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "SSAI_is_empty",
			args: args{
				rctx: models.RequestCtx{
					ProfileID:          1234,
					DisplayVersionID:   1234,
					StartTime:          1234,
					LoggerImpressionID: "1234",
					SSAI:               "",
					DeviceCtx: models.DeviceCtx{
						Platform: 1234,
					},
				},
			},
			want: map[string]string{
				"[PROFILE_ID]":            "1234",
				"[PROFILE_VERSION]":       "1234",
				"[UNIX_TIMESTAMP]":        "1234",
				"[PLATFORM]":              "1234",
				"[WRAPPER_IMPRESSION_ID]": "1234",
			},
		},
		{
			name: "SSAI_is_not_empty",
			args: args{
				rctx: models.RequestCtx{
					ProfileID:          1234,
					DisplayVersionID:   1234,
					StartTime:          1234,
					LoggerImpressionID: "1234",
					SSAI:               "1234",
					DeviceCtx: models.DeviceCtx{
						Platform: 1234,
					},
				},
			},
			want: map[string]string{
				"[PROFILE_ID]":            "1234",
				"[PROFILE_VERSION]":       "1234",
				"[UNIX_TIMESTAMP]":        "1234",
				"[PLATFORM]":              "1234",
				"[WRAPPER_IMPRESSION_ID]": "1234",
				"[SSAI]":                  "1234",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getVASTEventMacros(tt.args.rctx)
			assert.Equal(t, tt.want, got)

		})
	}
}

func TestUpdateAliasGVLIds(t *testing.T) {
	type args struct {
		aliasgvlids   map[string]uint16
		bidderCode    string
		partnerConfig map[string]string
	}
	type want struct {
		aliasgvlids map[string]uint16
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "vendorId not present in config",
			args: args{
				aliasgvlids:   map[string]uint16{},
				bidderCode:    "vastbidder1",
				partnerConfig: map[string]string{},
			},
			want: want{
				aliasgvlids: map[string]uint16{},
			},
		},
		{
			name: "Empty vendorID",
			args: args{
				aliasgvlids:   map[string]uint16{},
				bidderCode:    "vastbidder1",
				partnerConfig: map[string]string{models.VENDORID: ""},
			},
			want: want{
				aliasgvlids: map[string]uint16{},
			},
		},
		{
			name: "Error parsing vendorID",
			args: args{
				aliasgvlids:   map[string]uint16{},
				bidderCode:    "vastbidder1",
				partnerConfig: map[string]string{models.VENDORID: "abc"},
			},
		},
		{
			name: "VendorID is 0",
			args: args{
				aliasgvlids:   map[string]uint16{},
				bidderCode:    "vastbidder1",
				partnerConfig: map[string]string{models.VENDORID: "0"},
			},
			want: want{
				aliasgvlids: map[string]uint16{},
			},
		},
		{
			name: "Negative vendorID",
			args: args{
				aliasgvlids:   map[string]uint16{},
				bidderCode:    "vastbidder1",
				partnerConfig: map[string]string{models.VENDORID: "-76"},
			},
		},
		{
			name: "Valid vendorID",
			args: args{
				aliasgvlids:   map[string]uint16{},
				bidderCode:    "vastbidder1",
				partnerConfig: map[string]string{models.VENDORID: "76"},
			},
			want: want{
				aliasgvlids: map[string]uint16{"vastbidder1": uint16(76)},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateAliasGVLIds(tt.args.aliasgvlids, tt.args.bidderCode, tt.args.partnerConfig)
		})
	}
}

func TestOpenWrapSetTimeout(t *testing.T) {
	type fields struct {
		cfg          config.Config
		cache        cache.Cache
		metricEngine metrics.MetricsEngine
	}
	type args struct {
		rCtx       models.RequestCtx
		bidRequest *openrtb2.BidRequest
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int64
	}{
		{
			name: "Highest_priority_to_request_tmax_parameter",
			args: args{
				rCtx: models.RequestCtx{
					PartnerConfigMap: map[int]map[string]string{
						-1: {
							"ssTimeout": "250",
						},
					},
				},
				bidRequest: &openrtb2.BidRequest{
					TMax: 220,
				},
			},
			fields: fields{
				cfg: config.Config{
					Timeout: config.Timeout{
						MinTimeout: 200,
						MaxTimeout: 300,
					},
				},
			},
			want: 220,
		},
		{
			name: "tmax_parameter_less_than_minTimeout",
			args: args{
				rCtx: models.RequestCtx{
					PartnerConfigMap: map[int]map[string]string{
						-1: {
							"ssTimeout": "250",
						},
					},
				},
				bidRequest: &openrtb2.BidRequest{
					TMax: 10,
				},
			},
			fields: fields{
				cfg: config.Config{
					Timeout: config.Timeout{
						MinTimeout: 200,
						MaxTimeout: 300,
					},
				},
			},
			want: 200,
		},
		{
			name: "ssTimeout_greater_than_minTimeout_and_less_than_maxTimeout",
			args: args{
				rCtx: models.RequestCtx{
					PartnerConfigMap: map[int]map[string]string{
						-1: {
							"ssTimeout": "250",
						},
					},
				},
				bidRequest: &openrtb2.BidRequest{},
			},
			fields: fields{
				cfg: config.Config{
					Timeout: config.Timeout{
						MinTimeout: 200,
						MaxTimeout: 300,
					},
				},
			},
			want: 250,
		},
		{
			name: "ssTimeout_less_than_minTimeout",
			args: args{
				rCtx: models.RequestCtx{
					PartnerConfigMap: map[int]map[string]string{
						-1: {
							"ssTimeout": "250",
						},
					},
				},
				bidRequest: &openrtb2.BidRequest{},
			},
			fields: fields{
				cfg: config.Config{
					Timeout: config.Timeout{
						MinTimeout: 300,
						MaxTimeout: 400,
					},
				},
			},
			want: 300,
		},
		{
			name: "ssTimeout_greater_than_minTimeout_and_also_greater_than_maxTimeout",
			args: args{
				rCtx: models.RequestCtx{
					PartnerConfigMap: map[int]map[string]string{
						-1: {
							"ssTimeout": "500",
						},
					},
				},
				bidRequest: &openrtb2.BidRequest{},
			},
			fields: fields{
				cfg: config.Config{
					Timeout: config.Timeout{
						MinTimeout: 300,
						MaxTimeout: 400,
					},
				},
			},
			want: 400,
		},
		{
			name: "ssTimeout_greater_than_minTimeout_and_less_than_maxTimeout",
			args: args{
				rCtx: models.RequestCtx{
					PartnerConfigMap: map[int]map[string]string{
						-1: {
							"ssTimeout": "400",
						},
					},
				},
				bidRequest: &openrtb2.BidRequest{},
			},
			fields: fields{
				cfg: config.Config{
					Timeout: config.Timeout{
						MinTimeout: 300,
						MaxTimeout: 500,
					},
				},
			},
			want: 400,
		},
		//Below piece of code is applicable for older profiles where ssTimeout is not set
		//Here we will check the partner timeout and select max timeout considering timeout range
		{
			name: "at_lease_one_partner_timeout_greater_than_cofig_maxTimeout",
			args: args{
				rCtx: models.RequestCtx{
					PartnerConfigMap: map[int]map[string]string{
						1: {
							"timeout": "500",
						},
						2: {
							"timeout": "250",
						},
					},
				},
				bidRequest: &openrtb2.BidRequest{},
			},
			fields: fields{
				cfg: config.Config{
					Timeout: config.Timeout{
						MinTimeout: 200,
						MaxTimeout: 300,
					},
				},
			},
			want: 300,
		},
		{
			name: "all_partner_timeout_less_than_cofig_maxTimeout",
			args: args{
				rCtx: models.RequestCtx{
					PartnerConfigMap: map[int]map[string]string{
						1: {
							"timeout": "230",
						},
						2: {
							"timeout": "250",
						},
						3: {
							"timeout": "280",
						},
					},
				},
				bidRequest: &openrtb2.BidRequest{},
			},
			fields: fields{
				cfg: config.Config{
					Timeout: config.Timeout{
						MinTimeout: 200,
						MaxTimeout: 300,
					},
				},
			},
			want: 280,
		},
		{
			name: "all_partner_timeout_less_than_cofig_minTimeout",
			args: args{
				rCtx: models.RequestCtx{
					PartnerConfigMap: map[int]map[string]string{
						1: {
							"timeout": "100",
						},
						2: {
							"timeout": "150",
						},
						3: {
							"timeout": "180",
						},
					},
				},
				bidRequest: &openrtb2.BidRequest{},
			},
			fields: fields{
				cfg: config.Config{
					Timeout: config.Timeout{
						MinTimeout: 200,
						MaxTimeout: 300,
					},
				},
			},
			want: 200,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := OpenWrap{
				cfg:          tt.fields.cfg,
				cache:        tt.fields.cache,
				metricEngine: tt.fields.metricEngine,
			}
			got := m.setTimeout(tt.args.rCtx, tt.args.bidRequest)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsSendAllBids(t *testing.T) {
	type args struct {
		rctx models.RequestCtx
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "sendallbids_always_true_for_webs2s_endpoint",
			args: args{
				rctx: models.RequestCtx{
					Endpoint: models.EndpointWebS2S,
				},
			},
			want: true,
		},
		{
			name: "Don't_do_ssauction",
			args: args{
				rctx: models.RequestCtx{
					SSAuction: 0,
				},
			},
			want: true,
		},
		{
			name: "SSAuction_flag_not_set_In-app_sendAllbids_flag_1",
			args: args{
				rctx: models.RequestCtx{
					SSAuction: -1,
					Platform:  models.PLATFORM_APP,
					PartnerConfigMap: map[int]map[string]string{
						-1: {
							"sendAllBids": "1",
						},
					},
				},
			},
			want: true,
		},
		{
			name: "SSAuction_flag_not_set_In-app_sendAllbids_flag_other_than_1",
			args: args{
				rctx: models.RequestCtx{
					SSAuction: -1,
					Platform:  models.PLATFORM_APP,
					PartnerConfigMap: map[int]map[string]string{
						-1: {
							"sendAllBids": "5",
						},
					},
				},
			},
			want: false,
		},
		{
			name: "Random_value_of_ssauctionflag",
			args: args{
				rctx: models.RequestCtx{
					SSAuction: 5,
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSendAllBids(tt.args.rctx)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetValidLanguage(t *testing.T) {
	type args struct {
		language string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Language_of_length_less_than_2",
			args: args{
				language: "te",
			},
			want: "te",
		},
		{
			name: "Language_of_length_greater_than_2_and_it_is_valid_code",
			args: args{
				language: "hindi",
			},
			want: "hi",
		},
		{
			name: "Language_of_length_greater_than_2_and_it_is_Invalid_code",
			args: args{
				language: "xyz",
			},
			want: "xyz",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getValidLanguage(tt.args.language)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsSlotEnabled(t *testing.T) {
	type args struct {
		imp             openrtb2.Imp
		videoAdUnitCtx  models.AdUnitCtx
		bannerAdUnitCtx models.AdUnitCtx
		nativeAdUnitCtx models.AdUnitCtx
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Video_enabled_in_Video_adunit_context_but_video_impression_is_not_present_in_request",
			args: args{
				videoAdUnitCtx: models.AdUnitCtx{
					AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
						Video: &adunitconfig.Video{
							Enabled: ptrutil.ToPtr(true),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "Video_enabled_in_Video_adunit_context_video_impression_is_present_in_request",
			args: args{
				imp: openrtb2.Imp{
					Video: &openrtb2.Video{
						W: ptrutil.ToPtr[int64](1280),
						H: ptrutil.ToPtr[int64](1310),
					},
				},
				videoAdUnitCtx: models.AdUnitCtx{
					AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
						Video: &adunitconfig.Video{
							Enabled: ptrutil.ToPtr(true),
						},
					},
				},
			},
			want: true,
		},
		{
			name: "Banner_enabled_in_banner_adunit_context_but_banner_impression_is_not_present_in_request",
			args: args{
				bannerAdUnitCtx: models.AdUnitCtx{
					AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
						Banner: &adunitconfig.Banner{
							Enabled: ptrutil.ToPtr(true),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "Banner_enabled_in_banner_adunit_context_banner_impression_is_present_in_request",
			args: args{
				imp: openrtb2.Imp{
					Banner: &openrtb2.Banner{
						W: ptrutil.ToPtr[int64](10),
						H: ptrutil.ToPtr[int64](12),
					},
				},
				bannerAdUnitCtx: models.AdUnitCtx{
					AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
						Banner: &adunitconfig.Banner{
							Enabled: ptrutil.ToPtr(true),
						},
					},
				},
			},
			want: true,
		},
		{
			name: "both_banner_and_video_enabled_in_adunit_context_and_banner_and_video_impressions_present_in_the_request",
			args: args{
				imp: openrtb2.Imp{
					Video:  &openrtb2.Video{},
					Banner: &openrtb2.Banner{},
				},
				bannerAdUnitCtx: models.AdUnitCtx{
					AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
						Banner: &adunitconfig.Banner{
							Enabled: ptrutil.ToPtr(true),
						},
					},
				},
				videoAdUnitCtx: models.AdUnitCtx{
					AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
						Video: &adunitconfig.Video{
							Enabled: ptrutil.ToPtr(true),
						},
					},
				},
			},
			want: true,
		},
		{
			name: "both_banner_and_video_disabled_in_adunit_context_and_request_has_both_banner_and_video_impressions",
			args: args{
				imp: openrtb2.Imp{
					Video:  &openrtb2.Video{},
					Banner: &openrtb2.Banner{},
				},
				bannerAdUnitCtx: models.AdUnitCtx{
					AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
						Banner: &adunitconfig.Banner{
							Enabled: ptrutil.ToPtr(false),
						},
					},
				},
				videoAdUnitCtx: models.AdUnitCtx{
					AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
						Video: &adunitconfig.Video{
							Enabled: ptrutil.ToPtr(false),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "both_banner_and_video_enabled_in_adunit_context_both_banner_and_video_impressions_is_not_present_in_the_request",
			args: args{
				bannerAdUnitCtx: models.AdUnitCtx{
					AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
						Banner: &adunitconfig.Banner{
							Enabled: ptrutil.ToPtr(true),
						},
					},
				},
				videoAdUnitCtx: models.AdUnitCtx{
					AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
						Video: &adunitconfig.Video{
							Enabled: ptrutil.ToPtr(true),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "Banner_is_enabled_in_adunit_and_request_has_video_impressions",
			args: args{
				imp: openrtb2.Imp{
					Video: &openrtb2.Video{},
				},
				bannerAdUnitCtx: models.AdUnitCtx{
					AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
						Banner: &adunitconfig.Banner{
							Enabled: ptrutil.ToPtr(true),
						},
					},
				},
			},
			want: true,
		},
		{
			name: "Native_impression_is_present_in_request",
			args: args{
				imp: openrtb2.Imp{
					Native: &openrtb2.Native{},
				},
				bannerAdUnitCtx: models.AdUnitCtx{
					AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
						Banner: &adunitconfig.Banner{
							Enabled: ptrutil.ToPtr(false),
						},
					},
				},
				videoAdUnitCtx: models.AdUnitCtx{
					AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
						Video: &adunitconfig.Video{
							Enabled: ptrutil.ToPtr(false),
						},
					},
				},
			},
			want: true,
		},
		{
			name: "both_banner_and_video_context_are empty",
			args: args{},
			want: false,
		},
		{
			name: "Native_enabled_in_native_adunit_context_but_native_impression_is_not_present_in_request",
			args: args{
				bannerAdUnitCtx: models.AdUnitCtx{
					AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
						Native: &adunitconfig.Native{
							Enabled: ptrutil.ToPtr(true),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "Native_enabled_in_native_adunit_context_native_impression_present_in_request",
			args: args{
				imp: openrtb2.Imp{
					Native: &openrtb2.Native{
						Request: "Native_Reuqest",
					},
				},
				bannerAdUnitCtx: models.AdUnitCtx{
					AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
						Native: &adunitconfig.Native{
							Enabled: ptrutil.ToPtr(true),
						},
					},
				},
			},
			want: true,
		},
		{
			name: "banner_video_native_enabled_in_adunit_context_banner_and_video_native_impressions_is_not_present_in_the_request",
			args: args{
				bannerAdUnitCtx: models.AdUnitCtx{
					AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
						Banner: &adunitconfig.Banner{
							Enabled: ptrutil.ToPtr(true),
						},
					},
				},
				videoAdUnitCtx: models.AdUnitCtx{
					AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
						Video: &adunitconfig.Video{
							Enabled: ptrutil.ToPtr(true),
						},
					},
				},
				nativeAdUnitCtx: models.AdUnitCtx{
					AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
						Native: &adunitconfig.Native{
							Enabled: ptrutil.ToPtr(true),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "banner_video_and_native_enabled_in_adunit_context_and_banner_video_and_native_impressions_present_in_the_request",
			args: args{
				imp: openrtb2.Imp{
					Video:  &openrtb2.Video{},
					Banner: &openrtb2.Banner{},
					Native: &openrtb2.Native{},
				},
				bannerAdUnitCtx: models.AdUnitCtx{
					AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
						Banner: &adunitconfig.Banner{
							Enabled: ptrutil.ToPtr(true),
						},
					},
				},
				videoAdUnitCtx: models.AdUnitCtx{
					AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
						Video: &adunitconfig.Video{
							Enabled: ptrutil.ToPtr(true),
						},
					},
				},
				nativeAdUnitCtx: models.AdUnitCtx{
					AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
						Native: &adunitconfig.Native{
							Enabled: ptrutil.ToPtr(true),
						},
					},
				},
			},
			want: true,
		},
		{
			name: "banner_video_naitve_disabled_in_adunit_context_and_request_has_banner_video_native_impressions",
			args: args{
				imp: openrtb2.Imp{
					Video:  &openrtb2.Video{},
					Banner: &openrtb2.Banner{},
					Native: &openrtb2.Native{},
				},
				bannerAdUnitCtx: models.AdUnitCtx{
					AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
						Banner: &adunitconfig.Banner{
							Enabled: ptrutil.ToPtr(false),
						},
					},
				},
				videoAdUnitCtx: models.AdUnitCtx{
					AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
						Video: &adunitconfig.Video{
							Enabled: ptrutil.ToPtr(false),
						},
					},
				},
				nativeAdUnitCtx: models.AdUnitCtx{
					AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
						Native: &adunitconfig.Native{
							Enabled: ptrutil.ToPtr(false),
						},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSlotEnabled(tt.args.imp, tt.args.videoAdUnitCtx, tt.args.bannerAdUnitCtx, tt.args.nativeAdUnitCtx)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetPubID(t *testing.T) {
	type args struct {
		bidRequest openrtb2.BidRequest
	}
	type want struct {
		wantErr bool
		pubID   int
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "publisher_id_not_present_in_site_object_and_in_app_object",
			args: args{
				bidRequest: openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{},
					},
				},
			},
			want: want{
				wantErr: false,
				pubID:   0,
			},
		},
		{
			name: "publisher_id_present_in_site_object_and_it_is_valid_integer",
			args: args{
				bidRequest: openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{
							ID: "5890",
						},
					},
				},
			},
			want: want{
				wantErr: false,
				pubID:   5890,
			},
		},
		{
			name: "publisher_id_present_in_site_object_but_it_is_not_valid_integer",
			args: args{
				bidRequest: openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{
							ID: "test",
						},
					},
				},
			},
			want: want{
				wantErr: true,
				pubID:   0,
			},
		},
		{
			name: "publisher_id_present_in_App_object_and_it_is_valid_integer",
			args: args{
				bidRequest: openrtb2.BidRequest{
					App: &openrtb2.App{
						Publisher: &openrtb2.Publisher{
							ID: "5890",
						},
					},
				},
			},
			want: want{
				wantErr: false,
				pubID:   5890,
			},
		},
		{
			name: "publisher_id_present_in_App_object_but_it_is_not_valid_integer",
			args: args{
				bidRequest: openrtb2.BidRequest{
					App: &openrtb2.App{
						Publisher: &openrtb2.Publisher{
							ID: "test",
						},
					},
				},
			},
			want: want{
				wantErr: true,
				pubID:   0,
			},
		},
		{
			name: "publisher_id_present_in_both_Site_and_App_object",
			args: args{
				bidRequest: openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{
							ID: "1234",
						},
					},
					App: &openrtb2.App{
						Publisher: &openrtb2.Publisher{
							ID: "5890",
						},
					},
				},
			},
			want: want{
				wantErr: false,
				pubID:   1234,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getPubID(tt.args.bidRequest)
			if (err != nil) != tt.want.wantErr {
				assert.Equal(t, tt.want.wantErr, err != nil)
				return
			}
			if got != tt.want.pubID {
				assert.Equal(t, tt.want.pubID, got)
			}
		})
	}
}

func TestOpenWrapApplyProfileChanges(t *testing.T) {
	type fields struct {
		cfg          config.Config
		cache        cache.Cache
		metricEngine metrics.MetricsEngine
	}
	type args struct {
		rctx       models.RequestCtx
		bidRequest *openrtb2.BidRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *openrtb2.BidRequest
		wantErr bool
	}{
		{
			name: "Request_with_App_object",
			args: args{
				rctx: models.RequestCtx{
					IsTestRequest: 1,
					PartnerConfigMap: map[int]map[string]string{
						-1: {
							models.AdServerCurrency: "USD",
							models.SChainDBKey:      "1",
						},
					},
					TMax:      500,
					DeviceCtx: models.DeviceCtx{IP: "127.0.0.1"},
					Platform:  models.PLATFORM_APP,
					KADUSERCookie: &http.Cookie{
						Name:  "KADUSERCOOKIE",
						Value: "123456789",
					},
				},
				bidRequest: getTestBidRequest(false),
			},
			want: &openrtb2.BidRequest{
				ID:   "testID",
				Test: 1,
				Cur:  []string{"EUR", "USD"},
				TMax: 500,
				Source: &openrtb2.Source{
					TID: "testID",
				},
				Imp: []openrtb2.Imp{
					{
						ID: "testImp1",
						Banner: &openrtb2.Banner{
							W: ptrutil.ToPtr[int64](200),
							H: ptrutil.ToPtr[int64](300),
						},
						Video: &openrtb2.Video{
							W:     ptrutil.ToPtr[int64](200),
							H:     ptrutil.ToPtr[int64](300),
							Plcmt: 1,
						},
					},
				},
				Device: &openrtb2.Device{
					IP:         "127.0.0.1",
					Language:   "en",
					DeviceType: 1,
				},
				WLang: []string{"en", "hi"},
				User: &openrtb2.User{
					CustomData: "123456789",
				},
				App: &openrtb2.App{
					Publisher: &openrtb2.Publisher{
						ID: "1010",
					},
					Content: &openrtb2.Content{
						Language: "en",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Request_with_Site_object",
			args: args{
				rctx: models.RequestCtx{
					IsTestRequest: 1,
					PartnerConfigMap: map[int]map[string]string{
						-1: {
							models.AdServerCurrency: "USD",
							models.SChainDBKey:      "1",
						},
					},
					TMax:      500,
					DeviceCtx: models.DeviceCtx{IP: "127.0.0.1"},
					Platform:  models.PLATFORM_APP,
					KADUSERCookie: &http.Cookie{
						Name:  "KADUSERCOOKIE",
						Value: "123456789",
					},
				},
				bidRequest: getTestBidRequest(true),
			},
			want: &openrtb2.BidRequest{
				ID:   "testID",
				Test: 1,
				Cur:  []string{"EUR", "USD"},
				TMax: 500,
				Source: &openrtb2.Source{
					TID: "testID",
				},
				Imp: []openrtb2.Imp{
					{
						ID: "testImp1",
						Banner: &openrtb2.Banner{
							W: ptrutil.ToPtr[int64](200),
							H: ptrutil.ToPtr[int64](300),
						},
						Video: &openrtb2.Video{
							W:     ptrutil.ToPtr[int64](200),
							H:     ptrutil.ToPtr[int64](300),
							Plcmt: 1,
						},
					},
				},
				Device: &openrtb2.Device{
					IP:         "127.0.0.1",
					Language:   "en",
					DeviceType: 1,
				},
				WLang: []string{"en", "hi"},
				User: &openrtb2.User{
					CustomData: "123456789",
				},
				Site: &openrtb2.Site{
					Publisher: &openrtb2.Publisher{
						ID: "1010",
					},
					Content: &openrtb2.Content{
						Language: "en",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "For_amp_request_banner_can_not_be_disabled_through_adunit_config",
			args: args{
				rctx: models.RequestCtx{
					IsTestRequest: 1,
					PartnerConfigMap: map[int]map[string]string{
						-1: {
							models.AdServerCurrency: "USD",
							models.SChainDBKey:      "1",
						},
					},
					TMax:      500,
					DeviceCtx: models.DeviceCtx{IP: "127.0.0.1"},
					Platform:  models.PLATFORM_APP,
					KADUSERCookie: &http.Cookie{
						Name:  "KADUSERCOOKIE",
						Value: "123456789",
					},
					ImpBidCtx: map[string]models.ImpCtx{
						"testImp": {
							BannerAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
									Banner: &adunitconfig.Banner{
										Enabled: ptrutil.ToPtr(false),
									},
								},
							},
						},
					},
					Endpoint: models.EndpointAMP,
				},
				bidRequest: getTestBidRequest(true),
			},
			want: &openrtb2.BidRequest{
				ID:   "testID",
				Test: 1,
				Cur:  []string{"EUR", "USD"},
				TMax: 500,
				Source: &openrtb2.Source{
					TID: "testID",
				},
				Imp: []openrtb2.Imp{
					{
						ID: "testImp1",
						Banner: &openrtb2.Banner{
							W: ptrutil.ToPtr[int64](200),
							H: ptrutil.ToPtr[int64](300),
						},
						Video: &openrtb2.Video{
							W:     ptrutil.ToPtr[int64](200),
							H:     ptrutil.ToPtr[int64](300),
							Plcmt: 1,
						},
					},
				},
				Device: &openrtb2.Device{
					IP:         "127.0.0.1",
					Language:   "en",
					DeviceType: 1,
				},
				WLang: []string{"en", "hi"},
				User: &openrtb2.User{
					CustomData: "123456789",
				},
				Site: &openrtb2.Site{
					Publisher: &openrtb2.Publisher{
						ID: "1010",
					},
					Content: &openrtb2.Content{
						Language: "en",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Request_with_User_Ext_Eids",
			args: args{
				rctx: models.RequestCtx{
					IsTestRequest: 1,
					PartnerConfigMap: map[int]map[string]string{
						-1: {
							models.AdServerCurrency: "USD",
							models.SChainDBKey:      "1",
						},
					},
					TMax:      500,
					DeviceCtx: models.DeviceCtx{IP: "127.0.0.1"},
					Platform:  models.PLATFORM_APP,
					KADUSERCookie: &http.Cookie{
						Name:  "KADUSERCOOKIE",
						Value: "123456789",
					},
				},
				bidRequest: &openrtb2.BidRequest{
					ID:   "testID",
					Test: 1,
					Cur:  []string{"EUR"},
					TMax: 500,
					Source: &openrtb2.Source{
						TID: "testID",
					},
					Imp: []openrtb2.Imp{
						{
							ID: "testImp1",
							Video: &openrtb2.Video{
								W:     ptrutil.ToPtr[int64](200),
								H:     ptrutil.ToPtr[int64](300),
								Plcmt: 1,
							},
						},
					},
					Device: &openrtb2.Device{
						IP:         "127.0.0.1",
						Language:   "en",
						DeviceType: 1,
					},
					WLang: []string{"en", "hi"},
					User: &openrtb2.User{
						CustomData: "123456789",
						Ext:        json.RawMessage(`{"eids":[{"source":"uidapi.com","uids":[{"id":"UID2:testUID"},{"id":"testUID2"}]},{"source":"euid.eu","uids":[{"id":"testeuid"}]},{"source":"liveramp.com","uids":[{"id":""}]}]}`),
					},
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{
							ID: "1010",
						},
						Content: &openrtb2.Content{
							Language: "en",
						},
					},
				},
			},
			want: &openrtb2.BidRequest{
				ID:   "testID",
				Test: 1,
				Cur:  []string{"EUR", "USD"},
				TMax: 500,
				Source: &openrtb2.Source{
					TID: "testID",
				},
				Imp: []openrtb2.Imp{
					{
						ID: "testImp1",
						Video: &openrtb2.Video{
							W:     ptrutil.ToPtr[int64](200),
							H:     ptrutil.ToPtr[int64](300),
							Plcmt: 1,
						},
					},
				},
				Device: &openrtb2.Device{
					IP:         "127.0.0.1",
					Language:   "en",
					DeviceType: 1,
				},
				WLang: []string{"en", "hi"},
				User: &openrtb2.User{
					CustomData: "123456789",
					Ext:        json.RawMessage(`{"eids":[{"source":"uidapi.com","uids":[{"id":"testUID"},{"id":"testUID2"}]},{"source":"euid.eu","uids":[{"id":"testeuid"}]}]}`),
				},
				Site: &openrtb2.Site{
					Publisher: &openrtb2.Publisher{
						ID: "1010",
					},
					Content: &openrtb2.Content{
						Language: "en",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Valid_EIDs_in_User_object",
			args: args{
				rctx: models.RequestCtx{
					IsTestRequest: 1,
					PartnerConfigMap: map[int]map[string]string{
						-1: {
							models.AdServerCurrency: "USD",
							models.SChainDBKey:      "1",
						},
					},
					TMax:      500,
					DeviceCtx: models.DeviceCtx{IP: "127.0.0.1"},
					Platform:  models.PLATFORM_APP,
					KADUSERCookie: &http.Cookie{
						Name:  "KADUSERCOOKIE",
						Value: "123456789",
					},
				},
				bidRequest: &openrtb2.BidRequest{
					ID:   "testID",
					Test: 1,
					Cur:  []string{"EUR"},
					TMax: 500,
					Source: &openrtb2.Source{
						TID: "testID",
					},
					Imp: []openrtb2.Imp{
						{
							ID: "testImp1",
							Video: &openrtb2.Video{
								W:     ptrutil.ToPtr[int64](200),
								H:     ptrutil.ToPtr[int64](300),
								Plcmt: 1,
							},
						},
					},
					Device: &openrtb2.Device{
						IP:         "127.0.0.1",
						Language:   "en",
						DeviceType: 1,
					},
					WLang: []string{"en", "hi"},
					User: &openrtb2.User{
						CustomData: "123456789",
						EIDs: []openrtb2.EID{
							{
								Source: "uidapi.com",
								UIDs: []openrtb2.UID{
									{ID: "UID2:testUID"},
									{ID: "testUID2"},
									{ID: "UID2:"},
								},
							},
							{
								Source: "euid.eu",
								UIDs: []openrtb2.UID{
									{ID: "euid:testeuid"}},
							},
							{
								Source: "liveramp.com",
								UIDs: []openrtb2.UID{
									{ID: "IDL:"}},
							},
						},
					},
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{
							ID: "1010",
						},
						Content: &openrtb2.Content{
							Language: "en",
						},
					},
				},
			},
			want: &openrtb2.BidRequest{
				ID:   "testID",
				Test: 1,
				Cur:  []string{"EUR", "USD"},
				TMax: 500,
				Source: &openrtb2.Source{
					TID: "testID",
				},
				Imp: []openrtb2.Imp{
					{
						ID: "testImp1",
						Video: &openrtb2.Video{
							W:     ptrutil.ToPtr[int64](200),
							H:     ptrutil.ToPtr[int64](300),
							Plcmt: 1,
						},
					},
				},
				Device: &openrtb2.Device{
					IP:         "127.0.0.1",
					Language:   "en",
					DeviceType: 1,
				},
				WLang: []string{"en", "hi"},
				User: &openrtb2.User{
					CustomData: "123456789",
					EIDs: []openrtb2.EID{
						{
							Source: "uidapi.com",
							UIDs: []openrtb2.UID{
								{ID: "testUID"},
								{ID: "testUID2"},
							},
						},
						{
							Source: "euid.eu",
							UIDs: []openrtb2.UID{
								{ID: "testeuid"},
							},
						},
					},
				},
				Site: &openrtb2.Site{
					Publisher: &openrtb2.Publisher{
						ID: "1010",
					},
					Content: &openrtb2.Content{
						Language: "en",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Valid_EIDs_in_User_object_and_User_Ext",
			args: args{
				rctx: models.RequestCtx{
					IsTestRequest: 1,
					PartnerConfigMap: map[int]map[string]string{
						-1: {
							models.AdServerCurrency: "USD",
							models.SChainDBKey:      "1",
						},
					},
					TMax:      500,
					DeviceCtx: models.DeviceCtx{IP: "127.0.0.1"},
					Platform:  models.PLATFORM_APP,
					KADUSERCookie: &http.Cookie{
						Name:  "KADUSERCOOKIE",
						Value: "123456789",
					},
				},
				bidRequest: &openrtb2.BidRequest{
					ID:   "testID",
					Test: 1,
					Cur:  []string{"EUR"},
					TMax: 500,
					Source: &openrtb2.Source{
						TID: "testID",
					},
					Imp: []openrtb2.Imp{
						{
							ID: "testImp1",
							Video: &openrtb2.Video{
								W:     ptrutil.ToPtr[int64](200),
								H:     ptrutil.ToPtr[int64](300),
								Plcmt: 1,
							},
						},
					},
					Device: &openrtb2.Device{
						IP:         "127.0.0.1",
						Language:   "en",
						DeviceType: 1,
					},
					WLang: []string{"en", "hi"},
					User: &openrtb2.User{
						CustomData: "123456789",
						Ext:        json.RawMessage(`{"eids":[{"source":"uidapi.com","uids":[{"id":"testUID"},{"id":"testUID2"}]},{"source":"euid.eu","uids":[{"id":"testeuid"}]}]}`),
						EIDs: []openrtb2.EID{
							{
								Source: "uidapi.com",
								UIDs: []openrtb2.UID{
									{ID: "UID2:testUID"},
									{ID: "testUID2"},
								},
							},
							{
								Source: "euid.eu",
								UIDs: []openrtb2.UID{
									{ID: "euid:testeuid"}},
							},
						},
					},
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{
							ID: "1010",
						},
						Content: &openrtb2.Content{
							Language: "en",
						},
					},
				},
			},
			want: &openrtb2.BidRequest{
				ID:   "testID",
				Test: 1,
				Cur:  []string{"EUR", "USD"},
				TMax: 500,
				Source: &openrtb2.Source{
					TID: "testID",
				},
				Imp: []openrtb2.Imp{
					{
						ID: "testImp1",
						Video: &openrtb2.Video{
							W:     ptrutil.ToPtr[int64](200),
							H:     ptrutil.ToPtr[int64](300),
							Plcmt: 1,
						},
					},
				},
				Device: &openrtb2.Device{
					IP:         "127.0.0.1",
					Language:   "en",
					DeviceType: 1,
				},
				WLang: []string{"en", "hi"},
				User: &openrtb2.User{
					CustomData: "123456789",
					Ext:        json.RawMessage(`{"eids":[{"source":"uidapi.com","uids":[{"id":"testUID"},{"id":"testUID2"}]},{"source":"euid.eu","uids":[{"id":"testeuid"}]}]}`),
					EIDs: []openrtb2.EID{
						{
							Source: "uidapi.com",
							UIDs: []openrtb2.UID{
								{ID: "testUID"},
								{ID: "testUID2"},
							},
						},
						{
							Source: "euid.eu",
							UIDs: []openrtb2.UID{
								{ID: "testeuid"},
							},
						},
					},
				},
				Site: &openrtb2.Site{
					Publisher: &openrtb2.Publisher{
						ID: "1010",
					},
					Content: &openrtb2.Content{
						Language: "en",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid_EIDs_in_User_object",
			args: args{
				rctx: models.RequestCtx{
					IsTestRequest: 1,
					PartnerConfigMap: map[int]map[string]string{
						-1: {
							models.AdServerCurrency: "USD",
							models.SChainDBKey:      "1",
						},
					},
					TMax:      500,
					DeviceCtx: models.DeviceCtx{IP: "127.0.0.1"},
					Platform:  models.PLATFORM_APP,
					KADUSERCookie: &http.Cookie{
						Name:  "KADUSERCOOKIE",
						Value: "123456789",
					},
				},
				bidRequest: &openrtb2.BidRequest{
					ID:   "testID",
					Test: 1,
					Cur:  []string{"EUR"},
					TMax: 500,
					Source: &openrtb2.Source{
						TID: "testID",
					},
					Imp: []openrtb2.Imp{
						{
							ID: "testImp1",
							Video: &openrtb2.Video{
								W:     ptrutil.ToPtr[int64](200),
								H:     ptrutil.ToPtr[int64](300),
								Plcmt: 1,
							},
						},
					},
					Device: &openrtb2.Device{
						IP:         "127.0.0.1",
						Language:   "en",
						DeviceType: 1,
					},
					WLang: []string{"en", "hi"},
					User: &openrtb2.User{
						CustomData: "123456789",
						EIDs: []openrtb2.EID{
							{
								Source: "uidapi.com",
								UIDs: []openrtb2.UID{
									{ID: "UID2:"},
									{ID: ""},
								},
							},
							{
								Source: "euid.eu",
								UIDs: []openrtb2.UID{
									{ID: "euid:"}},
							},
							{
								Source: "liveramp.com",
								UIDs: []openrtb2.UID{
									{ID: "IDL:"}},
							},
						},
					},
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{
							ID: "1010",
						},
						Content: &openrtb2.Content{
							Language: "en",
						},
					},
				},
			},
			want: &openrtb2.BidRequest{
				ID:   "testID",
				Test: 1,
				Cur:  []string{"EUR", "USD"},
				TMax: 500,
				Source: &openrtb2.Source{
					TID: "testID",
				},
				Imp: []openrtb2.Imp{
					{
						ID: "testImp1",
						Video: &openrtb2.Video{
							W:     ptrutil.ToPtr[int64](200),
							H:     ptrutil.ToPtr[int64](300),
							Plcmt: 1,
						},
					},
				},
				Device: &openrtb2.Device{
					IP:         "127.0.0.1",
					Language:   "en",
					DeviceType: 1,
				},
				WLang: []string{"en", "hi"},
				User: &openrtb2.User{
					CustomData: "123456789",
				},
				Site: &openrtb2.Site{
					Publisher: &openrtb2.Publisher{
						ID: "1010",
					},
					Content: &openrtb2.Content{
						Language: "en",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Request_with_User_Ext_Eids_Invalid",
			args: args{
				rctx: models.RequestCtx{
					IsTestRequest: 1,
					PartnerConfigMap: map[int]map[string]string{
						-1: {
							models.AdServerCurrency: "USD",
							models.SChainDBKey:      "1",
						},
					},
					TMax:      500,
					DeviceCtx: models.DeviceCtx{IP: "127.0.0.1"},
					Platform:  models.PLATFORM_APP,
					KADUSERCookie: &http.Cookie{
						Name:  "KADUSERCOOKIE",
						Value: "123456789",
					},
				},
				bidRequest: &openrtb2.BidRequest{
					ID:   "testID",
					Test: 1,
					Cur:  []string{"EUR"},
					TMax: 500,
					Source: &openrtb2.Source{
						TID: "testID",
					},
					Imp: []openrtb2.Imp{
						{
							ID: "testImp1",
							Video: &openrtb2.Video{
								W:     ptrutil.ToPtr[int64](200),
								H:     ptrutil.ToPtr[int64](300),
								Plcmt: 1,
							},
						},
					},
					Device: &openrtb2.Device{
						IP:         "127.0.0.1",
						Language:   "en",
						DeviceType: 1,
					},
					WLang: []string{"en", "hi"},
					User: &openrtb2.User{
						CustomData: "123456789",
						Ext:        json.RawMessage(`{"eids":[{"source":"uidapi.com","uids":[{"id":"UID2:"},{"id":""}]},{"source":"euid.eu","uids":[{"id":""}]},{"source":"liveramp.com","uids":[{"id":"IDL:"}]}]}`),
					},
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{
							ID: "1010",
						},
						Content: &openrtb2.Content{
							Language: "en",
						},
					},
				},
			},
			want: &openrtb2.BidRequest{
				ID:   "testID",
				Test: 1,
				Cur:  []string{"EUR", "USD"},
				TMax: 500,
				Source: &openrtb2.Source{
					TID: "testID",
				},
				Imp: []openrtb2.Imp{
					{
						ID: "testImp1",
						Video: &openrtb2.Video{
							W:     ptrutil.ToPtr[int64](200),
							H:     ptrutil.ToPtr[int64](300),
							Plcmt: 1,
						},
					},
				},
				Device: &openrtb2.Device{
					IP:         "127.0.0.1",
					Language:   "en",
					DeviceType: 1,
				},
				WLang: []string{"en", "hi"},
				User: &openrtb2.User{
					CustomData: "123456789",
					Ext:        json.RawMessage(`{}`),
				},
				Site: &openrtb2.Site{
					Publisher: &openrtb2.Publisher{
						ID: "1010",
					},
					Content: &openrtb2.Content{
						Language: "en",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "AppLovinMax_request_with_storeurl_and_sourceapp_updated_from_DB",
			args: args{
				rctx: models.RequestCtx{
					PartnerConfigMap: map[int]map[string]string{
						-1: {
							models.AppStoreUrl: "https://itunes.apple.com/us/app/angry-birds/id343200656",
						},
					},
					Endpoint:    models.EndpointAppLovinMax,
					AppStoreUrl: "https://itunes.apple.com/us/app/angry-birds/id343200656",
					ImpBidCtx: map[string]models.ImpCtx{
						"testImp1": {
							NewExt: json.RawMessage(`{"skadn":{"sourceapp":"343200656"}}`),
						},
					},
				},
				bidRequest: &openrtb2.BidRequest{
					ID: "testID",
					Imp: []openrtb2.Imp{
						{
							ID: "testImp1",
							Video: &openrtb2.Video{
								W:     ptrutil.ToPtr[int64](200),
								H:     ptrutil.ToPtr[int64](300),
								Plcmt: 1,
							},
							Ext: json.RawMessage(`{"skadn": {}}`),
						},
					},
					Device: &openrtb2.Device{
						OS: "iOS",
					},
					App: &openrtb2.App{
						Publisher: &openrtb2.Publisher{
							ID: "1010",
						},
					},
				},
			},
			want: &openrtb2.BidRequest{
				ID: "testID",
				Imp: []openrtb2.Imp{
					{
						ID:     "testImp1",
						Secure: openrtb2.Int8Ptr(1),
						Video: &openrtb2.Video{
							W:     ptrutil.ToPtr[int64](200),
							H:     ptrutil.ToPtr[int64](300),
							Plcmt: 1,
						},
						Ext: json.RawMessage(`{"skadn":{"sourceapp":"343200656"}}`),
					},
				},
				Device: &openrtb2.Device{
					OS: "iOS",
				},
				User: &openrtb2.User{},
				App: &openrtb2.App{
					Publisher: &openrtb2.Publisher{
						ID: "1010",
					},
					StoreURL: "https://itunes.apple.com/us/app/angry-birds/id343200656",
				},
				Source: &openrtb2.Source{
					TID: "testID",
				},
			},
			wantErr: false,
		},
		{
			name: "GAM_Unwinding_Enabled",
			args: args{
				rctx: models.RequestCtx{
					IsTestRequest: 1,
					PartnerConfigMap: map[int]map[string]string{
						-1: {
							models.AdServerCurrency:           "USD",
							models.SChainDBKey:                "1",
							models.GoogleSSUFeatureEnabledKey: models.Enabled,
						},
					},
					TMax:      500,
					DeviceCtx: models.DeviceCtx{IP: "127.0.0.1"},
					Platform:  models.PLATFORM_APP,
					KADUSERCookie: &http.Cookie{
						Name:  "KADUSERCOOKIE",
						Value: "123456789",
					},
				},
				bidRequest: &openrtb2.BidRequest{
					ID:   "testID",
					Test: 1,
					Cur:  []string{"EUR"},
					TMax: 500,
					Source: &openrtb2.Source{
						TID: "testID",
					},
					Imp: []openrtb2.Imp{
						{
							ID: "testImp1",
							Video: &openrtb2.Video{
								W:         ptrutil.ToPtr[int64](200),
								H:         ptrutil.ToPtr[int64](300),
								Plcmt:     1,
								Protocols: []adcom1.MediaCreativeSubtype{1, 2, 3},
							},
						},
					},
					Device: &openrtb2.Device{
						IP:         "127.0.0.1",
						Language:   "en",
						DeviceType: 1,
					},
					WLang: []string{"en", "hi"},
					User: &openrtb2.User{
						CustomData: "123456789",
						Ext:        json.RawMessage(`{"eids":[{"source":"uidapi.com","uids":[{"id":"UID2:"},{"id":""}]},{"source":"euid.eu","uids":[{"id":""}]},{"source":"liveramp.com","uids":[{"id":"IDL:"}]}]}`),
					},
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{
							ID: "1010",
						},
						Content: &openrtb2.Content{
							Language: "en",
						},
					},
				},
			},
			want: &openrtb2.BidRequest{
				ID:   "testID",
				Test: 1,
				Cur:  []string{"EUR", "USD"},
				TMax: 500,
				Source: &openrtb2.Source{
					TID: "testID",
				},
				Imp: []openrtb2.Imp{
					{
						ID: "testImp1",
						Video: &openrtb2.Video{
							W:         ptrutil.ToPtr[int64](200),
							H:         ptrutil.ToPtr[int64](300),
							Plcmt:     1,
							Protocols: []adcom1.MediaCreativeSubtype{1, 2, 3, 6, 7, 8},
						},
					},
				},
				Device: &openrtb2.Device{
					IP:         "127.0.0.1",
					Language:   "en",
					DeviceType: 1,
				},
				WLang: []string{"en", "hi"},
				User: &openrtb2.User{
					CustomData: "123456789",
					Ext:        json.RawMessage(`{}`),
				},
				Site: &openrtb2.Site{
					Publisher: &openrtb2.Publisher{
						ID: "1010",
					},
					Content: &openrtb2.Content{
						Language: "en",
					},
				},
				Ext: json.RawMessage(`{"prebid":{"googlessufeature":true}}`),
			},
			wantErr: false,
		},
		{
			name: "GAM_Unwinding_Enabled_Multi_Imp",
			args: args{
				rctx: models.RequestCtx{
					IsTestRequest: 1,
					PartnerConfigMap: map[int]map[string]string{
						-1: {
							models.AdServerCurrency:           "USD",
							models.SChainDBKey:                "1",
							models.GoogleSSUFeatureEnabledKey: models.Enabled,
						},
					},
					TMax:      500,
					DeviceCtx: models.DeviceCtx{IP: "127.0.0.1"},
					Platform:  models.PLATFORM_APP,
					KADUSERCookie: &http.Cookie{
						Name:  "KADUSERCOOKIE",
						Value: "123456789",
					},
				},
				bidRequest: &openrtb2.BidRequest{
					ID:   "testID",
					Test: 1,
					Cur:  []string{"EUR"},
					TMax: 500,
					Source: &openrtb2.Source{
						TID: "testID",
					},
					Imp: []openrtb2.Imp{
						{
							ID: "testImp1",
							Video: &openrtb2.Video{
								W:         ptrutil.ToPtr[int64](200),
								H:         ptrutil.ToPtr[int64](300),
								Plcmt:     1,
								Protocols: []adcom1.MediaCreativeSubtype{1, 2, 3},
							},
						},
						{
							ID: "testImp2",
							Video: &openrtb2.Video{
								W:         ptrutil.ToPtr[int64](200),
								H:         ptrutil.ToPtr[int64](300),
								Plcmt:     1,
								Protocols: []adcom1.MediaCreativeSubtype{1},
							},
						},
					},
					Device: &openrtb2.Device{
						IP:         "127.0.0.1",
						Language:   "en",
						DeviceType: 1,
					},
					WLang: []string{"en", "hi"},
					User: &openrtb2.User{
						CustomData: "123456789",
						Ext:        json.RawMessage(`{"eids":[{"source":"uidapi.com","uids":[{"id":"UID2:"},{"id":""}]},{"source":"euid.eu","uids":[{"id":""}]},{"source":"liveramp.com","uids":[{"id":"IDL:"}]}]}`),
					},
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{
							ID: "1010",
						},
						Content: &openrtb2.Content{
							Language: "en",
						},
					},
				},
			},
			want: &openrtb2.BidRequest{
				ID:   "testID",
				Test: 1,
				Cur:  []string{"EUR", "USD"},
				TMax: 500,
				Source: &openrtb2.Source{
					TID: "testID",
				},
				Imp: []openrtb2.Imp{
					{
						ID: "testImp1",
						Video: &openrtb2.Video{
							W:         ptrutil.ToPtr[int64](200),
							H:         ptrutil.ToPtr[int64](300),
							Plcmt:     1,
							Protocols: []adcom1.MediaCreativeSubtype{1, 2, 3, 6, 7, 8},
						},
					},
					{
						ID: "testImp2",
						Video: &openrtb2.Video{
							W:         ptrutil.ToPtr[int64](200),
							H:         ptrutil.ToPtr[int64](300),
							Plcmt:     1,
							Protocols: []adcom1.MediaCreativeSubtype{1, 3, 6, 7, 8},
						},
					},
				},
				Device: &openrtb2.Device{
					IP:         "127.0.0.1",
					Language:   "en",
					DeviceType: 1,
				},
				WLang: []string{"en", "hi"},
				User: &openrtb2.User{
					CustomData: "123456789",
					Ext:        json.RawMessage(`{}`),
				},
				Site: &openrtb2.Site{
					Publisher: &openrtb2.Publisher{
						ID: "1010",
					},
					Content: &openrtb2.Content{
						Language: "en",
					},
				},
				Ext: json.RawMessage(`{"prebid":{"googlessufeature":true}}`),
			},
			wantErr: false,
		},
		{
			name: "GAM_Unwinding_Enabled_Empty_Protocols",
			args: args{
				rctx: models.RequestCtx{
					IsTestRequest: 1,
					PartnerConfigMap: map[int]map[string]string{
						-1: {
							models.AdServerCurrency:           "USD",
							models.SChainDBKey:                "1",
							models.GoogleSSUFeatureEnabledKey: models.Enabled,
						},
					},
					TMax:      500,
					DeviceCtx: models.DeviceCtx{IP: "127.0.0.1"},
					Platform:  models.PLATFORM_APP,
					KADUSERCookie: &http.Cookie{
						Name:  "KADUSERCOOKIE",
						Value: "123456789",
					},
				},
				bidRequest: &openrtb2.BidRequest{
					ID:   "testID",
					Test: 1,
					Cur:  []string{"EUR"},
					TMax: 500,
					Source: &openrtb2.Source{
						TID: "testID",
					},
					Imp: []openrtb2.Imp{
						{
							ID: "testImp1",
							Video: &openrtb2.Video{
								W:         ptrutil.ToPtr[int64](200),
								H:         ptrutil.ToPtr[int64](300),
								Plcmt:     1,
								Protocols: []adcom1.MediaCreativeSubtype{},
							},
						},
					},
					Device: &openrtb2.Device{
						IP:         "127.0.0.1",
						Language:   "en",
						DeviceType: 1,
					},
					WLang: []string{"en", "hi"},
					User: &openrtb2.User{
						CustomData: "123456789",
						Ext:        json.RawMessage(`{"eids":[{"source":"uidapi.com","uids":[{"id":"UID2:"},{"id":""}]},{"source":"euid.eu","uids":[{"id":""}]},{"source":"liveramp.com","uids":[{"id":"IDL:"}]}]}`),
					},
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{
							ID: "1010",
						},
						Content: &openrtb2.Content{
							Language: "en",
						},
					},
				},
			},
			want: &openrtb2.BidRequest{
				ID:   "testID",
				Test: 1,
				Cur:  []string{"EUR", "USD"},
				TMax: 500,
				Source: &openrtb2.Source{
					TID: "testID",
				},
				Imp: []openrtb2.Imp{
					{
						ID: "testImp1",
						Video: &openrtb2.Video{
							W:         ptrutil.ToPtr[int64](200),
							H:         ptrutil.ToPtr[int64](300),
							Plcmt:     1,
							Protocols: []adcom1.MediaCreativeSubtype{3, 6, 7, 8},
						},
					},
				},
				Device: &openrtb2.Device{
					IP:         "127.0.0.1",
					Language:   "en",
					DeviceType: 1,
				},
				WLang: []string{"en", "hi"},
				User: &openrtb2.User{
					CustomData: "123456789",
					Ext:        json.RawMessage(`{}`),
				},
				Site: &openrtb2.Site{
					Publisher: &openrtb2.Publisher{
						ID: "1010",
					},
					Content: &openrtb2.Content{
						Language: "en",
					},
				},
				Ext: json.RawMessage(`{"prebid":{"googlessufeature":true}}`),
			},
			wantErr: false,
		},
		{
			name: "GAM_Unwinding_Enabled_Protocols_Not_Present",
			args: args{
				rctx: models.RequestCtx{
					IsTestRequest: 1,
					PartnerConfigMap: map[int]map[string]string{
						-1: {
							models.AdServerCurrency:           "USD",
							models.SChainDBKey:                "1",
							models.GoogleSSUFeatureEnabledKey: models.Enabled,
						},
					},
					TMax:      500,
					DeviceCtx: models.DeviceCtx{IP: "127.0.0.1"},
					Platform:  models.PLATFORM_APP,
					KADUSERCookie: &http.Cookie{
						Name:  "KADUSERCOOKIE",
						Value: "123456789",
					},
				},
				bidRequest: &openrtb2.BidRequest{
					ID:   "testID",
					Test: 1,
					Cur:  []string{"EUR"},
					TMax: 500,
					Source: &openrtb2.Source{
						TID: "testID",
					},
					Imp: []openrtb2.Imp{
						{
							ID: "testImp1",
							Video: &openrtb2.Video{
								W:     ptrutil.ToPtr[int64](200),
								H:     ptrutil.ToPtr[int64](300),
								Plcmt: 1,
							},
						},
					},
					Device: &openrtb2.Device{
						IP:         "127.0.0.1",
						Language:   "en",
						DeviceType: 1,
					},
					WLang: []string{"en", "hi"},
					User: &openrtb2.User{
						CustomData: "123456789",
						Ext:        json.RawMessage(`{"eids":[{"source":"uidapi.com","uids":[{"id":"UID2:"},{"id":""}]},{"source":"euid.eu","uids":[{"id":""}]},{"source":"liveramp.com","uids":[{"id":"IDL:"}]}]}`),
					},
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{
							ID: "1010",
						},
						Content: &openrtb2.Content{
							Language: "en",
						},
					},
				},
			},
			want: &openrtb2.BidRequest{
				ID:   "testID",
				Test: 1,
				Cur:  []string{"EUR", "USD"},
				TMax: 500,
				Source: &openrtb2.Source{
					TID: "testID",
				},
				Imp: []openrtb2.Imp{
					{
						ID: "testImp1",
						Video: &openrtb2.Video{
							W:         ptrutil.ToPtr[int64](200),
							H:         ptrutil.ToPtr[int64](300),
							Plcmt:     1,
							Protocols: []adcom1.MediaCreativeSubtype{3, 6, 7, 8},
						},
					},
				},
				Device: &openrtb2.Device{
					IP:         "127.0.0.1",
					Language:   "en",
					DeviceType: 1,
				},
				WLang: []string{"en", "hi"},
				User: &openrtb2.User{
					CustomData: "123456789",
					Ext:        json.RawMessage(`{}`),
				},
				Site: &openrtb2.Site{
					Publisher: &openrtb2.Publisher{
						ID: "1010",
					},
					Content: &openrtb2.Content{
						Language: "en",
					},
				},
				Ext: json.RawMessage(`{"prebid":{"googlessufeature":true}}`),
			},
			wantErr: false,
		},
		{
			name: "GAM_Unwinding_Disabled",
			args: args{
				rctx: models.RequestCtx{
					IsTestRequest: 1,
					PartnerConfigMap: map[int]map[string]string{
						-1: {
							models.AdServerCurrency:           "USD",
							models.SChainDBKey:                "1",
							models.GoogleSSUFeatureEnabledKey: "0",
						},
					},
					TMax:      500,
					DeviceCtx: models.DeviceCtx{IP: "127.0.0.1"},
					Platform:  models.PLATFORM_APP,
					KADUSERCookie: &http.Cookie{
						Name:  "KADUSERCOOKIE",
						Value: "123456789",
					},
				},
				bidRequest: &openrtb2.BidRequest{
					ID:   "testID",
					Test: 1,
					Cur:  []string{"EUR"},
					TMax: 500,
					Source: &openrtb2.Source{
						TID: "testID",
					},
					Imp: []openrtb2.Imp{
						{
							ID: "testImp1",
							Video: &openrtb2.Video{
								W:         ptrutil.ToPtr[int64](200),
								H:         ptrutil.ToPtr[int64](300),
								Plcmt:     1,
								Protocols: []adcom1.MediaCreativeSubtype{1, 2, 3},
							},
						},
					},
					Device: &openrtb2.Device{
						IP:         "127.0.0.1",
						Language:   "en",
						DeviceType: 1,
					},
					WLang: []string{"en", "hi"},
					User: &openrtb2.User{
						CustomData: "123456789",
						Ext:        json.RawMessage(`{"eids":[{"source":"uidapi.com","uids":[{"id":"UID2:"},{"id":""}]},{"source":"euid.eu","uids":[{"id":""}]},{"source":"liveramp.com","uids":[{"id":"IDL:"}]}]}`),
					},
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{
							ID: "1010",
						},
						Content: &openrtb2.Content{
							Language: "en",
						},
					},
				},
			},
			want: &openrtb2.BidRequest{
				ID:   "testID",
				Test: 1,
				Cur:  []string{"EUR", "USD"},
				TMax: 500,
				Source: &openrtb2.Source{
					TID: "testID",
				},
				Imp: []openrtb2.Imp{
					{
						ID: "testImp1",
						Video: &openrtb2.Video{
							W:         ptrutil.ToPtr[int64](200),
							H:         ptrutil.ToPtr[int64](300),
							Plcmt:     1,
							Protocols: []adcom1.MediaCreativeSubtype{1, 2, 3},
						},
					},
				},
				Device: &openrtb2.Device{
					IP:         "127.0.0.1",
					Language:   "en",
					DeviceType: 1,
				},
				WLang: []string{"en", "hi"},
				User: &openrtb2.User{
					CustomData: "123456789",
					Ext:        json.RawMessage(`{}`),
				},
				Site: &openrtb2.Site{
					Publisher: &openrtb2.Publisher{
						ID: "1010",
					},
					Content: &openrtb2.Content{
						Language: "en",
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &OpenWrap{
				cfg:          tt.fields.cfg,
				cache:        tt.fields.cache,
				metricEngine: tt.fields.metricEngine,
			}
			got, err := m.applyProfileChanges(tt.args.rctx, tt.args.bidRequest)
			if (err != nil) != tt.wantErr {
				assert.Equal(t, tt.wantErr, err != nil)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOpenWrap_applyImpChanges(t *testing.T) {
	tests := []struct {
		name        string
		inputImp    *openrtb2.Imp
		rCtx        models.RequestCtx
		expectedImp *openrtb2.Imp
	}{
		{
			name: "empty_bidfloor",
			inputImp: &openrtb2.Imp{
				ID:          "imp1",
				BidFloor:    0,
				BidFloorCur: "USD",
			},
			rCtx: models.RequestCtx{},
			expectedImp: &openrtb2.Imp{
				ID:          "imp1",
				BidFloor:    0,
				BidFloorCur: "",
			},
		},
		{
			name: "empty_bidfloorcur",
			inputImp: &openrtb2.Imp{
				ID:          "imp1",
				BidFloor:    1.0,
				BidFloorCur: "",
			},
			rCtx: models.RequestCtx{},
			expectedImp: &openrtb2.Imp{
				ID:          "imp1",
				BidFloor:    1.0,
				BidFloorCur: "USD",
			},
		},
		{
			name: "apply_imp_video_changes",
			inputImp: &openrtb2.Imp{
				ID: "imp3",
				Video: &openrtb2.Video{
					Protocols: []adcom1.MediaCreativeSubtype{
						adcom1.CreativeVAST10,
						adcom1.CreativeVAST20},
					Placement: adcom1.VideoPlacementInStream,
				},
			},
			rCtx: models.RequestCtx{
				NewReqExt: &models.RequestExt{
					ExtRequest: openrtb_ext.ExtRequest{
						Prebid: openrtb_ext.ExtRequestPrebid{
							ExtOWRequestPrebid: openrtb_ext.ExtOWRequestPrebid{
								GoogleSSUFeatureEnabled: true,
							},
						},
					},
				},
			},
			expectedImp: &openrtb2.Imp{
				ID: "imp3",
				Video: &openrtb2.Video{
					Protocols: []adcom1.MediaCreativeSubtype{
						adcom1.CreativeVAST10,
						adcom1.CreativeVAST20,
						adcom1.CreativeVAST30,
						adcom1.CreativeVAST30Wrapper,
						adcom1.CreativeVAST40,
						adcom1.CreativeVAST40Wrapper},
					Placement: adcom1.VideoPlacementInStream,
					Plcmt:     adcom1.VideoPlcmtInstream,
				},
			},
		},
		{
			name: "Impression extensions updated",
			inputImp: &openrtb2.Imp{
				ID: "imp5",
			},
			rCtx: models.RequestCtx{
				ImpBidCtx: map[string]models.ImpCtx{
					"imp5": {
						NewExt: []byte(`{"key":"value"}`),
					},
				},
			},
			expectedImp: &openrtb2.Imp{
				ID:  "imp5",
				Ext: []byte(`{"key":"value"}`),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			m := &OpenWrap{}
			m.applyImpChanges(test.rCtx, test.inputImp)
			assert.Equal(t, test.expectedImp, test.inputImp)
		})
	}
}

func TestOpenWrap_applyImpVideoChanges(t *testing.T) {
	tests := []struct {
		name          string
		inputVideo    *openrtb2.Video
		rCtx          models.RequestCtx
		expectedVideo *openrtb2.Video
	}{
		{
			name: "googlessufeature_enabled",
			inputVideo: &openrtb2.Video{
				Protocols: []adcom1.MediaCreativeSubtype{
					adcom1.CreativeVAST10,
					adcom1.CreativeVAST20},
			},
			rCtx: models.RequestCtx{
				NewReqExt: &models.RequestExt{
					ExtRequest: openrtb_ext.ExtRequest{
						Prebid: openrtb_ext.ExtRequestPrebid{
							ExtOWRequestPrebid: openrtb_ext.ExtOWRequestPrebid{
								GoogleSSUFeatureEnabled: true,
							},
						},
					},
				},
			},
			expectedVideo: &openrtb2.Video{
				Protocols: []adcom1.MediaCreativeSubtype{
					adcom1.CreativeVAST10,
					adcom1.CreativeVAST20,
					adcom1.CreativeVAST30,
					adcom1.CreativeVAST30Wrapper,
					adcom1.CreativeVAST40,
					adcom1.CreativeVAST40Wrapper},
			},
		},
		{
			name: "video.plcmt_present",
			inputVideo: &openrtb2.Video{
				Placement: 0,
				Plcmt:     adcom1.VideoPlcmtInterstitial,
			},
			rCtx: models.RequestCtx{},
			expectedVideo: &openrtb2.Video{
				Placement: 0,
				Plcmt:     adcom1.VideoPlcmtInterstitial,
			},
		},
		{
			name: "video.placement_video.plcmt_present",
			inputVideo: &openrtb2.Video{
				Placement: adcom1.VideoPlacementAlwaysVisible,
				Plcmt:     adcom1.VideoPlcmtInterstitial,
			},
			rCtx: models.RequestCtx{},
			expectedVideo: &openrtb2.Video{
				Placement: adcom1.VideoPlacementAlwaysVisible,
				Plcmt:     adcom1.VideoPlcmtInterstitial,
			},
		},
		{
			name: "video.placement=instream",
			inputVideo: &openrtb2.Video{
				Placement: adcom1.VideoPlacementInStream,
			},
			rCtx: models.RequestCtx{},
			expectedVideo: &openrtb2.Video{
				Placement: adcom1.VideoPlacementInStream,
				Plcmt:     adcom1.VideoPlcmtInstream,
			},
		},
		{
			name: "video.placement=interstitial",
			inputVideo: &openrtb2.Video{
				Placement: adcom1.VideoPlacementAlwaysVisible,
			},
			rCtx: models.RequestCtx{},
			expectedVideo: &openrtb2.Video{
				Placement: adcom1.VideoPlacementAlwaysVisible,
				Plcmt:     adcom1.VideoPlcmtInterstitial,
			},
		},
		{
			name: "video.placement=inbanner",
			inputVideo: &openrtb2.Video{
				Placement: adcom1.VideoPlacementInBanner,
			},
			rCtx: models.RequestCtx{},
			expectedVideo: &openrtb2.Video{
				Placement: adcom1.VideoPlacementInBanner,
				Plcmt:     adcom1.VideoPlcmtNoContent,
			},
		},
		{
			name: "video.placement=inarticle",
			inputVideo: &openrtb2.Video{
				Placement: adcom1.VideoPlacementInArticle,
			},
			rCtx: models.RequestCtx{},
			expectedVideo: &openrtb2.Video{
				Placement: adcom1.VideoPlacementInArticle,
			},
		},
		{
			name: "video.placement=infeed",
			inputVideo: &openrtb2.Video{
				Placement: adcom1.VideoPlacementInFeed,
			},
			rCtx: models.RequestCtx{},
			expectedVideo: &openrtb2.Video{
				Placement: adcom1.VideoPlacementInFeed,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			m := &OpenWrap{}
			m.applyImpVideoChanges(test.rCtx, test.inputVideo)
			assert.Equal(t, test.expectedVideo, test.inputVideo)
		})
	}
}

func TestOpenWrap_applyVideoAdUnitConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockFeature := mock_feature.NewMockFeature(ctrl)

	type fields struct {
		cfg          config.Config
		cache        cache.Cache
		metricEngine metrics.MetricsEngine
	}
	type args struct {
		rCtx models.RequestCtx
		imp  *openrtb2.Imp
	}
	type want struct {
		rCtx models.RequestCtx
		imp  *openrtb2.Imp
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
		setup  func()
	}{
		{
			name: "imp.video_is_nil",
			args: args{
				imp: &openrtb2.Imp{
					Video: nil,
				},
			},
			want: want{
				imp: &openrtb2.Imp{
					Video: nil,
				},
			},
		},
		{
			name: "empty_adunitCfg",
			args: args{
				rCtx: models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"testImp": {
							VideoAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: nil,
							},
						},
					},
				},
				imp: &openrtb2.Imp{
					ID:    "testImp",
					Video: &openrtb2.Video{},
				},
			},
			want: want{
				imp: &openrtb2.Imp{
					ID:    "testImp",
					Video: &openrtb2.Video{},
				},
				rCtx: models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"testImp": {
							VideoAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: nil,
							},
						},
					},
				},
			},
		},
		{
			name: "imp.BidFloor_and_BidFloorCur_updated_from_adunit_config",
			args: args{
				rCtx: models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"testImp": {
							VideoAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
									BidFloor:    ptrutil.ToPtr(2.0),
									BidFloorCur: ptrutil.ToPtr("USD"),
								},
							},
						},
					},
				},
				imp: &openrtb2.Imp{
					ID:          "testImp",
					BidFloor:    0,
					BidFloorCur: "",
					Video:       &openrtb2.Video{},
				},
			},
			want: want{
				imp: &openrtb2.Imp{
					ID:          "testImp",
					Video:       &openrtb2.Video{},
					BidFloor:    2.0,
					BidFloorCur: "USD",
				},
				rCtx: models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"testImp": {
							VideoAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
									BidFloor:    ptrutil.ToPtr(2.0),
									BidFloorCur: ptrutil.ToPtr("USD"),
								},
							},
							BidFloor:    2,
							BidFloorCur: "USD",
						},
					},
				},
			},
		},
		{
			name: "imp.Exp_updated_from_adunit_config",
			args: args{
				rCtx: models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"testImp": {
							VideoAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
									Exp: ptrutil.ToPtr(10),
								},
							},
						},
					},
				},
				imp: &openrtb2.Imp{
					ID:    "testImp",
					Video: &openrtb2.Video{},
				},
			},
			want: want{
				imp: &openrtb2.Imp{
					ID:    "testImp",
					Video: &openrtb2.Video{},
					Exp:   10,
				},
				rCtx: models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"testImp": {
							VideoAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
									Exp: ptrutil.ToPtr(10),
								},
							},
						},
					},
				},
			},
		},
		{
			name: "imp_has_video_object_but_adunitConfig_video_is_nil._imp_video_will_not_be_updated",
			args: args{
				rCtx: models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"testImp": {
							VideoAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
									Video: nil,
								},
							},
						},
					},
				},
				imp: &openrtb2.Imp{
					ID: "testImp",
					Video: &openrtb2.Video{
						W: ptrutil.ToPtr[int64](200),
						H: ptrutil.ToPtr[int64](300),
					},
				},
			},
			want: want{
				imp: &openrtb2.Imp{
					ID: "testImp",
					Video: &openrtb2.Video{
						W: ptrutil.ToPtr[int64](200),
						H: ptrutil.ToPtr[int64](300),
					},
				},
				rCtx: models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"testImp": {
							VideoAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
									Video: nil,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "imp_has_video_object_but_video_is_disabled_from_adunitConfig_then_remove_video_object_from_imp",
			args: args{
				rCtx: models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"testImp": {
							VideoAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
									Video: &adunitconfig.Video{
										Enabled: ptrutil.ToPtr(false),
									},
								},
							},
						},
					},
				},
				imp: &openrtb2.Imp{
					ID: "testImp",
					Video: &openrtb2.Video{
						W: ptrutil.ToPtr[int64](200),
						H: ptrutil.ToPtr[int64](300),
					},
				},
			},
			want: want{
				imp: &openrtb2.Imp{
					ID:    "testImp",
					Video: nil,
				},
				rCtx: models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"testImp": {
							VideoAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
									Video: &adunitconfig.Video{
										Enabled: ptrutil.ToPtr(false),
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "imp_has_empty_video_object_and_adunitCofig_for_video_is_enable._all_absent_video_parameters_will_be_updated",
			args: args{
				rCtx: models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"testImp": {
							VideoAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
									Video: &adunitconfig.Video{
										Enabled: ptrutil.ToPtr(true),
										Config: &adunitconfig.VideoConfig{
											Video: openrtb2.Video{
												MinDuration:    10,
												MaxDuration:    40,
												Skip:           ptrutil.ToPtr(int8(1)),
												SkipMin:        5,
												SkipAfter:      10,
												Plcmt:          1,
												Placement:      1,
												MinBitRate:     100,
												MaxBitRate:     200,
												MaxExtended:    50,
												Linearity:      1,
												Protocol:       1,
												W:              ptrutil.ToPtr[int64](640),
												H:              ptrutil.ToPtr[int64](480),
												Sequence:       2,
												BoxingAllowed:  ptrutil.ToPtr[int8](1),
												PlaybackEnd:    2,
												MIMEs:          []string{"mimes"},
												API:            []adcom1.APIFramework{1, 2},
												Delivery:       []adcom1.DeliveryMethod{1, 2},
												PlaybackMethod: []adcom1.PlaybackMethod{1, 2},
												BAttr:          []adcom1.CreativeAttribute{1, 2},
												StartDelay:     ptrutil.ToPtr(adcom1.StartDelay(2)),
												Protocols:      []adcom1.MediaCreativeSubtype{1, 2},
												Pos:            ptrutil.ToPtr(adcom1.PlacementPosition(1)),
												CompanionType:  []adcom1.CompanionType{1, 2},
											},
										},
									},
								},
							},
						},
					},
				},
				imp: &openrtb2.Imp{
					ID:    "testImp",
					Video: &openrtb2.Video{},
				},
			},
			want: want{
				imp: &openrtb2.Imp{
					ID: "testImp",
					Video: &openrtb2.Video{
						W:              ptrutil.ToPtr[int64](640),
						H:              ptrutil.ToPtr[int64](480),
						MinDuration:    10,
						MaxDuration:    40,
						Skip:           ptrutil.ToPtr(int8(1)),
						SkipMin:        5,
						SkipAfter:      10,
						Plcmt:          1,
						Placement:      1,
						MinBitRate:     100,
						MaxBitRate:     200,
						MaxExtended:    50,
						Linearity:      1,
						Protocol:       1,
						Sequence:       2,
						BoxingAllowed:  ptrutil.ToPtr[int8](1),
						PlaybackEnd:    2,
						MIMEs:          []string{"mimes"},
						API:            []adcom1.APIFramework{1, 2},
						Delivery:       []adcom1.DeliveryMethod{1, 2},
						PlaybackMethod: []adcom1.PlaybackMethod{1, 2},
						BAttr:          []adcom1.CreativeAttribute{1, 2},
						StartDelay:     ptrutil.ToPtr(adcom1.StartDelay(2)),
						Protocols:      []adcom1.MediaCreativeSubtype{1, 2},
						Pos:            ptrutil.ToPtr(adcom1.PlacementPosition(1)),
						CompanionType:  []adcom1.CompanionType{1, 2},
					},
				},
				rCtx: models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"testImp": {
							VideoAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
									Video: &adunitconfig.Video{
										Enabled: ptrutil.ToPtr(true),
										Config: &adunitconfig.VideoConfig{
											Video: openrtb2.Video{
												MinDuration:    10,
												MaxDuration:    40,
												Skip:           ptrutil.ToPtr(int8(1)),
												SkipMin:        5,
												SkipAfter:      10,
												Plcmt:          1,
												Placement:      1,
												MinBitRate:     100,
												MaxBitRate:     200,
												MaxExtended:    50,
												Linearity:      1,
												Protocol:       1,
												W:              ptrutil.ToPtr[int64](640),
												H:              ptrutil.ToPtr[int64](480),
												Sequence:       2,
												BoxingAllowed:  ptrutil.ToPtr[int8](1),
												PlaybackEnd:    2,
												MIMEs:          []string{"mimes"},
												API:            []adcom1.APIFramework{1, 2},
												Delivery:       []adcom1.DeliveryMethod{1, 2},
												PlaybackMethod: []adcom1.PlaybackMethod{1, 2},
												BAttr:          []adcom1.CreativeAttribute{1, 2},
												StartDelay:     ptrutil.ToPtr(adcom1.StartDelay(2)),
												Protocols:      []adcom1.MediaCreativeSubtype{1, 2},
												Pos:            ptrutil.ToPtr(adcom1.PlacementPosition(1)),
												CompanionType:  []adcom1.CompanionType{1, 2},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "imp_has_video_object_and_adunitConfig_alos_have_parameter_present_then_priority_to_request",
			args: args{
				rCtx: models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"testImp": {
							VideoAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
									Video: &adunitconfig.Video{
										Enabled: ptrutil.ToPtr(true),
										Config: &adunitconfig.VideoConfig{
											Video: openrtb2.Video{
												MinDuration: 10,
												MaxDuration: 40,
												Skip:        ptrutil.ToPtr(int8(1)),
												SkipMin:     5,
												SkipAfter:   10,
											},
										},
									},
								},
							},
						},
					},
				},
				imp: &openrtb2.Imp{
					ID: "testImp",
					Video: &openrtb2.Video{
						W:           ptrutil.ToPtr[int64](640),
						H:           ptrutil.ToPtr[int64](480),
						MinDuration: 20,
						MaxDuration: 60,
						Skip:        ptrutil.ToPtr(int8(2)),
						SkipMin:     10,
						SkipAfter:   20,
					},
				},
			},
			want: want{
				imp: &openrtb2.Imp{
					ID: "testImp",
					Video: &openrtb2.Video{
						W:           ptrutil.ToPtr[int64](640),
						H:           ptrutil.ToPtr[int64](480),
						MinDuration: 20,
						MaxDuration: 60,
						Skip:        ptrutil.ToPtr(int8(2)),
						SkipMin:     10,
						SkipAfter:   20,
					},
				},
				rCtx: models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"testImp": {
							VideoAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
									Video: &adunitconfig.Video{
										Enabled: ptrutil.ToPtr(true),
										Config: &adunitconfig.VideoConfig{
											Video: openrtb2.Video{
												MinDuration: 10,
												MaxDuration: 40,
												Skip:        ptrutil.ToPtr(int8(1)),
												SkipMin:     5,
												SkipAfter:   10,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "imp.video_is_nil_but_AmpVideoEnabled_true_update_and_no_video_config_update_imp.Video_to_default_video",
			args: args{
				imp: &openrtb2.Imp{
					ID: "testImp",
					Banner: &openrtb2.Banner{
						W: ptrutil.ToPtr[int64](250),
						H: ptrutil.ToPtr[int64](300),
					},
				},
				rCtx: models.RequestCtx{
					PubID: 5890,
					ImpBidCtx: map[string]models.ImpCtx{
						"testImp": {
							VideoAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
									Video: &adunitconfig.Video{
										Enabled: ptrutil.ToPtr(true),
										Config: &adunitconfig.VideoConfig{
											Video: openrtb2.Video{
												MIMEs:          []string{"video/mpev"},
												MinDuration:    10,
												MaxDuration:    50,
												StartDelay:     adcom1.StartMidRoll.Ptr(),
												Protocols:      []adcom1.MediaCreativeSubtype{adcom1.CreativeVAST10, adcom1.CreativeVAST20, adcom1.CreativeVAST30, adcom1.CreativeVAST10Wrapper, adcom1.CreativeVAST20Wrapper},
												Placement:      adcom1.VideoPlacementInArticle,
												Plcmt:          adcom1.VideoPlcmtInstream,
												Linearity:      adcom1.LinearityNonLinear,
												Skip:           ptrutil.ToPtr[int8](1),
												SkipMin:        1,
												SkipAfter:      1,
												PlaybackMethod: []adcom1.PlaybackMethod{adcom1.PlaybackClickSoundOn},
												PlaybackEnd:    adcom1.PlaybackFloating,
												Delivery:       []adcom1.DeliveryMethod{adcom1.DeliveryProgressive},
												W:              ptrutil.ToPtr[int64](300),
												H:              ptrutil.ToPtr[int64](400),
											},
										},
									},
								},
							},
						},
					},
					AmpVideoEnabled: true,
				},
			},
			want: want{
				imp: &openrtb2.Imp{
					ID: "testImp",
					Banner: &openrtb2.Banner{
						W: ptrutil.ToPtr[int64](250),
						H: ptrutil.ToPtr[int64](300),
					},
					Video: &openrtb2.Video{
						MIMEs:          []string{"video/mpev"},
						MinDuration:    10,
						MaxDuration:    50,
						StartDelay:     adcom1.StartMidRoll.Ptr(),
						Protocols:      []adcom1.MediaCreativeSubtype{adcom1.CreativeVAST10, adcom1.CreativeVAST20, adcom1.CreativeVAST30, adcom1.CreativeVAST10Wrapper, adcom1.CreativeVAST20Wrapper},
						Placement:      adcom1.VideoPlacementInArticle,
						Plcmt:          adcom1.VideoPlcmtInstream,
						Linearity:      adcom1.LinearityNonLinear,
						Skip:           ptrutil.ToPtr[int8](1),
						SkipMin:        1,
						SkipAfter:      1,
						PlaybackMethod: []adcom1.PlaybackMethod{adcom1.PlaybackClickSoundOn},
						PlaybackEnd:    adcom1.PlaybackFloating,
						Delivery:       []adcom1.DeliveryMethod{adcom1.DeliveryProgressive},
						W:              ptrutil.ToPtr[int64](300),
						H:              ptrutil.ToPtr[int64](400),
					},
				},
				rCtx: models.RequestCtx{
					PubID: 5890,
					ImpBidCtx: map[string]models.ImpCtx{
						"testImp": {
							VideoAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
									Video: &adunitconfig.Video{
										Enabled: ptrutil.ToPtr(true),
										Config: &adunitconfig.VideoConfig{
											Video: openrtb2.Video{
												MIMEs:          []string{"video/mpev"},
												MinDuration:    10,
												MaxDuration:    50,
												StartDelay:     adcom1.StartMidRoll.Ptr(),
												Protocols:      []adcom1.MediaCreativeSubtype{adcom1.CreativeVAST10, adcom1.CreativeVAST20, adcom1.CreativeVAST30, adcom1.CreativeVAST10Wrapper, adcom1.CreativeVAST20Wrapper},
												Placement:      adcom1.VideoPlacementInArticle,
												Plcmt:          adcom1.VideoPlcmtInstream,
												Linearity:      adcom1.LinearityNonLinear,
												Skip:           ptrutil.ToPtr[int8](1),
												SkipMin:        1,
												SkipAfter:      1,
												PlaybackMethod: []adcom1.PlaybackMethod{adcom1.PlaybackClickSoundOn},
												PlaybackEnd:    adcom1.PlaybackFloating,
												Delivery:       []adcom1.DeliveryMethod{adcom1.DeliveryProgressive},
												W:              ptrutil.ToPtr[int64](300),
												H:              ptrutil.ToPtr[int64](400),
											},
										},
									},
								},
							},
						},
					},
					AmpVideoEnabled: true,
				},
			},
		},
		{
			name: "imp.video_is_nil_but_AmpVideoEnabled_true_update_and_video_config_is_also_non_nil_update_imp.Video_to_video_config",
			args: args{
				imp: &openrtb2.Imp{
					ID: "testImp",
					Banner: &openrtb2.Banner{
						W: ptrutil.ToPtr[int64](250),
						H: ptrutil.ToPtr[int64](300),
					},
				},
				rCtx: models.RequestCtx{
					PubID: 5890,
					ImpBidCtx: map[string]models.ImpCtx{
						"testImp": {
							VideoAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
									Video: &adunitconfig.Video{
										Enabled: ptrutil.ToPtr(true),
									},
								},
							},
						},
					},
					AmpVideoEnabled: true,
				},
			},
			want: want{
				imp: &openrtb2.Imp{
					ID: "testImp",
					Banner: &openrtb2.Banner{
						W: ptrutil.ToPtr[int64](250),
						H: ptrutil.ToPtr[int64](300),
					},
					Video: &openrtb2.Video{
						MIMEs:          []string{"video/mp4"},
						MinDuration:    0,
						MaxDuration:    30,
						StartDelay:     adcom1.StartPreRoll.Ptr(),
						Protocols:      []adcom1.MediaCreativeSubtype{adcom1.CreativeVAST10, adcom1.CreativeVAST20, adcom1.CreativeVAST30, adcom1.CreativeVAST10Wrapper, adcom1.CreativeVAST20Wrapper, adcom1.CreativeVAST30Wrapper, adcom1.CreativeVAST40, adcom1.CreativeVAST40Wrapper, adcom1.CreativeVAST41, adcom1.CreativeVAST41Wrapper, adcom1.CreativeVAST42, adcom1.CreativeVAST42Wrapper},
						Placement:      adcom1.VideoPlacementInBanner,
						Plcmt:          adcom1.VideoPlcmtNoContent,
						Linearity:      adcom1.LinearityLinear,
						Skip:           ptrutil.ToPtr[int8](0),
						SkipMin:        0,
						SkipAfter:      0,
						PlaybackMethod: []adcom1.PlaybackMethod{adcom1.PlaybackPageLoadSoundOff},
						PlaybackEnd:    adcom1.PlaybackCompletion,
						Delivery:       []adcom1.DeliveryMethod{adcom1.DeliveryProgressive, adcom1.DeliveryDownload},
						W:              ptrutil.ToPtr[int64](250),
						H:              ptrutil.ToPtr[int64](300),
					},
				},
				rCtx: models.RequestCtx{
					PubID: 5890,
					ImpBidCtx: map[string]models.ImpCtx{
						"testImp": {
							VideoAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
									Video: &adunitconfig.Video{
										Enabled: ptrutil.ToPtr(true),
									},
								},
							},
						},
					},
					AmpVideoEnabled: true,
				},
			},
		},
		{
			name: "imp.BidFloor_is_less_than_BidFloor_from_adunit_config_for_applovinmax_setMaxFloor_true",
			args: args{
				rCtx: models.RequestCtx{
					Endpoint:           models.EndpointAppLovinMax,
					IsMaxFloorsEnabled: true,
					ImpBidCtx: map[string]models.ImpCtx{
						"testImp": {
							VideoAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
									BidFloor:    ptrutil.ToPtr(2.0),
									BidFloorCur: ptrutil.ToPtr("USD"),
								},
							},
						},
					},
				},
				imp: &openrtb2.Imp{
					ID:          "testImp",
					BidFloor:    1,
					BidFloorCur: "USD",
					Video:       &openrtb2.Video{},
				},
			},
			want: want{
				imp: &openrtb2.Imp{
					ID:          "testImp",
					Video:       &openrtb2.Video{},
					BidFloor:    2.0,
					BidFloorCur: "USD",
				},
				rCtx: models.RequestCtx{
					Endpoint:           models.EndpointAppLovinMax,
					IsMaxFloorsEnabled: true,
					ImpBidCtx: map[string]models.ImpCtx{
						"testImp": {
							VideoAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
									BidFloor:    ptrutil.ToPtr(2.0),
									BidFloorCur: ptrutil.ToPtr("USD"),
								},
							},
							BidFloor:    2,
							BidFloorCur: "USD",
						},
					},
				},
			},
		},
		{
			name: "imp.BidFloor_is_less_than_BidFloor_from_adunit_config_for_applovinmax_setMaxFloor_false",
			args: args{
				rCtx: models.RequestCtx{
					Endpoint: models.EndpointAppLovinMax,
					ImpBidCtx: map[string]models.ImpCtx{
						"testImp": {
							VideoAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
									BidFloor:    ptrutil.ToPtr(2.0),
									BidFloorCur: ptrutil.ToPtr("USD"),
								},
							},
						},
					},
				},
				imp: &openrtb2.Imp{
					ID:          "testImp",
					BidFloor:    1,
					BidFloorCur: "USD",
					Video:       &openrtb2.Video{},
				},
			},
			want: want{
				imp: &openrtb2.Imp{
					ID:          "testImp",
					Video:       &openrtb2.Video{},
					BidFloor:    1.0,
					BidFloorCur: "USD",
				},
				rCtx: models.RequestCtx{
					Endpoint: models.EndpointAppLovinMax,
					ImpBidCtx: map[string]models.ImpCtx{
						"testImp": {
							VideoAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
									BidFloor:    ptrutil.ToPtr(2.0),
									BidFloorCur: ptrutil.ToPtr("USD"),
								},
							},
							BidFloor:    1,
							BidFloorCur: "USD",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &OpenWrap{
				cfg:           tt.fields.cfg,
				cache:         tt.fields.cache,
				metricEngine:  tt.fields.metricEngine,
				pubFeatures:   mockFeature,
				rateConvertor: currency.NewRateConverter(&http.Client{}, 60*time.Second, "", time.Duration(0)),
			}
			if tt.setup != nil {
				tt.setup()
			}
			m.applyVideoAdUnitConfig(tt.args.rCtx, tt.args.imp)
			assert.Equal(t, tt.args.imp, tt.want.imp, "Imp video is not upadted as expected from adunit config")
			assert.Equal(t, tt.args.rCtx, tt.want.rCtx, "rctx is not upadted as expected from adunit config")
		})
	}
}

func TestOpenWrap_applyBannerAdUnitConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockFeature := mock_feature.NewMockFeature(ctrl)
	type fields struct {
		cfg          config.Config
		cache        cache.Cache
		metricEngine metrics.MetricsEngine
	}
	type args struct {
		rCtx models.RequestCtx
		imp  *openrtb2.Imp
	}
	type want struct {
		rCtx models.RequestCtx
		imp  *openrtb2.Imp
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
		setup  func()
	}{
		{
			name: "imp.banner_is_nil",
			args: args{
				imp: &openrtb2.Imp{
					Banner: nil,
				},
			},
			want: want{
				imp: &openrtb2.Imp{
					Banner: nil,
				},
			},
		},
		{
			name: "empty_adunitCfg",
			args: args{
				rCtx: models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"testImp": {
							BannerAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: nil,
							},
						},
					},
				},
				imp: &openrtb2.Imp{
					ID:     "testImp",
					Banner: &openrtb2.Banner{},
				},
			},
			want: want{
				imp: &openrtb2.Imp{
					ID:     "testImp",
					Banner: &openrtb2.Banner{},
				},
				rCtx: models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"testImp": {
							BannerAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: nil,
							},
						},
					},
				},
			},
		},
		{
			name: "imp.BidFloor_and_BidFloorCur_updated_from_adunit_config",
			args: args{
				rCtx: models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"testImp": {
							BannerAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
									BidFloor:    ptrutil.ToPtr(2.0),
									BidFloorCur: ptrutil.ToPtr("USD"),
								},
							},
						},
					},
				},
				imp: &openrtb2.Imp{
					ID:          "testImp",
					BidFloor:    0,
					BidFloorCur: "",
					Banner:      &openrtb2.Banner{},
				},
			},
			want: want{
				imp: &openrtb2.Imp{
					ID:          "testImp",
					Banner:      &openrtb2.Banner{},
					BidFloor:    2.0,
					BidFloorCur: "USD",
				},
				rCtx: models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"testImp": {
							BannerAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
									BidFloor:    ptrutil.ToPtr(2.0),
									BidFloorCur: ptrutil.ToPtr("USD"),
								},
							},
							BidFloor:    2,
							BidFloorCur: "USD",
						},
					},
				},
			},
		},
		{
			name: "imp.Exp_updated_from_adunit_config",
			args: args{
				rCtx: models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"testImp": {
							BannerAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
									Exp: ptrutil.ToPtr(10),
								},
							},
						},
					},
				},
				imp: &openrtb2.Imp{
					ID:     "testImp",
					Banner: &openrtb2.Banner{},
				},
			},
			want: want{
				imp: &openrtb2.Imp{
					ID:     "testImp",
					Banner: &openrtb2.Banner{},
					Exp:    10,
				},
				rCtx: models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"testImp": {
							BannerAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
									Exp: ptrutil.ToPtr(10),
								},
							},
						},
					},
				},
			},
		},
		{
			name: "imp_has_banner_object_but_adunitConfig_banner_is_nil._imp_banner_will_not_be_updated",
			args: args{
				rCtx: models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"testImp": {
							BannerAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
									Banner: nil,
								},
							},
						},
					},
				},
				imp: &openrtb2.Imp{
					ID: "testImp",
					Banner: &openrtb2.Banner{
						W: ptrutil.ToPtr[int64](200),
						H: ptrutil.ToPtr[int64](300),
					},
				},
			},
			want: want{
				imp: &openrtb2.Imp{
					ID: "testImp",
					Banner: &openrtb2.Banner{
						W: ptrutil.ToPtr[int64](200),
						H: ptrutil.ToPtr[int64](300),
					},
				},
				rCtx: models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"testImp": {
							BannerAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
									Banner: nil,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "imp_has_banner_object_but_banner_is_disabled_from_adunitConfig_then_remove_banner_object_from_imp",
			args: args{
				rCtx: models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"testImp": {
							BannerAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
									Banner: &adunitconfig.Banner{
										Enabled: ptrutil.ToPtr(false),
									},
								},
							},
						},
					},
				},
				imp: &openrtb2.Imp{
					ID: "testImp",
					Banner: &openrtb2.Banner{
						W: ptrutil.ToPtr[int64](200),
						H: ptrutil.ToPtr[int64](300),
					},
				},
			},
			want: want{
				imp: &openrtb2.Imp{
					ID:     "testImp",
					Banner: nil,
				},
				rCtx: models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"testImp": {
							BannerAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
									Banner: &adunitconfig.Banner{
										Enabled: ptrutil.ToPtr(false),
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "imp.BidFloor_less_than_BidFloor_from_adunit_config_applovinmax_setMaxFloor_true",
			args: args{
				rCtx: models.RequestCtx{
					Endpoint:           models.EndpointAppLovinMax,
					IsMaxFloorsEnabled: true,
					ImpBidCtx: map[string]models.ImpCtx{
						"testImp": {
							BannerAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
									BidFloor:    ptrutil.ToPtr(2.0),
									BidFloorCur: ptrutil.ToPtr("USD"),
								},
							},
						},
					},
				},
				imp: &openrtb2.Imp{
					ID:          "testImp",
					BidFloor:    1,
					BidFloorCur: "USD",
					Banner:      &openrtb2.Banner{},
				},
			},
			want: want{
				imp: &openrtb2.Imp{
					ID:          "testImp",
					Banner:      &openrtb2.Banner{},
					BidFloor:    2.0,
					BidFloorCur: "USD",
				},
				rCtx: models.RequestCtx{
					Endpoint:           models.EndpointAppLovinMax,
					IsMaxFloorsEnabled: true,
					ImpBidCtx: map[string]models.ImpCtx{
						"testImp": {
							BannerAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
									BidFloor:    ptrutil.ToPtr(2.0),
									BidFloorCur: ptrutil.ToPtr("USD"),
								},
							},
							BidFloor:    2,
							BidFloorCur: "USD",
						},
					},
				},
			},
		},
		{
			name: "imp.BidFloor_less_than_BidFloor_from_adunit_config_applovinmax_setMaxFloor_false",
			args: args{
				rCtx: models.RequestCtx{
					Endpoint: models.EndpointAppLovinMax,
					ImpBidCtx: map[string]models.ImpCtx{
						"testImp": {
							BannerAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
									BidFloor:    ptrutil.ToPtr(2.0),
									BidFloorCur: ptrutil.ToPtr("USD"),
								},
							},
						},
					},
				},
				imp: &openrtb2.Imp{
					ID:          "testImp",
					BidFloor:    1,
					BidFloorCur: "USD",
					Banner:      &openrtb2.Banner{},
				},
			},
			want: want{
				imp: &openrtb2.Imp{
					ID:          "testImp",
					Banner:      &openrtb2.Banner{},
					BidFloor:    1.0,
					BidFloorCur: "USD",
				},
				rCtx: models.RequestCtx{
					Endpoint: models.EndpointAppLovinMax,
					ImpBidCtx: map[string]models.ImpCtx{
						"testImp": {
							BannerAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
									BidFloor:    ptrutil.ToPtr(2.0),
									BidFloorCur: ptrutil.ToPtr("USD"),
								},
							},
							BidFloor:    1,
							BidFloorCur: "USD",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &OpenWrap{
				cfg:           tt.fields.cfg,
				cache:         tt.fields.cache,
				metricEngine:  tt.fields.metricEngine,
				pubFeatures:   mockFeature,
				rateConvertor: currency.NewRateConverter(&http.Client{}, 60*time.Second, "", time.Duration(0)),
			}
			if tt.setup != nil {
				tt.setup()
			}
			m.applyBannerAdUnitConfig(tt.args.rCtx, tt.args.imp)
			assert.Equal(t, tt.args.imp, tt.want.imp, "Imp banner is not upadted as expected from adunit config")
			assert.Equal(t, tt.args.rCtx, tt.want.rCtx, "rctx is not upadted as expected from adunit config")
		})
	}
}

func TestGetDomainFromUrl(t *testing.T) {
	type args struct {
		pageUrl string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "test_case_1",
			args: args{
				pageUrl: "http://ebay.com/inte/automation/s2s/pwt_parameter_validation_muti_slot_multi_size.html?pwtvc=1&pwtv=1&profileid=3277",
			},
			want: "ebay.com",
		},
		{
			name: "test_case_2",
			args: args{
				pageUrl: "http://ebay.co.in/inte/automation/s2s/pwt_parameter_validation_muti_slot_multi_size.html?pwtvc=1&pwtv=1&profileid=3277",
			},
			want: "ebay.co.in",
		},
		{
			name: "test_case_3",
			args: args{
				pageUrl: "site@sit.com",
			},
			want: "",
		},
		{
			name: "test_case_4",
			args: args{
				pageUrl: " 12 44",
			},
			want: "",
		},
		{
			name: "test_case_5",
			args: args{
				pageUrl: " ",
			},
			want: "",
		},
		{
			name: "test_case_6",
			args: args{
				pageUrl: "",
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getDomainFromUrl(tt.args.pageUrl); got != tt.want {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestUpdateRequestExtBidderParamsPubmatic(t *testing.T) {
	type args struct {
		bidderParams      json.RawMessage
		cookies           []string
		loggerID          string
		bidderCode        string
		sendBurl          bool
		sdkSubIntegration *int
	}
	tests := []struct {
		name    string
		args    args
		want    json.RawMessage
		wantErr bool
	}{
		{
			name: "empty_cookie",
			args: args{
				bidderParams: json.RawMessage(`{"pubmatic":{"pmzoneid":"zone1","adSlot":"38519891"}}`),
				loggerID:     "b441a46e-8c1f-428b-9c29-44e2a408a954",
				bidderCode:   "pubmatic",
			},
			want:    json.RawMessage(`{"pubmatic":{"wiid":"b441a46e-8c1f-428b-9c29-44e2a408a954"}}`),
			wantErr: false,
		},
		{
			name: "nil_cookies_same_as_empty_no_cookie_key",
			args: args{
				bidderParams: json.RawMessage(`{"pubmatic":{"pmzoneid":"zone1","adSlot":"38519891"}}`),
				cookies:      nil,
				loggerID:     "b441a46e-8c1f-428b-9c29-44e2a408a954",
				bidderCode:   "pubmatic",
			},
			want: json.RawMessage(`{"pubmatic":{"wiid":"b441a46e-8c1f-428b-9c29-44e2a408a954"}}`),
		},
		{
			name: "empty_loggerID",
			args: args{
				bidderParams: json.RawMessage(`{"pubmatic":{"pmzoneid":"zone1","adSlot":"38519891"}}`),
				cookies:      []string{"test_cookie"},
				bidderCode:   "pubmatic",
			},
			want: json.RawMessage(`{"pubmatic":{"Cookie":["test_cookie"],"wiid":""}}`),
		},
		{
			name: "both_cookie_and_loogerID_are_empty",
			args: args{
				bidderParams: json.RawMessage(`{"pubmatic":{"pmzoneid":"zone1","adSlot":"38519891"}}`),
				cookies:      []string{},
				loggerID:     "",
				bidderCode:   "pubmatic",
			},
			want: json.RawMessage(`{"pubmatic":{"wiid":""}}`),
		},
		{
			name: "both_cookie_and_loogerID_are_present",
			args: args{
				bidderParams: json.RawMessage(`{"pubmatic":{"pmzoneid":"zone1","adSlot":"38519891"}}`),
				cookies:      []string{"test_cookie"},
				loggerID:     "b441a46e-8c1f-428b-9c29-44e2a408a954",
				bidderCode:   "pubmatic",
			},
			want: json.RawMessage(`{"pubmatic":{"Cookie":["test_cookie"],"wiid":"b441a46e-8c1f-428b-9c29-44e2a408a954"}}`),
		},
		{
			name: "multiple_cookies_sets_cookie_slice_on_bidder",
			args: args{
				bidderParams: json.RawMessage(`{"pubmatic":{"pmzoneid":"zone1"}}`),
				cookies:      []string{"KADUSERCOOKIE=abc", "other=1"},
				loggerID:     "wid-1",
				bidderCode:   "pubmatic",
			},
			want: json.RawMessage(`{"pubmatic":{"Cookie":["KADUSERCOOKIE=abc","other=1"],"wiid":"wid-1"}}`),
		},
		{
			name: "cookies_applied_under_custom_bidder_code",
			args: args{
				bidderParams: json.RawMessage(`{"pubmatic_alias":{"legacy":true}}`),
				cookies:      []string{"session=abc"},
				loggerID:     "lid-2",
				bidderCode:   "pubmatic_alias",
			},
			want: json.RawMessage(`{"pubmatic_alias":{"Cookie":["session=abc"],"wiid":"lid-2"}}`),
		},
		{
			name: "sendburl is true and both cookie and loggerID are present",
			args: args{
				bidderParams: json.RawMessage(`{"pubmatic":{"pmzoneid":"zone1","adSlot":"38519891"}}`),
				cookies:      []string{"test_cookie"},
				loggerID:     "b441a46e-8c1f-428b-9c29-44e2a408a954",
				bidderCode:   "pubmatic",
				sendBurl:     true,
			},
			want: json.RawMessage(`{"pubmatic":{"Cookie":["test_cookie"],"sendburl":true,"wiid":"b441a46e-8c1f-428b-9c29-44e2a408a954"}}`),
		},
		{
			name: "sendburl is true and both cookie and loggerID are empty",
			args: args{
				bidderParams: json.RawMessage(`{"pubmatic":{"pmzoneid":"zone1","adSlot":"38519891"}}`),
				cookies:      []string{},
				loggerID:     "",
				bidderCode:   "pubmatic",
				sendBurl:     true,
			},
			want: json.RawMessage(`{"pubmatic":{"sendburl":true,"wiid":""}}`),
		},
		{
			name: "sendburl is false and both cookie and loggerID are present",
			args: args{
				bidderParams: json.RawMessage(`{"pubmatic":{"pmzoneid":"zone1","adSlot":"38519891"}}`),
				cookies:      []string{"test_cookie"},
				loggerID:     "b441a46e-8c1f-428b-9c29-44e2a408a954",
				bidderCode:   "pubmatic",
				sendBurl:     false,
			},
			want: json.RawMessage(`{"pubmatic":{"Cookie":["test_cookie"],"wiid":"b441a46e-8c1f-428b-9c29-44e2a408a954"}}`),
		},
		{
			name: "sdksubintegration_on_pubmatic_bidderparams_alongside_wiid",
			args: args{
				bidderParams:      json.RawMessage(`{"pubmatic":{"pmzoneid":"zone1"}}`),
				loggerID:          "wid-sdk",
				bidderCode:        "pubmatic",
				sdkSubIntegration: ptrutil.ToPtr(16),
			},
			want: json.RawMessage(`{"pubmatic":{"wiid":"wid-sdk","wrapper":{"sdksubintegration":16}}}`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := updateRequestExtBidderParamsPubmatic(tt.args.bidderParams, tt.args.cookies, tt.args.loggerID, tt.args.bidderCode, tt.args.sendBurl, tt.args.sdkSubIntegration)
			if (err != nil) != tt.wantErr {
				assert.Equal(t, tt.wantErr, err != nil)
				return
			}
			assert.JSONEq(t, string(tt.want), string(got))
		})
	}
}

func TestOpenWrapHandleBeforeValidationHook(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockCache := mock_cache.NewMockCache(ctrl)
	mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)
	mockFeature := mock_feature.NewMockFeature(ctrl)
	mockProfileMetaData := mock_profilemetadata.NewMockProfileMetaData(ctrl)
	adapters.InitBidders("./static/bidder-params/")
	resetFakeUUID := openrtb_ext.SetTestFakeUUIDGenerator("30470a14-2949-4110-abce-b62d57304ad5")
	defer resetFakeUUID()

	type fields struct {
		cfg          config.Config
		cache        cache.Cache
		metricEngine metrics.MetricsEngine
	}
	type args struct {
		ctx        context.Context
		moduleCtx  hookstage.ModuleInvocationContext
		payload    hookstage.BeforeValidationRequestPayload
		bidrequest json.RawMessage
	}
	type want struct {
		hookResult            hookstage.HookResult[hookstage.BeforeValidationRequestPayload]
		bidRequest            json.RawMessage
		error                 bool
		nilCurrencyConversion bool
		doMutate              bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
		setup  func()
	}{
		{
			name: "request_with_sshb=1",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							Sshb: "1",
						},
					},
				},
				bidrequest: json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","banner":{"format":[{"w":728,"h":90},{"w":300,"h":250}],"w":700,"h":900},"video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"tagid":"adunit","bidfloor":4.3,"bidfloorcur":"USD","ext":{"bidder":{"pubmatic":{"keywords":[{"key":"pmzoneid","value":["val1","val2"]}]}},"prebid":{}}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}},"ext":{"prebid":{},"wrapper":{"test":123,"profileid":123,"versionid":1,"wiid":"test_display_wiid"}}}`),
			},
			want: want{
				hookResult: hookstage.HookResult[hookstage.BeforeValidationRequestPayload]{
					Reject: false,
				},
				error:                 false,
				nilCurrencyConversion: true,
			},
		},
		{
			name: "empty_module_context",
			args: args{
				ctx:        context.Background(),
				moduleCtx:  hookstage.ModuleInvocationContext{},
				bidrequest: json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","banner":{"format":[{"w":728,"h":90},{"w":300,"h":250}],"w":700,"h":900},"video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"tagid":"adunit","bidfloor":4.3,"bidfloorcur":"USD","ext":{"bidder":{"pubmatic":{"keywords":[{"key":"pmzoneid","value":["val1","val2"]}]}},"prebid":{}}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}},"ext":{"prebid":{},"wrapper":{"test":123,"profileid":123,"versionid":1,"wiid":"test_display_wiid"}}}`),
			},
			fields: fields{
				cache:        mockCache,
				metricEngine: mockEngine,
			},
			want: want{
				hookResult: hookstage.HookResult[hookstage.BeforeValidationRequestPayload]{
					Reject:        true,
					DebugMessages: []string{"error: module-ctx not found in handleBeforeValidationHook()"},
				},
				error:                 false,
				nilCurrencyConversion: true,
			},
		},
		{
			name: "rctx_is_not_present",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"test_rctx": "test",
					},
				},
				bidrequest: json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","banner":{"format":[{"w":728,"h":90},{"w":300,"h":250}],"w":700,"h":900},"video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"tagid":"adunit","bidfloor":4.3,"bidfloorcur":"USD","ext":{"bidder":{"pubmatic":{"keywords":[{"key":"pmzoneid","value":["val1","val2"]}]}},"prebid":{}}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}},"ext":{"prebid":{},"wrapper":{"test":123,"profileid":123,"versionid":1,"wiid":"test_display_wiid"}}}`),
			},
			fields: fields{
				cache:        mockCache,
				metricEngine: mockEngine,
			},
			want: want{
				hookResult: hookstage.HookResult[hookstage.BeforeValidationRequestPayload]{
					Reject:        true,
					DebugMessages: []string{"error: request-ctx not found in handleBeforeValidationHook()"},
				},
				error:                 false,
				nilCurrencyConversion: true,
			},
		},
		{
			name: "hybrid_request_module_should_not_reject_request_and_return_without_executing_module",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							Endpoint: models.EndpointHybrid,
						},
					},
				},
				bidrequest: json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","banner":{"format":[{"w":728,"h":90},{"w":300,"h":250}],"w":700,"h":900},"video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"tagid":"adunit","bidfloor":4.3,"bidfloorcur":"USD","ext":{"bidder":{"pubmatic":{"keywords":[{"key":"pmzoneid","value":["val1","val2"]}]}},"prebid":{}}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}},"ext":{"prebid":{},"wrapper":{"test":123,"profileid":123,"versionid":1,"wiid":"test_display_wiid"}}}`),
			},
			want: want{
				hookResult: hookstage.HookResult[hookstage.BeforeValidationRequestPayload]{
					Reject: false,
				},
				error:                 false,
				nilCurrencyConversion: true,
			},
		},
		{
			name: "Invalid_request_ext",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": rctx,
					},
				},
				bidrequest: json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","banner":{"format":[{"w":728,"h":90},{"w":300,"h":250}],"w":700,"h":900},"video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"tagid":"adunit","ext":{"wrapper":{"div":"div"},"bidder":{"pubmatic":{"keywords":[{"key":"pmzoneid","value":["val1","val2"]}]}},"prebid":{}}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}},"ext":1}`),
			},
			fields: fields{
				cache:        mockCache,
				metricEngine: mockEngine,
			},
			setup: func() {
				mockEngine.EXPECT().RecordPublisherProfileRequests("5890", "1234")
				mockEngine.EXPECT().RecordBadRequests(rctx.Endpoint, "5890", getPubmaticErrorCode(nbr.InvalidRequestExt))
				mockEngine.EXPECT().RecordNobidErrPrebidServerRequests("5890", int(nbr.InvalidRequestExt))
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false)
			},
			want: want{
				hookResult: hookstage.HookResult[hookstage.BeforeValidationRequestPayload]{
					Reject:  true,
					NbrCode: int(nbr.InvalidRequestExt),
					Errors:  []string{"failed to get request ext: failed to decode request.ext : json: cannot unmarshal number into Go value of type models.RequestExt"},
				},
				error:                 false,
				nilCurrencyConversion: true,
			},
		},
		{
			name: "Error_in_getting_profile_data",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": rctx,
					},
				},
				bidrequest: json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","banner":{"format":[{"w":728,"h":90},{"w":300,"h":250}],"w":700,"h":900},"video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"tagid":"adunit","ext":{"wrapper":{"div":"div"},"bidder":{"pubmatic":{"keywords":[{"key":"pmzoneid","value":["val1","val2"]}]}},"prebid":{}}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}},"ext":{"prebid":{},"wrapper":{"test":123,"profileid":123,"versionid":1,"wiid":"test_display_wiid"}}}`),
			},
			fields: fields{
				cache:        mockCache,
				metricEngine: mockEngine,
			},
			setup: func() {
				mockCache.EXPECT().GetPartnerConfigMap(gomock.Any(), gomock.Any(), gomock.Any()).Return(map[int]map[string]string{
					2: {
						models.PARTNER_ID:          "2",
						models.PREBID_PARTNER_NAME: "appnexus",
						models.BidderCode:          "appnexus",
						models.SERVER_SIDE_FLAG:    "1",
						models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
						models.TIMEOUT:             "200",
					},
					-1: {
						models.PLATFORM_KEY:     models.PLATFORM_APP,
						models.DisplayVersionID: "1",
					},
				}, errors.New("test"))
				//prometheus metrics
				mockEngine.EXPECT().RecordPublisherProfileRequests("5890", "1234")
				mockEngine.EXPECT().RecordBadRequests(rctx.Endpoint, rctx.PubIDStr, getPubmaticErrorCode(nbr.InvalidProfileConfiguration))
				mockEngine.EXPECT().RecordNobidErrPrebidServerRequests("5890", int(nbr.InvalidProfileConfiguration))
				mockEngine.EXPECT().RecordPublisherInvalidProfileRequests(rctx.Endpoint, "5890", rctx.ProfileIDStr)
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false)
				mockFeature.EXPECT().IsAnalyticsTrackingThrottled(gomock.Any(), gomock.Any()).Return(false, false)
			},
			want: want{
				hookResult: hookstage.HookResult[hookstage.BeforeValidationRequestPayload]{
					Reject:  true,
					NbrCode: int(nbr.InvalidProfileConfiguration),
					Errors:  []string{"failed to get profile data: test"},
				},
				error:                 true,
				nilCurrencyConversion: true,
			},
		},
		{
			name: "got_empty_profileData_from_DB",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": rctx,
					},
				},
				bidrequest: json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","banner":{"format":[{"w":728,"h":90},{"w":300,"h":250}],"w":700,"h":900},"video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"tagid":"adunit","ext":{"wrapper":{"div":"div"},"bidder":{"pubmatic":{"keywords":[{"key":"pmzoneid","value":["val1","val2"]}]}},"prebid":{}}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}},"ext":{"prebid":{},"wrapper":{"test":123,"profileid":123,"versionid":1,"wiid":"test_display_wiid"}}}`),
			},
			fields: fields{
				cache:        mockCache,
				metricEngine: mockEngine,
			},
			setup: func() {
				mockCache.EXPECT().GetPartnerConfigMap(gomock.Any(), gomock.Any(), gomock.Any()).Return(map[int]map[string]string{}, nil)
				//prometheus metrics
				mockEngine.EXPECT().RecordPublisherProfileRequests("5890", "1234")
				mockEngine.EXPECT().RecordBadRequests(rctx.Endpoint, rctx.PubIDStr, getPubmaticErrorCode(nbr.InvalidProfileConfiguration))
				mockEngine.EXPECT().RecordNobidErrPrebidServerRequests("5890", int(nbr.InvalidProfileConfiguration))
				mockEngine.EXPECT().RecordPublisherInvalidProfileRequests(rctx.Endpoint, "5890", rctx.ProfileIDStr)
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false)
				mockFeature.EXPECT().IsAnalyticsTrackingThrottled(gomock.Any(), gomock.Any()).Return(false, false)
			},
			want: want{
				hookResult: hookstage.HookResult[hookstage.BeforeValidationRequestPayload]{
					Reject:  true,
					NbrCode: int(nbr.InvalidProfileConfiguration),
					Errors:  []string{"invalid profile data"},
				},
				error:                 true,
				nilCurrencyConversion: true,
			},
		},
		{
			name: "platform_is_not_present_in_request_then_reject_the_request",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": rctx,
					},
				},
				bidrequest: json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","banner":{"format":[{"w":728,"h":90},{"w":300,"h":250}],"w":700,"h":900},"video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"tagid":"adunit","ext":{"wrapper":{"div":"div"},"bidder":{"pubmatic":{"keywords":[{"key":"pmzoneid","value":["val1","val2"]}]}},"prebid":{}}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}},"ext":{"prebid":{},"wrapper":{"test":123,"profileid":123,"versionid":1,"wiid":"test_display_wiid"}}}`),
			},
			fields: fields{
				cache:        mockCache,
				metricEngine: mockEngine,
			},
			setup: func() {
				mockCache.EXPECT().GetPartnerConfigMap(gomock.Any(), gomock.Any(), gomock.Any()).Return(map[int]map[string]string{
					2: {
						models.PARTNER_ID:          "2",
						models.PREBID_PARTNER_NAME: "appnexus",
						models.BidderCode:          "appnexus",
						models.SERVER_SIDE_FLAG:    "1",
						models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
						models.TIMEOUT:             "200",
					},
					-1: {
						models.DisplayVersionID: "1",
					},
				}, nil)
				//prometheus metrics
				mockEngine.EXPECT().RecordPublisherProfileRequests("5890", "1234")
				mockEngine.EXPECT().RecordBadRequests(rctx.Endpoint, rctx.PubIDStr, getPubmaticErrorCode(nbr.InvalidPlatform))
				mockEngine.EXPECT().RecordNobidErrPrebidServerRequests("5890", int(nbr.InvalidPlatform))
				mockEngine.EXPECT().RecordPublisherInvalidProfileRequests(rctx.Endpoint, "5890", rctx.ProfileIDStr)
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false)
				mockFeature.EXPECT().IsAnalyticsTrackingThrottled(gomock.Any(), gomock.Any()).Return(false, false)
			},
			want: want{
				hookResult: hookstage.HookResult[hookstage.BeforeValidationRequestPayload]{
					Reject:  true,
					NbrCode: int(nbr.InvalidPlatform),
					Errors:  []string{"failed to get platform data"},
				},
				error:                 true,
				nilCurrencyConversion: true,
			},
		},
		{
			name: "All_partners_throttled",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": rctx,
					},
				},
				bidrequest: json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","banner":{"format":[{"w":728,"h":90},{"w":300,"h":250}],"w":700,"h":900},"video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"tagid":"","ext":{"wrapper":{"div":"div"},"bidder":{"pubmatic":{"keywords":[{"key":"pmzoneid","value":["val1","val2"]}]}},"prebid":{}}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}},"ext":{"prebid":{},"wrapper":{"test":123,"profileid":123,"versionid":1,"wiid":"test_display_wiid"}}}`),
			},
			fields: fields{
				cache:        mockCache,
				metricEngine: mockEngine,
			},
			setup: func() {
				mockCache.EXPECT().GetPartnerConfigMap(gomock.Any(), gomock.Any(), gomock.Any()).Return(map[int]map[string]string{
					2: {
						models.PARTNER_ID:          "2",
						models.PREBID_PARTNER_NAME: "appnexus",
						models.BidderCode:          "appnexus",
						models.SERVER_SIDE_FLAG:    "1",
						models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
						models.TIMEOUT:             "200",
						models.THROTTLE:            "0",
					},
					-1: {
						models.DisplayVersionID: "1",
						models.PLATFORM_KEY:     models.PLATFORM_APP,
					},
				}, nil)
				mockCache.EXPECT().GetThrottlePartnersWithCriteria(gomock.Any()).Return(map[string]struct{}{"appnexus": {}}, nil)
				//prometheus metrics
				mockEngine.EXPECT().RecordPublisherProfileRequests("5890", "1234")
				mockEngine.EXPECT().RecordPartnerThrottledRequests("5890", "appnexus", models.PartnerLevelThrottlingFeatureID)
				mockEngine.EXPECT().RecordCountryLevelPartnerThrottledRequests(rctx.Endpoint, "appnexus", "")
				mockEngine.EXPECT().RecordBadRequests(rctx.Endpoint, rctx.PubIDStr, getPubmaticErrorCode(nbr.RequestBlockedGeoFiltered))
				mockEngine.EXPECT().RecordNobidErrPrebidServerRequests("5890", int(nbr.RequestBlockedGeoFiltered))
				mockEngine.EXPECT().RecordPublisherRequests(rctx.Endpoint, "5890", rctx.Platform)
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false)
				mockFeature.EXPECT().IsAnalyticsTrackingThrottled(gomock.Any(), gomock.Any()).Return(false, false)
				mockProfileMetaData.EXPECT().GetProfileTypePlatform(gomock.Any()).Return(0, false)
			},
			want: want{
				hookResult: hookstage.HookResult[hookstage.BeforeValidationRequestPayload]{
					Reject:  true,
					NbrCode: int(nbr.RequestBlockedGeoFiltered),
					Errors:  []string{"All adapters throttled"},
				},
				error:                 false,
				nilCurrencyConversion: true,
			},
		},
		{
			name: "Some_partners_filtered",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": rctx,
					},
				},
				bidrequest: json.RawMessage(`{"device":{"geo":{"country":"IN"}},"id":"123-456-789","imp":[{"id":"123","banner":{"format":[{"w":728,"h":90},{"w":300,"h":250}],"w":700,"h":900},"video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"tagid":"","ext":{"wrapper":{"div":"div"},"bidder":{"pubmatic":{"keywords":[{"key":"pmzoneid","value":["val1","val2"]}]}},"prebid":{}}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}},"ext":{"prebid":{},"wrapper":{"test":123,"profileid":123,"versionid":1,"wiid":"test_display_wiid"}}}`),
			},
			fields: fields{
				cache:        mockCache,
				metricEngine: mockEngine,
			},
			setup: func() {
				mockCache.EXPECT().GetPartnerConfigMap(gomock.Any(), gomock.Any(), gomock.Any()).Return(map[int]map[string]string{
					2: {
						models.PARTNER_ID:          "2",
						models.PREBID_PARTNER_NAME: "appnexus",
						models.BidderCode:          "appnexus",
						models.SERVER_SIDE_FLAG:    "1",
						models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
						models.TIMEOUT:             "200",
						models.THROTTLE:            "100",
						models.BidderFilters:       `{ "in": [{ "var": "country"}, ["USA"]]}`,
					},
					3: {
						models.PARTNER_ID:          "3",
						models.PREBID_PARTNER_NAME: "pubmatic",
						models.BidderCode:          "pubmatic",
						models.SERVER_SIDE_FLAG:    "1",
						models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
						models.TIMEOUT:             "200",
						models.THROTTLE:            "100",
					},
					-1: {
						models.DisplayVersionID: "1",
						models.PLATFORM_KEY:     models.PLATFORM_APP,
					},
				}, nil)
				mockCache.EXPECT().GetAdunitConfigFromCache(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&adunitconfig.AdUnitConfig{})
				mockCache.EXPECT().GetThrottlePartnersWithCriteria(gomock.Any()).Return(map[string]struct{}{}, nil)
				//prometheus metrics
				mockEngine.EXPECT().RecordPublisherProfileRequests("5890", "1234")
				mockEngine.EXPECT().RecordBadRequests(rctx.Endpoint, rctx.PubIDStr, getPubmaticErrorCode(nbr.InvalidImpressionTagID))
				mockEngine.EXPECT().RecordNobidErrPrebidServerRequests("5890", int(nbr.InvalidImpressionTagID))
				mockEngine.EXPECT().RecordPublisherRequests(rctx.Endpoint, "5890", rctx.Platform)
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false)
				mockFeature.EXPECT().IsAnalyticsTrackingThrottled(gomock.Any(), gomock.Any()).Return(false, false)
				mockProfileMetaData.EXPECT().GetProfileTypePlatform(gomock.Any()).Return(0, false)
			},
			want: want{
				hookResult: hookstage.HookResult[hookstage.BeforeValidationRequestPayload]{
					Reject:     true,
					NbrCode:    int(nbr.InvalidImpressionTagID),
					Errors:     []string{"tagid missing for imp: 123"},
					SeatNonBid: getNonBids(map[string][]openrtb_ext.NonBidParams{"appnexus": {{Bid: &openrtb2.Bid{ImpID: "123"}, NonBidReason: int(nbr.RequestBlockedPartnerFiltered)}}}),
				},
				error: true,
			},
		},
		{
			name: "All_partners_filtered",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": rctx,
					},
				},
				bidrequest: json.RawMessage(`{"device":{"geo":{"country":"in"}},"id":"123-456-789","imp":[{"id":"123","banner":{"format":[{"w":728,"h":90},{"w":300,"h":250}],"w":700,"h":900},"video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"tagid":"","ext":{"wrapper":{"div":"div"},"bidder":{"pubmatic":{"keywords":[{"key":"pmzoneid","value":["val1","val2"]}]}},"prebid":{}}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}},"ext":{"prebid":{},"wrapper":{"test":123,"profileid":123,"versionid":1,"wiid":"test_display_wiid"}}}`),
			},
			fields: fields{
				cache:        mockCache,
				metricEngine: mockEngine,
			},
			setup: func() {
				mockCache.EXPECT().GetPartnerConfigMap(gomock.Any(), gomock.Any(), gomock.Any()).Return(map[int]map[string]string{
					2: {
						models.PARTNER_ID:          "2",
						models.PREBID_PARTNER_NAME: "appnexus",
						models.BidderCode:          "appnexus",
						models.SERVER_SIDE_FLAG:    "1",
						models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
						models.TIMEOUT:             "200",
						models.THROTTLE:            "100",
						models.BidderFilters:       `{ "in": [{ "var": "country"}, ["USA"]]}`,
					},
					-1: {
						models.DisplayVersionID: "1",
						models.PLATFORM_KEY:     models.PLATFORM_APP,
					},
				}, nil)
				mockProfileMetaData.EXPECT().GetProfileTypePlatform(gomock.Any()).Return(0, false)
				mockCache.EXPECT().GetThrottlePartnersWithCriteria(gomock.Any()).Return(map[string]struct{}{}, nil)
				//prometheus metrics
				mockEngine.EXPECT().RecordPublisherProfileRequests("5890", "1234")
				mockEngine.EXPECT().RecordBadRequests(rctx.Endpoint, rctx.PubIDStr, getPubmaticErrorCode(nbr.AllPartnersFiltered))
				mockEngine.EXPECT().RecordNobidErrPrebidServerRequests("5890", int(nbr.AllPartnersFiltered))
				mockEngine.EXPECT().RecordPublisherRequests(rctx.Endpoint, "5890", rctx.Platform)
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false)
				mockFeature.EXPECT().IsAnalyticsTrackingThrottled(gomock.Any(), gomock.Any()).Return(false, false)
			},
			want: want{
				hookResult: hookstage.HookResult[hookstage.BeforeValidationRequestPayload]{
					Reject:     true,
					NbrCode:    int(nbr.AllPartnersFiltered),
					Errors:     []string{"All partners filtered"},
					SeatNonBid: getNonBids(map[string][]openrtb_ext.NonBidParams{"appnexus": {{Bid: &openrtb2.Bid{ImpID: "123"}, NonBidReason: int(nbr.RequestBlockedPartnerFiltered)}}}),
				},
				error:                 false,
				nilCurrencyConversion: true,
			},
		},
		{
			name: "TagID_not_present_in_imp",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": rctx,
					},
				},
				bidrequest: json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","banner":{"format":[{"w":728,"h":90},{"w":300,"h":250}],"w":700,"h":900},"video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"ext":{"wrapper":{"div":"div"},"bidder":{"pubmatic":{"keywords":[{"key":"pmzoneid","value":["val1","val2"]}]}},"prebid":{}}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}},"ext":{"prebid":{},"wrapper":{"test":123,"profileid":123,"versionid":1,"wiid":"test_display_wiid"}}}`),
			},
			fields: fields{
				cache:        mockCache,
				metricEngine: mockEngine,
			},
			setup: func() {
				mockCache.EXPECT().GetPartnerConfigMap(gomock.Any(), gomock.Any(), gomock.Any()).Return(map[int]map[string]string{
					2: {
						models.PARTNER_ID:          "2",
						models.PREBID_PARTNER_NAME: "appnexus",
						models.BidderCode:          "appnexus",
						models.SERVER_SIDE_FLAG:    "1",
						models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
						models.TIMEOUT:             "200",
					},
					-1: {
						models.DisplayVersionID: "1",
						models.PLATFORM_KEY:     models.PLATFORM_APP,
					},
				}, nil)
				mockCache.EXPECT().GetAdunitConfigFromCache(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&adunitconfig.AdUnitConfig{})
				mockCache.EXPECT().GetThrottlePartnersWithCriteria(gomock.Any()).Return(map[string]struct{}{}, nil)
				//prometheus metrics
				mockEngine.EXPECT().RecordPublisherProfileRequests("5890", "1234")
				mockEngine.EXPECT().RecordBadRequests(rctx.Endpoint, rctx.PubIDStr, getPubmaticErrorCode(nbr.InvalidImpressionTagID))
				mockEngine.EXPECT().RecordNobidErrPrebidServerRequests("5890", int(nbr.InvalidImpressionTagID))
				mockEngine.EXPECT().RecordPublisherRequests(rctx.Endpoint, "5890", rctx.Platform)
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false)
				mockFeature.EXPECT().IsAnalyticsTrackingThrottled(gomock.Any(), gomock.Any()).Return(false, false)
				mockProfileMetaData.EXPECT().GetProfileTypePlatform(gomock.Any()).Return(0, false)
			},
			want: want{
				hookResult: hookstage.HookResult[hookstage.BeforeValidationRequestPayload]{
					Reject:  true,
					NbrCode: int(nbr.InvalidImpressionTagID),
					Errors:  []string{"tagid missing for imp: 123"},
				},
				error:                 true,
				nilCurrencyConversion: false,
			},
		},
		{
			name: "TagID_not_present_in_imp_and_not_found_for_client_request",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": func() models.RequestCtx {
							testRctx := rctx
							testRctx.Endpoint = models.EndpointWebS2S
							return testRctx
						}(),
					},
				},
				bidrequest: json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","banner":{"format":[{"w":728,"h":90},{"w":300,"h":250}],"w":700,"h":900},"video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"ext":{"wrapper":{"div":"div"},"bidder":{"pubmatic":{"keywords":[{"key":"pmzoneid","value":["val1","val2"]}]}},"prebid":{}}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}},"ext":{"prebid":{},"wrapper":{"test":123,"profileid":123,"versionid":1,"wiid":"test_display_wiid"}}}`),
			},
			fields: fields{
				cache:        mockCache,
				metricEngine: mockEngine,
			},
			setup: func() {
				mockCache.EXPECT().GetPartnerConfigMap(gomock.Any(), gomock.Any(), gomock.Any()).Return(map[int]map[string]string{
					2: {
						models.PARTNER_ID:          "2",
						models.PREBID_PARTNER_NAME: "appnexus",
						models.BidderCode:          "appnexus",
						models.SERVER_SIDE_FLAG:    "1",
						models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
						models.TIMEOUT:             "200",
					},
					-1: {
						models.DisplayVersionID: "1",
						models.PLATFORM_KEY:     models.PLATFORM_APP,
					},
				}, nil)
				mockCache.EXPECT().GetAdunitConfigFromCache(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&adunitconfig.AdUnitConfig{})
				mockCache.EXPECT().GetThrottlePartnersWithCriteria(gomock.Any()).Return(map[string]struct{}{}, nil)

				//prometheus metrics
				mockEngine.EXPECT().RecordPublisherProfileRequests("5890", "1234")
				mockEngine.EXPECT().RecordBadRequests(models.EndpointWebS2S, "5890", getPubmaticErrorCode(nbr.InvalidImpressionTagID))
				mockEngine.EXPECT().RecordNobidErrPrebidServerRequests("5890", int(nbr.InvalidImpressionTagID))
				mockEngine.EXPECT().RecordPublisherRequests(models.EndpointWebS2S, "5890", rctx.Platform)
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false)
				mockFeature.EXPECT().IsAnalyticsTrackingThrottled(gomock.Any(), gomock.Any()).Return(false, false)
				mockProfileMetaData.EXPECT().GetProfileTypePlatform(gomock.Any()).Return(0, false)
			},
			want: want{
				hookResult: hookstage.HookResult[hookstage.BeforeValidationRequestPayload]{
					Reject:  true,
					NbrCode: int(nbr.InvalidImpressionTagID),
					Errors:  []string{"tagid missing for imp: 123"},
				},
				error:                 true,
				nilCurrencyConversion: false,
			},
		},
		{
			name: "invalid_impExt",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": rctx,
					},
				},
				bidrequest: json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","banner":{"format":[{"w":728,"h":90},{"w":300,"h":250}],"w":700,"h":900},"video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"tagid":"adunit","ext":"1"}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}},"ext":{"prebid":{},"wrapper":{"test":123,"profileid":123,"versionid":1,"wiid":"test_display_wiid"}}}`),
			},
			fields: fields{
				cache:        mockCache,
				metricEngine: mockEngine,
			},
			setup: func() {
				mockCache.EXPECT().GetPartnerConfigMap(gomock.Any(), gomock.Any(), gomock.Any()).Return(map[int]map[string]string{
					2: {
						models.PARTNER_ID:          "2",
						models.PREBID_PARTNER_NAME: "appnexus",
						models.BidderCode:          "appnexus",
						models.SERVER_SIDE_FLAG:    "1",
						models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
						models.TIMEOUT:             "200",
					},
					-1: {
						models.DisplayVersionID: "1",
						models.PLATFORM_KEY:     models.PLATFORM_APP,
					},
				}, nil)
				mockCache.EXPECT().GetAdunitConfigFromCache(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&adunitconfig.AdUnitConfig{})
				mockCache.EXPECT().GetThrottlePartnersWithCriteria(gomock.Any()).Return(map[string]struct{}{}, nil)
				//prometheus metrics
				mockEngine.EXPECT().RecordPublisherProfileRequests("5890", "1234")
				mockEngine.EXPECT().RecordBadRequests(rctx.Endpoint, rctx.PubIDStr, getPubmaticErrorCode(openrtb3.NoBidInvalidRequest))
				mockEngine.EXPECT().RecordNobidErrPrebidServerRequests("5890", int(openrtb3.NoBidInvalidRequest))
				mockEngine.EXPECT().RecordPublisherRequests(rctx.Endpoint, "5890", rctx.Platform)
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false)
				mockFeature.EXPECT().IsAnalyticsTrackingThrottled(gomock.Any(), gomock.Any()).Return(false, false)
				mockProfileMetaData.EXPECT().GetProfileTypePlatform(gomock.Any()).Return(0, false)
			},
			want: want{
				hookResult: hookstage.HookResult[hookstage.BeforeValidationRequestPayload]{
					Reject:  true,
					NbrCode: int(openrtb3.NoBidInvalidRequest),
					Errors:  []string{"failed to parse imp.ext: 123"},
				},
				error:                 true,
				nilCurrencyConversion: false,
			},
		},
		{
			name: "allSotsDisabled-native-not-present",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							PubIDStr:                  "5890",
							PubID:                     5890,
							ProfileID:                 1234,
							DisplayID:                 1,
							SSAuction:                 -1,
							Platform:                  "in-app",
							Debug:                     true,
							DeviceCtx:                 models.DeviceCtx{IP: "127.0.0.1", UA: "go-test"},
							IsCTVRequest:              false,
							TrackerEndpoint:           "t.pubmatic.com",
							VideoErrorTrackerEndpoint: "t.pubmatic.com/error",
							UidCookie: &http.Cookie{
								Name:  "uids",
								Value: `eyJ0ZW1wVUlEcyI6eyIzM2Fjcm9zcyI6eyJ1aWQiOiIxMTkxNzkxMDk5Nzc2NjEiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OTo0My4zODg4Nzk5NVoifSwiYWRmIjp7InVpZCI6IjgwNDQ2MDgzMzM3Nzg4MzkwNzgiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxMS4wMzMwNTQ3MjdaIn0sImFka2VybmVsIjp7InVpZCI6IkE5MTYzNTAwNzE0OTkyOTMyOTkwIiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MDkuMzczMzg1NjYyWiJ9LCJhZGtlcm5lbEFkbiI6eyJ1aWQiOiJBOTE2MzUwMDcxNDk5MjkzMjk5MCIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjEzLjQzNDkyNTg5NloifSwiYWRtaXhlciI6eyJ1aWQiOiIzNjZhMTdiMTJmMjI0ZDMwOGYzZTNiOGRhOGMzYzhhMCIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjA5LjU5MjkxNDgwMVoifSwiYWRueHMiOnsidWlkIjoiNDE5Mjg5ODUzMDE0NTExOTMiLCJleHBpcmVzIjoiMjAyMy0wMS0xOFQwOTo1MzowOC44MjU0NDI2NzZaIn0sImFqYSI6eyJ1aWQiOiJzMnN1aWQ2RGVmMFl0bjJveGQ1aG9zS1AxVmV3IiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MTMuMjM5MTc2MDU0WiJ9LCJlcGxhbm5pbmciOnsidWlkIjoiQUoxRjBTOE5qdTdTQ0xWOSIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjEwLjkyOTk2MDQ3M1oifSwiZ2Ftb3NoaSI6eyJ1aWQiOiJndXNyXzM1NmFmOWIxZDhjNjQyYjQ4MmNiYWQyYjdhMjg4MTYxIiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MTAuNTI0MTU3MjI1WiJ9LCJncmlkIjp7InVpZCI6IjRmYzM2MjUwLWQ4NTItNDU5Yy04NzcyLTczNTZkZTE3YWI5NyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjE0LjY5NjMxNjIyN1oifSwiZ3JvdXBtIjp7InVpZCI6IjdENzVEMjVGLUZBQzktNDQzRC1CMkQxLUIxN0ZFRTExRTAyNyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjM5LjIyNjIxMjUzMloifSwiaXgiOnsidWlkIjoiWW9ORlNENlc5QkphOEh6eEdtcXlCUUFBXHUwMDI2Mjk3IiwiZXhwaXJlcyI6IjIwMjMtMDUtMzFUMDc6NTM6MzguNTU1ODI3MzU0WiJ9LCJqaXhpZSI6eyJ1aWQiOiI3MzY3MTI1MC1lODgyLTExZWMtYjUzOC0xM2FjYjdhZjBkZTQiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxMi4xOTEwOTk3MzJaIn0sImxvZ2ljYWQiOnsidWlkIjoiQVZ4OVROQS11c25pa3M4QURzTHpWa3JvaDg4QUFBR0JUREh0UUEiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OTowOS40NTUxNDk2MTZaIn0sIm1lZGlhbmV0Ijp7InVpZCI6IjI5Nzg0MjM0OTI4OTU0MTAwMDBWMTAiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxMy42NzIyMTUxMjhaIn0sIm1naWQiOnsidWlkIjoibTU5Z1hyN0xlX1htIiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MTcuMDk3MDAxNDcxWiJ9LCJuYW5vaW50ZXJhY3RpdmUiOnsidWlkIjoiNmFlYzhjMTAzNzlkY2I3ODQxMmJjODBiNmRkOWM5NzMxNzNhYjdkNzEyZTQzMWE1YTVlYTcwMzRlNTZhNThhMCIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjE2LjcxNDgwNzUwNVoifSwib25ldGFnIjp7InVpZCI6IjdPelZoVzFOeC1LOGFVak1HMG52NXVNYm5YNEFHUXZQbnVHcHFrZ3k0ckEiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OTowOS4xNDE3NDEyNjJaIn0sIm9wZW54Ijp7InVpZCI6IjVkZWNlNjIyLTBhMjMtMGRhYi0zYTI0LTVhNzcwMTBlNDU4MiIsImV4cGlyZXMiOiIyMDIzLTA1LTMxVDA3OjUyOjQ3LjE0MDQxNzM2M1oifSwicHVibWF0aWMiOnsidWlkIjoiN0Q3NUQyNUYtRkFDOS00NDNELUIyRDEtQjE3RkVFMTFFMDI3IiwiZXhwaXJlcyI6IjIwMjItMTAtMzFUMDk6MTQ6MjUuNzM3MjU2ODk5WiJ9LCJyaWNoYXVkaWVuY2UiOnsidWlkIjoiY2I2YzYzMjAtMzNlMi00Nzc0LWIxNjAtMXp6MTY1NDg0MDc0OSIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjA5LjUyNTA3NDE4WiJ9LCJzbWFydHlhZHMiOnsidWlkIjoiMTJhZjE1ZTQ0ZjAwZDA3NjMwZTc0YzQ5MTU0Y2JmYmE0Zjg0N2U4ZDRhMTU0YzhjM2Q1MWY1OGNmNzJhNDYyNyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjEwLjgyNTAzMTg4NFoifSwic21pbGV3YW50ZWQiOnsidWlkIjoiZGQ5YzNmZTE4N2VmOWIwOWNhYTViNzExNDA0YzI4MzAiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxNC4yNTU2MDkzNjNaIn0sInN5bmFjb3JtZWRpYSI6eyJ1aWQiOiJHRFBSIiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MDkuOTc5NTgzNDM4WiJ9LCJ0cmlwbGVsaWZ0Ijp7InVpZCI6IjcwMjE5NzUwNTQ4MDg4NjUxOTQ2MyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjA4Ljk4OTY3MzU3NFoifSwidmFsdWVpbXByZXNzaW9uIjp7InVpZCI6IjlkMDgxNTVmLWQ5ZmUtNGI1OC04OThlLWUyYzU2MjgyYWIzZSIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjA5LjA2NzgzOTE2NFoifSwidmlzeCI6eyJ1aWQiOiIyN2UwYWMzYy1iNDZlLTQxYjMtOTkyYy1mOGQyNzE0OTQ5NWUiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxMi45ODk1MjM1NzNaIn0sInlpZWxkbGFiIjp7InVpZCI6IjY5NzE0ZDlmLWZiMDAtNGE1Zi04MTljLTRiZTE5MTM2YTMyNSIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjExLjMwMzAyNjYxNVoifSwieWllbGRtbyI6eyJ1aWQiOiJnOTZjMmY3MTlmMTU1MWIzMWY2MyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjEwLjExMDUyODYwOVoifSwieWllbGRvbmUiOnsidWlkIjoiMmE0MmZiZDMtMmM3MC00ZWI5LWIxYmQtMDQ2OTY2NTBkOTQ4IiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MTAuMzE4MzMzOTM5WiJ9LCJ6ZXJvY2xpY2tmcmF1ZCI6eyJ1aWQiOiJiOTk5NThmZS0yYTg3LTJkYTQtOWNjNC05NjFmZDExM2JlY2UiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxNS43MTk1OTQ1NjZaIn19LCJiZGF5IjoiMjAyMi0wNS0xN1QwNjo0ODozOC4wMTc5ODgyMDZaIn0=`,
							},
							KADUSERCookie: &http.Cookie{
								Name:  "KADUSERCOOKIE",
								Value: `7D75D25F-FAC9-443D-B2D1-B17FEE11E027`,
							},
							OriginCookie:             "go-test",
							Aliases:                  make(map[string]string),
							ImpBidCtx:                make(map[string]models.ImpCtx),
							PrebidBidderCode:         make(map[string]string),
							BidderResponseTimeMillis: make(map[string]int),
							ProfileIDStr:             "1234",
							Endpoint:                 models.EndpointV25,
							SeatNonBids:              make(map[string][]openrtb_ext.NonBid),
							MetricsEngine:            mockEngine,
							WakandaDebug:             &wakanda.Debug{},
						},
					},
				},
				bidrequest: json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","banner":{"format":[{"w":728,"h":90},{"w":300,"h":250}],"w":700,"h":900},"video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"tagid":"adunit","ext":{"bidder":{"pubmatic":{"keywords":[{"key":"pmzoneid","value":["val1","val2"]}]}},"prebid":{}}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}},"ext":{"prebid":{},"wrapper":{"test":123,"profileid":123,"versionid":1,"wiid":"test_display_wiid"}}}`),
			},
			fields: fields{
				cache:        mockCache,
				metricEngine: mockEngine,
			},
			setup: func() {
				mockCache.EXPECT().GetPartnerConfigMap(gomock.Any(), gomock.Any(), gomock.Any()).Return(map[int]map[string]string{
					2: {
						models.PARTNER_ID:          "2",
						models.PREBID_PARTNER_NAME: "appnexus",
						models.BidderCode:          "appnexus",
						models.SERVER_SIDE_FLAG:    "1",
						models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
						models.TIMEOUT:             "200",
					},
					-1: {
						models.DisplayVersionID: "1",
						models.PLATFORM_KEY:     models.PLATFORM_APP,
					},
				}, nil)
				mockCache.EXPECT().GetAdunitConfigFromCache(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&adunitconfig.AdUnitConfig{
					ConfigPattern: "_AU_@_W_x_H_",
					Config: map[string]*adunitconfig.AdConfig{
						"adunit@700x900": {
							Banner: &adunitconfig.Banner{
								Enabled: ptrutil.ToPtr(false),
							},
						},
						"adunit@640x480": {
							Video: &adunitconfig.Video{
								Enabled: ptrutil.ToPtr(false),
							},
						},
					},
				})
				mockCache.EXPECT().GetThrottlePartnersWithCriteria(gomock.Any()).Return(map[string]struct{}{}, nil)
				//prometheus metrics
				mockEngine.EXPECT().RecordPublisherProfileRequests("5890", "1234")
				mockEngine.EXPECT().RecordBadRequests(rctx.Endpoint, rctx.PubIDStr, getPubmaticErrorCode(nbr.AllSlotsDisabled))
				mockEngine.EXPECT().RecordNobidErrPrebidServerRequests("5890", int(nbr.AllSlotsDisabled))
				mockEngine.EXPECT().RecordPublisherRequests(rctx.Endpoint, "5890", rctx.Platform)
				mockEngine.EXPECT().RecordImpDisabledViaConfigStats(models.ImpTypeVideo, "5890", "1234")
				mockEngine.EXPECT().RecordImpDisabledViaConfigStats(models.ImpTypeBanner, "5890", "1234")
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false)
				mockFeature.EXPECT().IsAnalyticsTrackingThrottled(gomock.Any(), gomock.Any()).Return(false, false)
				mockProfileMetaData.EXPECT().GetProfileTypePlatform(gomock.Any()).Return(0, false)
			},
			want: want{
				hookResult: hookstage.HookResult[hookstage.BeforeValidationRequestPayload]{
					Reject:  true,
					NbrCode: int(nbr.AllSlotsDisabled),
					Errors:  []string{"All slots disabled"},
				},
				error:                 false,
				nilCurrencyConversion: false,
			},
		},
		{
			name: "allSotsDisabled-native-present",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							PubIDStr:                  "5890",
							PubID:                     5890,
							ProfileID:                 1234,
							DisplayID:                 1,
							SSAuction:                 -1,
							Platform:                  "in-app",
							Debug:                     true,
							DeviceCtx:                 models.DeviceCtx{UA: "go-test", IP: "127.0.0.1"},
							IsCTVRequest:              false,
							TrackerEndpoint:           "t.pubmatic.com",
							VideoErrorTrackerEndpoint: "t.pubmatic.com/error",
							UidCookie: &http.Cookie{
								Name:  "uids",
								Value: `eyJ0ZW1wVUlEcyI6eyIzM2Fjcm9zcyI6eyJ1aWQiOiIxMTkxNzkxMDk5Nzc2NjEiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OTo0My4zODg4Nzk5NVoifSwiYWRmIjp7InVpZCI6IjgwNDQ2MDgzMzM3Nzg4MzkwNzgiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxMS4wMzMwNTQ3MjdaIn0sImFka2VybmVsIjp7InVpZCI6IkE5MTYzNTAwNzE0OTkyOTMyOTkwIiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MDkuMzczMzg1NjYyWiJ9LCJhZGtlcm5lbEFkbiI6eyJ1aWQiOiJBOTE2MzUwMDcxNDk5MjkzMjk5MCIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjEzLjQzNDkyNTg5NloifSwiYWRtaXhlciI6eyJ1aWQiOiIzNjZhMTdiMTJmMjI0ZDMwOGYzZTNiOGRhOGMzYzhhMCIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjA5LjU5MjkxNDgwMVoifSwiYWRueHMiOnsidWlkIjoiNDE5Mjg5ODUzMDE0NTExOTMiLCJleHBpcmVzIjoiMjAyMy0wMS0xOFQwOTo1MzowOC44MjU0NDI2NzZaIn0sImFqYSI6eyJ1aWQiOiJzMnN1aWQ2RGVmMFl0bjJveGQ1aG9zS1AxVmV3IiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MTMuMjM5MTc2MDU0WiJ9LCJlcGxhbm5pbmciOnsidWlkIjoiQUoxRjBTOE5qdTdTQ0xWOSIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjEwLjkyOTk2MDQ3M1oifSwiZ2Ftb3NoaSI6eyJ1aWQiOiJndXNyXzM1NmFmOWIxZDhjNjQyYjQ4MmNiYWQyYjdhMjg4MTYxIiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MTAuNTI0MTU3MjI1WiJ9LCJncmlkIjp7InVpZCI6IjRmYzM2MjUwLWQ4NTItNDU5Yy04NzcyLTczNTZkZTE3YWI5NyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjE0LjY5NjMxNjIyN1oifSwiZ3JvdXBtIjp7InVpZCI6IjdENzVEMjVGLUZBQzktNDQzRC1CMkQxLUIxN0ZFRTExRTAyNyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjM5LjIyNjIxMjUzMloifSwiaXgiOnsidWlkIjoiWW9ORlNENlc5QkphOEh6eEdtcXlCUUFBXHUwMDI2Mjk3IiwiZXhwaXJlcyI6IjIwMjMtMDUtMzFUMDc6NTM6MzguNTU1ODI3MzU0WiJ9LCJqaXhpZSI6eyJ1aWQiOiI3MzY3MTI1MC1lODgyLTExZWMtYjUzOC0xM2FjYjdhZjBkZTQiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxMi4xOTEwOTk3MzJaIn0sImxvZ2ljYWQiOnsidWlkIjoiQVZ4OVROQS11c25pa3M4QURzTHpWa3JvaDg4QUFBR0JUREh0UUEiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OTowOS40NTUxNDk2MTZaIn0sIm1lZGlhbmV0Ijp7InVpZCI6IjI5Nzg0MjM0OTI4OTU0MTAwMDBWMTAiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxMy42NzIyMTUxMjhaIn0sIm1naWQiOnsidWlkIjoibTU5Z1hyN0xlX1htIiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MTcuMDk3MDAxNDcxWiJ9LCJuYW5vaW50ZXJhY3RpdmUiOnsidWlkIjoiNmFlYzhjMTAzNzlkY2I3ODQxMmJjODBiNmRkOWM5NzMxNzNhYjdkNzEyZTQzMWE1YTVlYTcwMzRlNTZhNThhMCIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjE2LjcxNDgwNzUwNVoifSwib25ldGFnIjp7InVpZCI6IjdPelZoVzFOeC1LOGFVak1HMG52NXVNYm5YNEFHUXZQbnVHcHFrZ3k0ckEiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OTowOS4xNDE3NDEyNjJaIn0sIm9wZW54Ijp7InVpZCI6IjVkZWNlNjIyLTBhMjMtMGRhYi0zYTI0LTVhNzcwMTBlNDU4MiIsImV4cGlyZXMiOiIyMDIzLTA1LTMxVDA3OjUyOjQ3LjE0MDQxNzM2M1oifSwicHVibWF0aWMiOnsidWlkIjoiN0Q3NUQyNUYtRkFDOS00NDNELUIyRDEtQjE3RkVFMTFFMDI3IiwiZXhwaXJlcyI6IjIwMjItMTAtMzFUMDk6MTQ6MjUuNzM3MjU2ODk5WiJ9LCJyaWNoYXVkaWVuY2UiOnsidWlkIjoiY2I2YzYzMjAtMzNlMi00Nzc0LWIxNjAtMXp6MTY1NDg0MDc0OSIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjA5LjUyNTA3NDE4WiJ9LCJzbWFydHlhZHMiOnsidWlkIjoiMTJhZjE1ZTQ0ZjAwZDA3NjMwZTc0YzQ5MTU0Y2JmYmE0Zjg0N2U4ZDRhMTU0YzhjM2Q1MWY1OGNmNzJhNDYyNyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjEwLjgyNTAzMTg4NFoifSwic21pbGV3YW50ZWQiOnsidWlkIjoiZGQ5YzNmZTE4N2VmOWIwOWNhYTViNzExNDA0YzI4MzAiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxNC4yNTU2MDkzNjNaIn0sInN5bmFjb3JtZWRpYSI6eyJ1aWQiOiJHRFBSIiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MDkuOTc5NTgzNDM4WiJ9LCJ0cmlwbGVsaWZ0Ijp7InVpZCI6IjcwMjE5NzUwNTQ4MDg4NjUxOTQ2MyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjA4Ljk4OTY3MzU3NFoifSwidmFsdWVpbXByZXNzaW9uIjp7InVpZCI6IjlkMDgxNTVmLWQ5ZmUtNGI1OC04OThlLWUyYzU2MjgyYWIzZSIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjA5LjA2NzgzOTE2NFoifSwidmlzeCI6eyJ1aWQiOiIyN2UwYWMzYy1iNDZlLTQxYjMtOTkyYy1mOGQyNzE0OTQ5NWUiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxMi45ODk1MjM1NzNaIn0sInlpZWxkbGFiIjp7InVpZCI6IjY5NzE0ZDlmLWZiMDAtNGE1Zi04MTljLTRiZTE5MTM2YTMyNSIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjExLjMwMzAyNjYxNVoifSwieWllbGRtbyI6eyJ1aWQiOiJnOTZjMmY3MTlmMTU1MWIzMWY2MyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjEwLjExMDUyODYwOVoifSwieWllbGRvbmUiOnsidWlkIjoiMmE0MmZiZDMtMmM3MC00ZWI5LWIxYmQtMDQ2OTY2NTBkOTQ4IiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MTAuMzE4MzMzOTM5WiJ9LCJ6ZXJvY2xpY2tmcmF1ZCI6eyJ1aWQiOiJiOTk5NThmZS0yYTg3LTJkYTQtOWNjNC05NjFmZDExM2JlY2UiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxNS43MTk1OTQ1NjZaIn19LCJiZGF5IjoiMjAyMi0wNS0xN1QwNjo0ODozOC4wMTc5ODgyMDZaIn0=`,
							},
							KADUSERCookie: &http.Cookie{
								Name:  "KADUSERCOOKIE",
								Value: `7D75D25F-FAC9-443D-B2D1-B17FEE11E027`,
							},
							OriginCookie:             "go-test",
							Aliases:                  make(map[string]string),
							ImpBidCtx:                make(map[string]models.ImpCtx),
							PrebidBidderCode:         make(map[string]string),
							BidderResponseTimeMillis: make(map[string]int),
							ProfileIDStr:             "1234",
							Endpoint:                 models.EndpointV25,
							SeatNonBids:              make(map[string][]openrtb_ext.NonBid),
							MetricsEngine:            mockEngine,
							WakandaDebug:             &wakanda.Debug{},
						},
					},
				},
				bidrequest: json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","native": {},"banner":{"format":[{"w":728,"h":90},{"w":300,"h":250}],"w":700,"h":900},"video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"tagid":"adunit","ext":{"bidder":{"pubmatic":{"keywords":[{"key":"pmzoneid","value":["val1","val2"]}]}},"prebid":{}}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}},"ext":{"prebid":{},"wrapper":{"test":123,"profileid":123,"versionid":1,"wiid":"test_display_wiid"}}}`),
			},
			fields: fields{
				cache:        mockCache,
				metricEngine: mockEngine,
			},
			setup: func() {
				mockCache.EXPECT().GetPartnerConfigMap(gomock.Any(), gomock.Any(), gomock.Any()).Return(map[int]map[string]string{
					2: {
						models.PARTNER_ID:          "2",
						models.PREBID_PARTNER_NAME: "appnexus",
						models.BidderCode:          "appnexus",
						models.SERVER_SIDE_FLAG:    "1",
						models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
						models.TIMEOUT:             "200",
					},
					-1: {
						models.DisplayVersionID: "1",
						models.PLATFORM_KEY:     models.PLATFORM_APP,
					},
				}, nil)
				mockCache.EXPECT().GetAdunitConfigFromCache(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&adunitconfig.AdUnitConfig{
					ConfigPattern: "_AU_@_W_x_H_",
					Config: map[string]*adunitconfig.AdConfig{
						"adunit@700x900": {
							Banner: &adunitconfig.Banner{
								Enabled: ptrutil.ToPtr(false),
							},
						},
						"adunit@640x480": {
							Video: &adunitconfig.Video{
								Enabled: ptrutil.ToPtr(false),
							},
						},
					},
				})
				mockCache.EXPECT().GetMappingsFromCacheV25(gomock.Any(), gomock.Any()).Return(map[string]models.SlotMapping{
					"adunit@700x900": {
						SlotName: "adunit@700x900",
						SlotMappings: map[string]interface{}{
							models.SITE_CACHE_KEY: "12313",
							models.TAG_CACHE_KEY:  "45343",
						},
					},
				})
				mockCache.EXPECT().GetSlotToHashValueMapFromCacheV25(gomock.Any(), gomock.Any()).Return(models.SlotMappingInfo{
					OrderedSlotList: []string{"adunit@700x900"},
					HashValueMap: map[string]string{
						"adunit@700x900": "1232433543534543",
					},
				})
				mockCache.EXPECT().GetThrottlePartnersWithCriteria(gomock.Any()).Return(map[string]struct{}{}, nil)
				//prometheus metrics
				mockEngine.EXPECT().RecordPublisherProfileRequests("5890", "1234")
				mockEngine.EXPECT().RecordPublisherRequests(rctx.Endpoint, "5890", rctx.Platform)
				mockEngine.EXPECT().RecordPlatformPublisherPartnerReqStats(rctx.Platform, "5890", "appnexus")
				mockEngine.EXPECT().RecordImpDisabledViaConfigStats(models.ImpTypeVideo, "5890", "1234")
				mockEngine.EXPECT().RecordImpDisabledViaConfigStats(models.ImpTypeBanner, "5890", "1234")
				mockEngine.EXPECT().RecordPreProcessingTimeStats(rctx.PubIDStr, gomock.Any())
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false)
				mockFeature.EXPECT().IsAnalyticsTrackingThrottled(gomock.Any(), gomock.Any()).Return(false, false)
				mockProfileMetaData.EXPECT().GetProfileTypePlatform(gomock.Any()).Return(0, false)
				mockFeature.EXPECT().IsDynamicFloorEnabledPublisher(gomock.Any()).Return(false)
			},
			want: want{
				hookResult: hookstage.HookResult[hookstage.BeforeValidationRequestPayload]{
					Reject:        false,
					NbrCode:       0,
					Message:       "",
					ChangeSet:     hookstage.ChangeSet[hookstage.BeforeValidationRequestPayload]{},
					DebugMessages: []string{`new imp: {"123":{"ImpID":"123","TagID":"adunit","DisplayManager":"","DisplayManagerVer":"","Div":"","SlotName":"adunit","AdUnitName":"adunit","Secure":0,"BidFloor":0,"BidFloorCur":"","IsRewardInventory":null,"Banner":true,"Video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"Native":{"request":""},"IncomingSlots":["700x900","728x90","300x250"],"Type":"video","Bidders":{"appnexus":{"PartnerID":2,"PrebidBidderCode":"appnexus","MatchedSlot":"adunit@700x900","KGP":"_AU_@_W_x_H_","KGPV":"","IsRegex":false,"Params":{"placementId":0,"site":"12313","adtag":"45343"},"VASTTagFlags":null}},"NonMapped":{},"NewExt":{"data":{"pbadslot":"adunit"},"prebid":{"bidder":{"appnexus":{"placementId":0,"site":"12313","adtag":"45343"}}}},"BidCtx":{},"BannerAdUnitCtx":{"MatchedSlot":"adunit@700x900","IsRegex":false,"MatchedRegex":"","SelectedSlotAdUnitConfig":{"banner":{"enabled":false}},"AppliedSlotAdUnitConfig":{"banner":{"enabled":false}},"UsingDefaultConfig":false,"AllowedConnectionTypes":null},"VideoAdUnitCtx":{"MatchedSlot":"adunit@640x480","IsRegex":false,"MatchedRegex":"","SelectedSlotAdUnitConfig":{"video":{"enabled":false}},"AppliedSlotAdUnitConfig":{"video":{"enabled":false}},"UsingDefaultConfig":false,"AllowedConnectionTypes":null},"NativeAdUnitCtx":{"MatchedSlot":"","IsRegex":false,"MatchedRegex":"","SelectedSlotAdUnitConfig":null,"AppliedSlotAdUnitConfig":null,"UsingDefaultConfig":false,"AllowedConnectionTypes":null},"BidderError":"","IsAdPodRequest":false,"AdpodConfig":null,"ImpAdPodCfg":null,"BidIDToAPRC":null,"AdserverURL":"","BidIDToDur":null}}`, `new request.ext: {"prebid":{"bidadjustmentfactors":{"appnexus":1},"bidderparams":{"pubmatic":{"wiid":""}},"debug":true,"floors":{"enforcement":{"enforcepbs":true,"floordeals":true},"enabled":true},"targeting":{"pricegranularity":{"precision":2,"ranges":[{"min":0,"max":5,"increment":0.05},{"min":5,"max":10,"increment":0.1},{"min":10,"max":20,"increment":0.5}]},"mediatypepricegranularity":{},"includewinners":true,"includebidderkeys":true},"macros":{"[PLATFORM]":"3","[PROFILE_ID]":"1234","[PROFILE_VERSION]":"1","[UNIX_TIMESTAMP]":"0","[WRAPPER_IMPRESSION_ID]":""}}}`},
					AnalyticsTags: hookanalytics.Analytics{Activities: nil},
				},
				error:                 false,
				doMutate:              true,
				nilCurrencyConversion: false,
				bidRequest:            json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","native":{"request":""},"tagid":"adunit","ext":{"data":{"pbadslot":"adunit"},"prebid":{"bidder":{"appnexus":{"placementId":0,"site":"12313","adtag":"45343"}}}}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","customdata":"7D75D25F-FAC9-443D-B2D1-B17FEE11E027","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"tid":"123-456-789","ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}},"ext":{"prebid":{"bidadjustmentfactors":{"appnexus":1},"bidderparams":{"pubmatic":{"wiid":""}},"debug":true,"floors":{"enforcement":{"enforcepbs":true,"floordeals":true},"enabled":true},"targeting":{"pricegranularity":{"precision":2,"ranges":[{"min":0,"max":5,"increment":0.05},{"min":5,"max":10,"increment":0.1},{"min":10,"max":20,"increment":0.5}]},"includewinners":true,"includebidderkeys":true},"macros":{"[PLATFORM]":"3","[PROFILE_ID]":"1234","[PROFILE_VERSION]":"1","[UNIX_TIMESTAMP]":"0","[WRAPPER_IMPRESSION_ID]":""}}}}`),
			},
		},
		{
			name: "no_serviceSideBidderPresent",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							PubIDStr:                  "5890",
							PubID:                     5890,
							ProfileID:                 1234,
							DisplayID:                 1,
							SSAuction:                 -1,
							Platform:                  "in-app",
							Debug:                     true,
							DeviceCtx:                 models.DeviceCtx{IP: "127.0.0.1", UA: "go-test"},
							IsCTVRequest:              false,
							TrackerEndpoint:           "t.pubmatic.com",
							VideoErrorTrackerEndpoint: "t.pubmatic.com/error",
							UidCookie: &http.Cookie{
								Name:  "uids",
								Value: `eyJ0ZW1wVUlEcyI6eyIzM2Fjcm9zcyI6eyJ1aWQiOiIxMTkxNzkxMDk5Nzc2NjEiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OTo0My4zODg4Nzk5NVoifSwiYWRmIjp7InVpZCI6IjgwNDQ2MDgzMzM3Nzg4MzkwNzgiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxMS4wMzMwNTQ3MjdaIn0sImFka2VybmVsIjp7InVpZCI6IkE5MTYzNTAwNzE0OTkyOTMyOTkwIiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MDkuMzczMzg1NjYyWiJ9LCJhZGtlcm5lbEFkbiI6eyJ1aWQiOiJBOTE2MzUwMDcxNDk5MjkzMjk5MCIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjEzLjQzNDkyNTg5NloifSwiYWRtaXhlciI6eyJ1aWQiOiIzNjZhMTdiMTJmMjI0ZDMwOGYzZTNiOGRhOGMzYzhhMCIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjA5LjU5MjkxNDgwMVoifSwiYWRueHMiOnsidWlkIjoiNDE5Mjg5ODUzMDE0NTExOTMiLCJleHBpcmVzIjoiMjAyMy0wMS0xOFQwOTo1MzowOC44MjU0NDI2NzZaIn0sImFqYSI6eyJ1aWQiOiJzMnN1aWQ2RGVmMFl0bjJveGQ1aG9zS1AxVmV3IiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MTMuMjM5MTc2MDU0WiJ9LCJlcGxhbm5pbmciOnsidWlkIjoiQUoxRjBTOE5qdTdTQ0xWOSIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjEwLjkyOTk2MDQ3M1oifSwiZ2Ftb3NoaSI6eyJ1aWQiOiJndXNyXzM1NmFmOWIxZDhjNjQyYjQ4MmNiYWQyYjdhMjg4MTYxIiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MTAuNTI0MTU3MjI1WiJ9LCJncmlkIjp7InVpZCI6IjRmYzM2MjUwLWQ4NTItNDU5Yy04NzcyLTczNTZkZTE3YWI5NyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjE0LjY5NjMxNjIyN1oifSwiZ3JvdXBtIjp7InVpZCI6IjdENzVEMjVGLUZBQzktNDQzRC1CMkQxLUIxN0ZFRTExRTAyNyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjM5LjIyNjIxMjUzMloifSwiaXgiOnsidWlkIjoiWW9ORlNENlc5QkphOEh6eEdtcXlCUUFBXHUwMDI2Mjk3IiwiZXhwaXJlcyI6IjIwMjMtMDUtMzFUMDc6NTM6MzguNTU1ODI3MzU0WiJ9LCJqaXhpZSI6eyJ1aWQiOiI3MzY3MTI1MC1lODgyLTExZWMtYjUzOC0xM2FjYjdhZjBkZTQiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxMi4xOTEwOTk3MzJaIn0sImxvZ2ljYWQiOnsidWlkIjoiQVZ4OVROQS11c25pa3M4QURzTHpWa3JvaDg4QUFBR0JUREh0UUEiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OTowOS40NTUxNDk2MTZaIn0sIm1lZGlhbmV0Ijp7InVpZCI6IjI5Nzg0MjM0OTI4OTU0MTAwMDBWMTAiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxMy42NzIyMTUxMjhaIn0sIm1naWQiOnsidWlkIjoibTU5Z1hyN0xlX1htIiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MTcuMDk3MDAxNDcxWiJ9LCJuYW5vaW50ZXJhY3RpdmUiOnsidWlkIjoiNmFlYzhjMTAzNzlkY2I3ODQxMmJjODBiNmRkOWM5NzMxNzNhYjdkNzEyZTQzMWE1YTVlYTcwMzRlNTZhNThhMCIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjE2LjcxNDgwNzUwNVoifSwib25ldGFnIjp7InVpZCI6IjdPelZoVzFOeC1LOGFVak1HMG52NXVNYm5YNEFHUXZQbnVHcHFrZ3k0ckEiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OTowOS4xNDE3NDEyNjJaIn0sIm9wZW54Ijp7InVpZCI6IjVkZWNlNjIyLTBhMjMtMGRhYi0zYTI0LTVhNzcwMTBlNDU4MiIsImV4cGlyZXMiOiIyMDIzLTA1LTMxVDA3OjUyOjQ3LjE0MDQxNzM2M1oifSwicHVibWF0aWMiOnsidWlkIjoiN0Q3NUQyNUYtRkFDOS00NDNELUIyRDEtQjE3RkVFMTFFMDI3IiwiZXhwaXJlcyI6IjIwMjItMTAtMzFUMDk6MTQ6MjUuNzM3MjU2ODk5WiJ9LCJyaWNoYXVkaWVuY2UiOnsidWlkIjoiY2I2YzYzMjAtMzNlMi00Nzc0LWIxNjAtMXp6MTY1NDg0MDc0OSIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjA5LjUyNTA3NDE4WiJ9LCJzbWFydHlhZHMiOnsidWlkIjoiMTJhZjE1ZTQ0ZjAwZDA3NjMwZTc0YzQ5MTU0Y2JmYmE0Zjg0N2U4ZDRhMTU0YzhjM2Q1MWY1OGNmNzJhNDYyNyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjEwLjgyNTAzMTg4NFoifSwic21pbGV3YW50ZWQiOnsidWlkIjoiZGQ5YzNmZTE4N2VmOWIwOWNhYTViNzExNDA0YzI4MzAiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxNC4yNTU2MDkzNjNaIn0sInN5bmFjb3JtZWRpYSI6eyJ1aWQiOiJHRFBSIiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MDkuOTc5NTgzNDM4WiJ9LCJ0cmlwbGVsaWZ0Ijp7InVpZCI6IjcwMjE5NzUwNTQ4MDg4NjUxOTQ2MyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjA4Ljk4OTY3MzU3NFoifSwidmFsdWVpbXByZXNzaW9uIjp7InVpZCI6IjlkMDgxNTVmLWQ5ZmUtNGI1OC04OThlLWUyYzU2MjgyYWIzZSIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjA5LjA2NzgzOTE2NFoifSwidmlzeCI6eyJ1aWQiOiIyN2UwYWMzYy1iNDZlLTQxYjMtOTkyYy1mOGQyNzE0OTQ5NWUiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxMi45ODk1MjM1NzNaIn0sInlpZWxkbGFiIjp7InVpZCI6IjY5NzE0ZDlmLWZiMDAtNGE1Zi04MTljLTRiZTE5MTM2YTMyNSIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjExLjMwMzAyNjYxNVoifSwieWllbGRtbyI6eyJ1aWQiOiJnOTZjMmY3MTlmMTU1MWIzMWY2MyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjEwLjExMDUyODYwOVoifSwieWllbGRvbmUiOnsidWlkIjoiMmE0MmZiZDMtMmM3MC00ZWI5LWIxYmQtMDQ2OTY2NTBkOTQ4IiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MTAuMzE4MzMzOTM5WiJ9LCJ6ZXJvY2xpY2tmcmF1ZCI6eyJ1aWQiOiJiOTk5NThmZS0yYTg3LTJkYTQtOWNjNC05NjFmZDExM2JlY2UiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxNS43MTk1OTQ1NjZaIn19LCJiZGF5IjoiMjAyMi0wNS0xN1QwNjo0ODozOC4wMTc5ODgyMDZaIn0=`,
							},
							KADUSERCookie: &http.Cookie{
								Name:  "KADUSERCOOKIE",
								Value: `7D75D25F-FAC9-443D-B2D1-B17FEE11E027`,
							},
							OriginCookie:             "go-test",
							Aliases:                  make(map[string]string),
							ImpBidCtx:                make(map[string]models.ImpCtx),
							PrebidBidderCode:         make(map[string]string),
							BidderResponseTimeMillis: make(map[string]int),
							ProfileIDStr:             "1234",
							Endpoint:                 models.EndpointV25,
							SeatNonBids:              make(map[string][]openrtb_ext.NonBid),
							MetricsEngine:            mockEngine,
							WakandaDebug:             &wakanda.Debug{},
						},
					},
				},
				bidrequest: json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","banner":{"format":[{"w":728,"h":90},{"w":300,"h":250}],"w":700,"h":900},"video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"tagid":"adunit","ext":{"bidder":{"pubmatic":{"keywords":[{"key":"pmzoneid","value":["val1","val2"]}]}},"prebid":{}}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}},"ext":{"prebid":{},"wrapper":{"test":123,"profileid":123,"versionid":1,"wiid":"test_display_wiid"}}}`),
			},
			fields: fields{
				cache:        mockCache,
				metricEngine: mockEngine,
			},
			setup: func() {
				mockCache.EXPECT().GetPartnerConfigMap(gomock.Any(), gomock.Any(), gomock.Any()).Return(map[int]map[string]string{
					2: {
						models.PREBID_PARTNER_NAME: "appnexus",
						models.BidderCode:          "appnexus",
						models.SERVER_SIDE_FLAG:    "1",
						models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
						models.TIMEOUT:             "200",
					},
					-1: {
						models.DisplayVersionID: "1",
						models.PLATFORM_KEY:     models.PLATFORM_APP,
					},
				}, nil)
				mockCache.EXPECT().GetAdunitConfigFromCache(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&adunitconfig.AdUnitConfig{})
				mockCache.EXPECT().GetThrottlePartnersWithCriteria(gomock.Any()).Return(map[string]struct{}{}, nil)
				//prometheus metrics
				mockEngine.EXPECT().RecordPublisherProfileRequests("5890", "1234")
				mockEngine.EXPECT().RecordBadRequests(rctx.Endpoint, rctx.PubIDStr, getPubmaticErrorCode(nbr.ServerSidePartnerNotConfigured))
				mockEngine.EXPECT().RecordNobidErrPrebidServerRequests("5890", int(nbr.ServerSidePartnerNotConfigured))
				mockEngine.EXPECT().RecordPublisherRequests(rctx.Endpoint, "5890", rctx.Platform)
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false)
				mockFeature.EXPECT().IsAnalyticsTrackingThrottled(gomock.Any(), gomock.Any()).Return(false, false)
				mockProfileMetaData.EXPECT().GetProfileTypePlatform(gomock.Any()).Return(0, false)
				mockFeature.EXPECT().IsDynamicFloorEnabledPublisher(gomock.Any()).Return(false)
			},
			want: want{
				hookResult: hookstage.HookResult[hookstage.BeforeValidationRequestPayload]{
					Reject:  true,
					NbrCode: int(nbr.ServerSidePartnerNotConfigured),
					Errors:  []string{"server side partner not found"},
				},
				error:                 false,
				nilCurrencyConversion: false,
			},
		},
		{
			name: "if_partner_is_alias_update_req.ext.prebid.aliasgvlid",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							PubIDStr:                  "5890",
							PubID:                     5890,
							ProfileID:                 1234,
							DisplayID:                 1,
							SSAuction:                 -1,
							Platform:                  "in-app",
							Debug:                     true,
							DeviceCtx:                 models.DeviceCtx{IP: "127.0.0.1", UA: "go-test"},
							IsCTVRequest:              false,
							TrackerEndpoint:           "t.pubmatic.com",
							VideoErrorTrackerEndpoint: "t.pubmatic.com/error",
							UidCookie: &http.Cookie{
								Name:  "uids",
								Value: `eyJ0ZW1wVUlEcyI6eyIzM2Fjcm9zcyI6eyJ1aWQiOiIxMTkxNzkxMDk5Nzc2NjEiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OTo0My4zODg4Nzk5NVoifSwiYWRmIjp7InVpZCI6IjgwNDQ2MDgzMzM3Nzg4MzkwNzgiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxMS4wMzMwNTQ3MjdaIn0sImFka2VybmVsIjp7InVpZCI6IkE5MTYzNTAwNzE0OTkyOTMyOTkwIiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MDkuMzczMzg1NjYyWiJ9LCJhZGtlcm5lbEFkbiI6eyJ1aWQiOiJBOTE2MzUwMDcxNDk5MjkzMjk5MCIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjEzLjQzNDkyNTg5NloifSwiYWRtaXhlciI6eyJ1aWQiOiIzNjZhMTdiMTJmMjI0ZDMwOGYzZTNiOGRhOGMzYzhhMCIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjA5LjU5MjkxNDgwMVoifSwiYWRueHMiOnsidWlkIjoiNDE5Mjg5ODUzMDE0NTExOTMiLCJleHBpcmVzIjoiMjAyMy0wMS0xOFQwOTo1MzowOC44MjU0NDI2NzZaIn0sImFqYSI6eyJ1aWQiOiJzMnN1aWQ2RGVmMFl0bjJveGQ1aG9zS1AxVmV3IiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MTMuMjM5MTc2MDU0WiJ9LCJlcGxhbm5pbmciOnsidWlkIjoiQUoxRjBTOE5qdTdTQ0xWOSIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjEwLjkyOTk2MDQ3M1oifSwiZ2Ftb3NoaSI6eyJ1aWQiOiJndXNyXzM1NmFmOWIxZDhjNjQyYjQ4MmNiYWQyYjdhMjg4MTYxIiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MTAuNTI0MTU3MjI1WiJ9LCJncmlkIjp7InVpZCI6IjRmYzM2MjUwLWQ4NTItNDU5Yy04NzcyLTczNTZkZTE3YWI5NyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjE0LjY5NjMxNjIyN1oifSwiZ3JvdXBtIjp7InVpZCI6IjdENzVEMjVGLUZBQzktNDQzRC1CMkQxLUIxN0ZFRTExRTAyNyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjM5LjIyNjIxMjUzMloifSwiaXgiOnsidWlkIjoiWW9ORlNENlc5QkphOEh6eEdtcXlCUUFBXHUwMDI2Mjk3IiwiZXhwaXJlcyI6IjIwMjMtMDUtMzFUMDc6NTM6MzguNTU1ODI3MzU0WiJ9LCJqaXhpZSI6eyJ1aWQiOiI3MzY3MTI1MC1lODgyLTExZWMtYjUzOC0xM2FjYjdhZjBkZTQiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxMi4xOTEwOTk3MzJaIn0sImxvZ2ljYWQiOnsidWlkIjoiQVZ4OVROQS11c25pa3M4QURzTHpWa3JvaDg4QUFBR0JUREh0UUEiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OTowOS40NTUxNDk2MTZaIn0sIm1lZGlhbmV0Ijp7InVpZCI6IjI5Nzg0MjM0OTI4OTU0MTAwMDBWMTAiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxMy42NzIyMTUxMjhaIn0sIm1naWQiOnsidWlkIjoibTU5Z1hyN0xlX1htIiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MTcuMDk3MDAxNDcxWiJ9LCJuYW5vaW50ZXJhY3RpdmUiOnsidWlkIjoiNmFlYzhjMTAzNzlkY2I3ODQxMmJjODBiNmRkOWM5NzMxNzNhYjdkNzEyZTQzMWE1YTVlYTcwMzRlNTZhNThhMCIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjE2LjcxNDgwNzUwNVoifSwib25ldGFnIjp7InVpZCI6IjdPelZoVzFOeC1LOGFVak1HMG52NXVNYm5YNEFHUXZQbnVHcHFrZ3k0ckEiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OTowOS4xNDE3NDEyNjJaIn0sIm9wZW54Ijp7InVpZCI6IjVkZWNlNjIyLTBhMjMtMGRhYi0zYTI0LTVhNzcwMTBlNDU4MiIsImV4cGlyZXMiOiIyMDIzLTA1LTMxVDA3OjUyOjQ3LjE0MDQxNzM2M1oifSwicHVibWF0aWMiOnsidWlkIjoiN0Q3NUQyNUYtRkFDOS00NDNELUIyRDEtQjE3RkVFMTFFMDI3IiwiZXhwaXJlcyI6IjIwMjItMTAtMzFUMDk6MTQ6MjUuNzM3MjU2ODk5WiJ9LCJyaWNoYXVkaWVuY2UiOnsidWlkIjoiY2I2YzYzMjAtMzNlMi00Nzc0LWIxNjAtMXp6MTY1NDg0MDc0OSIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjA5LjUyNTA3NDE4WiJ9LCJzbWFydHlhZHMiOnsidWlkIjoiMTJhZjE1ZTQ0ZjAwZDA3NjMwZTc0YzQ5MTU0Y2JmYmE0Zjg0N2U4ZDRhMTU0YzhjM2Q1MWY1OGNmNzJhNDYyNyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjEwLjgyNTAzMTg4NFoifSwic21pbGV3YW50ZWQiOnsidWlkIjoiZGQ5YzNmZTE4N2VmOWIwOWNhYTViNzExNDA0YzI4MzAiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxNC4yNTU2MDkzNjNaIn0sInN5bmFjb3JtZWRpYSI6eyJ1aWQiOiJHRFBSIiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MDkuOTc5NTgzNDM4WiJ9LCJ0cmlwbGVsaWZ0Ijp7InVpZCI6IjcwMjE5NzUwNTQ4MDg4NjUxOTQ2MyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjA4Ljk4OTY3MzU3NFoifSwidmFsdWVpbXByZXNzaW9uIjp7InVpZCI6IjlkMDgxNTVmLWQ5ZmUtNGI1OC04OThlLWUyYzU2MjgyYWIzZSIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjA5LjA2NzgzOTE2NFoifSwidmlzeCI6eyJ1aWQiOiIyN2UwYWMzYy1iNDZlLTQxYjMtOTkyYy1mOGQyNzE0OTQ5NWUiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxMi45ODk1MjM1NzNaIn0sInlpZWxkbGFiIjp7InVpZCI6IjY5NzE0ZDlmLWZiMDAtNGE1Zi04MTljLTRiZTE5MTM2YTMyNSIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjExLjMwMzAyNjYxNVoifSwieWllbGRtbyI6eyJ1aWQiOiJnOTZjMmY3MTlmMTU1MWIzMWY2MyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjEwLjExMDUyODYwOVoifSwieWllbGRvbmUiOnsidWlkIjoiMmE0MmZiZDMtMmM3MC00ZWI5LWIxYmQtMDQ2OTY2NTBkOTQ4IiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MTAuMzE4MzMzOTM5WiJ9LCJ6ZXJvY2xpY2tmcmF1ZCI6eyJ1aWQiOiJiOTk5NThmZS0yYTg3LTJkYTQtOWNjNC05NjFmZDExM2JlY2UiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxNS43MTk1OTQ1NjZaIn19LCJiZGF5IjoiMjAyMi0wNS0xN1QwNjo0ODozOC4wMTc5ODgyMDZaIn0=`,
							},
							KADUSERCookie: &http.Cookie{
								Name:  "KADUSERCOOKIE",
								Value: `7D75D25F-FAC9-443D-B2D1-B17FEE11E027`,
							},
							OriginCookie:             "go-test",
							Aliases:                  make(map[string]string),
							ImpBidCtx:                make(map[string]models.ImpCtx),
							PrebidBidderCode:         make(map[string]string),
							BidderResponseTimeMillis: make(map[string]int),
							ProfileIDStr:             "1234",
							Endpoint:                 models.EndpointV25,
							SeatNonBids:              make(map[string][]openrtb_ext.NonBid),
							MetricsEngine:            mockEngine,
							WakandaDebug:             &wakanda.Debug{},
						},
					},
				},
				bidrequest: json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","banner":{"format":[{"w":728,"h":90},{"w":300,"h":250}],"w":700,"h":900},"video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"tagid":"adunit","bidfloor":4.3,"bidfloorcur":"USD","ext":{"bidder":{"pubmatic":{"keywords":[{"key":"pmzoneid","value":["val1","val2"]}]}},"prebid":{}}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}},"ext":{"prebid":{},"wrapper":{"test":123,"profileid":123,"versionid":1,"wiid":"test_display_wiid"}}}`),
			},
			fields: fields{
				cache:        mockCache,
				metricEngine: mockEngine,
			},
			setup: func() {
				mockCache.EXPECT().GetMappingsFromCacheV25(gomock.Any(), gomock.Any()).Return(map[string]models.SlotMapping{
					"adunit@700x900": {
						SlotName: "adunit@700x900",
						SlotMappings: map[string]interface{}{
							models.SITE_CACHE_KEY: "12313",
							models.TAG_CACHE_KEY:  "45343",
						},
					},
				}).Times(3)
				mockCache.EXPECT().GetSlotToHashValueMapFromCacheV25(gomock.Any(), gomock.Any()).Return(models.SlotMappingInfo{
					OrderedSlotList: []string{"adunit@700x900"},
					HashValueMap: map[string]string{
						"adunit@700x900": "1232433543534543",
					},
				}).Times(3)
				mockCache.EXPECT().GetPartnerConfigMap(gomock.Any(), gomock.Any(), gomock.Any()).Return(map[int]map[string]string{
					1: {
						models.PARTNER_ID:          "1",
						models.PREBID_PARTNER_NAME: "pubmatic2",
						models.BidderCode:          "pub2-alias",
						models.IsAlias:             "1",
						models.TIMEOUT:             "200",
						models.PubID:               "301",
						models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
						models.SERVER_SIDE_FLAG:    "1",
						models.VENDORID:            "130",
					},
					2: {
						models.PARTNER_ID:          "2",
						models.PREBID_PARTNER_NAME: "appnexus",
						models.BidderCode:          "appnexus",
						models.SERVER_SIDE_FLAG:    "1",
						models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
						models.TIMEOUT:             "200",
						models.VENDORID:            "100",
					},
					3: {
						models.PARTNER_ID:          "3",
						models.PREBID_PARTNER_NAME: "districtm",
						models.BidderCode:          "dm-alias",
						models.IsAlias:             "1",
						models.TIMEOUT:             "200",
						models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
						models.SERVER_SIDE_FLAG:    "1",
						models.VENDORID:            "99",
					},
					-1: {
						models.DisplayVersionID: "1",
						models.PLATFORM_KEY:     models.PLATFORM_APP,
					},
				}, nil)
				mockCache.EXPECT().GetAdunitConfigFromCache(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&adunitconfig.AdUnitConfig{})
				mockCache.EXPECT().GetThrottlePartnersWithCriteria(gomock.Any()).Return(map[string]struct{}{}, nil)
				//prometheus metrics
				mockEngine.EXPECT().RecordPublisherProfileRequests("5890", "1234")
				mockEngine.EXPECT().RecordPublisherRequests(rctx.Endpoint, "5890", rctx.Platform)
				mockEngine.EXPECT().RecordPlatformPublisherPartnerReqStats(rctx.Platform, "5890", "pub2-alias")
				mockEngine.EXPECT().RecordPlatformPublisherPartnerReqStats(rctx.Platform, "5890", "appnexus")
				mockEngine.EXPECT().RecordPlatformPublisherPartnerReqStats(rctx.Platform, "5890", "dm-alias")
				mockEngine.EXPECT().RecordPreProcessingTimeStats(rctx.PubIDStr, gomock.Any())
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false)
				mockFeature.EXPECT().IsAnalyticsTrackingThrottled(gomock.Any(), gomock.Any()).Return(false, false)
				mockProfileMetaData.EXPECT().GetProfileTypePlatform(gomock.Any()).Return(0, false)
				mockFeature.EXPECT().IsDynamicFloorEnabledPublisher(gomock.Any()).Return(false)
			},
			want: want{
				hookResult: hookstage.HookResult[hookstage.BeforeValidationRequestPayload]{
					Reject:        false,
					NbrCode:       0,
					ChangeSet:     hookstage.ChangeSet[hookstage.BeforeValidationRequestPayload]{},
					DebugMessages: []string{`new imp: {"123":{"ImpID":"123","TagID":"adunit","DisplayManager":"","DisplayManagerVer":"","Div":"","SlotName":"adunit","AdUnitName":"adunit","Secure":0,"BidFloor":4.3,"BidFloorCur":"USD","IsRewardInventory":null,"Banner":true,"Video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"Native":null,"IncomingSlots":["700x900","728x90","300x250","640x480"],"Type":"video","Bidders":{"appnexus":{"PartnerID":2,"PrebidBidderCode":"appnexus","MatchedSlot":"adunit@700x900","KGP":"_AU_@_W_x_H_","KGPV":"","IsRegex":false,"Params":{"placementId":0,"adtag":"45343","site":"12313"},"VASTTagFlags":null},"dm-alias":{"PartnerID":3,"PrebidBidderCode":"districtm","MatchedSlot":"adunit@700x900","KGP":"_AU_@_W_x_H_","KGPV":"","IsRegex":false,"Params":{"placementId":0,"site":"12313","adtag":"45343"},"VASTTagFlags":null},"pub2-alias":{"PartnerID":1,"PrebidBidderCode":"pubmatic2","MatchedSlot":"adunit@700x900","KGP":"_AU_@_W_x_H_","KGPV":"","IsRegex":false,"Params":{"publisherId":"301","adSlot":"adunit@700x900"},"VASTTagFlags":null}},"NonMapped":{},"NewExt":{"data":{"pbadslot":"adunit"},"prebid":{"bidder":{"appnexus":{"placementId":0,"adtag":"45343","site":"12313"},"dm-alias":{"placementId":0,"site":"12313","adtag":"45343"},"pub2-alias":{"publisherId":"301","adSlot":"adunit@700x900"}}}},"BidCtx":{},"BannerAdUnitCtx":{"MatchedSlot":"","IsRegex":false,"MatchedRegex":"","SelectedSlotAdUnitConfig":null,"AppliedSlotAdUnitConfig":null,"UsingDefaultConfig":false,"AllowedConnectionTypes":null},"VideoAdUnitCtx":{"MatchedSlot":"","IsRegex":false,"MatchedRegex":"","SelectedSlotAdUnitConfig":null,"AppliedSlotAdUnitConfig":null,"UsingDefaultConfig":false,"AllowedConnectionTypes":null},"NativeAdUnitCtx":{"MatchedSlot":"","IsRegex":false,"MatchedRegex":"","SelectedSlotAdUnitConfig":null,"AppliedSlotAdUnitConfig":null,"UsingDefaultConfig":false,"AllowedConnectionTypes":null},"BidderError":"","IsAdPodRequest":false,"AdpodConfig":null,"ImpAdPodCfg":null,"BidIDToAPRC":null,"AdserverURL":"","BidIDToDur":null}}`, `new request.ext: {"prebid":{"aliases":{"dm-alias":"appnexus","pub2-alias":"pubmatic"},"aliasgvlids":{"dm-alias":99,"pub2-alias":130},"bidadjustmentfactors":{"appnexus":1,"dm-alias":1,"pub2-alias":1},"bidderparams":{"pub2-alias":{"wiid":""},"pubmatic":{"wiid":""}},"debug":true,"floors":{"enforcement":{"enforcepbs":true,"floordeals":true},"enabled":true},"targeting":{"pricegranularity":{"precision":2,"ranges":[{"min":0,"max":5,"increment":0.05},{"min":5,"max":10,"increment":0.1},{"min":10,"max":20,"increment":0.5}]},"mediatypepricegranularity":{},"includewinners":true,"includebidderkeys":true},"macros":{"[PLATFORM]":"3","[PROFILE_ID]":"1234","[PROFILE_VERSION]":"1","[UNIX_TIMESTAMP]":"0","[WRAPPER_IMPRESSION_ID]":""}}}`}, AnalyticsTags: hookanalytics.Analytics{},
				},
				bidRequest:            json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","banner":{"format":[{"w":728,"h":90},{"w":300,"h":250}],"w":700,"h":900},"video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"tagid":"adunit","bidfloor":4.3,"bidfloorcur":"USD","ext":{"data":{"pbadslot":"adunit"},"prebid":{"bidder":{"appnexus":{"placementId":0,"site":"12313","adtag":"45343"},"dm-alias":{"placementId":0,"site":"12313","adtag":"45343"},"pub2-alias":{"publisherId":"301","adSlot":"adunit@700x900"}}}}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","customdata":"7D75D25F-FAC9-443D-B2D1-B17FEE11E027","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"tid":"123-456-789","ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}},"ext":{"prebid":{"aliases":{"dm-alias":"appnexus","pub2-alias":"pubmatic"},"aliasgvlids":{"dm-alias":99,"pub2-alias":130},"bidadjustmentfactors":{"appnexus":1,"dm-alias":1,"pub2-alias":1},"bidderparams":{"pub2-alias":{"wiid":""},"pubmatic":{"wiid":""}},"debug":true,"floors":{"enforcement":{"enforcepbs":true,"floordeals":true},"enabled":true},"targeting":{"pricegranularity":{"precision":2,"ranges":[{"min":0,"max":5,"increment":0.05},{"min":5,"max":10,"increment":0.1},{"min":10,"max":20,"increment":0.5}]},"includewinners":true,"includebidderkeys":true},"macros":{"[PLATFORM]":"3","[PROFILE_ID]":"1234","[PROFILE_VERSION]":"1","[UNIX_TIMESTAMP]":"0","[WRAPPER_IMPRESSION_ID]":""}}}}`),
				error:                 false,
				doMutate:              true,
				nilCurrencyConversion: false,
			},
		},
		{
			name: "happy_path_request_not_rejected_and_successfully_updted_from_DB",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							PubIDStr:                  "5890",
							PubID:                     5890,
							ProfileID:                 1234,
							DisplayID:                 1,
							SSAuction:                 -1,
							Platform:                  "in-app",
							Debug:                     true,
							DeviceCtx:                 models.DeviceCtx{IP: "127.0.0.1", UA: "go-test"},
							IsCTVRequest:              false,
							TrackerEndpoint:           "t.pubmatic.com",
							VideoErrorTrackerEndpoint: "t.pubmatic.com/error",
							UidCookie: &http.Cookie{
								Name:  "uids",
								Value: `eyJ0ZW1wVUlEcyI6eyIzM2Fjcm9zcyI6eyJ1aWQiOiIxMTkxNzkxMDk5Nzc2NjEiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OTo0My4zODg4Nzk5NVoifSwiYWRmIjp7InVpZCI6IjgwNDQ2MDgzMzM3Nzg4MzkwNzgiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxMS4wMzMwNTQ3MjdaIn0sImFka2VybmVsIjp7InVpZCI6IkE5MTYzNTAwNzE0OTkyOTMyOTkwIiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MDkuMzczMzg1NjYyWiJ9LCJhZGtlcm5lbEFkbiI6eyJ1aWQiOiJBOTE2MzUwMDcxNDk5MjkzMjk5MCIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjEzLjQzNDkyNTg5NloifSwiYWRtaXhlciI6eyJ1aWQiOiIzNjZhMTdiMTJmMjI0ZDMwOGYzZTNiOGRhOGMzYzhhMCIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjA5LjU5MjkxNDgwMVoifSwiYWRueHMiOnsidWlkIjoiNDE5Mjg5ODUzMDE0NTExOTMiLCJleHBpcmVzIjoiMjAyMy0wMS0xOFQwOTo1MzowOC44MjU0NDI2NzZaIn0sImFqYSI6eyJ1aWQiOiJzMnN1aWQ2RGVmMFl0bjJveGQ1aG9zS1AxVmV3IiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MTMuMjM5MTc2MDU0WiJ9LCJlcGxhbm5pbmciOnsidWlkIjoiQUoxRjBTOE5qdTdTQ0xWOSIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjEwLjkyOTk2MDQ3M1oifSwiZ2Ftb3NoaSI6eyJ1aWQiOiJndXNyXzM1NmFmOWIxZDhjNjQyYjQ4MmNiYWQyYjdhMjg4MTYxIiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MTAuNTI0MTU3MjI1WiJ9LCJncmlkIjp7InVpZCI6IjRmYzM2MjUwLWQ4NTItNDU5Yy04NzcyLTczNTZkZTE3YWI5NyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjE0LjY5NjMxNjIyN1oifSwiZ3JvdXBtIjp7InVpZCI6IjdENzVEMjVGLUZBQzktNDQzRC1CMkQxLUIxN0ZFRTExRTAyNyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjM5LjIyNjIxMjUzMloifSwiaXgiOnsidWlkIjoiWW9ORlNENlc5QkphOEh6eEdtcXlCUUFBXHUwMDI2Mjk3IiwiZXhwaXJlcyI6IjIwMjMtMDUtMzFUMDc6NTM6MzguNTU1ODI3MzU0WiJ9LCJqaXhpZSI6eyJ1aWQiOiI3MzY3MTI1MC1lODgyLTExZWMtYjUzOC0xM2FjYjdhZjBkZTQiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxMi4xOTEwOTk3MzJaIn0sImxvZ2ljYWQiOnsidWlkIjoiQVZ4OVROQS11c25pa3M4QURzTHpWa3JvaDg4QUFBR0JUREh0UUEiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OTowOS40NTUxNDk2MTZaIn0sIm1lZGlhbmV0Ijp7InVpZCI6IjI5Nzg0MjM0OTI4OTU0MTAwMDBWMTAiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxMy42NzIyMTUxMjhaIn0sIm1naWQiOnsidWlkIjoibTU5Z1hyN0xlX1htIiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MTcuMDk3MDAxNDcxWiJ9LCJuYW5vaW50ZXJhY3RpdmUiOnsidWlkIjoiNmFlYzhjMTAzNzlkY2I3ODQxMmJjODBiNmRkOWM5NzMxNzNhYjdkNzEyZTQzMWE1YTVlYTcwMzRlNTZhNThhMCIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjE2LjcxNDgwNzUwNVoifSwib25ldGFnIjp7InVpZCI6IjdPelZoVzFOeC1LOGFVak1HMG52NXVNYm5YNEFHUXZQbnVHcHFrZ3k0ckEiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OTowOS4xNDE3NDEyNjJaIn0sIm9wZW54Ijp7InVpZCI6IjVkZWNlNjIyLTBhMjMtMGRhYi0zYTI0LTVhNzcwMTBlNDU4MiIsImV4cGlyZXMiOiIyMDIzLTA1LTMxVDA3OjUyOjQ3LjE0MDQxNzM2M1oifSwicHVibWF0aWMiOnsidWlkIjoiN0Q3NUQyNUYtRkFDOS00NDNELUIyRDEtQjE3RkVFMTFFMDI3IiwiZXhwaXJlcyI6IjIwMjItMTAtMzFUMDk6MTQ6MjUuNzM3MjU2ODk5WiJ9LCJyaWNoYXVkaWVuY2UiOnsidWlkIjoiY2I2YzYzMjAtMzNlMi00Nzc0LWIxNjAtMXp6MTY1NDg0MDc0OSIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjA5LjUyNTA3NDE4WiJ9LCJzbWFydHlhZHMiOnsidWlkIjoiMTJhZjE1ZTQ0ZjAwZDA3NjMwZTc0YzQ5MTU0Y2JmYmE0Zjg0N2U4ZDRhMTU0YzhjM2Q1MWY1OGNmNzJhNDYyNyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjEwLjgyNTAzMTg4NFoifSwic21pbGV3YW50ZWQiOnsidWlkIjoiZGQ5YzNmZTE4N2VmOWIwOWNhYTViNzExNDA0YzI4MzAiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxNC4yNTU2MDkzNjNaIn0sInN5bmFjb3JtZWRpYSI6eyJ1aWQiOiJHRFBSIiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MDkuOTc5NTgzNDM4WiJ9LCJ0cmlwbGVsaWZ0Ijp7InVpZCI6IjcwMjE5NzUwNTQ4MDg4NjUxOTQ2MyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjA4Ljk4OTY3MzU3NFoifSwidmFsdWVpbXByZXNzaW9uIjp7InVpZCI6IjlkMDgxNTVmLWQ5ZmUtNGI1OC04OThlLWUyYzU2MjgyYWIzZSIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjA5LjA2NzgzOTE2NFoifSwidmlzeCI6eyJ1aWQiOiIyN2UwYWMzYy1iNDZlLTQxYjMtOTkyYy1mOGQyNzE0OTQ5NWUiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxMi45ODk1MjM1NzNaIn0sInlpZWxkbGFiIjp7InVpZCI6IjY5NzE0ZDlmLWZiMDAtNGE1Zi04MTljLTRiZTE5MTM2YTMyNSIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjExLjMwMzAyNjYxNVoifSwieWllbGRtbyI6eyJ1aWQiOiJnOTZjMmY3MTlmMTU1MWIzMWY2MyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjEwLjExMDUyODYwOVoifSwieWllbGRvbmUiOnsidWlkIjoiMmE0MmZiZDMtMmM3MC00ZWI5LWIxYmQtMDQ2OTY2NTBkOTQ4IiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MTAuMzE4MzMzOTM5WiJ9LCJ6ZXJvY2xpY2tmcmF1ZCI6eyJ1aWQiOiJiOTk5NThmZS0yYTg3LTJkYTQtOWNjNC05NjFmZDExM2JlY2UiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxNS43MTk1OTQ1NjZaIn19LCJiZGF5IjoiMjAyMi0wNS0xN1QwNjo0ODozOC4wMTc5ODgyMDZaIn0=`,
							},
							KADUSERCookie: &http.Cookie{
								Name:  "KADUSERCOOKIE",
								Value: `7D75D25F-FAC9-443D-B2D1-B17FEE11E027`,
							},
							OriginCookie:             "go-test",
							Aliases:                  make(map[string]string),
							ImpBidCtx:                make(map[string]models.ImpCtx),
							PrebidBidderCode:         make(map[string]string),
							BidderResponseTimeMillis: make(map[string]int),
							ProfileIDStr:             "1234",
							Endpoint:                 models.EndpointV25,
							SeatNonBids:              make(map[string][]openrtb_ext.NonBid),
							MetricsEngine:            mockEngine,
							WakandaDebug:             &wakanda.Debug{},
						},
					},
				},
				bidrequest: json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","banner":{"format":[{"w":728,"h":90},{"w":300,"h":250}],"w":700,"h":900},"video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"tagid":"adunit","bidfloor":4.3,"bidfloorcur":"USD","ext":{"bidder":{"pubmatic":{"keywords":[{"key":"pmzoneid","value":["val1","val2"]}]}},"prebid":{}}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}},"ext":{"prebid":{},"wrapper":{"test":123,"profileid":123,"versionid":1,"wiid":"test_display_wiid"}}}`),
			},
			fields: fields{
				cache:        mockCache,
				metricEngine: mockEngine,
			},
			setup: func() {
				mockCache.EXPECT().GetMappingsFromCacheV25(gomock.Any(), gomock.Any()).Return(map[string]models.SlotMapping{
					"adunit@700x900": {
						SlotName: "adunit@700x900",
						SlotMappings: map[string]interface{}{
							models.SITE_CACHE_KEY: "12313",
							models.TAG_CACHE_KEY:  "45343",
						},
					},
				})
				mockCache.EXPECT().GetSlotToHashValueMapFromCacheV25(gomock.Any(), gomock.Any()).Return(models.SlotMappingInfo{
					OrderedSlotList: []string{"adunit@700x900"},
					HashValueMap: map[string]string{
						"adunit@700x900": "1232433543534543",
					},
				})
				mockCache.EXPECT().GetPartnerConfigMap(gomock.Any(), gomock.Any(), gomock.Any()).Return(map[int]map[string]string{
					2: {
						models.PARTNER_ID:          "2",
						models.PREBID_PARTNER_NAME: "appnexus",
						models.BidderCode:          "appnexus",
						models.SERVER_SIDE_FLAG:    "1",
						models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
						models.TIMEOUT:             "200",
					},
					-1: {
						models.DisplayVersionID:   "1",
						models.PLATFORM_KEY:       models.PLATFORM_APP,
						models.AllBidderSChainObj: `[{"bidders":["bidderA"],"schain":{"ver":"1.0","complete":1,"nodes":[{"asi":"example.com"}]}}]`,
						models.SChainObjectDBKey:  `{"validation":"off","config":{"ver":"2.0","complete":1,"nodes":[{"asi":"indirectseller-1.com","sid":"00001","hp":1}]}}`,
					},
				}, nil)
				mockCache.EXPECT().GetAdunitConfigFromCache(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&adunitconfig.AdUnitConfig{})
				mockCache.EXPECT().GetThrottlePartnersWithCriteria(gomock.Any()).Return(map[string]struct{}{}, nil)
				//prometheus metrics
				mockEngine.EXPECT().RecordPublisherRequests(models.EndpointV25, "5890", "in-app")
				mockEngine.EXPECT().RecordPublisherProfileRequests("5890", "1234")
				mockEngine.EXPECT().RecordPlatformPublisherPartnerReqStats(rctx.Platform, "5890", "appnexus")
				mockEngine.EXPECT().RecordPreProcessingTimeStats(rctx.PubIDStr, gomock.Any())
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false)
				mockFeature.EXPECT().IsAnalyticsTrackingThrottled(gomock.Any(), gomock.Any()).Return(false, false)
				mockProfileMetaData.EXPECT().GetProfileTypePlatform(gomock.Any()).Return(0, false)
				mockFeature.EXPECT().IsDynamicFloorEnabledPublisher(gomock.Any()).Return(false)
			},
			want: want{
				hookResult: hookstage.HookResult[hookstage.BeforeValidationRequestPayload]{
					Reject:        false,
					NbrCode:       0,
					ChangeSet:     hookstage.ChangeSet[hookstage.BeforeValidationRequestPayload]{},
					DebugMessages: []string{`new imp: {"123":{"ImpID":"123","TagID":"adunit","DisplayManager":"","DisplayManagerVer":"","Div":"","SlotName":"adunit","AdUnitName":"adunit","Secure":0,"BidFloor":4.3,"BidFloorCur":"USD","IsRewardInventory":null,"Banner":true,"Video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"Native":null,"IncomingSlots":["700x900","728x90","300x250","640x480"],"Type":"video","Bidders":{"appnexus":{"PartnerID":2,"PrebidBidderCode":"appnexus","MatchedSlot":"adunit@700x900","KGP":"_AU_@_W_x_H_","KGPV":"","IsRegex":false,"Params":{"placementId":0,"site":"12313","adtag":"45343"},"VASTTagFlags":null}},"NonMapped":{},"NewExt":{"data":{"pbadslot":"adunit"},"prebid":{"bidder":{"appnexus":{"placementId":0,"site":"12313","adtag":"45343"}}}},"BidCtx":{},"BannerAdUnitCtx":{"MatchedSlot":"","IsRegex":false,"MatchedRegex":"","SelectedSlotAdUnitConfig":null,"AppliedSlotAdUnitConfig":null,"UsingDefaultConfig":false,"AllowedConnectionTypes":null},"VideoAdUnitCtx":{"MatchedSlot":"adunit","IsRegex":false,"MatchedRegex":"","SelectedSlotAdUnitConfig":{"video":{"enabled":true,"amptrafficpercentage":100,"config":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480}}},"AppliedSlotAdUnitConfig":{"video":{"enabled":true,"amptrafficpercentage":100,"config":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480}}},"UsingDefaultConfig":false,"AllowedConnectionTypes":null},"NativeAdUnitCtx":{"MatchedSlot":"","IsRegex":false,"MatchedRegex":"","SelectedSlotAdUnitConfig":null,"AppliedSlotAdUnitConfig":null,"UsingDefaultConfig":false,"AllowedConnectionTypes":null},"BidderError":"","IsAdPodRequest":false,"AdpodConfig":null,"ImpAdPodCfg":null,"BidIDToAPRC":null,"AdserverURL":"","BidIDToDur":null}}`, `new request.ext: {"prebid":{"bidadjustmentfactors":{"appnexus":1},"bidderparams":{"pubmatic":{"wiid":""}},"debug":true,"floors":{"enforcement":{"enforcepbs":true,"floordeals":true},"enabled":true},"targeting":{"pricegranularity":{"precision":2,"ranges":[{"min":0,"max":5,"increment":0.05},{"min":5,"max":10,"increment":0.1},{"min":10,"max":20,"increment":0.5}]},"mediatypepricegranularity":{},"includewinners":true,"includebidderkeys":true},"macros":{"[PLATFORM]":"3","[PROFILE_ID]":"1234","[PROFILE_VERSION]":"1","[UNIX_TIMESTAMP]":"0","[WRAPPER_IMPRESSION_ID]":""}}}`}, AnalyticsTags: hookanalytics.Analytics{},
				},
				bidRequest:            json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","banner":{"format":[{"w":728,"h":90},{"w":300,"h":250}],"w":700,"h":900},"video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"tagid":"adunit","bidfloor":4.3,"bidfloorcur":"USD","ext":{"data":{"pbadslot":"adunit"},"prebid":{"bidder":{"appnexus":{"placementId":0,"site":"12313","adtag":"45343"}}}}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","customdata":"7D75D25F-FAC9-443D-B2D1-B17FEE11E027","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"tid":"123-456-789","ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1","schain":{"ver":"2.0","complete":1,"nodes":[{"asi":"indirectseller-1.com","sid":"00001","hp":1}]}}},"ext":{"prebid":{"bidadjustmentfactors":{"appnexus":1},"bidderparams":{"pubmatic":{"wiid":""}},"debug":true,"floors":{"enforcement":{"enforcepbs":true,"floordeals":true},"enabled":true},"schains":[{"bidders":["bidderA"],"schain":{"ver":"1.0","complete":1,"nodes":[{"asi":"example.com","sid":""}]}}],"targeting":{"pricegranularity":{"precision":2,"ranges":[{"min":0,"max":5,"increment":0.05},{"min":5,"max":10,"increment":0.1},{"min":10,"max":20,"increment":0.5}]},"includewinners":true,"includebidderkeys":true},"macros":{"[PLATFORM]":"3","[PROFILE_ID]":"1234","[PROFILE_VERSION]":"1","[UNIX_TIMESTAMP]":"0","[WRAPPER_IMPRESSION_ID]":""}}}}`),
				error:                 false,
				doMutate:              true,
				nilCurrencyConversion: false,
			},
		},
		{
			name: "prebid-validation-errors-imp-missing",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							PubIDStr: "1234",
							Endpoint: models.EndpointV25,
						},
					},
				},
				bidrequest: json.RawMessage(`{}`),
			},
			fields: fields{
				cache:        mockCache,
				metricEngine: mockEngine,
			},
			setup: func() {
				mockEngine.EXPECT().RecordBadRequests(models.EndpointV25, "1234", 18)
				mockEngine.EXPECT().RecordNobidErrPrebidServerRequests("1234", 604)
			},
			want: want{
				hookResult:            hookstage.HookResult[hookstage.BeforeValidationRequestPayload]{},
				error:                 false,
				nilCurrencyConversion: true,
			},
		},
		{
			name: "prebid-validation-errors-site-and-app-missing",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							PubIDStr: "1234",
							Endpoint: models.EndpointV25,
						},
					},
				},
				bidrequest: json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","banner":{"format":[{"w":728,"h":90},{"w":300,"h":250}],"w":700,"h":900},"video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"tagid":"adunit","bidfloor":4.3,"bidfloorcur":"USD","ext":{"bidder":{"pubmatic":{"keywords":[{"key":"pmzoneid","value":["val1","val2"]}]}},"prebid":{}}}],"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}},"ext":{"prebid":{},"wrapper":{"test":123,"profileid":123,"versionid":1,"wiid":"test_display_wiid"}}}`),
			},
			fields: fields{
				cache:        mockCache,
				metricEngine: mockEngine,
			},
			setup: func() {
				mockEngine.EXPECT().RecordBadRequests(models.EndpointV25, "1234", 18)
				mockEngine.EXPECT().RecordNobidErrPrebidServerRequests("1234", 604)
			},
			want: want{
				hookResult:            hookstage.HookResult[hookstage.BeforeValidationRequestPayload]{},
				error:                 false,
				nilCurrencyConversion: true,
			},
		},
		{
			name: "AMP_request_successfully_update_video_object_from_adunit_config_and_updated_remaining_feilds_from_default",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							PubIDStr:                  "5890",
							ProfileID:                 1234,
							PubID:                     5890,
							DisplayID:                 1,
							SSAuction:                 -1,
							Platform:                  "amp",
							Debug:                     true,
							DeviceCtx:                 models.DeviceCtx{IP: "127.0.0.1", UA: "go-test"},
							IsCTVRequest:              false,
							TrackerEndpoint:           "t.pubmatic.com",
							VideoErrorTrackerEndpoint: "t.pubmatic.com/error",
							UidCookie: &http.Cookie{
								Name:  "uids",
								Value: `eyJ0ZW1wVUlEcyI6eyIzM2Fjcm9zcyI6eyJ1aWQiOiIxMTkxNzkxMDk5Nzc2NjEiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OTo0My4zODg4Nzk5NVoifSwiYWRmIjp7InVpZCI6IjgwNDQ2MDgzMzM3Nzg4MzkwNzgiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxMS4wMzMwNTQ3MjdaIn0sImFka2VybmVsIjp7InVpZCI6IkE5MTYzNTAwNzE0OTkyOTMyOTkwIiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MDkuMzczMzg1NjYyWiJ9LCJhZGtlcm5lbEFkbiI6eyJ1aWQiOiJBOTE2MzUwMDcxNDk5MjkzMjk5MCIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjEzLjQzNDkyNTg5NloifSwiYWRtaXhlciI6eyJ1aWQiOiIzNjZhMTdiMTJmMjI0ZDMwOGYzZTNiOGRhOGMzYzhhMCIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjA5LjU5MjkxNDgwMVoifSwiYWRueHMiOnsidWlkIjoiNDE5Mjg5ODUzMDE0NTExOTMiLCJleHBpcmVzIjoiMjAyMy0wMS0xOFQwOTo1MzowOC44MjU0NDI2NzZaIn0sImFqYSI6eyJ1aWQiOiJzMnN1aWQ2RGVmMFl0bjJveGQ1aG9zS1AxVmV3IiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MTMuMjM5MTc2MDU0WiJ9LCJlcGxhbm5pbmciOnsidWlkIjoiQUoxRjBTOE5qdTdTQ0xWOSIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjEwLjkyOTk2MDQ3M1oifSwiZ2Ftb3NoaSI6eyJ1aWQiOiJndXNyXzM1NmFmOWIxZDhjNjQyYjQ4MmNiYWQyYjdhMjg4MTYxIiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MTAuNTI0MTU3MjI1WiJ9LCJncmlkIjp7InVpZCI6IjRmYzM2MjUwLWQ4NTItNDU5Yy04NzcyLTczNTZkZTE3YWI5NyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjE0LjY5NjMxNjIyN1oifSwiZ3JvdXBtIjp7InVpZCI6IjdENzVEMjVGLUZBQzktNDQzRC1CMkQxLUIxN0ZFRTExRTAyNyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjM5LjIyNjIxMjUzMloifSwiaXgiOnsidWlkIjoiWW9ORlNENlc5QkphOEh6eEdtcXlCUUFBXHUwMDI2Mjk3IiwiZXhwaXJlcyI6IjIwMjMtMDUtMzFUMDc6NTM6MzguNTU1ODI3MzU0WiJ9LCJqaXhpZSI6eyJ1aWQiOiI3MzY3MTI1MC1lODgyLTExZWMtYjUzOC0xM2FjYjdhZjBkZTQiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxMi4xOTEwOTk3MzJaIn0sImxvZ2ljYWQiOnsidWlkIjoiQVZ4OVROQS11c25pa3M4QURzTHpWa3JvaDg4QUFBR0JUREh0UUEiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OTowOS40NTUxNDk2MTZaIn0sIm1lZGlhbmV0Ijp7InVpZCI6IjI5Nzg0MjM0OTI4OTU0MTAwMDBWMTAiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxMy42NzIyMTUxMjhaIn0sIm1naWQiOnsidWlkIjoibTU5Z1hyN0xlX1htIiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MTcuMDk3MDAxNDcxWiJ9LCJuYW5vaW50ZXJhY3RpdmUiOnsidWlkIjoiNmFlYzhjMTAzNzlkY2I3ODQxMmJjODBiNmRkOWM5NzMxNzNhYjdkNzEyZTQzMWE1YTVlYTcwMzRlNTZhNThhMCIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjE2LjcxNDgwNzUwNVoifSwib25ldGFnIjp7InVpZCI6IjdPelZoVzFOeC1LOGFVak1HMG52NXVNYm5YNEFHUXZQbnVHcHFrZ3k0ckEiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OTowOS4xNDE3NDEyNjJaIn0sIm9wZW54Ijp7InVpZCI6IjVkZWNlNjIyLTBhMjMtMGRhYi0zYTI0LTVhNzcwMTBlNDU4MiIsImV4cGlyZXMiOiIyMDIzLTA1LTMxVDA3OjUyOjQ3LjE0MDQxNzM2M1oifSwicHVibWF0aWMiOnsidWlkIjoiN0Q3NUQyNUYtRkFDOS00NDNELUIyRDEtQjE3RkVFMTFFMDI3IiwiZXhwaXJlcyI6IjIwMjItMTAtMzFUMDk6MTQ6MjUuNzM3MjU2ODk5WiJ9LCJyaWNoYXVkaWVuY2UiOnsidWlkIjoiY2I2YzYzMjAtMzNlMi00Nzc0LWIxNjAtMXp6MTY1NDg0MDc0OSIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjA5LjUyNTA3NDE4WiJ9LCJzbWFydHlhZHMiOnsidWlkIjoiMTJhZjE1ZTQ0ZjAwZDA3NjMwZTc0YzQ5MTU0Y2JmYmE0Zjg0N2U4ZDRhMTU0YzhjM2Q1MWY1OGNmNzJhNDYyNyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjEwLjgyNTAzMTg4NFoifSwic21pbGV3YW50ZWQiOnsidWlkIjoiZGQ5YzNmZTE4N2VmOWIwOWNhYTViNzExNDA0YzI4MzAiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxNC4yNTU2MDkzNjNaIn0sInN5bmFjb3JtZWRpYSI6eyJ1aWQiOiJHRFBSIiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MDkuOTc5NTgzNDM4WiJ9LCJ0cmlwbGVsaWZ0Ijp7InVpZCI6IjcwMjE5NzUwNTQ4MDg4NjUxOTQ2MyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjA4Ljk4OTY3MzU3NFoifSwidmFsdWVpbXByZXNzaW9uIjp7InVpZCI6IjlkMDgxNTVmLWQ5ZmUtNGI1OC04OThlLWUyYzU2MjgyYWIzZSIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjA5LjA2NzgzOTE2NFoifSwidmlzeCI6eyJ1aWQiOiIyN2UwYWMzYy1iNDZlLTQxYjMtOTkyYy1mOGQyNzE0OTQ5NWUiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxMi45ODk1MjM1NzNaIn0sInlpZWxkbGFiIjp7InVpZCI6IjY5NzE0ZDlmLWZiMDAtNGE1Zi04MTljLTRiZTE5MTM2YTMyNSIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjExLjMwMzAyNjYxNVoifSwieWllbGRtbyI6eyJ1aWQiOiJnOTZjMmY3MTlmMTU1MWIzMWY2MyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjEwLjExMDUyODYwOVoifSwieWllbGRvbmUiOnsidWlkIjoiMmE0MmZiZDMtMmM3MC00ZWI5LWIxYmQtMDQ2OTY2NTBkOTQ4IiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MTAuMzE4MzMzOTM5WiJ9LCJ6ZXJvY2xpY2tmcmF1ZCI6eyJ1aWQiOiJiOTk5NThmZS0yYTg3LTJkYTQtOWNjNC05NjFmZDExM2JlY2UiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxNS43MTk1OTQ1NjZaIn19LCJiZGF5IjoiMjAyMi0wNS0xN1QwNjo0ODozOC4wMTc5ODgyMDZaIn0=`,
							},
							KADUSERCookie: &http.Cookie{
								Name:  "KADUSERCOOKIE",
								Value: `7D75D25F-FAC9-443D-B2D1-B17FEE11E027`,
							},
							OriginCookie:             "go-test",
							Aliases:                  make(map[string]string),
							ImpBidCtx:                make(map[string]models.ImpCtx),
							PrebidBidderCode:         make(map[string]string),
							BidderResponseTimeMillis: make(map[string]int),
							ProfileIDStr:             "1234",
							Endpoint:                 models.EndpointAMP,
							SeatNonBids:              make(map[string][]openrtb_ext.NonBid),
							MetricsEngine:            mockEngine,
							WakandaDebug:             &wakanda.Debug{},
						},
					},
				},
				bidrequest: json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","banner":{"format":[{"w":728,"h":90},{"w":300,"h":250}],"w":700,"h":900},"tagid":"adunit","bidfloor":4.3,"bidfloorcur":"USD","ext":{"bidder":{"pubmatic":{"keywords":[{"key":"pmzoneid","value":["val1","val2"]}]}},"prebid":{}}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}},"ext":{"prebid":{},"wrapper":{"test":123,"profileid":123,"versionid":1,"wiid":"test_display_wiid"}}}`),
			},
			fields: fields{
				cache:        mockCache,
				metricEngine: mockEngine,
			},
			setup: func() {
				mockCache.EXPECT().GetMappingsFromCacheV25(gomock.Any(), gomock.Any()).Return(map[string]models.SlotMapping{
					"adunit@700x900": {
						SlotName: "adunit@700x900",
						SlotMappings: map[string]interface{}{
							models.SITE_CACHE_KEY: "12313",
							models.TAG_CACHE_KEY:  "45343",
						},
					},
				})
				mockCache.EXPECT().GetSlotToHashValueMapFromCacheV25(gomock.Any(), gomock.Any()).Return(models.SlotMappingInfo{
					OrderedSlotList: []string{"adunit@700x900"},
					HashValueMap: map[string]string{
						"adunit@700x900": "1232433543534543",
					},
				})
				mockCache.EXPECT().GetPartnerConfigMap(gomock.Any(), gomock.Any(), gomock.Any()).Return(map[int]map[string]string{
					2: {
						models.PARTNER_ID:          "2",
						models.PREBID_PARTNER_NAME: "appnexus",
						models.BidderCode:          "appnexus",
						models.SERVER_SIDE_FLAG:    "1",
						models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
						models.TIMEOUT:             "200",
					},
					-1: {
						models.DisplayVersionID: "1",
						models.PLATFORM_KEY:     models.PLATFORM_AMP,
					},
				}, nil)
				mockCache.EXPECT().GetAdunitConfigFromCache(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&adunitconfig.AdUnitConfig{
					ConfigPattern: "_AU_",
					Config: map[string]*adunitconfig.AdConfig{
						"adunit": {
							Video: &adunitconfig.Video{
								Enabled:              ptrutil.ToPtr(true),
								AmpTrafficPercentage: ptrutil.ToPtr(100),
								Config: &adunitconfig.VideoConfig{
									Video: openrtb2.Video{
										MIMEs: []string{"video/mp4", "video/mpeg"},
										W:     ptrutil.ToPtr[int64](640),
										H:     ptrutil.ToPtr[int64](480),
									},
								},
							},
						},
					},
				})
				mockCache.EXPECT().GetThrottlePartnersWithCriteria(gomock.Any()).Return(map[string]struct{}{}, nil)
				//prometheus metrics
				mockEngine.EXPECT().RecordPublisherProfileRequests("5890", "1234")
				mockEngine.EXPECT().RecordPublisherRequests(models.EndpointAMP, "5890", "amp")
				mockEngine.EXPECT().RecordPlatformPublisherPartnerReqStats("amp", "5890", "appnexus")
				mockEngine.EXPECT().RecordPreProcessingTimeStats(rctx.PubIDStr, gomock.Any())
				mockFeature.EXPECT().IsAmpMultiformatEnabled(5890).Return(true)
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false)
				mockFeature.EXPECT().IsAnalyticsTrackingThrottled(gomock.Any(), gomock.Any()).Return(false, false)
				mockProfileMetaData.EXPECT().GetProfileTypePlatform(gomock.Any()).Return(0, false)
			},
			want: want{
				hookResult: hookstage.HookResult[hookstage.BeforeValidationRequestPayload]{
					Reject:        false,
					NbrCode:       0,
					ChangeSet:     hookstage.ChangeSet[hookstage.BeforeValidationRequestPayload]{},
					DebugMessages: []string{`new imp: {"123":{"ImpID":"123","TagID":"adunit","DisplayManager":"","DisplayManagerVer":"","Div":"","SlotName":"adunit","AdUnitName":"adunit","Secure":0,"BidFloor":4.3,"BidFloorCur":"USD","IsRewardInventory":null,"Banner":true,"Video":{"mimes":null},"Native":null,"IncomingSlots":["640x480","700x900","728x90","300x250"],"Type":"video","Bidders":{"appnexus":{"PartnerID":2,"PrebidBidderCode":"appnexus","MatchedSlot":"adunit@700x900","KGP":"_AU_@_W_x_H_","KGPV":"","IsRegex":false,"Params":{"placementId":0,"site":"12313","adtag":"45343"},"VASTTagFlags":null}},"NonMapped":{},"NewExt":{"data":{"pbadslot":"adunit"},"prebid":{"bidder":{"appnexus":{"placementId":0,"site":"12313","adtag":"45343"}}}},"BidCtx":{},"BannerAdUnitCtx":{"MatchedSlot":"","IsRegex":false,"MatchedRegex":"","SelectedSlotAdUnitConfig":null,"AppliedSlotAdUnitConfig":null,"UsingDefaultConfig":false,"AllowedConnectionTypes":null},"VideoAdUnitCtx":{"MatchedSlot":"adunit","IsRegex":false,"MatchedRegex":"","SelectedSlotAdUnitConfig":{"video":{"enabled":true,"amptrafficpercentage":100,"config":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480}}},"AppliedSlotAdUnitConfig":{"video":{"enabled":true,"amptrafficpercentage":100,"config":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480}}},"UsingDefaultConfig":false,"AllowedConnectionTypes":null},"NativeAdUnitCtx":{"MatchedSlot":"adunit","IsRegex":false,"MatchedRegex":"","SelectedSlotAdUnitConfig":{"video":{"enabled":true,"amptrafficpercentage":100,"config":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480}}},"AppliedSlotAdUnitConfig":{"video":{"enabled":true,"amptrafficpercentage":100,"config":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480}}},"UsingDefaultConfig":false,"AllowedConnectionTypes":null},"BidderError":"","IsAdPodRequest":false,"AdpodConfig":null,"ImpAdPodCfg":null,"BidIDToAPRC":null,"AdserverURL":"","BidIDToDur":null}}`, `new request.ext: {"prebid":{"bidadjustmentfactors":{"appnexus":1},"bidderparams":{"pubmatic":{"wiid":""}},"debug":true,"floors":{"enforcement":{"enforcepbs":true},"enabled":true},"targeting":{"pricegranularity":{"precision":2,"ranges":[{"min":0,"max":5,"increment":0.05},{"min":5,"max":10,"increment":0.1},{"min":10,"max":20,"increment":0.5}]},"mediatypepricegranularity":{},"includewinners":true,"includebidderkeys":true},"macros":{"[PLATFORM]":"2","[PROFILE_ID]":"1234","[PROFILE_VERSION]":"1","[UNIX_TIMESTAMP]":"0","[WRAPPER_IMPRESSION_ID]":""}}}`},
				},
				bidRequest:            json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","banner":{"format":[{"w":728,"h":90},{"w":300,"h":250}],"w":700,"h":900},"video":{"mimes":["video/mp4","video/mpeg"],"maxduration":30,"startdelay":0,"protocols":[1,2,3,4,5,6,7,8,11,12,13,14],"w":640,"h":480,"placement":2,"plcmt":4,"linearity":1,"skip":0,"playbackmethod":[2],"playbackend":1,"delivery":[2,3]},"tagid":"adunit","bidfloor":4.3,"bidfloorcur":"USD","ext":{"data":{"pbadslot":"adunit"},"prebid":{"bidder":{"appnexus":{"placementId":0,"site":"12313","adtag":"45343"}}}}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","customdata":"7D75D25F-FAC9-443D-B2D1-B17FEE11E027","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"tid":"123-456-789","ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}},"ext":{"prebid":{"bidadjustmentfactors":{"appnexus":1},"bidderparams":{"pubmatic":{"wiid":""}},"debug":true,"floors":{"enforcement":{"enforcepbs":true},"enabled":true},"targeting":{"pricegranularity":{"precision":2,"ranges":[{"min":0,"max":5,"increment":0.05},{"min":5,"max":10,"increment":0.1},{"min":10,"max":20,"increment":0.5}]},"includewinners":true,"includebidderkeys":true},"macros":{"[PLATFORM]":"2","[PROFILE_ID]":"1234","[PROFILE_VERSION]":"1","[UNIX_TIMESTAMP]":"0","[WRAPPER_IMPRESSION_ID]":""}}}}`),
				error:                 false,
				doMutate:              true,
				nilCurrencyConversion: false,
			},
		},
		{
			name: "Read_adpod_configs_from_UI_and_update_request_impressions_with_configs",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							PubID:        5890,
							PubIDStr:     "5890",
							ProfileID:    4444,
							ProfileIDStr: "4444",
							DisplayID:    1,
							SSAuction:    1,
							Platform:     "video",
							Debug:        true,
							DeviceCtx:    models.DeviceCtx{IP: "127.0.0.1", UA: "go-test"},
							IsCTVRequest: true,
							UidCookie: &http.Cookie{
								Name:  "uids",
								Value: `eyJ0ZW1wVUlEcyI6eyIzM2Fjcm9zcyI6eyJ1aWQiOiIxMTkxNzkxMDk5Nzc2NjEiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OTo0My4zODg4Nzk5NVoifSwiYWRmIjp7InVpZCI6IjgwNDQ2MDgzMzM3Nzg4MzkwNzgiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxMS4wMzMwNTQ3MjdaIn0sImFka2VybmVsIjp7InVpZCI6IkE5MTYzNTAwNzE0OTkyOTMyOTkwIiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MDkuMzczMzg1NjYyWiJ9LCJhZGtlcm5lbEFkbiI6eyJ1aWQiOiJBOTE2MzUwMDcxNDk5MjkzMjk5MCIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjEzLjQzNDkyNTg5NloifSwiYWRtaXhlciI6eyJ1aWQiOiIzNjZhMTdiMTJmMjI0ZDMwOGYzZTNiOGRhOGMzYzhhMCIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjA5LjU5MjkxNDgwMVoifSwiYWRueHMiOnsidWlkIjoiNDE5Mjg5ODUzMDE0NTExOTMiLCJleHBpcmVzIjoiMjAyMy0wMS0xOFQwOTo1MzowOC44MjU0NDI2NzZaIn0sImFqYSI6eyJ1aWQiOiJzMnN1aWQ2RGVmMFl0bjJveGQ1aG9zS1AxVmV3IiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MTMuMjM5MTc2MDU0WiJ9LCJlcGxhbm5pbmciOnsidWlkIjoiQUoxRjBTOE5qdTdTQ0xWOSIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjEwLjkyOTk2MDQ3M1oifSwiZ2Ftb3NoaSI6eyJ1aWQiOiJndXNyXzM1NmFmOWIxZDhjNjQyYjQ4MmNiYWQyYjdhMjg4MTYxIiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MTAuNTI0MTU3MjI1WiJ9LCJncmlkIjp7InVpZCI6IjRmYzM2MjUwLWQ4NTItNDU5Yy04NzcyLTczNTZkZTE3YWI5NyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjE0LjY5NjMxNjIyN1oifSwiZ3JvdXBtIjp7InVpZCI6IjdENzVEMjVGLUZBQzktNDQzRC1CMkQxLUIxN0ZFRTExRTAyNyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjM5LjIyNjIxMjUzMloifSwiaXgiOnsidWlkIjoiWW9ORlNENlc5QkphOEh6eEdtcXlCUUFBXHUwMDI2Mjk3IiwiZXhwaXJlcyI6IjIwMjMtMDUtMzFUMDc6NTM6MzguNTU1ODI3MzU0WiJ9LCJqaXhpZSI6eyJ1aWQiOiI3MzY3MTI1MC1lODgyLTExZWMtYjUzOC0xM2FjYjdhZjBkZTQiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxMi4xOTEwOTk3MzJaIn0sImxvZ2ljYWQiOnsidWlkIjoiQVZ4OVROQS11c25pa3M4QURzTHpWa3JvaDg4QUFBR0JUREh0UUEiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OTowOS40NTUxNDk2MTZaIn0sIm1lZGlhbmV0Ijp7InVpZCI6IjI5Nzg0MjM0OTI4OTU0MTAwMDBWMTAiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxMy42NzIyMTUxMjhaIn0sIm1naWQiOnsidWlkIjoibTU5Z1hyN0xlX1htIiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MTcuMDk3MDAxNDcxWiJ9LCJuYW5vaW50ZXJhY3RpdmUiOnsidWlkIjoiNmFlYzhjMTAzNzlkY2I3ODQxMmJjODBiNmRkOWM5NzMxNzNhYjdkNzEyZTQzMWE1YTVlYTcwMzRlNTZhNThhMCIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjE2LjcxNDgwNzUwNVoifSwib25ldGFnIjp7InVpZCI6IjdPelZoVzFOeC1LOGFVak1HMG52NXVNYm5YNEFHUXZQbnVHcHFrZ3k0ckEiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OTowOS4xNDE3NDEyNjJaIn0sIm9wZW54Ijp7InVpZCI6IjVkZWNlNjIyLTBhMjMtMGRhYi0zYTI0LTVhNzcwMTBlNDU4MiIsImV4cGlyZXMiOiIyMDIzLTA1LTMxVDA3OjUyOjQ3LjE0MDQxNzM2M1oifSwicHVibWF0aWMiOnsidWlkIjoiN0Q3NUQyNUYtRkFDOS00NDNELUIyRDEtQjE3RkVFMTFFMDI3IiwiZXhwaXJlcyI6IjIwMjItMTAtMzFUMDk6MTQ6MjUuNzM3MjU2ODk5WiJ9LCJyaWNoYXVkaWVuY2UiOnsidWlkIjoiY2I2YzYzMjAtMzNlMi00Nzc0LWIxNjAtMXp6MTY1NDg0MDc0OSIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjA5LjUyNTA3NDE4WiJ9LCJzbWFydHlhZHMiOnsidWlkIjoiMTJhZjE1ZTQ0ZjAwZDA3NjMwZTc0YzQ5MTU0Y2JmYmE0Zjg0N2U4ZDRhMTU0YzhjM2Q1MWY1OGNmNzJhNDYyNyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjEwLjgyNTAzMTg4NFoifSwic21pbGV3YW50ZWQiOnsidWlkIjoiZGQ5YzNmZTE4N2VmOWIwOWNhYTViNzExNDA0YzI4MzAiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxNC4yNTU2MDkzNjNaIn0sInN5bmFjb3JtZWRpYSI6eyJ1aWQiOiJHRFBSIiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MDkuOTc5NTgzNDM4WiJ9LCJ0cmlwbGVsaWZ0Ijp7InVpZCI6IjcwMjE5NzUwNTQ4MDg4NjUxOTQ2MyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjA4Ljk4OTY3MzU3NFoifSwidmFsdWVpbXByZXNzaW9uIjp7InVpZCI6IjlkMDgxNTVmLWQ5ZmUtNGI1OC04OThlLWUyYzU2MjgyYWIzZSIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjA5LjA2NzgzOTE2NFoifSwidmlzeCI6eyJ1aWQiOiIyN2UwYWMzYy1iNDZlLTQxYjMtOTkyYy1mOGQyNzE0OTQ5NWUiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxMi45ODk1MjM1NzNaIn0sInlpZWxkbGFiIjp7InVpZCI6IjY5NzE0ZDlmLWZiMDAtNGE1Zi04MTljLTRiZTE5MTM2YTMyNSIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjExLjMwMzAyNjYxNVoifSwieWllbGRtbyI6eyJ1aWQiOiJnOTZjMmY3MTlmMTU1MWIzMWY2MyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjEwLjExMDUyODYwOVoifSwieWllbGRvbmUiOnsidWlkIjoiMmE0MmZiZDMtMmM3MC00ZWI5LWIxYmQtMDQ2OTY2NTBkOTQ4IiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MTAuMzE4MzMzOTM5WiJ9LCJ6ZXJvY2xpY2tmcmF1ZCI6eyJ1aWQiOiJiOTk5NThmZS0yYTg3LTJkYTQtOWNjNC05NjFmZDExM2JlY2UiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxNS43MTk1OTQ1NjZaIn19LCJiZGF5IjoiMjAyMi0wNS0xN1QwNjo0ODozOC4wMTc5ODgyMDZaIn0=`,
							},
							KADUSERCookie: &http.Cookie{
								Name:  "KADUSERCOOKIE",
								Value: `7D75D25F-FAC9-443D-B2D1-B17FEE11E027`,
							},
							OriginCookie:             "go-test",
							Endpoint:                 models.EndpointJson,
							Method:                   "POST",
							Aliases:                  make(map[string]string),
							ImpBidCtx:                make(map[string]models.ImpCtx),
							PrebidBidderCode:         make(map[string]string),
							BidderResponseTimeMillis: make(map[string]int),
							SeatNonBids:              make(map[string][]openrtb_ext.NonBid),
							MetricsEngine:            mockEngine,
							AdruleFlag:               false,
							AdpodProfileConfig:       nil,
							ImpAdPodConfig:           make(map[string][]models.PodConfig),
							WakandaDebug:             &wakanda.Debug{Enabled: true},
						},
					},
				},
				bidrequest: json.RawMessage(`{"id":"1559039248176","test":0,"at":1,"tmax":500,"source":{"fd":1,"tid":"1559039248176","pchain":"pchaintagid"},"imp":[{"id":"28635736ddc2bb2","ext":{"bidder":{"appnexus":{"keywords":[],"dealtier":{"prefix":"apnx","mindealtier":4}}}},"displaymanager":"PubMaticSDK","displaymanagerver":"PubMaticSDK-1.0","instl":1,"tagid":"adunit","bidfloor":1,"bidfloorcur":"USD","secure":0,"iframebuster":["1"],"exp":1,"video":{"mimes":["video/3gpp","video/mp4"],"minduration":10,"maxduration":40,"protocols":[2,3,5,6],"w":280,"h":360,"startdelay":1,"placement":5,"linearity":1,"skip":1,"skipmin":5,"skipafter":10,"sequence":1,"battr":[5,6,7],"maxextended":10,"minbitrate":1000,"maxbitrate":2000,"playbackmethod":[1],"delivery":[2],"pos":7,"api":[1,2],"companiontype":[1,2,3],"playbackend":1}}],"site":{"id":"123","name":"EbayShopping","domain":"ebay.com","cat":["IAB1-5"],"sectioncat":["IAB1-5"],"pagecat":["IAB1-5"],"page":"http://ebay.com/inte/automation/s2s/pwt_parameter_validation_muti_slot_multi_size.html?appnexus_deal_priority=6&appnexus_video_fixedbid=10&videobid=4&cat=IAB20-3","ref":"http://ebay.com/home","search":"NewCloths","mobile":1,"privacypolicy":1,"keywords":"Cloths","publisher":{"id":"5890","name":"Test Publisher","cat":["IAB1-5"],"domain":"publisher.com"},"content":{"id":"381d2e0b-548d-4f27-bfdd-e6e66f43557e","episode":1,"title":"StarWars","series":"StarWars","season":"Season3","artist":"GeorgeLucas","genre":"Action","album":"Action","isrc":"2","url":"http://www.pubmatic.com/test/","cat":["IAB1-1","IAB1-2"],"prodq":1,"videoquality":1,"context":1,"contentrating":"MPAA","userrating":"9-Stars","qagmediarating":1,"keywords":"ActionMovies","livestream":1,"sourcerelationship":1,"len":12000,"language":"en","embeddable":1,"producer":{"id":"123","name":"GaryKurtz","cat":["IAB1-5","IAB1-6"],"domain":"producer.com"}}},"device":{"ua":"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.169 Safari/537.36","dnt":0,"lmt":0,"ip":"172.16.8.74","ipv6":"2001:db8::8a2e:370:7334","devicetype":1,"make":"Apple","model":"iPhone X","os":"iOS","osv":"10","hwv":"10x","h":768,"w":1366,"ppi":4096,"pxratio":1.3,"js":1,"flashver":"1.1","language":"en","carrier":"VERIZON","mccmnc":"310-005","connectiontype":2,"ifa":"EA7583CD-A667-48BC-B806-42ECB2B48606","didsha1":"EA7583CD-A667-48BC-B806-42ECB2B48606","didmd5":"EA7583CD-A667-48BC-B806-42ECB2B48606","dpidsha1":"EA7583CD-A667-48BC-B806-42ECB2B48606","dpidmd5":"EA7583CD-A667-48BC-B806-42ECB2B48606","macsha1":"EA7583CD-A667-48BC-B806-42ECB2B48606","macmd5":"EA7583CD-A667-48BC-B806-42ECB2B48606"},"user":{"id":"45067fec-eab7-4ca0-ad3a-87b01f21846a","buyeruid":"45067fec-eab7-4ca0-ad3a-87b01f21846a","yob":1990,"gender":"M","keywords":"Movies","customdata":"StarWars"},"regs":{"coppa":0,"ext":{"gdpr":1,"us_privacy":"1YNN"}},"ext":{"wrapper":{"profileid":4444,"versionid":1,"ssauction":1,"sumry_disable":0,"clientconfig":1,"supportdeals":true}}}`),
			},
			fields: fields{
				cache:        mockCache,
				metricEngine: mockEngine,
			},
			setup: func() {
				mockCache.EXPECT().GetMappingsFromCacheV25(gomock.Any(), gomock.Any()).Return(map[string]models.SlotMapping{
					"adunit@0x0": {
						SlotName: "adunit@0x0",
						SlotMappings: map[string]interface{}{
							models.SITE_CACHE_KEY: "12313",
							models.TAG_CACHE_KEY:  "45343",
						},
					},
				})
				mockCache.EXPECT().GetSlotToHashValueMapFromCacheV25(gomock.Any(), gomock.Any()).Return(models.SlotMappingInfo{
					OrderedSlotList: []string{"adunit@0x0"},
					HashValueMap: map[string]string{
						"adunit@0x0": "1232433543534543",
					},
				})
				mockCache.EXPECT().GetPartnerConfigMap(gomock.Any(), gomock.Any(), gomock.Any()).Return(map[int]map[string]string{
					2: {
						models.PARTNER_ID:          "2",
						models.PREBID_PARTNER_NAME: "appnexus",
						models.BidderCode:          "appnexus",
						models.SERVER_SIDE_FLAG:    "1",
						models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
						models.TIMEOUT:             "200",
					},
					-1: {
						models.DisplayVersionID: "1",
						models.PLATFORM_KEY:     models.PLATFORM_VIDEO,
					},
				}, nil)
				mockCache.EXPECT().GetAdunitConfigFromCache(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&adunitconfig.AdUnitConfig{
					ConfigPattern: "_AU_",
					Config: map[string]*adunitconfig.AdConfig{
						"adunit": {
							Video: &adunitconfig.Video{
								Enabled: ptrutil.ToPtr(true),
								Config: &adunitconfig.VideoConfig{
									Video: openrtb2.Video{
										MIMEs: []string{"video/mp4", "video/mpeg"},
										W:     ptrutil.ToPtr[int64](640),
										H:     ptrutil.ToPtr[int64](480),
									},
								},
								UsePodConfig: ptrutil.ToPtr(true),
							},
						},
					},
				})
				mockCache.EXPECT().GetAdpodConfig(gomock.Any(), gomock.Any(), gomock.Any()).Return(&adpodconfig.AdpodConfig{
					Dynamic: []adpodconfig.Dynamic{
						{
							PodDur:      60,
							MaxSeq:      3,
							MinDuration: 20,
							MaxDuration: 30,
						},
					},
					Structured: []adpodconfig.Structured{
						{
							MinDuration: 15,
							MaxDuration: 15,
						},
					},
				}, nil)
				mockCache.EXPECT().GetThrottlePartnersWithCriteria(gomock.Any()).Return(map[string]struct{}{}, nil)
				//prometheus metrics
				mockEngine.EXPECT().RecordPublisherProfileRequests("5890", "4444")
				mockEngine.EXPECT().RecordPreProcessingTimeStats(rctx.PubIDStr, gomock.Any())
				mockEngine.EXPECT().RecordPublisherRequests(models.EndpointJson, "5890", "video")
				mockEngine.EXPECT().RecordPlatformPublisherPartnerReqStats("video", "5890", "appnexus")
				mockEngine.EXPECT().RecordCTVRequests("json", models.PLATFORM_DISPLAY)
				mockEngine.EXPECT().RecordCTVHTTPMethodRequests("json", "5890", "POST")
				mockEngine.EXPECT().RecordVideoInstlImpsStats("5890", "4444")
				mockEngine.EXPECT().RecordReqImpsWithContentCount("5890", models.ContentTypeSite)
				mockEngine.EXPECT().RecordPreProcessingTimeStats(rctx.PubIDStr, gomock.Any())
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false)
				mockFeature.EXPECT().IsAnalyticsTrackingThrottled(gomock.Any(), gomock.Any()).Return(false, false)
				mockProfileMetaData.EXPECT().GetProfileTypePlatform(gomock.Any()).Return(0, false)
				mockFeature.EXPECT().IsDynamicFloorEnabledPublisher(gomock.Any()).Return(false)
			},
			want: want{
				hookResult: hookstage.HookResult[hookstage.BeforeValidationRequestPayload]{
					Reject:        false,
					NbrCode:       0,
					ChangeSet:     hookstage.ChangeSet[hookstage.BeforeValidationRequestPayload]{},
					DebugMessages: []string{`new imp: {"28635736ddc2bb2":{"ImpID":"28635736ddc2bb2","TagID":"adunit","DisplayManager":"PubMaticSDK","DisplayManagerVer":"PubMaticSDK-1.0","Div":"","SlotName":"adunit","AdUnitName":"adunit","Secure":0,"BidFloor":1,"BidFloorCur":"USD","IsRewardInventory":null,"Banner":false,"Video":{"mimes":["video/3gpp","video/mp4"],"minduration":10,"maxduration":40,"startdelay":1,"protocols":[2,3,5,6],"w":280,"h":360,"placement":5,"linearity":1,"skip":1,"skipmin":5,"skipafter":10,"sequence":1,"battr":[5,6,7],"maxextended":10,"minbitrate":1000,"maxbitrate":2000,"playbackmethod":[1],"playbackend":1,"delivery":[2],"pos":7,"api":[1,2],"companiontype":[1,2,3]},"Native":null,"IncomingSlots":["280x360"],"Type":"video","Bidders":{"appnexus":{"PartnerID":2,"PrebidBidderCode":"appnexus","MatchedSlot":"adunit@0x0","KGP":"_AU_@_W_x_H_","KGPV":"","IsRegex":false,"Params":{"placementId":0,"site":"12313","adtag":"45343"},"VASTTagFlags":null}},"NonMapped":{},"NewExt":{"data":{"pbadslot":"adunit"},"prebid":{"bidder":{"appnexus":{"placementId":0,"site":"12313","adtag":"45343"}}}},"BidCtx":{},"BannerAdUnitCtx":{"MatchedSlot":"adunit","IsRegex":false,"MatchedRegex":"","SelectedSlotAdUnitConfig":{"video":{"enabled":true,"config":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"usepodconfig":true}},"AppliedSlotAdUnitConfig":{"video":{"enabled":true,"config":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"usepodconfig":true}},"UsingDefaultConfig":false,"AllowedConnectionTypes":null},"VideoAdUnitCtx":{"MatchedSlot":"adunit","IsRegex":false,"MatchedRegex":"","SelectedSlotAdUnitConfig":{"video":{"enabled":true,"config":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"usepodconfig":true}},"AppliedSlotAdUnitConfig":{"video":{"enabled":true,"config":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"usepodconfig":true}},"UsingDefaultConfig":false,"AllowedConnectionTypes":null},"NativeAdUnitCtx":{"MatchedSlot":"adunit","IsRegex":false,"MatchedRegex":"","SelectedSlotAdUnitConfig":{"video":{"enabled":true,"config":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480}},"adrule":[{"mimes":null,"minduration":20,"maxduration":30,"maxseq":3,"poddur":60,"podid":"dynamic-1"},{"mimes":null,"minduration":15,"maxduration":15,"podid":"structured-1"}]},"AppliedSlotAdUnitConfig":{"video":{"enabled":true,"config":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480}},"adrule":[{"mimes":null,"minduration":20,"maxduration":30,"maxseq":3,"poddur":60,"podid":"dynamic-1"},{"mimes":null,"minduration":15,"maxduration":15,"podid":"structured-1"}]},"UsingDefaultConfig":false,"AllowedConnectionTypes":null},"BidderError":"","IsAdPodRequest":false,"AdpodConfig":null,"ImpAdPodCfg":null,"BidIDToAPRC":null,"AdserverURL":"","BidIDToDur":null}}`, `new request.ext: {"prebid":{"bidadjustmentfactors":{"appnexus":1},"bidderparams":{"pubmatic":{"wiid":""}},"debug":true,"floors":{"enforcement":{"enforcepbs":true},"enabled":true},"targeting":{"pricegranularity":{"precision":2,"ranges":[{"min":0,"max":5,"increment":0.05},{"min":5,"max":10,"increment":0.1},{"min":10,"max":20,"increment":0.5}]},"mediatypepricegranularity":{},"includewinners":true,"includebidderkeys":true,"includebrandcategory":{"primaryadserver":0,"publisher":"","withcategory":false,"translatecategories":false,"skipdedup":true}},"macros":{"[PLATFORM]":"2","[PROFILE_ID]":"4444","[PROFILE_VERSION]":"1","[UNIX_TIMESTAMP]":"0","[WRAPPER_IMPRESSION_ID]":""}}}`},
					AnalyticsTags: hookanalytics.Analytics{},
				},
				bidRequest:            json.RawMessage(`{"id":"1559039248176","imp":[{"id":"28635736ddc2bb2-dynamic-1-0","video":{"mimes":["video/3gpp","video/mp4"],"minduration":20,"maxduration":30,"startdelay":1,"maxseq":3,"poddur":60,"protocols":[2,3,5,6],"w":280,"h":360,"podid":"dynamic-1","placement":5,"plcmt":3,"linearity":1,"skip":1,"skipmin":5,"skipafter":10,"sequence":1,"battr":[5,6,7],"maxextended":10,"minbitrate":1000,"maxbitrate":2000,"playbackmethod":[1],"playbackend":1,"delivery":[2],"pos":7,"api":[1,2],"companiontype":[1,2,3]},"displaymanager":"PubMaticSDK","displaymanagerver":"PubMaticSDK-1.0","instl":1,"tagid":"adunit","bidfloor":1,"bidfloorcur":"USD","secure":0,"iframebuster":["1"],"exp":1,"ext":{"data":{"pbadslot":"adunit"},"prebid":{"bidder":{"appnexus":{"placementId":0,"site":"12313","adtag":"45343"}}}}},{"id":"28635736ddc2bb2-structured-1-1","video":{"mimes":["video/3gpp","video/mp4"],"minduration":15,"maxduration":15,"startdelay":1,"protocols":[2,3,5,6],"w":280,"h":360,"podid":"structured-1","placement":5,"plcmt":3,"linearity":1,"skip":1,"skipmin":5,"skipafter":10,"sequence":1,"battr":[5,6,7],"maxextended":10,"minbitrate":1000,"maxbitrate":2000,"playbackmethod":[1],"playbackend":1,"delivery":[2],"pos":7,"api":[1,2],"companiontype":[1,2,3]},"displaymanager":"PubMaticSDK","displaymanagerver":"PubMaticSDK-1.0","instl":1,"tagid":"adunit","bidfloor":1,"bidfloorcur":"USD","secure":0,"iframebuster":["1"],"exp":1,"ext":{"data":{"pbadslot":"adunit"},"prebid":{"bidder":{"appnexus":{"placementId":0,"site":"12313","adtag":"45343"}}}}}],"site":{"id":"123","name":"EbayShopping","domain":"ebay.com","cat":["IAB1-5"],"sectioncat":["IAB1-5"],"pagecat":["IAB1-5"],"page":"http://ebay.com/inte/automation/s2s/pwt_parameter_validation_muti_slot_multi_size.html?appnexus_deal_priority=6\u0026appnexus_video_fixedbid=10\u0026videobid=4\u0026cat=IAB20-3","ref":"http://ebay.com/home","search":"NewCloths","mobile":1,"privacypolicy":1,"publisher":{"id":"5890","name":"Test Publisher","cat":["IAB1-5"],"domain":"publisher.com"},"content":{"id":"381d2e0b-548d-4f27-bfdd-e6e66f43557e","episode":1,"title":"StarWars","series":"StarWars","season":"Season3","artist":"GeorgeLucas","genre":"Action","album":"Action","isrc":"2","producer":{"id":"123","name":"GaryKurtz","cat":["IAB1-5","IAB1-6"],"domain":"producer.com"},"url":"http://www.pubmatic.com/test/","cat":["IAB1-1","IAB1-2"],"prodq":1,"videoquality":1,"context":1,"contentrating":"MPAA","userrating":"9-Stars","qagmediarating":1,"keywords":"ActionMovies","livestream":1,"sourcerelationship":1,"len":12000,"language":"en","embeddable":1},"keywords":"Cloths"},"device":{"dnt":0,"lmt":0,"ua":"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.169 Safari/537.36","ip":"172.16.8.74","ipv6":"2001:db8::8a2e:370:7334","devicetype":1,"make":"Apple","model":"iPhone X","os":"iOS","osv":"10","hwv":"10x","h":768,"w":1366,"ppi":4096,"pxratio":1.3,"js":1,"flashver":"1.1","language":"en","carrier":"VERIZON","mccmnc":"310-005","connectiontype":2,"ifa":"EA7583CD-A667-48BC-B806-42ECB2B48606","didsha1":"EA7583CD-A667-48BC-B806-42ECB2B48606","didmd5":"EA7583CD-A667-48BC-B806-42ECB2B48606","dpidsha1":"EA7583CD-A667-48BC-B806-42ECB2B48606","dpidmd5":"EA7583CD-A667-48BC-B806-42ECB2B48606","macsha1":"EA7583CD-A667-48BC-B806-42ECB2B48606","macmd5":"EA7583CD-A667-48BC-B806-42ECB2B48606"},"user":{"id":"45067fec-eab7-4ca0-ad3a-87b01f21846a","buyeruid":"45067fec-eab7-4ca0-ad3a-87b01f21846a","yob":1990,"gender":"M","keywords":"Movies","customdata":"StarWars"},"at":1,"tmax":500,"source":{"fd":1,"tid":"1559039248176","pchain":"pchaintagid"},"regs":{"ext":{"gdpr":1,"us_privacy":"1YNN"}},"ext":{"prebid":{"bidadjustmentfactors":{"appnexus":1},"bidderparams":{"pubmatic":{"wiid":""}},"debug":true,"debug_override":true,"floors":{"enforcement":{"enforcepbs":true},"enabled":true},"targeting":{"pricegranularity":{"precision":2,"ranges":[{"min":0,"max":5,"increment":0.05},{"min":5,"max":10,"increment":0.1},{"min":10,"max":20,"increment":0.5}]},"includewinners":true,"includebidderkeys":true,"includebrandcategory":{"withcategory":false,"translatecategories":false,"skipdedup":true}},"macros":{"[PLATFORM]":"2","[PROFILE_ID]":"4444","[PROFILE_VERSION]":"1","[UNIX_TIMESTAMP]":"0","[WRAPPER_IMPRESSION_ID]":""}}}}`),
				error:                 false,
				nilCurrencyConversion: false,
				doMutate:              true,
			},
		},
		{
			name: "Read_adpod_configs_from_Adrule_and_update_request_impressions_with_configs",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							PubID:        5890,
							PubIDStr:     "5890",
							ProfileID:    4444,
							ProfileIDStr: "4444",
							DisplayID:    1,
							SSAuction:    1,
							Platform:     "video",
							Debug:        true,
							DeviceCtx:    models.DeviceCtx{IP: "127.0.0.1", UA: "go-test"},
							IsCTVRequest: true,
							UidCookie: &http.Cookie{
								Name:  "uids",
								Value: `eyJ0ZW1wVUlEcyI6eyIzM2Fjcm9zcyI6eyJ1aWQiOiIxMTkxNzkxMDk5Nzc2NjEiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OTo0My4zODg4Nzk5NVoifSwiYWRmIjp7InVpZCI6IjgwNDQ2MDgzMzM3Nzg4MzkwNzgiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxMS4wMzMwNTQ3MjdaIn0sImFka2VybmVsIjp7InVpZCI6IkE5MTYzNTAwNzE0OTkyOTMyOTkwIiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MDkuMzczMzg1NjYyWiJ9LCJhZGtlcm5lbEFkbiI6eyJ1aWQiOiJBOTE2MzUwMDcxNDk5MjkzMjk5MCIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjEzLjQzNDkyNTg5NloifSwiYWRtaXhlciI6eyJ1aWQiOiIzNjZhMTdiMTJmMjI0ZDMwOGYzZTNiOGRhOGMzYzhhMCIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjA5LjU5MjkxNDgwMVoifSwiYWRueHMiOnsidWlkIjoiNDE5Mjg5ODUzMDE0NTExOTMiLCJleHBpcmVzIjoiMjAyMy0wMS0xOFQwOTo1MzowOC44MjU0NDI2NzZaIn0sImFqYSI6eyJ1aWQiOiJzMnN1aWQ2RGVmMFl0bjJveGQ1aG9zS1AxVmV3IiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MTMuMjM5MTc2MDU0WiJ9LCJlcGxhbm5pbmciOnsidWlkIjoiQUoxRjBTOE5qdTdTQ0xWOSIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjEwLjkyOTk2MDQ3M1oifSwiZ2Ftb3NoaSI6eyJ1aWQiOiJndXNyXzM1NmFmOWIxZDhjNjQyYjQ4MmNiYWQyYjdhMjg4MTYxIiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MTAuNTI0MTU3MjI1WiJ9LCJncmlkIjp7InVpZCI6IjRmYzM2MjUwLWQ4NTItNDU5Yy04NzcyLTczNTZkZTE3YWI5NyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjE0LjY5NjMxNjIyN1oifSwiZ3JvdXBtIjp7InVpZCI6IjdENzVEMjVGLUZBQzktNDQzRC1CMkQxLUIxN0ZFRTExRTAyNyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjM5LjIyNjIxMjUzMloifSwiaXgiOnsidWlkIjoiWW9ORlNENlc5QkphOEh6eEdtcXlCUUFBXHUwMDI2Mjk3IiwiZXhwaXJlcyI6IjIwMjMtMDUtMzFUMDc6NTM6MzguNTU1ODI3MzU0WiJ9LCJqaXhpZSI6eyJ1aWQiOiI3MzY3MTI1MC1lODgyLTExZWMtYjUzOC0xM2FjYjdhZjBkZTQiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxMi4xOTEwOTk3MzJaIn0sImxvZ2ljYWQiOnsidWlkIjoiQVZ4OVROQS11c25pa3M4QURzTHpWa3JvaDg4QUFBR0JUREh0UUEiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OTowOS40NTUxNDk2MTZaIn0sIm1lZGlhbmV0Ijp7InVpZCI6IjI5Nzg0MjM0OTI4OTU0MTAwMDBWMTAiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxMy42NzIyMTUxMjhaIn0sIm1naWQiOnsidWlkIjoibTU5Z1hyN0xlX1htIiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MTcuMDk3MDAxNDcxWiJ9LCJuYW5vaW50ZXJhY3RpdmUiOnsidWlkIjoiNmFlYzhjMTAzNzlkY2I3ODQxMmJjODBiNmRkOWM5NzMxNzNhYjdkNzEyZTQzMWE1YTVlYTcwMzRlNTZhNThhMCIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjE2LjcxNDgwNzUwNVoifSwib25ldGFnIjp7InVpZCI6IjdPelZoVzFOeC1LOGFVak1HMG52NXVNYm5YNEFHUXZQbnVHcHFrZ3k0ckEiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OTowOS4xNDE3NDEyNjJaIn0sIm9wZW54Ijp7InVpZCI6IjVkZWNlNjIyLTBhMjMtMGRhYi0zYTI0LTVhNzcwMTBlNDU4MiIsImV4cGlyZXMiOiIyMDIzLTA1LTMxVDA3OjUyOjQ3LjE0MDQxNzM2M1oifSwicHVibWF0aWMiOnsidWlkIjoiN0Q3NUQyNUYtRkFDOS00NDNELUIyRDEtQjE3RkVFMTFFMDI3IiwiZXhwaXJlcyI6IjIwMjItMTAtMzFUMDk6MTQ6MjUuNzM3MjU2ODk5WiJ9LCJyaWNoYXVkaWVuY2UiOnsidWlkIjoiY2I2YzYzMjAtMzNlMi00Nzc0LWIxNjAtMXp6MTY1NDg0MDc0OSIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjA5LjUyNTA3NDE4WiJ9LCJzbWFydHlhZHMiOnsidWlkIjoiMTJhZjE1ZTQ0ZjAwZDA3NjMwZTc0YzQ5MTU0Y2JmYmE0Zjg0N2U4ZDRhMTU0YzhjM2Q1MWY1OGNmNzJhNDYyNyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjEwLjgyNTAzMTg4NFoifSwic21pbGV3YW50ZWQiOnsidWlkIjoiZGQ5YzNmZTE4N2VmOWIwOWNhYTViNzExNDA0YzI4MzAiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxNC4yNTU2MDkzNjNaIn0sInN5bmFjb3JtZWRpYSI6eyJ1aWQiOiJHRFBSIiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MDkuOTc5NTgzNDM4WiJ9LCJ0cmlwbGVsaWZ0Ijp7InVpZCI6IjcwMjE5NzUwNTQ4MDg4NjUxOTQ2MyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjA4Ljk4OTY3MzU3NFoifSwidmFsdWVpbXByZXNzaW9uIjp7InVpZCI6IjlkMDgxNTVmLWQ5ZmUtNGI1OC04OThlLWUyYzU2MjgyYWIzZSIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjA5LjA2NzgzOTE2NFoifSwidmlzeCI6eyJ1aWQiOiIyN2UwYWMzYy1iNDZlLTQxYjMtOTkyYy1mOGQyNzE0OTQ5NWUiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxMi45ODk1MjM1NzNaIn0sInlpZWxkbGFiIjp7InVpZCI6IjY5NzE0ZDlmLWZiMDAtNGE1Zi04MTljLTRiZTE5MTM2YTMyNSIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjExLjMwMzAyNjYxNVoifSwieWllbGRtbyI6eyJ1aWQiOiJnOTZjMmY3MTlmMTU1MWIzMWY2MyIsImV4cGlyZXMiOiIyMDIyLTA2LTI0VDA1OjU5OjEwLjExMDUyODYwOVoifSwieWllbGRvbmUiOnsidWlkIjoiMmE0MmZiZDMtMmM3MC00ZWI5LWIxYmQtMDQ2OTY2NTBkOTQ4IiwiZXhwaXJlcyI6IjIwMjItMDYtMjRUMDU6NTk6MTAuMzE4MzMzOTM5WiJ9LCJ6ZXJvY2xpY2tmcmF1ZCI6eyJ1aWQiOiJiOTk5NThmZS0yYTg3LTJkYTQtOWNjNC05NjFmZDExM2JlY2UiLCJleHBpcmVzIjoiMjAyMi0wNi0yNFQwNTo1OToxNS43MTk1OTQ1NjZaIn19LCJiZGF5IjoiMjAyMi0wNS0xN1QwNjo0ODozOC4wMTc5ODgyMDZaIn0=`,
							},
							KADUSERCookie: &http.Cookie{
								Name:  "KADUSERCOOKIE",
								Value: `7D75D25F-FAC9-443D-B2D1-B17FEE11E027`,
							},
							OriginCookie:             "go-test",
							Endpoint:                 models.EndpointJson,
							Method:                   "POST",
							Aliases:                  make(map[string]string),
							ImpBidCtx:                make(map[string]models.ImpCtx),
							PrebidBidderCode:         make(map[string]string),
							BidderResponseTimeMillis: make(map[string]int),
							SeatNonBids:              make(map[string][]openrtb_ext.NonBid),
							MetricsEngine:            mockEngine,
							AdruleFlag:               true,
							AdpodProfileConfig:       nil,
							ImpAdPodConfig:           make(map[string][]models.PodConfig),
							WakandaDebug:             &wakanda.Debug{},
						},
					},
				},
				bidrequest: json.RawMessage(`{"id":"1559039248176","test":0,"at":1,"tmax":500,"source":{"fd":1,"tid":"1559039248176","pchain":"pchaintagid"},"imp":[{"id":"28635736ddc2bb2","ext":{"bidder":{"appnexus":{"keywords":[],"dealtier":{"prefix":"apnx","mindealtier":4}}}},"displaymanager":"PubMaticSDK","displaymanagerver":"PubMaticSDK-1.0","instl":1,"tagid":"adunit","bidfloor":1,"bidfloorcur":"USD","secure":0,"iframebuster":["1"],"exp":1,"video":{"mimes":["video/3gpp","video/mp4"],"minduration":10,"maxduration":40,"protocols":[2,3,5,6],"w":280,"h":360,"startdelay":1,"placement":5,"linearity":1,"skip":1,"skipmin":5,"skipafter":10,"sequence":1,"battr":[5,6,7],"maxextended":10,"minbitrate":1000,"maxbitrate":2000,"playbackmethod":[1],"delivery":[2],"pos":7,"api":[1,2],"companiontype":[1,2,3],"playbackend":1}}],"site":{"id":"123","name":"EbayShopping","domain":"ebay.com","cat":["IAB1-5"],"sectioncat":["IAB1-5"],"pagecat":["IAB1-5"],"page":"http://ebay.com/inte/automation/s2s/pwt_parameter_validation_muti_slot_multi_size.html?appnexus_deal_priority=6&appnexus_video_fixedbid=10&videobid=4&cat=IAB20-3","ref":"http://ebay.com/home","search":"NewCloths","mobile":1,"privacypolicy":1,"keywords":"Cloths","publisher":{"id":"5890","name":"Test Publisher","cat":["IAB1-5"],"domain":"publisher.com"},"content":{"id":"381d2e0b-548d-4f27-bfdd-e6e66f43557e","episode":1,"title":"StarWars","series":"StarWars","season":"Season3","artist":"GeorgeLucas","genre":"Action","album":"Action","isrc":"2","url":"http://www.pubmatic.com/test/","cat":["IAB1-1","IAB1-2"],"prodq":1,"videoquality":1,"context":1,"contentrating":"MPAA","userrating":"9-Stars","qagmediarating":1,"keywords":"ActionMovies","livestream":1,"sourcerelationship":1,"len":12000,"language":"en","embeddable":1,"producer":{"id":"123","name":"GaryKurtz","cat":["IAB1-5","IAB1-6"],"domain":"producer.com"}}},"device":{"ua":"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.169 Safari/537.36","dnt":0,"lmt":0,"ip":"172.16.8.74","ipv6":"2001:db8::8a2e:370:7334","devicetype":1,"make":"Apple","model":"iPhone X","os":"iOS","osv":"10","hwv":"10x","h":768,"w":1366,"ppi":4096,"pxratio":1.3,"js":1,"flashver":"1.1","language":"en","carrier":"VERIZON","mccmnc":"310-005","connectiontype":2,"ifa":"EA7583CD-A667-48BC-B806-42ECB2B48606","didsha1":"EA7583CD-A667-48BC-B806-42ECB2B48606","didmd5":"EA7583CD-A667-48BC-B806-42ECB2B48606","dpidsha1":"EA7583CD-A667-48BC-B806-42ECB2B48606","dpidmd5":"EA7583CD-A667-48BC-B806-42ECB2B48606","macsha1":"EA7583CD-A667-48BC-B806-42ECB2B48606","macmd5":"EA7583CD-A667-48BC-B806-42ECB2B48606"},"user":{"id":"45067fec-eab7-4ca0-ad3a-87b01f21846a","buyeruid":"45067fec-eab7-4ca0-ad3a-87b01f21846a","yob":1990,"gender":"M","keywords":"Movies","customdata":"StarWars"},"regs":{"coppa":0,"ext":{"gdpr":1,"us_privacy":"1YNN"}},"ext":{"wrapper":{"profileid":4444,"versionid":1,"ssauction":1,"sumry_disable":0,"clientconfig":1,"supportdeals":true}}}`),
			},
			fields: fields{
				cache:        mockCache,
				metricEngine: mockEngine,
			},
			setup: func() {
				mockCache.EXPECT().GetMappingsFromCacheV25(gomock.Any(), gomock.Any()).Return(map[string]models.SlotMapping{
					"adunit@0x0": {
						SlotName: "adunit@0x0",
						SlotMappings: map[string]interface{}{
							models.SITE_CACHE_KEY: "12313",
							models.TAG_CACHE_KEY:  "45343",
						},
					},
				})
				mockCache.EXPECT().GetSlotToHashValueMapFromCacheV25(gomock.Any(), gomock.Any()).Return(models.SlotMappingInfo{
					OrderedSlotList: []string{"adunit@0x0"},
					HashValueMap: map[string]string{
						"adunit@0x0": "1232433543534543",
					},
				})
				mockCache.EXPECT().GetPartnerConfigMap(gomock.Any(), gomock.Any(), gomock.Any()).Return(map[int]map[string]string{
					2: {
						models.PARTNER_ID:          "2",
						models.PREBID_PARTNER_NAME: "appnexus",
						models.BidderCode:          "appnexus",
						models.SERVER_SIDE_FLAG:    "1",
						models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
						models.TIMEOUT:             "200",
					},
					-1: {
						models.DisplayVersionID: "1",
						models.PLATFORM_KEY:     models.PLATFORM_VIDEO,
					},
				}, nil)
				mockCache.EXPECT().GetAdunitConfigFromCache(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&adunitconfig.AdUnitConfig{
					ConfigPattern: "_AU_",
					Config: map[string]*adunitconfig.AdConfig{
						"adunit": {
							Video: &adunitconfig.Video{
								Enabled: ptrutil.ToPtr(true),
								Config: &adunitconfig.VideoConfig{
									Video: openrtb2.Video{
										MIMEs: []string{"video/mp4", "video/mpeg"},
										W:     ptrutil.ToPtr[int64](640),
										H:     ptrutil.ToPtr[int64](480),
									},
								},
							},
							Adrule: []*openrtb2.Video{
								{
									PodID:       "dynamic-1",
									PodDur:      60,
									MaxSeq:      3,
									MinDuration: 20,
									MaxDuration: 30,
								},
								{
									PodID:       "structured-1",
									MinDuration: 15,
									MaxDuration: 15,
								},
							},
						},
					},
				})
				mockCache.EXPECT().GetAdpodConfig(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
				mockCache.EXPECT().GetThrottlePartnersWithCriteria(gomock.Any()).Return(map[string]struct{}{}, nil)
				//prometheus metrics
				mockEngine.EXPECT().RecordPublisherProfileRequests("5890", "4444")
				mockEngine.EXPECT().RecordPublisherRequests(models.EndpointJson, "5890", "video")
				mockEngine.EXPECT().RecordPlatformPublisherPartnerReqStats("video", "5890", "appnexus")
				mockEngine.EXPECT().RecordCTVRequests("json", models.PLATFORM_DISPLAY)
				mockEngine.EXPECT().RecordCTVHTTPMethodRequests("json", "5890", "POST")
				mockEngine.EXPECT().RecordVideoInstlImpsStats("5890", "4444")
				mockEngine.EXPECT().RecordReqImpsWithContentCount("5890", models.ContentTypeSite)
				mockEngine.EXPECT().RecordPreProcessingTimeStats(rctx.PubIDStr, gomock.Any())
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false)
				mockFeature.EXPECT().IsAnalyticsTrackingThrottled(gomock.Any(), gomock.Any()).Return(false, false)
				mockProfileMetaData.EXPECT().GetProfileTypePlatform(gomock.Any()).Return(0, false)
				mockFeature.EXPECT().IsDynamicFloorEnabledPublisher(gomock.Any()).Return(false)
			},
			want: want{
				hookResult: hookstage.HookResult[hookstage.BeforeValidationRequestPayload]{
					Reject:        false,
					NbrCode:       0,
					ChangeSet:     hookstage.ChangeSet[hookstage.BeforeValidationRequestPayload]{},
					DebugMessages: []string{`new imp: {"28635736ddc2bb2":{"AdpodConfig":null,"AdserverURL":"","AdUnitName":"adunit","Banner":false,"BannerAdUnitCtx":{"AllowedConnectionTypes":null,"AppliedSlotAdUnitConfig":{"adrule":[{"maxduration":30,"maxseq":3,"mimes":null,"minduration":20,"poddur":60,"podid":"dynamic-1"},{"maxduration":15,"mimes":null,"minduration":15,"podid":"structured-1"}],"video":{"config":{"h":480,"mimes":["video/mp4","video/mpeg"],"w":640},"enabled":true}},"IsRegex":false,"MatchedRegex":"","MatchedSlot":"adunit","SelectedSlotAdUnitConfig":{"adrule":[{"maxduration":30,"maxseq":3,"mimes":null,"minduration":20,"poddur":60,"podid":"dynamic-1"},{"maxduration":15,"mimes":null,"minduration":15,"podid":"structured-1"}],"video":{"config":{"h":480,"mimes":["video/mp4","video/mpeg"],"w":640},"enabled":true}},"UsingDefaultConfig":false},"BidCtx":{},"BidderError":"","Bidders":{"appnexus":{"IsRegex":false,"KGP":"_AU_@_W_x_H_","KGPV":"","MatchedSlot":"adunit@0x0","Params":{"adtag":"45343","placementId":0,"site":"12313"},"PartnerID":2,"PrebidBidderCode":"appnexus","VASTTagFlags":null}},"BidFloor":1,"BidFloorCur":"USD","BidIDToAPRC":null,"BidIDToDur":null,"Div":"","ImpAdPodCfg":null,"ImpID":"28635736ddc2bb2","IncomingSlots":["280x360"],"IsAdPodRequest":false,"IsRewardInventory":null,"Native":null,"NewExt":{"data":{"pbadslot":"adunit"},"prebid":{"bidder":{"appnexus":{"adtag":"45343","placementId":0,"site":"12313"}}}},"NonMapped":{},"Secure":0,"SlotName":"adunit","TagID":"adunit","DisplayManager":"PubMaticSDK","DisplayManagerVer":"PubMaticSDK-1.0","Type":"video","Video":{"api":[1,2],"battr":[5,6,7],"companiontype":[1,2,3],"delivery":[2],"h":360,"linearity":1,"maxbitrate":2000,"maxduration":40,"maxextended":10,"mimes":["video/3gpp","video/mp4"],"minbitrate":1000,"minduration":10,"placement":5,"playbackend":1,"playbackmethod":[1],"pos":7,"protocols":[2,3,5,6],"sequence":1,"skip":1,"skipafter":10,"skipmin":5,"startdelay":1,"w":280},"VideoAdUnitCtx":{"AllowedConnectionTypes":null,"AppliedSlotAdUnitConfig":{"adrule":[{"maxduration":30,"maxseq":3,"mimes":null,"minduration":20,"poddur":60,"podid":"dynamic-1"},{"maxduration":15,"mimes":null,"minduration":15,"podid":"structured-1"}],"video":{"config":{"h":480,"mimes":["video/mp4","video/mpeg"],"w":640},"enabled":true}},"IsRegex":false,"MatchedRegex":"","MatchedSlot":"adunit","SelectedSlotAdUnitConfig":{"adrule":[{"maxduration":30,"maxseq":3,"mimes":null,"minduration":20,"poddur":60,"podid":"dynamic-1"},{"maxduration":15,"mimes":null,"minduration":15,"podid":"structured-1"}],"video":{"config":{"h":480,"mimes":["video/mp4","video/mpeg"],"w":640},"enabled":true}},"UsingDefaultConfig":false},"NativeAdUnitCtx":{"MatchedSlot":"adunit","IsRegex":false,"MatchedRegex":"","SelectedSlotAdUnitConfig":{"video":{"enabled":true,"config":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480}},"adrule":[{"mimes":null,"minduration":20,"maxduration":30,"maxseq":3,"poddur":60,"podid":"dynamic-1"},{"mimes":null,"minduration":15,"maxduration":15,"podid":"structured-1"}]},"AppliedSlotAdUnitConfig":{"video":{"enabled":true,"config":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480}},"adrule":[{"mimes":null,"minduration":20,"maxduration":30,"maxseq":3,"poddur":60,"podid":"dynamic-1"},{"mimes":null,"minduration":15,"maxduration":15,"podid":"structured-1"}]},"UsingDefaultConfig":false,"AllowedConnectionTypes":null}}}`, `new request.ext: {"prebid":{"bidadjustmentfactors":{"appnexus":1},"bidderparams":{"pubmatic":{"wiid":""}},"debug":true,"floors":{"enforcement":{"enforcepbs":true},"enabled":true},"targeting":{"pricegranularity":{"precision":2,"ranges":[{"min":0,"max":5,"increment":0.05},{"min":5,"max":10,"increment":0.1},{"min":10,"max":20,"increment":0.5}]},"mediatypepricegranularity":{},"includewinners":true,"includebidderkeys":true,"includebrandcategory":{"primaryadserver":0,"publisher":"","withcategory":false,"translatecategories":false,"skipdedup":true}},"macros":{"[PLATFORM]":"2","[PROFILE_ID]":"4444","[PROFILE_VERSION]":"1","[UNIX_TIMESTAMP]":"0","[WRAPPER_IMPRESSION_ID]":""}}}`},
				},
				bidRequest:            json.RawMessage(`{"id":"1559039248176","imp":[{"id":"28635736ddc2bb2-dynamic-1-0","video":{"mimes":["video/3gpp","video/mp4"],"minduration":20,"maxduration":30,"startdelay":1,"maxseq":3,"poddur":60,"protocols":[2,3,5,6],"w":280,"h":360,"podid":"dynamic-1","placement":5,"plcmt":3,"linearity":1,"skip":1,"skipmin":5,"skipafter":10,"sequence":1,"battr":[5,6,7],"maxextended":10,"minbitrate":1000,"maxbitrate":2000,"playbackmethod":[1],"playbackend":1,"delivery":[2],"pos":7,"api":[1,2],"companiontype":[1,2,3]},"displaymanager":"PubMaticSDK","displaymanagerver":"PubMaticSDK-1.0","instl":1,"tagid":"adunit","bidfloor":1,"bidfloorcur":"USD","secure":0,"iframebuster":["1"],"exp":1,"ext":{"data":{"pbadslot":"adunit"},"prebid":{"bidder":{"appnexus":{"placementId":0,"site":"12313","adtag":"45343"}}}}},{"id":"28635736ddc2bb2-structured-1-1","video":{"mimes":["video/3gpp","video/mp4"],"minduration":15,"maxduration":15,"startdelay":1,"protocols":[2,3,5,6],"w":280,"h":360,"podid":"structured-1","placement":5,"plcmt":3,"linearity":1,"skip":1,"skipmin":5,"skipafter":10,"sequence":1,"battr":[5,6,7],"maxextended":10,"minbitrate":1000,"maxbitrate":2000,"playbackmethod":[1],"playbackend":1,"delivery":[2],"pos":7,"api":[1,2],"companiontype":[1,2,3]},"displaymanager":"PubMaticSDK","displaymanagerver":"PubMaticSDK-1.0","instl":1,"tagid":"adunit","bidfloor":1,"bidfloorcur":"USD","secure":0,"iframebuster":["1"],"exp":1,"ext":{"data":{"pbadslot":"adunit"},"prebid":{"bidder":{"appnexus":{"placementId":0,"site":"12313","adtag":"45343"}}}}}],"site":{"id":"123","name":"EbayShopping","domain":"ebay.com","cat":["IAB1-5"],"sectioncat":["IAB1-5"],"pagecat":["IAB1-5"],"page":"http://ebay.com/inte/automation/s2s/pwt_parameter_validation_muti_slot_multi_size.html?appnexus_deal_priority=6\u0026appnexus_video_fixedbid=10\u0026videobid=4\u0026cat=IAB20-3","ref":"http://ebay.com/home","search":"NewCloths","mobile":1,"privacypolicy":1,"publisher":{"id":"5890","name":"Test Publisher","cat":["IAB1-5"],"domain":"publisher.com"},"content":{"id":"381d2e0b-548d-4f27-bfdd-e6e66f43557e","episode":1,"title":"StarWars","series":"StarWars","season":"Season3","artist":"GeorgeLucas","genre":"Action","album":"Action","isrc":"2","producer":{"id":"123","name":"GaryKurtz","cat":["IAB1-5","IAB1-6"],"domain":"producer.com"},"url":"http://www.pubmatic.com/test/","cat":["IAB1-1","IAB1-2"],"prodq":1,"videoquality":1,"context":1,"contentrating":"MPAA","userrating":"9-Stars","qagmediarating":1,"keywords":"ActionMovies","livestream":1,"sourcerelationship":1,"len":12000,"language":"en","embeddable":1},"keywords":"Cloths"},"device":{"dnt":0,"lmt":0,"ua":"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.169 Safari/537.36","ip":"172.16.8.74","ipv6":"2001:db8::8a2e:370:7334","devicetype":1,"make":"Apple","model":"iPhone X","os":"iOS","osv":"10","hwv":"10x","h":768,"w":1366,"ppi":4096,"pxratio":1.3,"js":1,"flashver":"1.1","language":"en","carrier":"VERIZON","mccmnc":"310-005","connectiontype":2,"ifa":"EA7583CD-A667-48BC-B806-42ECB2B48606","didsha1":"EA7583CD-A667-48BC-B806-42ECB2B48606","didmd5":"EA7583CD-A667-48BC-B806-42ECB2B48606","dpidsha1":"EA7583CD-A667-48BC-B806-42ECB2B48606","dpidmd5":"EA7583CD-A667-48BC-B806-42ECB2B48606","macsha1":"EA7583CD-A667-48BC-B806-42ECB2B48606","macmd5":"EA7583CD-A667-48BC-B806-42ECB2B48606"},"user":{"id":"45067fec-eab7-4ca0-ad3a-87b01f21846a","buyeruid":"45067fec-eab7-4ca0-ad3a-87b01f21846a","yob":1990,"gender":"M","keywords":"Movies","customdata":"StarWars"},"at":1,"tmax":500,"source":{"fd":1,"tid":"1559039248176","pchain":"pchaintagid"},"regs":{"ext":{"gdpr":1,"us_privacy":"1YNN"}},"ext":{"prebid":{"bidadjustmentfactors":{"appnexus":1},"bidderparams":{"pubmatic":{"wiid":""}},"debug":true,"floors":{"enforcement":{"enforcepbs":true},"enabled":true},"targeting":{"pricegranularity":{"precision":2,"ranges":[{"min":0,"max":5,"increment":0.05},{"min":5,"max":10,"increment":0.1},{"min":10,"max":20,"increment":0.5}]},"includewinners":true,"includebidderkeys":true,"includebrandcategory":{"withcategory":false,"translatecategories":false,"skipdedup":true}},"macros":{"[PLATFORM]":"2","[PROFILE_ID]":"4444","[PROFILE_VERSION]":"1","[UNIX_TIMESTAMP]":"0","[WRAPPER_IMPRESSION_ID]":""}}}}`),
				error:                 false,
				nilCurrencyConversion: false,
				doMutate:              true,
			},
		},
		{
			name: "valid_request_for_applovinmax",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": func() models.RequestCtx {
							testrctx := rctx
							testrctx.Endpoint = models.EndpointAppLovinMax
							return testrctx
						}(),
					},
				},
				bidrequest: json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","banner":{"format":[{"w":728,"h":90},{"w":300,"h":250}],"w":700,"h":900},"video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"tagid":"adunit","bidfloor":4.3,"bidfloorcur":"USD","ext":{"bidder":{"pubmatic":{"keywords":[{"key":"pmzoneid","value":["val1","val2"]}]}},"prebid":{}}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}},"ext":{"prebid":{},"wrapper":{"test":123,"profileid":123,"versionid":1,"wiid":"test_display_wiid"}}}`),
			},
			fields: fields{
				cache:        mockCache,
				metricEngine: mockEngine,
			},
			setup: func() {
				mockCache.EXPECT().GetMappingsFromCacheV25(gomock.Any(), gomock.Any()).Return(map[string]models.SlotMapping{
					"adunit@700x900": {
						SlotName: "adunit@700x900",
						SlotMappings: map[string]interface{}{
							models.SITE_CACHE_KEY: "12313",
							models.TAG_CACHE_KEY:  "45343",
						},
					},
				})
				mockCache.EXPECT().GetSlotToHashValueMapFromCacheV25(gomock.Any(), gomock.Any()).Return(models.SlotMappingInfo{
					OrderedSlotList: []string{"adunit@700x900"},
					HashValueMap: map[string]string{
						"adunit@700x900": "1232433543534543",
					},
				})
				mockCache.EXPECT().GetPartnerConfigMap(gomock.Any(), gomock.Any(), gomock.Any()).Return(map[int]map[string]string{
					2: {
						models.PARTNER_ID:          "2",
						models.PREBID_PARTNER_NAME: "appnexus",
						models.BidderCode:          "appnexus",
						models.SERVER_SIDE_FLAG:    "1",
						models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
						models.TIMEOUT:             "200",
					},
					-1: {
						models.DisplayVersionID:   "1",
						models.PLATFORM_KEY:       models.PLATFORM_APP,
						models.AllBidderSChainObj: `[{"bidders":["bidderA"],"schain":{"ver":"1.0","complete":1,"nodes":[{"asi":"example.com"}]}}]`,
						models.SChainObjectDBKey:  `{"validation":"off","config":{"ver":"2.0","complete":1,"nodes":[{"asi":"indirectseller-1.com","sid":"00001","hp":1}]}}`,
					},
				}, nil)
				mockCache.EXPECT().GetAdunitConfigFromCache(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&adunitconfig.AdUnitConfig{})
				mockCache.EXPECT().GetThrottlePartnersWithCriteria(gomock.Any()).Return(map[string]struct{}{}, nil)
				//prometheus metrics
				mockEngine.EXPECT().RecordPublisherProfileRequests("5890", "1234")
				mockEngine.EXPECT().RecordPublisherRequests(models.EndpointAppLovinMax, "5890", rctx.Platform)
				mockEngine.EXPECT().RecordPlatformPublisherPartnerReqStats(rctx.Platform, "5890", "appnexus")
				mockEngine.EXPECT().RecordMBMFRequests(models.EndpointAppLovinMax, rctx.PubIDStr, models.MBMFInvalidAdFormat)
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false)
				mockFeature.EXPECT().IsAnalyticsTrackingThrottled(gomock.Any(), gomock.Any()).Return(false, false)
				mockFeature.EXPECT().IsMaxFloorsEnabled(gomock.Any()).Return(false)
				mockEngine.EXPECT().RecordPreProcessingTimeStats(rctx.PubIDStr, gomock.Any())
				mockFeature.EXPECT().IsMBMFCountryForPublisher(gomock.Any(), gomock.Any()).Return(true)
				mockFeature.EXPECT().IsMBMFPublisherEnabled(gomock.Any()).Return(true)
				mockProfileMetaData.EXPECT().GetProfileTypePlatform(gomock.Any()).Return(0, false)
				mockFeature.EXPECT().IsDynamicFloorEnabledPublisher(gomock.Any()).Return(false)
			},
			want: want{
				hookResult: hookstage.HookResult[hookstage.BeforeValidationRequestPayload]{
					Reject:        false,
					NbrCode:       0,
					ChangeSet:     hookstage.ChangeSet[hookstage.BeforeValidationRequestPayload]{},
					DebugMessages: []string{`new imp: {"123":{"ImpID":"123","TagID":"adunit","DisplayManager":"","DisplayManagerVer":"","Div":"","SlotName":"adunit","AdUnitName":"adunit","Secure":0,"BidFloor":4.3,"BidFloorCur":"USD","IsRewardInventory":null,"Banner":true,"Video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"Native":null,"IncomingSlots":["700x900","728x90","300x250","640x480"],"Type":"video","Bidders":{"appnexus":{"PartnerID":2,"PrebidBidderCode":"appnexus","MatchedSlot":"adunit@700x900","KGP":"_AU_@_W_x_H_","KGPV":"","IsRegex":false,"Params":{"placementId":0,"site":"12313","adtag":"45343"},"VASTTagFlags":null}},"NonMapped":{},"NewExt":{"data":{"pbadslot":"adunit"},"gpid":"adunit","prebid":{"bidder":{"appnexus":{"placementId":0,"site":"12313","adtag":"45343"}}}},"BidCtx":{},"BannerAdUnitCtx":{"MatchedSlot":"","IsRegex":false,"MatchedRegex":"","SelectedSlotAdUnitConfig":null,"AppliedSlotAdUnitConfig":null,"UsingDefaultConfig":false,"AllowedConnectionTypes":null},"VideoAdUnitCtx":{"MatchedSlot":"","IsRegex":false,"MatchedRegex":"","SelectedSlotAdUnitConfig":null,"AppliedSlotAdUnitConfig":null,"UsingDefaultConfig":false,"AllowedConnectionTypes":null},"NativeAdUnitCtx":{"MatchedSlot":"","IsRegex":false,"MatchedRegex":"","SelectedSlotAdUnitConfig":null,"AppliedSlotAdUnitConfig":null,"UsingDefaultConfig":false,"AllowedConnectionTypes":null},"BidderError":"","IsAdPodRequest":false,"AdpodConfig":null,"ImpAdPodCfg":null,"BidIDToAPRC":null,"AdserverURL":"","BidIDToDur":null}}`, `new request.ext: {"prebid":{"bidadjustmentfactors":{"appnexus":1},"bidderparams":{"pubmatic":{"wiid":""}},"debug":true,"floors":{"enforcement":{"enforcepbs":true,"floordeals":true},"enabled":true},"targeting":{"pricegranularity":{"precision":2,"ranges":[{"min":0,"max":5,"increment":0.05},{"min":5,"max":10,"increment":0.1},{"min":10,"max":20,"increment":0.5}]},"mediatypepricegranularity":{},"includewinners":true,"includebidderkeys":true},"macros":{"[PLATFORM]":"3","[PROFILE_ID]":"1234","[PROFILE_VERSION]":"1","[UNIX_TIMESTAMP]":"0","[WRAPPER_IMPRESSION_ID]":""}}}`},
					AnalyticsTags: hookanalytics.Analytics{},
				},
				bidRequest:            json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","secure":1,"banner":{"format":[{"w":728,"h":90},{"w":300,"h":250}],"w":700,"h":900},"video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"tagid":"adunit","bidfloor":4.3,"bidfloorcur":"USD","ext":{"data":{"pbadslot":"adunit"},"prebid":{"bidder":{"appnexus":{"placementId":0,"site":"12313","adtag":"45343"}}},"gpid":"adunit"}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","customdata":"7D75D25F-FAC9-443D-B2D1-B17FEE11E027","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"tid":"123-456-789","ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1","schain":{"ver":"2.0","complete":1,"nodes":[{"asi":"indirectseller-1.com","sid":"00001","hp":1}]}}},"ext":{"prebid":{"bidadjustmentfactors":{"appnexus":1},"bidderparams":{"pubmatic":{"wiid":""}},"debug":true,"floors":{"enforcement":{"enforcepbs":true,"floordeals":true},"enabled":true},"schains":[{"bidders":["bidderA"],"schain":{"ver":"1.0","complete":1,"nodes":[{"asi":"example.com","sid":""}]}}],"targeting":{"pricegranularity":{"precision":2,"ranges":[{"min":0,"max":5,"increment":0.05},{"min":5,"max":10,"increment":0.1},{"min":10,"max":20,"increment":0.5}]},"includewinners":true,"includebidderkeys":true},"macros":{"[PLATFORM]":"3","[PROFILE_ID]":"1234","[PROFILE_VERSION]":"1","[UNIX_TIMESTAMP]":"0","[WRAPPER_IMPRESSION_ID]":""}}}}`),
				error:                 false,
				doMutate:              true,
				nilCurrencyConversion: false,
			},
		},
		{
			name: "valid_request_for_applovinmax_interstitial_banner_sizes",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": func() models.RequestCtx {
							testrctx := rctx
							testrctx.Endpoint = models.EndpointAppLovinMax
							return testrctx
						}(),
					},
				},
				bidrequest: json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","instl":1,"banner":{"format":[{"w":320,"h":480},{"w":300,"h":250}],"w":700,"h":900},"video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"tagid":"adunit","bidfloor":4.3,"bidfloorcur":"USD","ext":{"bidder":{"pubmatic":{"keywords":[{"key":"pmzoneid","value":["val1","val2"]}]}},"prebid":{}}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}},"ext":{"prebid":{},"wrapper":{"test":123,"profileid":123,"versionid":1,"wiid":"test_display_wiid"}}}`),
			},
			fields: fields{
				cache:        mockCache,
				metricEngine: mockEngine,
			},
			setup: func() {
				mockCache.EXPECT().GetMappingsFromCacheV25(gomock.Any(), gomock.Any()).Return(map[string]models.SlotMapping{
					"adunit@700x900": {
						SlotName: "adunit@700x900",
						SlotMappings: map[string]interface{}{
							models.SITE_CACHE_KEY: "12313",
							models.TAG_CACHE_KEY:  "45343",
						},
					},
				})
				mockCache.EXPECT().GetSlotToHashValueMapFromCacheV25(gomock.Any(), gomock.Any()).Return(models.SlotMappingInfo{
					OrderedSlotList: []string{"adunit@700x900"},
					HashValueMap: map[string]string{
						"adunit@700x900": "1232433543534543",
					},
				})
				mockCache.EXPECT().GetPartnerConfigMap(gomock.Any(), gomock.Any(), gomock.Any()).Return(map[int]map[string]string{
					2: {
						models.PARTNER_ID:          "2",
						models.PREBID_PARTNER_NAME: "appnexus",
						models.BidderCode:          "appnexus",
						models.SERVER_SIDE_FLAG:    "1",
						models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
						models.TIMEOUT:             "200",
					},
					-1: {
						models.DisplayVersionID:   "1",
						models.PLATFORM_KEY:       models.PLATFORM_APP,
						models.AllBidderSChainObj: `[{"bidders":["bidderA"],"schain":{"ver":"1.0","complete":1,"nodes":[{"asi":"example.com"}]}}]`,
						models.SChainObjectDBKey:  `{"validation":"off","config":{"ver":"2.0","complete":1,"nodes":[{"asi":"indirectseller-1.com","sid":"00001","hp":1}]}}`,
					},
				}, nil)
				mockCache.EXPECT().GetAdunitConfigFromCache(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&adunitconfig.AdUnitConfig{})
				mockCache.EXPECT().GetThrottlePartnersWithCriteria(gomock.Any()).Return(map[string]struct{}{}, nil)
				//prometheus metrics
				mockEngine.EXPECT().RecordPublisherProfileRequests("5890", "1234")
				mockEngine.EXPECT().RecordPublisherRequests(models.EndpointAppLovinMax, "5890", rctx.Platform)
				mockEngine.EXPECT().RecordPlatformPublisherPartnerReqStats(rctx.Platform, "5890", "appnexus")
				mockEngine.EXPECT().RecordMBMFRequests(models.EndpointAppLovinMax, rctx.PubIDStr, models.MBMFAdUnitFormatNotFound)
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false)
				mockFeature.EXPECT().IsAnalyticsTrackingThrottled(gomock.Any(), gomock.Any()).Return(false, false)
				mockFeature.EXPECT().IsMaxFloorsEnabled(gomock.Any()).Return(false)
				mockFeature.EXPECT().IsMBMFCountryForPublisher(gomock.Any(), gomock.Any()).Return(true)
				mockFeature.EXPECT().IsMBMFPublisherEnabled(gomock.Any()).Return(true)
				mockProfileMetaData.EXPECT().GetProfileTypePlatform(gomock.Any()).Return(0, false)
				mockFeature.EXPECT().IsMBMFEnabledForAdUnitFormat(gomock.Any(), gomock.Any()).Return(true)
				mockFeature.EXPECT().GetProfileAdUnitMultiFloors(gomock.Any())
				mockFeature.EXPECT().GetMBMFFloorsForAdUnitFormat(gomock.Any(), gomock.Any())
				mockEngine.EXPECT().RecordVideoInstlImpsStats(gomock.Any(), gomock.Any())
				mockFeature.EXPECT().IsDynamicFloorEnabledPublisher(gomock.Any()).Return(false)
			},
			want: want{
				hookResult: hookstage.HookResult[hookstage.BeforeValidationRequestPayload]{
					Reject:        false,
					NbrCode:       0,
					ChangeSet:     hookstage.ChangeSet[hookstage.BeforeValidationRequestPayload]{},
					DebugMessages: []string{`new imp: {"123":{"ImpID":"123","TagID":"adunit","DisplayManager":"","DisplayManagerVer":"","Div":"","SlotName":"adunit","AdUnitName":"adunit","Secure":0,"BidFloor":4.3,"BidFloorCur":"USD","IsRewardInventory":null,"Banner":true,"Video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"Native":null,"IncomingSlots":["700x900","728x90","300x250","640x480"],"Type":"video","Bidders":{"appnexus":{"PartnerID":2,"PrebidBidderCode":"appnexus","MatchedSlot":"adunit@700x900","KGP":"_AU_@_W_x_H_","KGPV":"","IsRegex":false,"Params":{"placementId":0,"site":"12313","adtag":"45343"},"VASTTagFlags":null}},"NonMapped":{},"NewExt":{"data":{"pbadslot":"adunit"},"gpid":"adunit","prebid":{"bidder":{"appnexus":{"placementId":0,"site":"12313","adtag":"45343"}}}},"BidCtx":{},"BannerAdUnitCtx":{"MatchedSlot":"","IsRegex":false,"MatchedRegex":"","SelectedSlotAdUnitConfig":null,"AppliedSlotAdUnitConfig":null,"UsingDefaultConfig":false,"AllowedConnectionTypes":null},"VideoAdUnitCtx":{"MatchedSlot":"","IsRegex":false,"MatchedRegex":"","SelectedSlotAdUnitConfig":null,"AppliedSlotAdUnitConfig":null,"UsingDefaultConfig":false,"AllowedConnectionTypes":null},"NativeAdUnitCtx":{"MatchedSlot":"","IsRegex":false,"MatchedRegex":"","SelectedSlotAdUnitConfig":null,"AppliedSlotAdUnitConfig":null,"UsingDefaultConfig":false,"AllowedConnectionTypes":null},"BidderError":"","IsAdPodRequest":false,"AdpodConfig":null,"ImpAdPodCfg":null,"BidIDToAPRC":null,"AdserverURL":"","BidIDToDur":null}}`, `new request.ext: {"prebid":{"bidadjustmentfactors":{"appnexus":1},"bidderparams":{"pubmatic":{"wiid":""}},"debug":true,"floors":{"enforcement":{"enforcepbs":true,"floordeals":true},"enabled":true},"targeting":{"pricegranularity":{"precision":2,"ranges":[{"min":0,"max":5,"increment":0.05},{"min":5,"max":10,"increment":0.1},{"min":10,"max":20,"increment":0.5}]},"mediatypepricegranularity":{},"includewinners":true,"includebidderkeys":true},"macros":{"[PLATFORM]":"3","[PROFILE_ID]":"1234","[PROFILE_VERSION]":"1","[UNIX_TIMESTAMP]":"0","[WRAPPER_IMPRESSION_ID]":""}}}`},
					AnalyticsTags: hookanalytics.Analytics{},
				},
				bidRequest:            json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","instl":1,"secure":1,"banner":{"format":[{"w":320,"h":480},{"w":300,"h":250},{"w":300,"h":600}],"w":700,"h":900},"video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"tagid":"adunit","bidfloor":4.3,"bidfloorcur":"USD","ext":{"data":{"pbadslot":"adunit"},"prebid":{"bidder":{"appnexus":{"placementId":0,"site":"12313","adtag":"45343"}}},"gpid":"adunit"}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","customdata":"7D75D25F-FAC9-443D-B2D1-B17FEE11E027","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"tid":"123-456-789","ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1","schain":{"ver":"2.0","complete":1,"nodes":[{"asi":"indirectseller-1.com","sid":"00001","hp":1}]}}},"ext":{"prebid":{"bidadjustmentfactors":{"appnexus":1},"bidderparams":{"pubmatic":{"wiid":""}},"debug":true,"floors":{"enforcement":{"enforcepbs":true,"floordeals":true},"enabled":true},"schains":[{"bidders":["bidderA"],"schain":{"ver":"1.0","complete":1,"nodes":[{"asi":"example.com","sid":""}]}}],"targeting":{"pricegranularity":{"precision":2,"ranges":[{"min":0,"max":5,"increment":0.05},{"min":5,"max":10,"increment":0.1},{"min":10,"max":20,"increment":0.5}]},"includewinners":true,"includebidderkeys":true},"macros":{"[PLATFORM]":"3","[PROFILE_ID]":"1234","[PROFILE_VERSION]":"1","[UNIX_TIMESTAMP]":"0","[WRAPPER_IMPRESSION_ID]":""}}}}`),
				error:                 false,
				doMutate:              true,
				nilCurrencyConversion: false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			m := OpenWrap{
				cfg:             tt.fields.cfg,
				cache:           tt.fields.cache,
				metricEngine:    tt.fields.metricEngine,
				pubFeatures:     mockFeature,
				rateConvertor:   &currency.RateConverter{},
				profileMetaData: mockProfileMetaData,
			}

			bidrequest := &openrtb2.BidRequest{}
			json.Unmarshal(tt.args.bidrequest, bidrequest)
			tt.args.payload.BidRequest = bidrequest
			got, err := m.handleBeforeValidationHook(tt.args.ctx, tt.args.moduleCtx, tt.args.payload)
			assert.Equal(t, tt.want.error, err != nil)
			// get updated rtcx
			iRctx := tt.args.moduleCtx.ModuleContext["rctx"]
			if rctx, ok := iRctx.(models.RequestCtx); ok {
				assert.Equal(t, tt.want.nilCurrencyConversion, rctx.CurrencyConversion == nil, "mismatched CurrencyConversion")
			}
			assert.Equal(t, tt.want.hookResult.Reject, got.Reject, "mismatched Reject")
			assert.Equal(t, tt.want.hookResult.NbrCode, got.NbrCode, "mismatched NbrCode")
			assert.Equal(t, tt.want.hookResult.SeatNonBid, got.SeatNonBid, "mismatched SeatNonBid")
			// for i := 0; i < len(got.DebugMessages); i++ {
			// 	gotDebugMessage, _ := json.Marshal(got.DebugMessages[i])
			// 	wantDebugMessage, _ := json.Marshal(tt.want.hookResult.DebugMessages[i])
			// 	sort.Slice(gotDebugMessage, func(i, j int) bool {
			// 		return gotDebugMessage[i] < gotDebugMessage[j]
			// 	})
			// 	sort.Slice(wantDebugMessage, func(i, j int) bool {
			// 		return wantDebugMessage[i] < wantDebugMessage[j]
			// 	})

			// 	assert.Equal(t, wantDebugMessage, gotDebugMessage)
			// }

			if tt.want.doMutate {
				mutations := got.ChangeSet.Mutations()
				assert.NotEmpty(t, mutations, tt.name)
				for _, mut := range mutations {
					result, err := mut.Apply(tt.args.payload)
					assert.Nil(t, err, tt.name)
					gotBidRequest, _ := json.Marshal(result.BidRequest)
					assert.JSONEq(t, string(tt.want.bidRequest), string(gotBidRequest), "mismatched BidRequest in %s", tt.name)
				}
			}
		})
	}
}

func TestCurrencyConverion(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockCache := mock_cache.NewMockCache(ctrl)
	mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)
	mockFeature := mock_feature.NewMockFeature(ctrl)
	mockProfileMetaData := mock_profilemetadata.NewMockProfileMetaData(ctrl)

	type fields struct {
		cfg          config.Config
		cache        cache.Cache
		metricEngine metrics.MetricsEngine
	}
	type args struct {
		ctx        context.Context
		moduleCtx  hookstage.ModuleInvocationContext
		bidrequest json.RawMessage
		fromCur    string
		toCur      string
		value      float64
	}
	type want struct {
		convertedValue float64
		error          error
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
		setup  func()
	}{
		{
			name: "GetRate returns error",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": rctx,
					},
				},
				bidrequest: json.RawMessage(`{"id":"123-456-789","imp":[{"ext":"invalid","id":"imp","tagid":"tag","banner":{"format":[{"w":300,"h":250}]}}],` +
					`"site":{"publisher":{"id":"5890"}},"ext":{"wrapper":{"profileid":33485},"prebid":{"currency":{"usepbsrates":true,"rates":{"USD":{"EUR":50}}}}}}`),
			},
			fields: fields{
				cache:        mockCache,
				metricEngine: mockEngine,
			},
			setup: func() {
				mockEngine.EXPECT().RecordPublisherProfileRequests("5890", "1234")
				mockEngine.EXPECT().RecordPublisherRequests(models.EndpointV25, "5890", "amp")
				mockEngine.EXPECT().RecordBadRequests(rctx.Endpoint, rctx.PubIDStr, getPubmaticErrorCode(openrtb3.NoBidInvalidRequest))
				mockEngine.EXPECT().RecordNobidErrPrebidServerRequests("5890", int(openrtb3.NoBidInvalidRequest))
				mockFeature.EXPECT().IsTBFFeatureEnabled(5890, 1234).Return(false)
				mockFeature.EXPECT().IsAnalyticsTrackingThrottled(5890, 1234).Return(false, false)
				mockCache.EXPECT().GetThrottlePartnersWithCriteria(gomock.Any()).Return(map[string]struct{}{}, nil)
				mockCache.EXPECT().GetPartnerConfigMap(5890, 1234, 1).Return(map[int]map[string]string{
					2: {
						models.PARTNER_ID:          "2",
						models.PREBID_PARTNER_NAME: "appnexus",
						models.BidderCode:          "appnexus",
						models.SERVER_SIDE_FLAG:    "1",
						models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
						models.TIMEOUT:             "200",
					},
					-1: {
						models.DisplayVersionID: "1",
						models.PLATFORM_KEY:     models.PLATFORM_AMP,
					},
				}, nil)
				mockCache.EXPECT().GetAdunitConfigFromCache(gomock.Any(), 5890, 1234, 1).Return(nil)
				mockProfileMetaData.EXPECT().GetProfileTypePlatform(models.TypeAmp).Return(0, false)
			},
			want: want{
				convertedValue: 0,
				error:          errors.New("currency: tag is not well-formed"),
			},
		},
		{
			name: "successful currency conversion",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": rctx,
					},
				},
				bidrequest: json.RawMessage(`{"id":"123-456-789","imp":[{"ext":"invalid","id":"imp","tagid":"tag","banner":{"format":[{"w":300,"h":250}]}}],` +
					`"site":{"publisher":{"id":"5890"}},"ext":{"wrapper":{"profileid":33485},"prebid":{"currency":{"usepbsrates":true,"rates":{"USD":{"EUR":50}}}}}}`),
				fromCur: "USD",
				toCur:   "EUR",
				value:   2,
			},
			fields: fields{
				cache:        mockCache,
				metricEngine: mockEngine,
			},
			setup: func() {
				mockEngine.EXPECT().RecordPublisherProfileRequests("5890", "1234")
				mockEngine.EXPECT().RecordPublisherRequests(models.EndpointV25, "5890", "amp")
				mockEngine.EXPECT().RecordBadRequests(rctx.Endpoint, rctx.PubIDStr, getPubmaticErrorCode(openrtb3.NoBidInvalidRequest))
				mockEngine.EXPECT().RecordNobidErrPrebidServerRequests("5890", int(openrtb3.NoBidInvalidRequest))
				mockFeature.EXPECT().IsTBFFeatureEnabled(5890, 1234).Return(false)
				mockFeature.EXPECT().IsAnalyticsTrackingThrottled(5890, 1234).Return(false, false)
				mockCache.EXPECT().GetThrottlePartnersWithCriteria(gomock.Any()).Return(map[string]struct{}{}, nil)
				mockCache.EXPECT().GetPartnerConfigMap(5890, 1234, 1).Return(map[int]map[string]string{
					2: {
						models.PARTNER_ID:          "2",
						models.PREBID_PARTNER_NAME: "appnexus",
						models.BidderCode:          "appnexus",
						models.SERVER_SIDE_FLAG:    "1",
						models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
						models.TIMEOUT:             "200",
					},
					-1: {
						models.DisplayVersionID: "1",
						models.PLATFORM_KEY:     models.PLATFORM_AMP,
					},
				}, nil)
				mockCache.EXPECT().GetAdunitConfigFromCache(gomock.Any(), 5890, 1234, 1).Return(nil)
				mockProfileMetaData.EXPECT().GetProfileTypePlatform(models.TypeAmp).Return(0, false)
			},
			want: want{
				convertedValue: 100,
				error:          nil,
			},
		},
	}
	for _, tt := range tests {
		if tt.setup != nil {
			tt.setup()
		}
		m := OpenWrap{
			cfg:             tt.fields.cfg,
			cache:           tt.fields.cache,
			metricEngine:    tt.fields.metricEngine,
			pubFeatures:     mockFeature,
			profileMetaData: mockProfileMetaData,
		}
		payload := &openrtb2.BidRequest{}
		err := json.Unmarshal(tt.args.bidrequest, payload)
		assert.Nil(t, err, "error should be nil")
		m.handleBeforeValidationHook(tt.args.ctx, tt.args.moduleCtx, hookstage.BeforeValidationRequestPayload{BidRequest: payload})
		iRctx := tt.args.moduleCtx.ModuleContext["rctx"]
		assert.NotNil(t, iRctx, "rctx should not be nil")
		rctx := iRctx.(models.RequestCtx)
		// verify currency-conversion logic
		convertedValue, err := rctx.CurrencyConversion(tt.args.fromCur, tt.args.toCur, tt.args.value)
		assert.Equal(t, tt.want.convertedValue, convertedValue, "mismatched convertedValue")
		assert.Equal(t, tt.want.error, err, "mismatched error")
	}
}

func TestUserAgent_handleBeforeValidationHook(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockCache := mock_cache.NewMockCache(ctrl)
	mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)
	mockFeature := mock_feature.NewMockFeature(ctrl)

	type fields struct {
		cfg          config.Config
		cache        cache.Cache
		metricEngine metrics.MetricsEngine
	}
	type args struct {
		ctx        context.Context
		moduleCtx  hookstage.ModuleInvocationContext
		payload    hookstage.BeforeValidationRequestPayload
		bidrequest json.RawMessage
	}
	type want struct {
		rctx        *models.RequestCtx
		moduleError []string
		err         error
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
		setup  func()
	}{
		{
			name: "bidRequest.Device.UA_is_present",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": rctx,
					},
				},
				bidrequest: json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","banner":{"format":[{"w":728,"h":90},{"w":300,"h":250}],"w":700,"h":900},"video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"tagid":"adunit","ext":{"wrapper":{"div":"div"},"bidder":{"pubmatic":{"keywords":[{"key":"pmzoneid","value":["val1","val2"]}]}},"prebid":{}}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}},"ext":1}`),
			},
			fields: fields{
				cache:        mockCache,
				metricEngine: mockEngine,
			},
			setup: func() {
				mockEngine.EXPECT().RecordPublisherProfileRequests("5890", "1234")
				mockEngine.EXPECT().RecordBadRequests(rctx.Endpoint, rctx.PubIDStr, getPubmaticErrorCode(nbr.InvalidRequestExt))
				mockEngine.EXPECT().RecordNobidErrPrebidServerRequests("5890", int(nbr.InvalidRequestExt))
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false)

			},
			want: want{
				rctx: &models.RequestCtx{
					DeviceCtx: models.DeviceCtx{UA: "Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36"},
				},
				moduleError: []string{"failed to get request ext: failed to decode request.ext : json: cannot unmarshal number into Go value of type models.RequestExt"},
				err:         nil,
			},
		},
		{
			name: "bidRequest.Device.UA_is_absent",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": rctx,
					},
				},
				bidrequest: json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","banner":{"format":[{"w":728,"h":90},{"w":300,"h":250}],"w":700,"h":900},"video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"tagid":"adunit","ext":{"wrapper":{"div":"div"},"bidder":{"pubmatic":{"keywords":[{"key":"pmzoneid","value":["val1","val2"]}]}},"prebid":{}}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}},"ext":1}`),
			},
			fields: fields{
				cache:        mockCache,
				metricEngine: mockEngine,
			},
			setup: func() {
				mockEngine.EXPECT().RecordPublisherProfileRequests("5890", "1234")
				mockEngine.EXPECT().RecordBadRequests(rctx.Endpoint, rctx.PubIDStr, getPubmaticErrorCode(nbr.InvalidRequestExt))
				mockEngine.EXPECT().RecordNobidErrPrebidServerRequests("5890", int(nbr.InvalidRequestExt))
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false)
			},
			want: want{
				rctx: &models.RequestCtx{
					DeviceCtx: models.DeviceCtx{UA: "go-test"},
					PubID:     1,
				},
				moduleError: []string{"failed to get request ext: failed to decode request.ext : json: cannot unmarshal number into Go value of type models.RequestExt"},
				err:         nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			adapters.InitBidders("./static/bidder-params/")
			m := OpenWrap{
				cfg:          tt.fields.cfg,
				cache:        tt.fields.cache,
				metricEngine: tt.fields.metricEngine,
				pubFeatures:  mockFeature,
			}
			tt.args.payload.BidRequest = &openrtb2.BidRequest{}
			json.Unmarshal(tt.args.bidrequest, tt.args.payload.BidRequest)

			result, err := m.handleBeforeValidationHook(tt.args.ctx, tt.args.moduleCtx, tt.args.payload)
			if err != nil {
				assert.Equal(t, tt.want.err, err, "different error returned from handleBeforeValidationHook")
			}
			if len(result.Errors) > 0 {
				assert.Equal(t, tt.want.moduleError, result.Errors, "mismatched error received from handleBeforeValidationHook execution")
			}
			iRctx := tt.args.moduleCtx.ModuleContext["rctx"]
			assert.Equal(t, tt.want.rctx == nil, iRctx == nil, "mismatched rctx received from handleBeforeValidationHook")
			gotRctx := iRctx.(models.RequestCtx)
			assert.Equal(t, tt.want.rctx.DeviceCtx.UA, gotRctx.DeviceCtx.UA, "mismatched rctx.UA received from handleBeforeValidationHook")
		})
	}
}

func TestVASTUnwrap_handleBeforeValidationHook(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockCache := mock_cache.NewMockCache(ctrl)
	mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)
	mockFeature := mock_feature.NewMockFeature(ctrl)
	mockProfileMetaData := mock_profilemetadata.NewMockProfileMetaData(ctrl)

	type fields struct {
		cfg          config.Config
		cache        cache.Cache
		metricEngine metrics.MetricsEngine
	}
	type args struct {
		ctx          context.Context
		moduleCtx    hookstage.ModuleInvocationContext
		payload      hookstage.BeforeValidationRequestPayload
		bidrequest   json.RawMessage
		randomNumber int
	}
	type want struct {
		rctx  *models.RequestCtx
		error bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
		setup  func()
	}{
		{
			name: "VAST Unwrap Disabled in DB, traffic percent  present in config",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": rctx,
					},
				},
				bidrequest: json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","banner":{"format":[{"w":728,"h":90},{"w":300,"h":250}],"w":700,"h":900},"video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"tagid":"adunit","bidfloor":4.3,"bidfloorcur":"USD","ext":{"bidder":{"pubmatic":{"keywords":[{"key":"pmzoneid","value":["val1","val2"]}]}},"prebid":{}}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}},"ext":{"prebid":{},"wrapper":{"test":123,"profileid":123,"versionid":1,"wiid":"test_display_wiid"}}}`),
			},
			fields: fields{
				cache:        mockCache,
				metricEngine: mockEngine,
				cfg: config.Config{
					Features: config.FeatureToggle{
						VASTUnwrapPercent: 10,
					},
				},
			},
			setup: func() {
				mockCache.EXPECT().GetMappingsFromCacheV25(gomock.Any(), gomock.Any()).Return(map[string]models.SlotMapping{
					"adunit@700x900": {
						SlotName: "adunit@700x900",
						SlotMappings: map[string]interface{}{
							models.SITE_CACHE_KEY: "12313",
							models.TAG_CACHE_KEY:  "45343",
						},
					},
				})
				mockCache.EXPECT().GetSlotToHashValueMapFromCacheV25(gomock.Any(), gomock.Any()).Return(models.SlotMappingInfo{
					OrderedSlotList: []string{"adunit@700x900"},
					HashValueMap: map[string]string{
						"adunit@700x900": "1232433543534543",
					},
				})
				mockCache.EXPECT().GetPartnerConfigMap(gomock.Any(), gomock.Any(), gomock.Any()).Return(map[int]map[string]string{
					2: {
						models.PARTNER_ID:          "2",
						models.PREBID_PARTNER_NAME: "appnexus",
						models.BidderCode:          "appnexus",
						models.SERVER_SIDE_FLAG:    "1",
						models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
						models.TIMEOUT:             "200",
					},
					-1: {
						models.DisplayVersionID:       "1",
						models.PLATFORM_KEY:           models.PLATFORM_APP,
						models.VastUnwrapperEnableKey: "0",
					},
				}, nil)
				mockCache.EXPECT().GetAdunitConfigFromCache(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&adunitconfig.AdUnitConfig{})
				mockCache.EXPECT().GetThrottlePartnersWithCriteria(gomock.Any()).Return(map[string]struct{}{}, nil)
				mockEngine.EXPECT().RecordPublisherProfileRequests("5890", "1234")
				mockEngine.EXPECT().RecordPublisherRequests(rctx.Endpoint, "5890", rctx.Platform)
				mockEngine.EXPECT().RecordPlatformPublisherPartnerReqStats(rctx.Platform, "5890", "appnexus")
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false)
				mockFeature.EXPECT().IsAnalyticsTrackingThrottled(gomock.Any(), gomock.Any()).Return(false, false)
				mockProfileMetaData.EXPECT().GetProfileTypePlatform(gomock.Any()).Return(0, false)
				mockFeature.EXPECT().IsDynamicFloorEnabledPublisher(gomock.Any()).Return(false)
			},
			want: want{
				rctx: &models.RequestCtx{
					VastUnWrap: models.VastUnWrap{
						Enabled: false,
					},
				},
				error: false,
			},
		},
		{
			name: "VAST Unwrap Enabled in DB, traffic percent not present in config and DB",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": rctx,
					},
				},
				bidrequest:   json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","banner":{"format":[{"w":728,"h":90},{"w":300,"h":250}],"w":700,"h":900},"video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"tagid":"adunit","bidfloor":4.3,"bidfloorcur":"USD","ext":{"bidder":{"pubmatic":{"keywords":[{"key":"pmzoneid","value":["val1","val2"]}]}},"prebid":{}}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}},"ext":{"prebid":{},"wrapper":{"test":123,"profileid":123,"versionid":1,"wiid":"test_display_wiid"}}}`),
				randomNumber: 20,
			},
			fields: fields{
				cache:        mockCache,
				metricEngine: mockEngine,
			},
			setup: func() {
				mockCache.EXPECT().GetMappingsFromCacheV25(gomock.Any(), gomock.Any()).Return(map[string]models.SlotMapping{
					"adunit@700x900": {
						SlotName: "adunit@700x900",
						SlotMappings: map[string]interface{}{
							models.SITE_CACHE_KEY: "12313",
							models.TAG_CACHE_KEY:  "45343",
						},
					},
				})
				mockCache.EXPECT().GetSlotToHashValueMapFromCacheV25(gomock.Any(), gomock.Any()).Return(models.SlotMappingInfo{
					OrderedSlotList: []string{"adunit@700x900"},
					HashValueMap: map[string]string{
						"adunit@700x900": "1232433543534543",
					},
				})
				mockCache.EXPECT().GetPartnerConfigMap(gomock.Any(), gomock.Any(), gomock.Any()).Return(map[int]map[string]string{
					2: {
						models.PARTNER_ID:          "2",
						models.PREBID_PARTNER_NAME: "appnexus",
						models.BidderCode:          "appnexus",
						models.SERVER_SIDE_FLAG:    "1",
						models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
						models.TIMEOUT:             "200",
					},
					-1: {
						models.DisplayVersionID:       "1",
						models.PLATFORM_KEY:           models.PLATFORM_APP,
						models.VastUnwrapperEnableKey: "1",
					},
				}, nil)
				mockCache.EXPECT().GetAdunitConfigFromCache(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&adunitconfig.AdUnitConfig{})
				mockCache.EXPECT().GetThrottlePartnersWithCriteria(gomock.Any()).Return(map[string]struct{}{}, nil)
				mockEngine.EXPECT().RecordPublisherProfileRequests("5890", "1234")
				mockEngine.EXPECT().RecordPublisherRequests(rctx.Endpoint, "5890", rctx.Platform)
				mockEngine.EXPECT().RecordPlatformPublisherPartnerReqStats(rctx.Platform, "5890", "appnexus")
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false)
				mockFeature.EXPECT().IsAnalyticsTrackingThrottled(gomock.Any(), gomock.Any()).Return(false, false)
				mockProfileMetaData.EXPECT().GetProfileTypePlatform(gomock.Any()).Return(0, false)
				mockFeature.EXPECT().IsDynamicFloorEnabledPublisher(gomock.Any()).Return(false)
			},
			want: want{
				rctx: &models.RequestCtx{
					VastUnWrap: models.VastUnWrap{
						Enabled: false,
					},
				},
				error: false,
			},
		},
		{
			name: "VAST Unwrap Enabled in DB, traffic percent present in config",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": rctx,
					},
				},
				bidrequest:   json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","banner":{"format":[{"w":728,"h":90},{"w":300,"h":250}],"w":700,"h":900},"video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"tagid":"adunit","bidfloor":4.3,"bidfloorcur":"USD","ext":{"bidder":{"pubmatic":{"keywords":[{"key":"pmzoneid","value":["val1","val2"]}]}},"prebid":{}}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}},"ext":{"prebid":{},"wrapper":{"test":123,"profileid":123,"versionid":1,"wiid":"test_display_wiid"}}}`),
				randomNumber: 20,
			},
			fields: fields{
				cache:        mockCache,
				metricEngine: mockEngine,
				cfg: config.Config{
					Features: config.FeatureToggle{
						VASTUnwrapPercent: 100,
					},
				},
			},
			setup: func() {
				mockCache.EXPECT().GetMappingsFromCacheV25(gomock.Any(), gomock.Any()).Return(map[string]models.SlotMapping{
					"adunit@700x900": {
						SlotName: "adunit@700x900",
						SlotMappings: map[string]interface{}{
							models.SITE_CACHE_KEY: "12313",
							models.TAG_CACHE_KEY:  "45343",
						},
					},
				})
				mockCache.EXPECT().GetSlotToHashValueMapFromCacheV25(gomock.Any(), gomock.Any()).Return(models.SlotMappingInfo{
					OrderedSlotList: []string{"adunit@700x900"},
					HashValueMap: map[string]string{
						"adunit@700x900": "1232433543534543",
					},
				})
				mockCache.EXPECT().GetPartnerConfigMap(gomock.Any(), gomock.Any(), gomock.Any()).Return(map[int]map[string]string{
					2: {
						models.PARTNER_ID:          "2",
						models.PREBID_PARTNER_NAME: "appnexus",
						models.BidderCode:          "appnexus",
						models.SERVER_SIDE_FLAG:    "1",
						models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
						models.TIMEOUT:             "200",
					},
					-1: {
						models.DisplayVersionID:       "1",
						models.PLATFORM_KEY:           models.PLATFORM_APP,
						models.VastUnwrapperEnableKey: "1",
					},
				}, nil)
				mockCache.EXPECT().GetAdunitConfigFromCache(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&adunitconfig.AdUnitConfig{})
				mockCache.EXPECT().GetThrottlePartnersWithCriteria(gomock.Any()).Return(map[string]struct{}{}, nil)
				mockEngine.EXPECT().RecordPublisherProfileRequests("5890", "1234")
				mockEngine.EXPECT().RecordPublisherRequests(rctx.Endpoint, "5890", rctx.Platform)
				mockEngine.EXPECT().RecordPlatformPublisherPartnerReqStats(rctx.Platform, "5890", "appnexus")
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false)
				mockFeature.EXPECT().IsAnalyticsTrackingThrottled(gomock.Any(), gomock.Any()).Return(false, false)
				mockProfileMetaData.EXPECT().GetProfileTypePlatform(gomock.Any()).Return(0, false)
				mockFeature.EXPECT().IsDynamicFloorEnabledPublisher(gomock.Any()).Return(false)
			},
			want: want{
				rctx: &models.RequestCtx{
					VastUnWrap: models.VastUnWrap{
						Enabled: true,
					},
				},
				error: false,
			},
		},
		{
			name: "VAST Unwrap Enabled in DB, traffic percent present in config and DB",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": rctx,
					},
				},
				bidrequest:   json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","banner":{"format":[{"w":728,"h":90},{"w":300,"h":250}],"w":700,"h":900},"video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"tagid":"adunit","bidfloor":4.3,"bidfloorcur":"USD","ext":{"bidder":{"pubmatic":{"keywords":[{"key":"pmzoneid","value":["val1","val2"]}]}},"prebid":{}}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}},"ext":{"prebid":{},"wrapper":{"test":123,"profileid":123,"versionid":1,"wiid":"test_display_wiid"}}}`),
				randomNumber: 20,
			},
			fields: fields{
				cache:        mockCache,
				metricEngine: mockEngine,
				cfg: config.Config{
					Features: config.FeatureToggle{
						VASTUnwrapPercent: 10,
					},
				},
			},
			setup: func() {
				mockCache.EXPECT().GetMappingsFromCacheV25(gomock.Any(), gomock.Any()).Return(map[string]models.SlotMapping{
					"adunit@700x900": {
						SlotName: "adunit@700x900",
						SlotMappings: map[string]interface{}{
							models.SITE_CACHE_KEY: "12313",
							models.TAG_CACHE_KEY:  "45343",
						},
					},
				})
				mockCache.EXPECT().GetSlotToHashValueMapFromCacheV25(gomock.Any(), gomock.Any()).Return(models.SlotMappingInfo{
					OrderedSlotList: []string{"adunit@700x900"},
					HashValueMap: map[string]string{
						"adunit@700x900": "1232433543534543",
					},
				})
				mockCache.EXPECT().GetPartnerConfigMap(gomock.Any(), gomock.Any(), gomock.Any()).Return(map[int]map[string]string{
					2: {
						models.PARTNER_ID:          "2",
						models.PREBID_PARTNER_NAME: "appnexus",
						models.BidderCode:          "appnexus",
						models.SERVER_SIDE_FLAG:    "1",
						models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
						models.TIMEOUT:             "200",
					},
					-1: {
						models.DisplayVersionID:            "1",
						models.PLATFORM_KEY:                models.PLATFORM_APP,
						models.VastUnwrapperEnableKey:      "1",
						models.VastUnwrapTrafficPercentKey: "50",
					},
				}, nil)
				mockCache.EXPECT().GetAdunitConfigFromCache(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&adunitconfig.AdUnitConfig{})
				mockCache.EXPECT().GetThrottlePartnersWithCriteria(gomock.Any()).Return(map[string]struct{}{}, nil)
				mockEngine.EXPECT().RecordPublisherProfileRequests("5890", "1234")
				mockEngine.EXPECT().RecordPublisherRequests(rctx.Endpoint, "5890", rctx.Platform)
				mockEngine.EXPECT().RecordPlatformPublisherPartnerReqStats(rctx.Platform, "5890", "appnexus")
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false)
				mockFeature.EXPECT().IsAnalyticsTrackingThrottled(gomock.Any(), gomock.Any()).Return(false, false)
				mockProfileMetaData.EXPECT().GetProfileTypePlatform(gomock.Any()).Return(0, false)
				mockFeature.EXPECT().IsDynamicFloorEnabledPublisher(gomock.Any()).Return(false)
			},
			want: want{
				rctx: &models.RequestCtx{
					VastUnWrap: models.VastUnWrap{
						Enabled: true,
					},
				},
				error: false,
			},
		},
		{
			name: "VAST Unwrap Enabled DB, traffic percent not present in config",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": rctx,
					},
				},
				bidrequest:   json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","banner":{"format":[{"w":728,"h":90},{"w":300,"h":250}],"w":700,"h":900},"video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"tagid":"adunit","bidfloor":4.3,"bidfloorcur":"USD","ext":{"bidder":{"pubmatic":{"keywords":[{"key":"pmzoneid","value":["val1","val2"]}]}},"prebid":{}}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}},"ext":{"prebid":{},"wrapper":{"test":123,"profileid":123,"versionid":1,"wiid":"test_display_wiid"}}}`),
				randomNumber: 20,
			},
			fields: fields{
				cache:        mockCache,
				metricEngine: mockEngine,
			},
			setup: func() {
				mockCache.EXPECT().GetMappingsFromCacheV25(gomock.Any(), gomock.Any()).Return(map[string]models.SlotMapping{
					"adunit@700x900": {
						SlotName: "adunit@700x900",
						SlotMappings: map[string]interface{}{
							models.SITE_CACHE_KEY: "12313",
							models.TAG_CACHE_KEY:  "45343",
						},
					},
				})
				mockCache.EXPECT().GetSlotToHashValueMapFromCacheV25(gomock.Any(), gomock.Any()).Return(models.SlotMappingInfo{
					OrderedSlotList: []string{"adunit@700x900"},
					HashValueMap: map[string]string{
						"adunit@700x900": "1232433543534543",
					},
				})
				mockCache.EXPECT().GetPartnerConfigMap(gomock.Any(), gomock.Any(), gomock.Any()).Return(map[int]map[string]string{
					2: {
						models.PARTNER_ID:          "2",
						models.PREBID_PARTNER_NAME: "appnexus",
						models.BidderCode:          "appnexus",
						models.SERVER_SIDE_FLAG:    "1",
						models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
						models.TIMEOUT:             "200",
					},
					-1: {
						models.DisplayVersionID:            "1",
						models.PLATFORM_KEY:                models.PLATFORM_APP,
						models.VastUnwrapperEnableKey:      "1",
						models.VastUnwrapTrafficPercentKey: "100",
					},
				}, nil)
				mockCache.EXPECT().GetAdunitConfigFromCache(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&adunitconfig.AdUnitConfig{})
				mockCache.EXPECT().GetThrottlePartnersWithCriteria(gomock.Any()).Return(map[string]struct{}{}, nil)
				mockEngine.EXPECT().RecordPublisherProfileRequests("5890", "1234")
				mockEngine.EXPECT().RecordPublisherRequests(rctx.Endpoint, "5890", rctx.Platform)
				mockEngine.EXPECT().RecordPlatformPublisherPartnerReqStats(rctx.Platform, "5890", "appnexus")
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false)
				mockFeature.EXPECT().IsAnalyticsTrackingThrottled(gomock.Any(), gomock.Any()).Return(false, false)
				mockProfileMetaData.EXPECT().GetProfileTypePlatform(gomock.Any()).Return(0, false)
				mockFeature.EXPECT().IsDynamicFloorEnabledPublisher(gomock.Any()).Return(false)
			},
			want: want{
				rctx: &models.RequestCtx{
					VastUnWrap: models.VastUnWrap{
						Enabled: true,
					},
				},
				error: false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			GetRandomNumberIn1To100 = func() int {
				return tt.args.randomNumber
			}

			adapters.InitBidders("./static/bidder-params/")
			m := OpenWrap{
				cfg:             tt.fields.cfg,
				cache:           tt.fields.cache,
				metricEngine:    tt.fields.metricEngine,
				pubFeatures:     mockFeature,
				profileMetaData: mockProfileMetaData,
			}
			tt.args.payload.BidRequest = &openrtb2.BidRequest{}
			json.Unmarshal(tt.args.bidrequest, tt.args.payload.BidRequest)

			_, err := m.handleBeforeValidationHook(tt.args.ctx, tt.args.moduleCtx, tt.args.payload)
			assert.Equal(t, tt.want.error, err != nil, "mismatched error received from handleBeforeValidationHook")
			iRctx := tt.args.moduleCtx.ModuleContext["rctx"]
			assert.Equal(t, tt.want.rctx == nil, iRctx == nil, "mismatched rctx received from handleBeforeValidationHook")
			gotRctx := iRctx.(models.RequestCtx)
			assert.Equal(t, tt.want.rctx.VastUnWrap.Enabled, gotRctx.VastUnWrap.Enabled, "mismatched rctx.VastUnWrap.Enabled received from handleBeforeValidationHook")
		})
	}
}
func TestImpBidCtx_handleBeforeValidationHook(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockCache := mock_cache.NewMockCache(ctrl)
	mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)
	mockFeature := mock_feature.NewMockFeature(ctrl)
	mockProfileMetaData := mock_profilemetadata.NewMockProfileMetaData(ctrl)
	type fields struct {
		cfg          config.Config
		cache        cache.Cache
		metricEngine metrics.MetricsEngine
	}
	type args struct {
		ctx        context.Context
		moduleCtx  hookstage.ModuleInvocationContext
		payload    hookstage.BeforeValidationRequestPayload
		bidrequest json.RawMessage
	}
	type want struct {
		rctx  *models.RequestCtx
		error bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
		setup  func()
	}{
		{
			name: "default_impctx_if_getProfileData_fails",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": rctx,
					},
				},
				bidrequest: json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"tagid":"adunit","ext":{"wrapper":{"div":"div"},"bidder":{"pubmatic":{"keywords":[{"key":"pmzoneid","value":["val1","val2"]}]}},"prebid":{}}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}},"ext":{"prebid":{},"wrapper":{"test":123,"profileid":123,"versionid":1,"wiid":"test_display_wiid"}}}`),
			},
			fields: fields{
				cache:        mockCache,
				metricEngine: mockEngine,
			},
			setup: func() {
				mockCache.EXPECT().GetPartnerConfigMap(gomock.Any(), gomock.Any(), gomock.Any()).Return(map[int]map[string]string{
					2: {
						models.PARTNER_ID:          "2",
						models.PREBID_PARTNER_NAME: "appnexus",
						models.BidderCode:          "appnexus",
						models.SERVER_SIDE_FLAG:    "1",
						models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
						models.TIMEOUT:             "200",
					},
					-1: {
						models.PLATFORM_KEY:     models.PLATFORM_APP,
						models.DisplayVersionID: "1",
					},
				}, errors.New("test"))
				mockCache.EXPECT().GetThrottlePartnersWithCriteria(gomock.Any()).Return(map[string]struct{}{}, nil)
				mockEngine.EXPECT().RecordPublisherProfileRequests("5890", "1234")
				mockEngine.EXPECT().RecordBadRequests(rctx.Endpoint, rctx.PubIDStr, getPubmaticErrorCode(nbr.InvalidProfileConfiguration))
				mockEngine.EXPECT().RecordNobidErrPrebidServerRequests("5890", int(nbr.InvalidProfileConfiguration))
				mockEngine.EXPECT().RecordPublisherInvalidProfileRequests(rctx.Endpoint, "5890", rctx.ProfileIDStr)
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false)
				mockFeature.EXPECT().IsAnalyticsTrackingThrottled(gomock.Any(), gomock.Any()).Return(false, false)
				mockProfileMetaData.EXPECT().GetProfileTypePlatform(gomock.Any()).Return(0, false)
			},
			want: want{
				rctx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"123": {
							IncomingSlots: []string{
								"640x480",
							},
							SlotName:   "adunit",
							AdUnitName: "adunit",
						},
					},
				},
				error: true,
			},
		},
		{
			name: "default_impctx_if_platform_is_missing",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": rctx,
					},
				},
				bidrequest: json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"tagid":"adunit","ext":{"wrapper":{"div":"div"},"bidder":{"pubmatic":{"keywords":[{"key":"pmzoneid","value":["val1","val2"]}]}},"prebid":{}}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}},"ext":{"prebid":{},"wrapper":{"test":123,"profileid":123,"versionid":1,"wiid":"test_display_wiid"}}}`),
			},
			fields: fields{
				cache:        mockCache,
				metricEngine: mockEngine,
			},
			setup: func() {
				mockCache.EXPECT().GetPartnerConfigMap(gomock.Any(), gomock.Any(), gomock.Any()).Return(map[int]map[string]string{
					2: {
						models.PARTNER_ID:          "2",
						models.PREBID_PARTNER_NAME: "appnexus",
						models.BidderCode:          "appnexus",
						models.SERVER_SIDE_FLAG:    "1",
						models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
						models.TIMEOUT:             "200",
					},
					-1: {
						models.DisplayVersionID: "1",
					},
				}, nil)
				//prometheus metrics
				mockEngine.EXPECT().RecordPublisherProfileRequests("5890", "1234")
				mockEngine.EXPECT().RecordBadRequests(rctx.Endpoint, rctx.PubIDStr, getPubmaticErrorCode(nbr.InvalidPlatform))
				mockEngine.EXPECT().RecordNobidErrPrebidServerRequests("5890", int(nbr.InvalidPlatform))
				mockEngine.EXPECT().RecordPublisherInvalidProfileRequests(rctx.Endpoint, "5890", rctx.ProfileIDStr)
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false)
				mockFeature.EXPECT().IsAnalyticsTrackingThrottled(gomock.Any(), gomock.Any()).Return(false, false)
				mockProfileMetaData.EXPECT().GetProfileTypePlatform(gomock.Any()).Return(0, false)
			},
			want: want{
				rctx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"123": {
							IncomingSlots: []string{
								"640x480",
							},
							SlotName:   "adunit",
							AdUnitName: "adunit",
						},
					},
				},
				error: true,
			},
		},
		{
			name: "default_impctx_if_all_partners_throttled",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": rctx,
					},
				},
				bidrequest: json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"tagid":"adunit","ext":{"wrapper":{"div":"div"},"bidder":{"pubmatic":{"keywords":[{"key":"pmzoneid","value":["val1","val2"]}]}},"prebid":{}}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432","buyeruid":"1rwe432","yob":1980,"gender":"F","geo":{"country":"US","region":"CA","metro":"90001","city":"Alamo"}},"wseat":["Wseat_0","Wseat_1"],"bseat":["Bseat_0","Bseat_1"],"cur":["cur_0","cur_1"],"wlang":["Wlang_0","Wlang_1"],"bcat":["bcat_0","bcat_1"],"badv":["badv_0","badv_1"],"bapp":["bapp_0","bapp_1"],"source":{"ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}},"ext":{"prebid":{},"wrapper":{"test":123,"profileid":123,"versionid":1,"wiid":"test_display_wiid"}}}`),
			},
			fields: fields{
				cache:        mockCache,
				metricEngine: mockEngine,
			},
			setup: func() {
				mockCache.EXPECT().GetPartnerConfigMap(gomock.Any(), gomock.Any(), gomock.Any()).Return(map[int]map[string]string{
					2: {
						models.PARTNER_ID:          "2",
						models.PREBID_PARTNER_NAME: "appnexus",
						models.BidderCode:          "appnexus",
						models.SERVER_SIDE_FLAG:    "1",
						models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
						models.TIMEOUT:             "200",
						models.THROTTLE:            "0",
					},
					-1: {
						models.DisplayVersionID: "1",
						models.PLATFORM_KEY:     models.PLATFORM_APP,
					},
				}, nil)
				mockCache.EXPECT().GetThrottlePartnersWithCriteria(gomock.Any()).Return(map[string]struct{}{}, nil)
				//prometheus metrics
				mockEngine.EXPECT().RecordPublisherProfileRequests("5890", "1234")
				mockEngine.EXPECT().RecordBadRequests(rctx.Endpoint, rctx.PubIDStr, getPubmaticErrorCode(nbr.AllPartnerThrottled))
				mockEngine.EXPECT().RecordNobidErrPrebidServerRequests("5890", int(nbr.AllPartnerThrottled))
				mockEngine.EXPECT().RecordPublisherRequests(rctx.Endpoint, "5890", rctx.Platform)
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false)
				mockFeature.EXPECT().IsAnalyticsTrackingThrottled(gomock.Any(), gomock.Any()).Return(false, false)
				mockProfileMetaData.EXPECT().GetProfileTypePlatform(gomock.Any()).Return(0, false)
			},
			want: want{
				error: false,
				rctx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"123": {
							IncomingSlots: []string{
								"640x480",
							},
							SlotName:   "adunit",
							AdUnitName: "adunit",
						},
					},
				},
			},
		},
		{
			name: "empty_impctx_if_TagID_not_present_in_imp",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": rctx,
					},
				},
				bidrequest: json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"ext":{"wrapper":{"div":"div"},"bidder":{"pubmatic":{"keywords":[{"key":"pmzoneid","value":["val1","val2"]}]}},"prebid":{}}},{"id":"456","video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"ext":{"wrapper":{"div":"div"},"bidder":{"pubmatic":{"keywords":[{"key":"pmzoneid","value":["val1","val2"]}]}},"prebid":{}}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432"},"source":{"ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}},"ext":{"prebid":{},"wrapper":{"test":123,"profileid":123,"versionid":1,"wiid":"test_display_wiid"}}}`),
			},
			fields: fields{
				cache:        mockCache,
				metricEngine: mockEngine,
			},
			setup: func() {
				mockCache.EXPECT().GetPartnerConfigMap(gomock.Any(), gomock.Any(), gomock.Any()).Return(map[int]map[string]string{
					2: {
						models.PARTNER_ID:          "2",
						models.PREBID_PARTNER_NAME: "appnexus",
						models.BidderCode:          "appnexus",
						models.SERVER_SIDE_FLAG:    "1",
						models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
						models.TIMEOUT:             "200",
					},
					-1: {
						models.DisplayVersionID: "1",
						models.PLATFORM_KEY:     models.PLATFORM_APP,
					},
				}, nil)
				mockCache.EXPECT().GetAdunitConfigFromCache(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&adunitconfig.AdUnitConfig{})
				mockCache.EXPECT().GetThrottlePartnersWithCriteria(gomock.Any()).Return(map[string]struct{}{}, nil)
				//prometheus metrics
				mockEngine.EXPECT().RecordPublisherProfileRequests("5890", "1234")
				mockEngine.EXPECT().RecordBadRequests(rctx.Endpoint, rctx.PubIDStr, getPubmaticErrorCode(nbr.InvalidImpressionTagID))
				mockEngine.EXPECT().RecordNobidErrPrebidServerRequests("5890", int(nbr.InvalidImpressionTagID))
				mockEngine.EXPECT().RecordPublisherRequests(rctx.Endpoint, "5890", rctx.Platform)
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false)
				mockFeature.EXPECT().IsAnalyticsTrackingThrottled(gomock.Any(), gomock.Any()).Return(false, false)
			},
			want: want{
				rctx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{},
				},
				error: true,
			},
		},
		{
			name: "empty_impctx_if_imp_ext_parse_fails",
			args: args{
				ctx: context.Background(),
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": rctx,
					},
				},
				bidrequest: json.RawMessage(`{"id":"123-456-789","imp":[{"id":"123","video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"ext":{"wrapper":"invalid","bidder":{"pubmatic":{"keywords":[{"key":"pmzoneid","value":["val1","val2"]}]}},"prebid":{}}},{"id":"456","video":{"mimes":["video/mp4","video/mpeg"],"w":640,"h":480},"ext":{"wrapper":{"div":"div"},"bidder":{"pubmatic":{"keywords":[{"key":"pmzoneid","value":["val1","val2"]}]}},"prebid":{}}}],"site":{"domain":"test.com","page":"www.test.com","publisher":{"id":"5890"}},"device":{"ua":"Mozilla/5.0(X11;Linuxx86_64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/52.0.2743.82Safari/537.36","ip":"123.145.167.10"},"user":{"id":"119208432"},"source":{"ext":{"omidpn":"MyIntegrationPartner","omidpv":"7.1"}},"ext":{"prebid":{},"wrapper":{"test":123,"profileid":123,"versionid":1,"wiid":"test_display_wiid"}}}`),
			},
			fields: fields{
				cache:        mockCache,
				metricEngine: mockEngine,
			},
			setup: func() {
				mockCache.EXPECT().GetPartnerConfigMap(gomock.Any(), gomock.Any(), gomock.Any()).Return(map[int]map[string]string{
					2: {
						models.PARTNER_ID:          "2",
						models.PREBID_PARTNER_NAME: "appnexus",
						models.BidderCode:          "appnexus",
						models.SERVER_SIDE_FLAG:    "1",
						models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
						models.TIMEOUT:             "200",
					},
					-1: {
						models.DisplayVersionID: "1",
						models.PLATFORM_KEY:     models.PLATFORM_APP,
					},
				}, nil)
				mockCache.EXPECT().GetAdunitConfigFromCache(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&adunitconfig.AdUnitConfig{})
				//prometheus metrics
				mockEngine.EXPECT().RecordPublisherProfileRequests("5890", "1234")
				mockEngine.EXPECT().RecordBadRequests(rctx.Endpoint, rctx.PubIDStr, getPubmaticErrorCode(openrtb3.NoBidInvalidRequest))
				mockEngine.EXPECT().RecordNobidErrPrebidServerRequests("5890", int(openrtb3.NoBidInvalidRequest))
				mockEngine.EXPECT().RecordPublisherRequests(rctx.Endpoint, "5890", rctx.Platform)
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false)
				mockFeature.EXPECT().IsAnalyticsTrackingThrottled(gomock.Any(), gomock.Any()).Return(false, false)
			},
			want: want{
				rctx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{},
				},
				error: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			adapters.InitBidders("./static/bidder-params/")
			m := OpenWrap{
				cfg:             tt.fields.cfg,
				cache:           tt.fields.cache,
				metricEngine:    tt.fields.metricEngine,
				pubFeatures:     mockFeature,
				profileMetaData: mockProfileMetaData,
			}
			tt.args.payload.BidRequest = &openrtb2.BidRequest{}
			json.Unmarshal(tt.args.bidrequest, tt.args.payload.BidRequest)

			_, err := m.handleBeforeValidationHook(tt.args.ctx, tt.args.moduleCtx, tt.args.payload)
			assert.Equal(t, tt.want.error, err != nil, "mismatched error")
			iRctx := tt.args.moduleCtx.ModuleContext["rctx"]
			gotRctx := iRctx.(models.RequestCtx)
			assert.Equal(t, tt.want.rctx.ImpBidCtx, gotRctx.ImpBidCtx, "mismatched rctx.ImpBidCtx")
		})
	}
}

func TestGetSlotName(t *testing.T) {
	type args struct {
		tagId  string
		impExt *models.ImpExtension
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Slot_name_from_gpid",
			args: args{
				tagId: "some-tagid",
				impExt: &models.ImpExtension{
					GpId: "some-gpid",
				},
			},
			want: "some-gpid",
		},
		{
			name: "Slot_name_from_tagid",
			args: args{
				tagId: "some-tagid",
				impExt: &models.ImpExtension{
					Data: openrtb_ext.ExtImpData{
						PbAdslot: "some-pbadslot",
					},
				},
			},
			want: "some-tagid",
		},
		{
			name: "Slot_name_from_pbadslot",
			args: args{
				tagId: "",
				impExt: &models.ImpExtension{
					Data: openrtb_ext.ExtImpData{
						PbAdslot: "some-pbadslot",
					},
				},
			},
			want: "some-pbadslot",
		},
		{
			name: "Slot_name_from_stored_request_id",
			args: args{
				tagId: "",
				impExt: &models.ImpExtension{
					Prebid: openrtb_ext.ExtImpPrebid{
						StoredRequest: &openrtb_ext.ExtStoredRequest{
							ID: "stored-req-id",
						},
					},
				},
			},
			want: "stored-req-id",
		},
		{
			name: "imp_ext_nil_slot_name_from_tag_id",
			args: args{
				tagId:  "some-tagid",
				impExt: nil,
			},
			want: "some-tagid",
		},
		{
			name: "empty_slot_name",
			args: args{
				tagId:  "",
				impExt: &models.ImpExtension{},
			},
			want: "",
		},
		{
			name: "all_level_information_is_present_slot_name_picked_by_preference",
			args: args{
				tagId: "some-tagid",
				impExt: &models.ImpExtension{
					GpId: "some-gpid",
					Data: openrtb_ext.ExtImpData{
						PbAdslot: "some-pbadslot",
					},
					Prebid: openrtb_ext.ExtImpPrebid{
						StoredRequest: &openrtb_ext.ExtStoredRequest{
							ID: "stored-req-id",
						},
					},
				},
			},
			want: "some-gpid",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getSlotName(tt.args.tagId, tt.args.impExt)
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}

func TestGetAdunitName(t *testing.T) {
	type args struct {
		tagId  string
		impExt *models.ImpExtension
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "adunit_from_adserver_slot",
			args: args{
				tagId: "some-tagid",
				impExt: &models.ImpExtension{
					Data: openrtb_ext.ExtImpData{
						PbAdslot: "some-pbadslot",
						AdServer: &openrtb_ext.ExtImpDataAdServer{
							Name:   models.GamAdServer,
							AdSlot: "gam-unit",
						},
					},
				},
			},
			want: "gam-unit",
		},
		{
			name: "adunit_from_pbadslot",
			args: args{
				tagId: "some-tagid",
				impExt: &models.ImpExtension{
					Data: openrtb_ext.ExtImpData{
						PbAdslot: "some-pbadslot",
						AdServer: &openrtb_ext.ExtImpDataAdServer{
							Name:   models.GamAdServer,
							AdSlot: "",
						},
					},
				},
			},
			want: "some-pbadslot",
		},
		{
			name: "adunit_from_pbadslot_when_gam_is_absent",
			args: args{
				tagId: "some-tagid",
				impExt: &models.ImpExtension{
					Data: openrtb_ext.ExtImpData{
						PbAdslot: "some-pbadslot",
						AdServer: &openrtb_ext.ExtImpDataAdServer{
							Name:   "freewheel",
							AdSlot: "freewheel-unit",
						},
					},
				},
			},
			want: "some-pbadslot",
		},
		{
			name: "adunit_from_TagId",
			args: args{
				tagId: "some-tagid",
				impExt: &models.ImpExtension{
					Data: openrtb_ext.ExtImpData{
						PbAdslot: "",
						AdServer: &openrtb_ext.ExtImpDataAdServer{
							Name:   models.GamAdServer,
							AdSlot: "",
						},
					},
				},
			},
			want: "some-tagid",
		},
		{
			name: "adunit_from_TagId_imp_ext_nil",
			args: args{
				tagId:  "some-tagid",
				impExt: nil,
			},
			want: "some-tagid",
		},
		{
			name: "adunit_from_TagId_imp_ext_nil",
			args: args{
				tagId:  "some-tagid",
				impExt: &models.ImpExtension{},
			},
			want: "some-tagid",
		},
		{
			name: "all_level_information_is_present_adunit_name_picked_by_preference",
			args: args{
				tagId: "some-tagid",
				impExt: &models.ImpExtension{
					GpId: "some-gpid",
					Data: openrtb_ext.ExtImpData{
						PbAdslot: "some-pbadslot",
						AdServer: &openrtb_ext.ExtImpDataAdServer{
							Name:   models.GamAdServer,
							AdSlot: "gam-unit",
						},
					},
				},
			},
			want: "gam-unit",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getAdunitName(tt.args.tagId, tt.args.impExt)
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}

func TestGetTagID(t *testing.T) {
	type args struct {
		imp    openrtb2.Imp
		impExt *models.ImpExtension
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "tagId_not_found",
			args: args{
				imp:    openrtb2.Imp{},
				impExt: &models.ImpExtension{},
			},
			want: "",
		},
		{
			name: "tagId_present_in_gpid",
			args: args{
				imp: openrtb2.Imp{},
				impExt: &models.ImpExtension{
					GpId: "/7578294/adunit1",
				},
			},
			want: "/7578294/adunit1",
		},
		{
			name: "tagId_set_by_publisher_on_page",
			args: args{
				imp: openrtb2.Imp{
					TagID: "/7578294/adunit1",
				},
				impExt: &models.ImpExtension{},
			},
			want: "/7578294/adunit1",
		},
		{
			name: "tagId_present_in_pbadslot",
			args: args{
				imp: openrtb2.Imp{},
				impExt: &models.ImpExtension{
					Data: openrtb_ext.ExtImpData{
						PbAdslot: "/7578294/adunit1",
					},
				},
			},
			want: "/7578294/adunit1",
		},
		{
			name: "tagId_present_in_pbadslot_and_gpid",
			args: args{
				imp: openrtb2.Imp{},
				impExt: &models.ImpExtension{
					GpId: "/7578294/adunit123",
					Data: openrtb_ext.ExtImpData{
						PbAdslot: "/7578294/adunit",
					},
				},
			},
			want: "/7578294/adunit123",
		},
		{
			name: "tagId_present_in_imp.TagId_and_gpid",
			args: args{
				imp: openrtb2.Imp{
					TagID: "/7578294/adunit",
				},
				impExt: &models.ImpExtension{
					GpId: "/7578294/adunit123",
				},
			},
			want: "/7578294/adunit123",
		},
		{
			name: "tagId_present_in_imp.TagId_and_pbadslot",
			args: args{
				imp: openrtb2.Imp{
					TagID: "/7578294/adunit123",
				},
				impExt: &models.ImpExtension{
					Data: openrtb_ext.ExtImpData{
						PbAdslot: "/7578294/adunit",
					},
				},
			},
			want: "/7578294/adunit123",
		},
		{
			name: "tagId_present_in_imp.TagId_and_pbadslot_and_gpid",
			args: args{
				imp: openrtb2.Imp{
					TagID: "/7578294/adunit",
				},
				impExt: &models.ImpExtension{
					GpId: "/7578294/adunit123",
					Data: openrtb_ext.ExtImpData{
						PbAdslot: "/7578294/adunit12345",
					},
				},
			},
			want: "/7578294/adunit123",
		},
		{
			name: "GpId_contains_'#'",
			args: args{
				imp: openrtb2.Imp{
					TagID: "/7578294/adunit",
				},
				impExt: &models.ImpExtension{
					GpId: "/43743431/DMDemo#Div1",
					Data: openrtb_ext.ExtImpData{
						PbAdslot: "/7578294/adunit12345",
					},
				},
			},
			want: "/43743431/DMDemo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getTagID(tt.args.imp, tt.args.impExt); got != tt.want {
				t.Errorf("getTagID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpdateImpVideoWithVideoConfig(t *testing.T) {
	type args struct {
		imp                    *openrtb2.Imp
		configObjInVideoConfig *adunitconfig.VideoConfig
	}
	tests := []struct {
		name         string
		args         args
		wantImpVideo *openrtb2.Video
	}{
		{
			name: "imp video object is empty updated from adunit config",
			args: args{
				imp: &openrtb2.Imp{
					ID:    "123",
					Video: &openrtb2.Video{},
				},
				configObjInVideoConfig: &adunitconfig.VideoConfig{
					Video: openrtb2.Video{
						W:              ptrutil.ToPtr[int64](300),
						H:              ptrutil.ToPtr[int64](250),
						MIMEs:          []string{"MP4"},
						Linearity:      adcom1.LinearityNonLinear,
						StartDelay:     adcom1.StartMidRoll.Ptr(),
						MinDuration:    20,
						MaxDuration:    50,
						Placement:      adcom1.VideoPlacementInStream,
						Plcmt:          adcom1.VideoPlcmtAccompanyingContent,
						Protocols:      []adcom1.MediaCreativeSubtype{adcom1.CreativeVAST10, adcom1.CreativeVAST20, adcom1.CreativeVAST30, adcom1.CreativeVAST10Wrapper, adcom1.CreativeVAST20Wrapper, adcom1.CreativeVAST30Wrapper, adcom1.CreativeVAST40, adcom1.CreativeVAST40Wrapper, adcom1.CreativeVAST41, adcom1.CreativeVAST41Wrapper, adcom1.CreativeVAST42, adcom1.CreativeVAST42Wrapper},
						Skip:           ptrutil.ToPtr(int8(1)),
						SkipMin:        10,
						SkipAfter:      5,
						BoxingAllowed:  ptrutil.ToPtr[int8](2),
						PlaybackMethod: []adcom1.PlaybackMethod{adcom1.PlaybackPageLoadSoundOn},
						PlaybackEnd:    adcom1.PlaybackCompletion,
						Delivery:       []adcom1.DeliveryMethod{adcom1.DeliveryProgressive},
						Protocol:       adcom1.CreativeVAST10,
						Sequence:       1,
						CompanionType:  []adcom1.CompanionType{adcom1.CompanionHTML},
						Pos:            adcom1.PositionAboveFold.Ptr(),
						API:            []adcom1.APIFramework{adcom1.APIVPAID10},
						CompanionAd:    []openrtb2.Banner{},
						BAttr:          []adcom1.CreativeAttribute{adcom1.AttrAudioAuto},
						MaxExtended:    100,
					},
				},
			},
			wantImpVideo: &openrtb2.Video{
				W:              ptrutil.ToPtr[int64](300),
				H:              ptrutil.ToPtr[int64](250),
				MIMEs:          []string{"MP4"},
				Linearity:      adcom1.LinearityNonLinear,
				StartDelay:     adcom1.StartMidRoll.Ptr(),
				MinDuration:    20,
				MaxDuration:    50,
				Placement:      adcom1.VideoPlacementInStream,
				Plcmt:          adcom1.VideoPlcmtAccompanyingContent,
				Protocols:      []adcom1.MediaCreativeSubtype{adcom1.CreativeVAST10, adcom1.CreativeVAST20, adcom1.CreativeVAST30, adcom1.CreativeVAST10Wrapper, adcom1.CreativeVAST20Wrapper, adcom1.CreativeVAST30Wrapper, adcom1.CreativeVAST40, adcom1.CreativeVAST40Wrapper, adcom1.CreativeVAST41, adcom1.CreativeVAST41Wrapper, adcom1.CreativeVAST42, adcom1.CreativeVAST42Wrapper},
				Skip:           ptrutil.ToPtr[int8](1),
				SkipMin:        10,
				SkipAfter:      5,
				BoxingAllowed:  ptrutil.ToPtr[int8](2),
				PlaybackMethod: []adcom1.PlaybackMethod{adcom1.PlaybackPageLoadSoundOn},
				PlaybackEnd:    adcom1.PlaybackCompletion,
				Delivery:       []adcom1.DeliveryMethod{adcom1.DeliveryProgressive},
				Protocol:       adcom1.CreativeVAST10,
				Sequence:       1,
				CompanionType:  []adcom1.CompanionType{adcom1.CompanionHTML},
				Pos:            ptrutil.ToPtr(adcom1.PositionAboveFold),
				API:            []adcom1.APIFramework{adcom1.APIVPAID10},
				CompanionAd:    []openrtb2.Banner{},
				BAttr:          []adcom1.CreativeAttribute{adcom1.AttrAudioAuto},
				MaxExtended:    100,
			},
		},
		{
			name: "imp video object is not empty and adunit config is also not empty priority to request level parameters",
			args: args{
				imp: &openrtb2.Imp{
					ID: "123",
					Video: &openrtb2.Video{
						W: ptrutil.ToPtr[int64](300),
						H: ptrutil.ToPtr[int64](250),
					},
				},

				configObjInVideoConfig: &adunitconfig.VideoConfig{
					Video: openrtb2.Video{
						W:              ptrutil.ToPtr[int64](400),
						H:              ptrutil.ToPtr[int64](300),
						MIMEs:          []string{"MP4"},
						Linearity:      adcom1.LinearityNonLinear,
						StartDelay:     adcom1.StartMidRoll.Ptr(),
						MinDuration:    20,
						MaxDuration:    50,
						Placement:      adcom1.VideoPlacementInStream,
						Plcmt:          adcom1.VideoPlcmtAccompanyingContent,
						Protocols:      []adcom1.MediaCreativeSubtype{adcom1.CreativeVAST10, adcom1.CreativeVAST20, adcom1.CreativeVAST30, adcom1.CreativeVAST10Wrapper, adcom1.CreativeVAST20Wrapper, adcom1.CreativeVAST30Wrapper, adcom1.CreativeVAST40, adcom1.CreativeVAST40Wrapper, adcom1.CreativeVAST41, adcom1.CreativeVAST41Wrapper, adcom1.CreativeVAST42, adcom1.CreativeVAST42Wrapper},
						Skip:           ptrutil.ToPtr(int8(1)),
						SkipMin:        10,
						SkipAfter:      5,
						BoxingAllowed:  ptrutil.ToPtr[int8](2),
						PlaybackMethod: []adcom1.PlaybackMethod{adcom1.PlaybackPageLoadSoundOn},
						PlaybackEnd:    adcom1.PlaybackCompletion,
						Delivery:       []adcom1.DeliveryMethod{adcom1.DeliveryProgressive},
						Protocol:       adcom1.CreativeVAST10,
						Sequence:       1,
						CompanionType:  []adcom1.CompanionType{adcom1.CompanionHTML},
						Pos:            adcom1.PositionAboveFold.Ptr(),
						API:            []adcom1.APIFramework{adcom1.APIVPAID10},
						CompanionAd:    []openrtb2.Banner{},
						BAttr:          []adcom1.CreativeAttribute{adcom1.AttrAudioAuto},
						MaxExtended:    100,
					},
				},
			},
			wantImpVideo: &openrtb2.Video{
				W:              ptrutil.ToPtr[int64](300),
				H:              ptrutil.ToPtr[int64](250),
				MIMEs:          []string{"MP4"},
				Linearity:      adcom1.LinearityNonLinear,
				StartDelay:     adcom1.StartMidRoll.Ptr(),
				MinDuration:    20,
				MaxDuration:    50,
				Placement:      adcom1.VideoPlacementInStream,
				Plcmt:          adcom1.VideoPlcmtAccompanyingContent,
				Protocols:      []adcom1.MediaCreativeSubtype{adcom1.CreativeVAST10, adcom1.CreativeVAST20, adcom1.CreativeVAST30, adcom1.CreativeVAST10Wrapper, adcom1.CreativeVAST20Wrapper, adcom1.CreativeVAST30Wrapper, adcom1.CreativeVAST40, adcom1.CreativeVAST40Wrapper, adcom1.CreativeVAST41, adcom1.CreativeVAST41Wrapper, adcom1.CreativeVAST42, adcom1.CreativeVAST42Wrapper},
				Skip:           ptrutil.ToPtr[int8](1),
				SkipMin:        10,
				SkipAfter:      5,
				BoxingAllowed:  ptrutil.ToPtr[int8](2),
				PlaybackMethod: []adcom1.PlaybackMethod{adcom1.PlaybackPageLoadSoundOn},
				PlaybackEnd:    adcom1.PlaybackCompletion,
				Delivery:       []adcom1.DeliveryMethod{adcom1.DeliveryProgressive},
				Protocol:       adcom1.CreativeVAST10,
				Sequence:       1,
				CompanionType:  []adcom1.CompanionType{adcom1.CompanionHTML},
				Pos:            ptrutil.ToPtr(adcom1.PositionAboveFold),
				API:            []adcom1.APIFramework{adcom1.APIVPAID10},
				CompanionAd:    []openrtb2.Banner{},
				BAttr:          []adcom1.CreativeAttribute{adcom1.AttrAudioAuto},
				MaxExtended:    100,
			},
		},
	}
	for _, tt := range tests {
		updateImpVideoWithVideoConfig(tt.args.imp, tt.args.configObjInVideoConfig)
		assert.Equal(t, tt.wantImpVideo, tt.args.imp.Video, tt.name)
	}
}

func TestUpdateAmpImpVideoWithDefault(t *testing.T) {
	type args struct {
		imp *openrtb2.Imp
	}
	tests := []struct {
		name string
		args args
		want *openrtb2.Video
	}{
		{
			name: "banner has the width and height",
			args: args{
				imp: &openrtb2.Imp{
					Banner: &openrtb2.Banner{
						W: ptrutil.ToPtr[int64](300),
						H: ptrutil.ToPtr[int64](250),
					},
					Video: &openrtb2.Video{},
				},
			},
			want: &openrtb2.Video{
				MIMEs:          []string{"video/mp4"},
				MinDuration:    0,
				MaxDuration:    30,
				StartDelay:     adcom1.StartPreRoll.Ptr(),
				Protocols:      []adcom1.MediaCreativeSubtype{adcom1.CreativeVAST10, adcom1.CreativeVAST20, adcom1.CreativeVAST30, adcom1.CreativeVAST10Wrapper, adcom1.CreativeVAST20Wrapper, adcom1.CreativeVAST30Wrapper, adcom1.CreativeVAST40, adcom1.CreativeVAST40Wrapper, adcom1.CreativeVAST41, adcom1.CreativeVAST41Wrapper, adcom1.CreativeVAST42, adcom1.CreativeVAST42Wrapper},
				Placement:      adcom1.VideoPlacementInBanner,
				Plcmt:          adcom1.VideoPlcmtNoContent,
				Linearity:      adcom1.LinearityLinear,
				Skip:           ptrutil.ToPtr[int8](0),
				SkipMin:        0,
				SkipAfter:      0,
				BoxingAllowed:  ptrutil.ToPtr[int8](1),
				PlaybackMethod: []adcom1.PlaybackMethod{adcom1.PlaybackPageLoadSoundOff},
				PlaybackEnd:    adcom1.PlaybackCompletion,
				Delivery:       []adcom1.DeliveryMethod{adcom1.DeliveryProgressive, adcom1.DeliveryDownload},
				W:              ptrutil.ToPtr[int64](300),
				H:              ptrutil.ToPtr[int64](250),
			},
		},
		{
			name: "banner has the width and height in the banner format object",
			args: args{
				imp: &openrtb2.Imp{
					Banner: &openrtb2.Banner{
						Format: []openrtb2.Format{
							{
								W: 300,
								H: 250,
							},
							{
								W: 400,
								H: 300,
							},
						},
					},
					Video: &openrtb2.Video{},
				},
			},
			want: &openrtb2.Video{
				MIMEs:          []string{"video/mp4"},
				MinDuration:    0,
				MaxDuration:    30,
				StartDelay:     adcom1.StartPreRoll.Ptr(),
				Protocols:      []adcom1.MediaCreativeSubtype{adcom1.CreativeVAST10, adcom1.CreativeVAST20, adcom1.CreativeVAST30, adcom1.CreativeVAST10Wrapper, adcom1.CreativeVAST20Wrapper, adcom1.CreativeVAST30Wrapper, adcom1.CreativeVAST40, adcom1.CreativeVAST40Wrapper, adcom1.CreativeVAST41, adcom1.CreativeVAST41Wrapper, adcom1.CreativeVAST42, adcom1.CreativeVAST42Wrapper},
				Placement:      adcom1.VideoPlacementInBanner,
				Plcmt:          adcom1.VideoPlcmtNoContent,
				Linearity:      adcom1.LinearityLinear,
				Skip:           ptrutil.ToPtr[int8](0),
				SkipMin:        0,
				SkipAfter:      0,
				BoxingAllowed:  ptrutil.ToPtr[int8](1),
				PlaybackMethod: []adcom1.PlaybackMethod{adcom1.PlaybackPageLoadSoundOff},
				PlaybackEnd:    adcom1.PlaybackCompletion,
				Delivery:       []adcom1.DeliveryMethod{adcom1.DeliveryProgressive, adcom1.DeliveryDownload},
				W:              ptrutil.ToPtr[int64](300),
				H:              ptrutil.ToPtr[int64](250),
			},
		},
		{
			name: "banner has the width and height in in the both banner and format object",
			args: args{
				imp: &openrtb2.Imp{
					Banner: &openrtb2.Banner{
						W: ptrutil.ToPtr[int64](300),
						H: ptrutil.ToPtr[int64](250),
						Format: []openrtb2.Format{
							{
								W: 200,
								H: 150,
							},
							{
								W: 400,
								H: 300,
							},
						},
					},
					Video: &openrtb2.Video{},
				},
			},
			want: &openrtb2.Video{
				MIMEs:          []string{"video/mp4"},
				MinDuration:    0,
				MaxDuration:    30,
				StartDelay:     adcom1.StartPreRoll.Ptr(),
				Protocols:      []adcom1.MediaCreativeSubtype{adcom1.CreativeVAST10, adcom1.CreativeVAST20, adcom1.CreativeVAST30, adcom1.CreativeVAST10Wrapper, adcom1.CreativeVAST20Wrapper, adcom1.CreativeVAST30Wrapper, adcom1.CreativeVAST40, adcom1.CreativeVAST40Wrapper, adcom1.CreativeVAST41, adcom1.CreativeVAST41Wrapper, adcom1.CreativeVAST42, adcom1.CreativeVAST42Wrapper},
				Placement:      adcom1.VideoPlacementInBanner,
				Plcmt:          adcom1.VideoPlcmtNoContent,
				Linearity:      adcom1.LinearityLinear,
				Skip:           ptrutil.ToPtr[int8](0),
				SkipMin:        0,
				SkipAfter:      0,
				BoxingAllowed:  ptrutil.ToPtr[int8](1),
				PlaybackMethod: []adcom1.PlaybackMethod{adcom1.PlaybackPageLoadSoundOff},
				PlaybackEnd:    adcom1.PlaybackCompletion,
				Delivery:       []adcom1.DeliveryMethod{adcom1.DeliveryProgressive, adcom1.DeliveryDownload},
				W:              ptrutil.ToPtr[int64](300),
				H:              ptrutil.ToPtr[int64](250),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateAmpImpVideoWithDefault(tt.args.imp)
		})
	}
}

func TestGetW(t *testing.T) {
	type args struct {
		imp *openrtb2.Imp
	}
	tests := []struct {
		name string
		args args
		want *int64
	}{
		{
			name: "Empty banner and format",
			args: args{
				imp: &openrtb2.Imp{
					Banner: &openrtb2.Banner{
						Format: nil,
					},
				},
			},
			want: nil,
		},
		{
			name: "both banner and format are present",
			args: args{
				imp: &openrtb2.Imp{
					Banner: &openrtb2.Banner{
						W: ptrutil.ToPtr[int64](300),
						Format: []openrtb2.Format{
							{
								W: 400,
							},
						},
					},
				},
			},
			want: ptrutil.ToPtr[int64](300),
		},
		{
			name: "only format is present",
			args: args{
				imp: &openrtb2.Imp{
					Banner: &openrtb2.Banner{
						Format: []openrtb2.Format{
							{
								W: 400,
							},
						},
					},
				},
			},
			want: ptrutil.ToPtr[int64](400),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getW(tt.args.imp)
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}

func TestGetH(t *testing.T) {
	type args struct {
		imp *openrtb2.Imp
	}
	tests := []struct {
		name string
		args args
		want *int64
	}{
		{
			name: "Empty banner and format",
			args: args{
				imp: &openrtb2.Imp{
					Banner: &openrtb2.Banner{
						Format: nil,
					},
				},
			},
			want: nil,
		},
		{
			name: "both banner and format are present",
			args: args{
				imp: &openrtb2.Imp{
					Banner: &openrtb2.Banner{
						H: ptrutil.ToPtr[int64](300),
						Format: []openrtb2.Format{
							{
								H: 400,
							},
						},
					},
				},
			},
			want: ptrutil.ToPtr[int64](300),
		},
		{
			name: "only format is present",
			args: args{
				imp: &openrtb2.Imp{
					Banner: &openrtb2.Banner{
						Format: []openrtb2.Format{
							{
								H: 400,
							},
						},
					},
				},
			},
			want: ptrutil.ToPtr[int64](400),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getH(tt.args.imp)
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}

func TestIsVastUnwrapEnabled(t *testing.T) {

	type args struct {
		PartnerConfigMap  map[int]map[string]string
		VASTUnwrapTraffic int
	}
	tests := []struct {
		name         string
		args         args
		randomNumber int
		want         bool
	}{
		{
			name: "vastunwrap is enabled and traffic percent in DB and config, DB percent should be preferred",
			args: args{
				PartnerConfigMap: map[int]map[string]string{
					-1: {
						models.VastUnwrapperEnableKey:      "1",
						models.VastUnwrapTrafficPercentKey: "90",
					},
				},
				VASTUnwrapTraffic: 9,
			},
			randomNumber: 10,
			want:         true,
		},
		{
			name: "vastunwrap is enabled and DB traffic percent is less than random number",
			args: args{
				PartnerConfigMap: map[int]map[string]string{
					-1: {
						models.VastUnwrapperEnableKey:      "1",
						models.VastUnwrapTrafficPercentKey: "90",
					},
				},
				VASTUnwrapTraffic: 0,
			},
			randomNumber: 91,
			want:         false,
		},
		{
			name: "vastunwrap is dissabled and config traffic percent is less than random number",
			args: args{
				PartnerConfigMap: map[int]map[string]string{
					-1: {
						models.VastUnwrapperEnableKey: "0",
					},
				},
				VASTUnwrapTraffic: 5,
			},
			randomNumber: 7,
			want:         false,
		},
		{
			name: "vastunwrap is enabled and traffic percent not present in DB, random num higher than traffic percent",
			args: args{
				PartnerConfigMap: map[int]map[string]string{
					-1: {
						models.VastUnwrapperEnableKey: "1",
					},
				},
				VASTUnwrapTraffic: 5,
			},
			randomNumber: 10,
			want:         false,
		},

		{
			name: "vastunwrap is enabled and traffic percent not present in DB, random num less than traffic percent",
			args: args{
				PartnerConfigMap: map[int]map[string]string{
					-1: {
						models.VastUnwrapperEnableKey: "1",
					},
				},
				VASTUnwrapTraffic: 10,
			},
			randomNumber: 9,
			want:         true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			GetRandomNumberIn1To100 = func() int {
				return tt.randomNumber
			}
			got := isVastUnwrapEnabled(tt.args.PartnerConfigMap, tt.args.VASTUnwrapTraffic)
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}

func TestGetImpBidFloorParams(t *testing.T) {
	type args struct {
		rCtx        models.RequestCtx
		adUnitCfg   *adunitconfig.AdConfig
		imp         *openrtb2.Imp
		conversions currency.Conversions
	}
	tests := []struct {
		name           string
		args           args
		expBidfloor    float64
		expBidfloorCur string
	}{
		{
			name: "imp_bid_floor_present_IsMaxFloorsEnabled_false",
			args: args{
				rCtx: models.RequestCtx{
					IsMaxFloorsEnabled: false,
				},
				adUnitCfg: &adunitconfig.AdConfig{
					BidFloor:    ptrutil.ToPtr(2.0),
					BidFloorCur: ptrutil.ToPtr("USD"),
				},
				imp: &openrtb2.Imp{
					BidFloor:    0.6,
					BidFloorCur: "USD",
				},
			},
			expBidfloor:    0.6,
			expBidfloorCur: "USD",
		},
		{
			name: "imp_bid_floor_higher_than_adunit_IsMaxFloorsEnabled_true",
			args: args{
				rCtx: models.RequestCtx{
					IsMaxFloorsEnabled: true,
				},
				adUnitCfg: &adunitconfig.AdConfig{
					BidFloor:    ptrutil.ToPtr(2.0),
					BidFloorCur: ptrutil.ToPtr("USD"),
				},
				imp: &openrtb2.Imp{
					BidFloor:    0.6,
					BidFloorCur: "USD",
				},
			},
			expBidfloor:    2,
			expBidfloorCur: "USD",
		},
		{
			name: "imp_bid_floor_less_than_adunit_IsMaxFloorsEnabled_true",
			args: args{
				rCtx: models.RequestCtx{
					IsMaxFloorsEnabled: true,
				},
				adUnitCfg: &adunitconfig.AdConfig{
					BidFloor:    ptrutil.ToPtr(2.0),
					BidFloorCur: ptrutil.ToPtr("USD"),
				},
				imp: &openrtb2.Imp{
					BidFloor:    2.6,
					BidFloorCur: "USD",
				},
			},
			expBidfloor:    2.6,
			expBidfloorCur: "USD",
		},
		{
			name: "bidfloor_is_present_but_bidfloorcur_is_not_present_set_default_currency_USD",
			args: args{
				adUnitCfg: &adunitconfig.AdConfig{
					BidFloor: ptrutil.ToPtr(2.0),
				},
				imp: &openrtb2.Imp{
					BidFloor: 2.6,
				},
			},
			expBidfloor:    2.6,
			expBidfloorCur: "USD",
		},
		{
			name: "imp_bidfloor_and_bidfloorcur_present_adunit_bidfloor_bidfloorcur_present",
			args: args{
				adUnitCfg: &adunitconfig.AdConfig{
					BidFloor:    ptrutil.ToPtr(2.0),
					BidFloorCur: ptrutil.ToPtr("EUR"),
				},
				imp: &openrtb2.Imp{
					BidFloor:    2.6,
					BidFloorCur: "INR",
				},
			},
			expBidfloor:    2.6,
			expBidfloorCur: "INR",
		},
		{
			name: "imp_bidfloor_present_and_bidfloorcur_notpresent_adunit_bidfloor_bidfloorcur_present",
			args: args{
				adUnitCfg: &adunitconfig.AdConfig{
					BidFloor:    ptrutil.ToPtr(2.0),
					BidFloorCur: ptrutil.ToPtr("INR"),
				},
				imp: &openrtb2.Imp{
					BidFloor: 2.6,
				},
			},
			expBidfloor:    2.6,
			expBidfloorCur: "USD",
		},
		{
			name: "imp_level_floor_notpresent_adunit_bidfloor_bidfloorcur_present",
			args: args{
				adUnitCfg: &adunitconfig.AdConfig{
					BidFloor:    ptrutil.ToPtr(2.0),
					BidFloorCur: ptrutil.ToPtr("INR"),
				},
				imp: &openrtb2.Imp{},
			},
			expBidfloor:    2.0,
			expBidfloorCur: "INR",
		},
		{
			name: "imp_bidfloor_notpresent_bidfloorcur_present_adunit_bidfloor_bidfloorcur_present",
			args: args{
				adUnitCfg: &adunitconfig.AdConfig{
					BidFloor:    ptrutil.ToPtr(2.0),
					BidFloorCur: ptrutil.ToPtr("INR"),
				},
				imp: &openrtb2.Imp{
					BidFloorCur: "EUR",
				},
			},
			expBidfloor:    2.0,
			expBidfloorCur: "INR",
		},
		{
			name: "imp_adunit_bidfloor_notpresent_and_imp_adunit_bidfloorcur_present",
			args: args{
				adUnitCfg: &adunitconfig.AdConfig{
					BidFloorCur: ptrutil.ToPtr("INR"),
				},
				imp: &openrtb2.Imp{
					BidFloorCur: "EUR",
				},
			},
			expBidfloor:    0,
			expBidfloorCur: "",
		},
		{
			name: "imp_bidfloor_and_adunit_bidfloorcur_notpresent_and_imp_bidfloorcur_and_adunit_bidfloor_present",
			args: args{
				adUnitCfg: &adunitconfig.AdConfig{
					BidFloor: ptrutil.ToPtr(2.0),
				},
				imp: &openrtb2.Imp{
					BidFloorCur: "EUR",
				},
			},
			expBidfloor:    2.0,
			expBidfloorCur: "USD",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bidfloor, bidfloorCur := getImpBidFloorParams(tt.args.rCtx, tt.args.adUnitCfg, tt.args.imp, tt.args.conversions)
			assert.Equal(t, tt.expBidfloor, bidfloor, tt.name)
			assert.Equal(t, tt.expBidfloorCur, bidfloorCur, tt.name)
		})
	}
}

func TestGetProfileAppStoreUrl(t *testing.T) {
	type args struct {
		rctx models.RequestCtx
	}
	tests := []struct {
		name  string
		args  args
		want  string
		want1 bool
	}{
		{
			name: "AppStoreUrl missing in DB",
			args: args{
				rctx: models.RequestCtx{
					PartnerConfigMap: map[int]map[string]string{
						models.VersionLevelConfigID: {},
					},
				},
			},
			want:  "",
			want1: false,
		},
		{
			name: "Invalid AppStoreUrl",
			args: args{
				rctx: models.RequestCtx{
					PartnerConfigMap: map[int]map[string]string{
						models.VersionLevelConfigID: {
							models.AppStoreUrl: "invalid-url",
						},
					},
				},
			},
			want:  "invalid-url",
			want1: false,
		},
		{
			name: "Valid AppStoreUrl",
			args: args{
				rctx: models.RequestCtx{
					PartnerConfigMap: map[int]map[string]string{
						models.VersionLevelConfigID: {
							models.AppStoreUrl: "https://apps.apple.com/app/id123456789",
						},
					},
				},
			},
			want:  "https://apps.apple.com/app/id123456789",
			want1: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := getProfileAppStoreUrl(tt.args.rctx)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.want1, got1)
		})
	}
}

func TestUpdateSkadnSourceapp(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	type feilds struct {
		m *OpenWrap
	}
	tests := []struct {
		name            string
		rctx            models.RequestCtx
		bidRequest      *openrtb2.BidRequest
		impExt          *models.ImpExtension
		wantAppStoreURL string
		wantSourceApp   string
		fe              feilds
		setup           func() *mock_metrics.MockMetricsEngine
	}{
		{
			name: "Valid AppStoreUrl os is ios and SKAdnetwork is present in imp.ext",
			rctx: models.RequestCtx{
				PartnerConfigMap: map[int]map[string]string{
					models.VersionLevelConfigID: {
						models.AppStoreUrl: "https://apps.apple.com/app/id123456789",
					},
				},
				AppStoreUrl: "https://apps.apple.com/app/id123456789",
			},
			bidRequest: &openrtb2.BidRequest{
				App: &openrtb2.App{},
				Device: &openrtb2.Device{
					OS: "ios",
				},
				Imp: []openrtb2.Imp{
					{
						Ext: json.RawMessage(`{"skadn": {}}`),
					},
				},
			},
			impExt: &models.ImpExtension{
				SKAdnetwork: json.RawMessage(`{}`),
			},
			setup: func() *mock_metrics.MockMetricsEngine {
				return mock_metrics.NewMockMetricsEngine(ctrl)
			},
			wantAppStoreURL: "https://apps.apple.com/app/id123456789",
			wantSourceApp:   "123456789",
		},
		{
			name: "Valid AppStoreUrl os is Android and SKAdnetwork is present in imp.ext",
			rctx: models.RequestCtx{
				PartnerConfigMap: map[int]map[string]string{
					models.VersionLevelConfigID: {
						models.AppStoreUrl: "https://apps.apple.com/app/id",
					},
				},
				AppStoreUrl: "https://apps.apple.com/app/id",
			},
			bidRequest: &openrtb2.BidRequest{
				App: &openrtb2.App{},
				Device: &openrtb2.Device{
					OS: "Android",
				},
				Imp: []openrtb2.Imp{
					{
						Ext: json.RawMessage(`{"skadn": {}}`),
					},
				},
			},
			setup: func() *mock_metrics.MockMetricsEngine {
				return mock_metrics.NewMockMetricsEngine(ctrl)
			},
			wantAppStoreURL: "https://apps.apple.com/app/id",
		},
		{
			name: "Valid AppStoreUrl os is ios but SKAdnetwork missing in imp.ext",
			rctx: models.RequestCtx{
				PartnerConfigMap: map[int]map[string]string{
					models.VersionLevelConfigID: {
						models.AppStoreUrl: "https://apps.apple.com/app/id123456789",
					},
				},
				AppStoreUrl: "https://apps.apple.com/app/id123456789",
			},
			bidRequest: &openrtb2.BidRequest{
				App: &openrtb2.App{},
				Device: &openrtb2.Device{
					OS: "ios",
				},
				Imp: []openrtb2.Imp{
					{
						Ext: json.RawMessage(`{}`),
					},
				},
			},
			setup: func() *mock_metrics.MockMetricsEngine {
				return mock_metrics.NewMockMetricsEngine(ctrl)
			},
			impExt:          &models.ImpExtension{},
			wantAppStoreURL: "https://apps.apple.com/app/id123456789",
		},
		{
			name: "Valid AppStoreUrl os is ios but Itunes ID missing in AppStoreUrl(url is of Android)",
			rctx: models.RequestCtx{
				PartnerConfigMap: map[int]map[string]string{
					models.VersionLevelConfigID: {
						models.AppStoreUrl: "https://apps.apple.com/app/",
					},
				},
				AppStoreUrl:  "https://apps.apple.com/app/",
				PubIDStr:     "5890",
				ProfileIDStr: "1234",
			},
			bidRequest: &openrtb2.BidRequest{
				App: &openrtb2.App{},
				Device: &openrtb2.Device{
					OS: "ios",
				},
				Imp: []openrtb2.Imp{
					{
						Ext: json.RawMessage(`{"skadn": {}}`),
					},
				},
			},
			setup: func() *mock_metrics.MockMetricsEngine {
				mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)
				mockEngine.EXPECT().RecordFailedParsingItuneID("5890", "1234")
				return mockEngine
			},
			impExt: &models.ImpExtension{
				SKAdnetwork: json.RawMessage(`{}`),
			},
			wantAppStoreURL: "https://apps.apple.com/app/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metricsEngine := tt.setup()
			tt.fe.m = &OpenWrap{
				metricEngine: metricsEngine,
			}
			tt.fe.m.updateSkadnSourceapp(tt.rctx, tt.bidRequest, tt.impExt)
			if tt.impExt != nil {
				if tt.impExt.SKAdnetwork != nil {
					var skAdnetwork map[string]interface{}
					if err := json.Unmarshal(tt.impExt.SKAdnetwork, &skAdnetwork); err == nil {
						if _, ok := skAdnetwork["sourceapp"]; ok {
							assert.Equal(t, tt.wantSourceApp, skAdnetwork["sourceapp"])
						}
					}

				}
			}
		})
	}
}

func TestExtractItunesIdFromAppStoreUrl(t *testing.T) {
	type args struct {
		url string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "app_store_url_with_id1",
			args: args{
				url: "https://itunes.apple.com/.../id1175273098",
			},
			want: "1175273098",
		},
		{
			name: "app_store_url_with_id2",
			args: args{
				url: "https://itunes.apple.com/...?id=361285480",
			},
			want: "361285480",
		},
		{
			name: "app_store_url_with_id3",
			args: args{
				url: "https://itunes.apple.com/.../1175273098",
			},
			want: "1175273098",
		},
		{
			name: "app_store_url_with_id4",
			args: args{
				url: "https://itunes.apple.com/.../12345id1175273098",
			},
			want: "1175273098",
		},
		{
			name: "app_store_url_with_id5",
			args: args{
				url: "https://itunes.apple.com/.../id-1175273098",
			},
			want: "1175273098",
		},
		{
			name: "itunes_url_with_no_id",
			args: args{
				url: "https://itunes.apple.com/.../id",
			},
			want: "",
		},
		{
			name: "app_store_url_with_id_and_multiple_ids",
			args: args{
				url: "https://itunes.apple.com/us/app/example-app/id123456789/id987654321",
			},
			want: "987654321",
		},
		{
			name: "app_store_url_with_id_and_text",
			args: args{
				url: "https://itunes.apple.com/us/app/example-app/id123456789text",
			},
			want: "",
		},
		{
			name: "app_store_url_with_id_and_trailing_slash",
			args: args{
				url: "https://itunes.apple.com/us/app/example-app/id123456789/",
			},
			want: "123456789",
		},
		{
			name: "app_store_url_with_id_and_leading_slash",
			args: args{
				url: "https://itunes.apple.com/us/app/example-app//id123456789",
			},
			want: "123456789",
		},
		{
			name: "app_store_url_with_id_and_multiple_slashes",
			args: args{
				url: "https://itunes.apple.com/us/app/example-app/id123456789///",
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractItunesIdFromAppStoreUrl(tt.args.url)
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}

func TestOpenWrapapplyNativeAdUnitConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockCache := mock_cache.NewMockCache(ctrl)
	mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)

	type args struct {
		rCtx models.RequestCtx
		imp  *openrtb2.Imp
	}
	tests := []struct {
		name string
		args args
		want *openrtb2.Imp
	}{
		{
			name: "native_ad_unit_config_is _nil",
			args: args{
				rCtx: models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							NativeAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: nil,
							},
						},
					},
				},
				imp: &openrtb2.Imp{
					ID:     "imp1",
					Native: &openrtb2.Native{},
				},
			},
			want: &openrtb2.Imp{
				ID:     "imp1",
				Native: &openrtb2.Native{},
			},
		},
		{
			name: "native_ad_unit_config_is_disabled",
			args: args{
				rCtx: models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							NativeAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
									Native: &adunitconfig.Native{
										Enabled: ptrutil.ToPtr(false),
									},
								},
							},
						},
					},
				},
				imp: &openrtb2.Imp{
					ID:     "imp1",
					Native: &openrtb2.Native{},
				},
			},
			want: &openrtb2.Imp{
				ID: "imp1",
			},
		},
		{
			name: "native_ad_unit_config_is_enabled",
			args: args{
				rCtx: models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							NativeAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
									Native: &adunitconfig.Native{
										Enabled: ptrutil.ToPtr(true),
									},
								},
							},
						},
					},
				},
				imp: &openrtb2.Imp{
					ID:     "imp1",
					Native: &openrtb2.Native{},
				},
			},
			want: &openrtb2.Imp{
				ID:     "imp1",
				Native: &openrtb2.Native{},
			},
		},
		{
			name: "native_ad_unit_config_with_bid_floor",
			args: args{
				rCtx: models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							NativeAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
									BidFloor:    ptrutil.ToPtr(1.5),
									BidFloorCur: ptrutil.ToPtr("USD"),
									Native:      &adunitconfig.Native{},
								},
							},
						},
					},
				},
				imp: &openrtb2.Imp{
					ID:     "imp1",
					Native: &openrtb2.Native{},
				},
			},
			want: &openrtb2.Imp{
				ID:          "imp1",
				Native:      &openrtb2.Native{},
				BidFloor:    1.5,
				BidFloorCur: "USD",
			},
		},
		{
			name: "native_ad_unit_config_with_expiration",
			args: args{
				rCtx: models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							NativeAdUnitCtx: models.AdUnitCtx{
								AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
									Exp: ptrutil.ToPtr(30),
									Native: &adunitconfig.Native{
										Enabled: ptrutil.ToPtr(true),
									},
								},
							},
						},
					},
				},
				imp: &openrtb2.Imp{
					ID:     "imp1",
					Native: &openrtb2.Native{},
				},
			},
			want: &openrtb2.Imp{
				ID:     "imp1",
				Native: &openrtb2.Native{},
				Exp:    30,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := OpenWrap{
				rateConvertor: currency.NewRateConverter(&http.Client{}, 60*time.Second, "", time.Duration(0)),
				cache:         mockCache,
				metricEngine:  mockEngine,
			}
			m.applyNativeAdUnitConfig(tt.args.rCtx, tt.args.imp)
			assert.Equal(t, tt.want, tt.args.imp)
		})
	}
}

func TestOpenWrap_updateAppLovinMaxRequestSchain(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockFeature := mock_feature.NewMockFeature(ctrl)
	mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)

	originalOw := ow
	defer func() { ow = originalOw }()
	ow = &OpenWrap{pubFeatures: mockFeature}

	tests := []struct {
		name              string
		rctx              *models.RequestCtx
		maxRequest        *openrtb2.BidRequest
		want              *openrtb2.BidRequest
		wantABTestEnabled bool
		setup             func()
	}{
		{
			name: "schain_not_present_in_request",
			rctx: &models.RequestCtx{
				Endpoint: models.EndpointAppLovinMax,
			},
			maxRequest: &openrtb2.BidRequest{
				Source: &openrtb2.Source{},
			},
			setup: func() {
			},
			want: &openrtb2.BidRequest{
				Source: &openrtb2.Source{
					SChain: nil,
				},
			},
			wantABTestEnabled: false,
		},
		{
			name: "schain_removed_from_request",
			rctx: &models.RequestCtx{
				Endpoint: models.EndpointAppLovinMax,
			},
			maxRequest: &openrtb2.BidRequest{
				Source: &openrtb2.Source{
					SChain: &openrtb2.SupplyChain{
						Complete: 1,
						Nodes: []openrtb2.SupplyChainNode{
							{
								ASI: "applovin.com",
								SID: "53bf468f18c5a0e2b7d4e3f748c677c1",
								RID: "494dbe15a3ce08c54f4e456363f35a022247f997",
								HP:  openrtb2.Int8Ptr(1),
							},
						},
					},
				},
			},
			setup: func() {
				mockEngine.EXPECT().RecordRequestWithSchainABTestEnabled()
			},
			want: &openrtb2.BidRequest{
				Source: &openrtb2.Source{
					SChain: &openrtb2.SupplyChain{
						Complete: 1,
						Nodes:    []openrtb2.SupplyChainNode{},
					},
				},
			},
			wantABTestEnabled: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			m := OpenWrap{
				pubFeatures:  mockFeature,
				metricEngine: mockEngine,
			}
			m.updateAppLovinMaxRequestSchain(tt.rctx, tt.maxRequest)
			assert.Equal(t, tt.want, tt.maxRequest)
			assert.Equal(t, tt.wantABTestEnabled, (tt.rctx.ABTestConfigApplied == 1))
		})
	}
}

// Benchmark for extractItunesIdFromAppStoreUrl
func BenchmarkExtractItunesIdFromAppStoreUrl(b *testing.B) {
	testURL := "https://apps.apple.com/us/app/example-app/id=123456789"
	for i := 0; i < b.N; i++ {
		extractItunesIdFromAppStoreUrl(testURL)
	}
}

func TestLogHookBidRequest(t *testing.T) {
	type args struct {
		stage      string
		rCtx       models.RequestCtx
		bidRequest *openrtb2.BidRequest
		nbrCode    int
	}
	tests := []struct {
		name        string
		args        args
		description string
	}{
		{
			name: "log_hook_start_with_valid_bidrequest",
			args: args{
				stage: "hook_start",
				rCtx: models.RequestCtx{
					PubID:     5890,
					ProfileID: 1234,
					Endpoint:  models.EndpointV25,
				},
				bidRequest: &openrtb2.BidRequest{
					ID: "test-request-id",
					Imp: []openrtb2.Imp{
						{ID: "imp1"},
					},
				},
				nbrCode: 0,
			},
			description: "Should log hook start without nbr code",
		},
		{
			name: "log_hook_end_rejected_with_nbr_code",
			args: args{
				stage: "hook_end_rejected",
				rCtx: models.RequestCtx{
					PubID:     5890,
					ProfileID: 1234,
					Endpoint:  models.EndpointV25,
				},
				bidRequest: &openrtb2.BidRequest{
					ID: "test-request-id",
					Imp: []openrtb2.Imp{
						{ID: "imp1"},
					},
				},
				nbrCode: int(nbr.InvalidImpressionTagID),
			},
			description: "Should log hook end rejected with nbr code",
		},
		{
			name: "log_hook_end_success_without_nbr_code",
			args: args{
				stage: "hook_end_success",
				rCtx: models.RequestCtx{
					PubID:     5890,
					ProfileID: 1234,
					Endpoint:  models.EndpointV25,
				},
				bidRequest: &openrtb2.BidRequest{
					ID: "test-request-id",
					Imp: []openrtb2.Imp{
						{ID: "imp1"},
					},
				},
				nbrCode: 0,
			},
			description: "Should log hook end success without nbr code",
		},
		{
			name: "log_with_empty_bidrequest",
			args: args{
				stage: "hook_start",
				rCtx: models.RequestCtx{
					PubID:     5890,
					ProfileID: 1234,
					Endpoint:  models.EndpointAMP,
				},
				bidRequest: &openrtb2.BidRequest{
					ID: "empty-request",
				},
				nbrCode: 0,
			},
			description: "Should handle empty bidrequest gracefully",
		},
		{
			name: "log_with_different_endpoint",
			args: args{
				stage: "hook_start",
				rCtx: models.RequestCtx{
					PubID:     1234,
					ProfileID: 5678,
					Endpoint:  models.EndpointJson,
				},
				bidRequest: &openrtb2.BidRequest{
					ID: "json-endpoint-request",
					Imp: []openrtb2.Imp{
						{ID: "imp1", TagID: "div1"},
					},
				},
				nbrCode: 0,
			},
			description: "Should log with different endpoint types",
		},
		{
			name: "log_with_high_nbr_code",
			args: args{
				stage: "hook_end_rejected",
				rCtx: models.RequestCtx{
					PubID:     9999,
					ProfileID: 8888,
					Endpoint:  models.EndpointWebS2S,
				},
				bidRequest: &openrtb2.BidRequest{
					ID: "high-nbr-request",
				},
				nbrCode: int(nbr.AllPartnerThrottled),
			},
			description: "Should log with high nbr code value",
		},
		{
			name: "log_with_nil_bidrequest",
			args: args{
				stage: "hook_start",
				rCtx: models.RequestCtx{
					PubID:     5890,
					ProfileID: 1234,
					Endpoint:  models.EndpointV25,
				},
				bidRequest: nil,
				nbrCode:    0,
			},
			description: "Should handle nil bidrequest gracefully",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test verifies that logHookBidRequest doesn't panic
			// and can handle various input combinations
			// The actual logging output is verified through glog which is tested separately
			assert.NotPanics(t, func() {
				logHookBidRequest(tt.args.stage, tt.args.rCtx, tt.args.bidRequest, tt.args.nbrCode)
			}, tt.description)
		})
	}
}

func TestLogHookBidRequest_NilBidRequest(t *testing.T) {
	// Test that function handles nil BidRequest gracefully
	rCtx := models.RequestCtx{
		PubID:     5890,
		ProfileID: 1234,
		Endpoint:  models.EndpointV25,
	}

	// Should not panic with nil BidRequest
	assert.NotPanics(t, func() {
		logHookBidRequest("hook_start", rCtx, nil, 0)
	}, "Should handle nil BidRequest without panicking")

	// Should not panic with empty BidRequest
	assert.NotPanics(t, func() {
		logHookBidRequest("hook_start", rCtx, &openrtb2.BidRequest{}, 0)
	}, "Should handle empty BidRequest without panicking")
}

func TestApplyNativeVideoAssetRulesFromAdUnitConfig(t *testing.T) {
	minDur := int64(5)
	maxDur := int64(15)

	makeNativeRequest := func(t *testing.T, req nativeRequests.Request) string {
		b, err := json.Marshal(&req)
		assert.NoError(t, err)
		return string(b)
	}

	type args struct {
		nativeCfg *adunitconfig.NativeConfig
		impNative *openrtb2.Native
		pubID     int
		profileID int
	}

	tests := []struct {
		name          string
		args          args
		wantImpNative *openrtb2.Native
	}{
		{
			name: "video_disabled_removes_all_video_assets_with_multiple_assets",
			args: args{
				nativeCfg: &adunitconfig.NativeConfig{Video: adunitconfig.NativeVideo{Enabled: ptrutil.ToPtr(false)}},
				impNative: &openrtb2.Native{Request: makeNativeRequest(t, nativeRequests.Request{Assets: []nativeRequests.Asset{
					{ID: 1, Video: &nativeRequests.Video{MinDuration: 1, MaxDuration: 2}},
					{ID: 2, Title: &nativeRequests.Title{Len: 10}},
					{ID: 3, Video: &nativeRequests.Video{MinDuration: 3, MaxDuration: 4}},
				}})},
				pubID:     rctx.PubID,
				profileID: rctx.ProfileID,
			},
			wantImpNative: &openrtb2.Native{Request: makeNativeRequest(t, nativeRequests.Request{Assets: []nativeRequests.Asset{
				{ID: 2, Title: &nativeRequests.Title{Len: 10}},
			}})},
		},
		{
			name: "video_enabled_updates_durations_only_for_video_assets_with_multiple_assets",
			args: args{
				nativeCfg: &adunitconfig.NativeConfig{Video: adunitconfig.NativeVideo{Enabled: ptrutil.ToPtr(true), Config: adunitconfig.NativeVideoConfig{MinDuration: &minDur, MaxDuration: &maxDur}}},
				impNative: &openrtb2.Native{Request: makeNativeRequest(t, nativeRequests.Request{Assets: []nativeRequests.Asset{
					{ID: 1, Title: &nativeRequests.Title{Len: 25}},
					{ID: 2, Video: &nativeRequests.Video{MinDuration: 1, MaxDuration: 2}},
					{ID: 3, Video: &nativeRequests.Video{MinDuration: 100, MaxDuration: 200}},
				}})},
				pubID:     rctx.PubID,
				profileID: rctx.ProfileID,
			},
			wantImpNative: &openrtb2.Native{Request: makeNativeRequest(t, nativeRequests.Request{Assets: []nativeRequests.Asset{
				{ID: 1, Title: &nativeRequests.Title{Len: 25}},
				{ID: 2, Video: &nativeRequests.Video{MinDuration: minDur, MaxDuration: maxDur}},
				{ID: 3, Video: &nativeRequests.Video{MinDuration: minDur, MaxDuration: maxDur}},
			}})},
		},
		{
			name: "video_enabled_with_no_duration_config_does_not_modify_request",
			args: args{
				nativeCfg: &adunitconfig.NativeConfig{Video: adunitconfig.NativeVideo{Enabled: ptrutil.ToPtr(true)}},
				impNative: &openrtb2.Native{Request: makeNativeRequest(t, nativeRequests.Request{Assets: []nativeRequests.Asset{
					{ID: 1, Video: &nativeRequests.Video{MinDuration: 1, MaxDuration: 2}},
					{ID: 2, Title: &nativeRequests.Title{Len: 10}},
				}})},
				pubID:     rctx.PubID,
				profileID: rctx.ProfileID,
			},
			wantImpNative: &openrtb2.Native{Request: makeNativeRequest(t, nativeRequests.Request{Assets: []nativeRequests.Asset{
				{ID: 1, Video: &nativeRequests.Video{MinDuration: 1, MaxDuration: 2}},
				{ID: 2, Title: &nativeRequests.Title{Len: 10}},
			}})},
		},
		{
			name: "invalid_native_request_json_is_noop",
			args: args{
				nativeCfg: &adunitconfig.NativeConfig{Video: adunitconfig.NativeVideo{Enabled: ptrutil.ToPtr(true), Config: adunitconfig.NativeVideoConfig{MinDuration: &minDur, MaxDuration: &maxDur}}},
				impNative: &openrtb2.Native{Request: "{"},
				pubID:     rctx.PubID,
				profileID: rctx.ProfileID,
			},
			wantImpNative: &openrtb2.Native{Request: "{"},
		},
		{
			name: "no_change_in_video_assets_if_video_config_not_disabled_explicitely",
			args: args{
				nativeCfg: &adunitconfig.NativeConfig{},
				impNative: &openrtb2.Native{Request: makeNativeRequest(t, nativeRequests.Request{Assets: []nativeRequests.Asset{
					{ID: 1, Video: &nativeRequests.Video{MinDuration: 1, MaxDuration: 2}},
					{ID: 2, Title: &nativeRequests.Title{Len: 10}},
				}})},
				pubID:     rctx.PubID,
				profileID: rctx.ProfileID,
			},
			wantImpNative: &openrtb2.Native{Request: makeNativeRequest(t, nativeRequests.Request{Assets: []nativeRequests.Asset{
				{ID: 1, Video: &nativeRequests.Video{MinDuration: 1, MaxDuration: 2}},
				{ID: 2, Title: &nativeRequests.Title{Len: 10}},
			}})},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applyNativeVideoAssetRulesFromAdUnitConfig(tt.args.nativeCfg, tt.args.impNative, tt.args.pubID, tt.args.profileID)
			assert.Equal(t, tt.wantImpNative, tt.args.impNative)
		})
	}
}
