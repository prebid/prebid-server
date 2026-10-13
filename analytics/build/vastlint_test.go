package build

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/prebid/prebid-server/v4/analytics"
	"github.com/prebid/prebid-server/v4/analytics/vastlint"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/privacy"
	"github.com/prebid/prebid-server/v4/util/ptrutil"
	"github.com/stretchr/testify/require"
)

type creativeModule struct {
	sampleModule
	requires bool
}

func (m *creativeModule) RequiresCreativeDataPermissions() bool { return m.requires }

func TestCreativeTransmissionActivities(t *testing.T) {
	for _, activity := range []string{"reportAnalytics", "transmitUfpd", "transmitPreciseGeo", "transmitUniqueRequestIds", "transmitTid", "allowed", "optional_false", "legacy"} {
		t.Run(activity, func(t *testing.T) {
			count := 0
			var module analytics.Module = &creativeModule{sampleModule: sampleModule{count: &count}, requires: activity != "optional_false"}
			if activity == "legacy" {
				module = &sampleModule{count: &count}
			}
			cfg := &config.AccountPrivacy{AllowActivities: &config.AllowActivities{}}
			deny := config.Activity{Default: ptrutil.ToPtr(false)}
			switch activity {
			case "reportAnalytics":
				cfg.AllowActivities.ReportAnalytics = deny
			case "transmitUfpd", "optional_false", "legacy":
				cfg.AllowActivities.TransmitUserFPD = deny
			case "transmitPreciseGeo":
				cfg.AllowActivities.TransmitPreciseGeo = deny
			case "transmitUniqueRequestIds":
				cfg.AllowActivities.TransmitUniqueRequestIds = deny
			case "transmitTid":
				cfg.AllowActivities.TransmitTids = deny
			}
			ac := privacy.NewActivityControl(cfg)
			runner := enabledAnalytics{"vastlint": module}
			request := func() *openrtb_ext.RequestWrapper {
				return &openrtb_ext.RequestWrapper{BidRequest: getDefaultBidRequest()}
			}
			runner.LogAuctionObject(&analytics.AuctionObject{RequestWrapper: request()}, ac)
			runner.LogAmpObject(&analytics.AmpObject{RequestWrapper: request()}, ac)
			runner.LogVideoObject(&analytics.VideoObject{RequestWrapper: request()}, ac)
			if activity == "allowed" || activity == "optional_false" || activity == "legacy" {
				require.Equal(t, 3, count)
			} else {
				require.Zero(t, count)
			}
		})
	}
}

func TestVastlintBuildOptIn(t *testing.T) {
	invalid := New(&config.Analytics{Vastlint: config.VastlintAnalytics{Enabled: true}}).(enabledAnalytics)
	require.Empty(t, invalid)
	valid := New(&config.Analytics{Vastlint: config.VastlintAnalytics{Enabled: true, Endpoint: "localhost:50051", Accounts: []string{"host-account"}}}).(enabledAnalytics)
	require.Contains(t, valid, "vastlint")
	require.IsType(t, &vastlint.Module{}, valid["vastlint"])
	valid.Shutdown()
}

func TestNoAnalyticsSettingsAvoidRequestCopy(t *testing.T) {
	request := getDefaultBidRequest()
	request.Ext = json.RawMessage(`{"prebid":{"targeting":{}}}`)
	request.User.Ext = json.RawMessage(`{"padding":"` + strings.Repeat("x", 65536) + `"}`)
	wrapper := &openrtb_ext.RequestWrapper{BidRequest: request}
	require.Nil(t, updateReqWrapperForAnalytics(wrapper, "vastlint", false))
	allocations := testing.AllocsPerRun(100, func() { require.Nil(t, updateReqWrapperForAnalytics(wrapper, "vastlint", false)) })
	require.Zero(t, allocations)
}
