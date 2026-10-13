package vastlint

import (
	"encoding/json"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/ptrutil"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestCreativePrivacySuppressions(t *testing.T) {
	for name, mutate := range map[string]func(*openrtb_ext.RequestWrapper){
		"missing_regs": func(r *openrtb_ext.RequestWrapper) { r.Regs = nil },
		"unknown_gdpr": func(r *openrtb_ext.RequestWrapper) { r.Regs.GDPR = nil },
		"gdpr":         func(r *openrtb_ext.RequestWrapper) { r.Regs.GDPR = ptrutil.ToPtr(int8(1)) },
		"invalid_gdpr": func(r *openrtb_ext.RequestWrapper) { r.Regs.GDPR = ptrutil.ToPtr(int8(-1)) },
		"gpc":          func(r *openrtb_ext.RequestWrapper) { r.Regs.Ext = json.RawMessage(`{"gpc":"1"}`) },
		"coppa":        func(r *openrtb_ext.RequestWrapper) { r.Regs.COPPA = 1 },
		"gpp":          func(r *openrtb_ext.RequestWrapper) { r.Regs.GPP = "consent" },
		"gpp_sections": func(r *openrtb_ext.RequestWrapper) { r.Regs.GPPSID = []int8{7} },
		"us_privacy":   func(r *openrtb_ext.RequestWrapper) { r.Regs.USPrivacy = "1NNN" },
		"consent":      func(r *openrtb_ext.RequestWrapper) { r.User = &openrtb2.User{Consent: "consent"} },
		"legacy_consent": func(r *openrtb_ext.RequestWrapper) {
			r.User = &openrtb2.User{Ext: json.RawMessage(`{"consent":"consent"}`)}
		},
		"legacy_gdpr":           func(r *openrtb_ext.RequestWrapper) { r.Regs.Ext = json.RawMessage(`{"gdpr":1}`) },
		"malformed_regulations": func(r *openrtb_ext.RequestWrapper) { r.Regs.Ext = json.RawMessage(`{`) },
		"oversized_regulations": func(r *openrtb_ext.RequestWrapper) { r.Regs.Ext = json.RawMessage(strings.Repeat("x", 4097)) },
		"malformed_user":        func(r *openrtb_ext.RequestWrapper) { r.User = &openrtb2.User{Ext: json.RawMessage(`{`)} },
		"dnt":                   func(r *openrtb_ext.RequestWrapper) { r.Device = &openrtb2.Device{DNT: ptrutil.ToPtr(int8(1))} },
		"lmt":                   func(r *openrtb_ext.RequestWrapper) { r.Device = &openrtb2.Device{Lmt: ptrutil.ToPtr(int8(1))} },
	} {
		t.Run(name, func(t *testing.T) {
			m := testModule(t)
			event := eventWith(video("<VAST/>"))
			mutate(event.RequestWrapper)
			m.LogAuctionObject(event)
			require.Empty(t, m.jobs)
			require.EqualValues(t, 1, m.Stats()["privacy_suppressed"])
		})
	}
	for _, legacy := range []bool{false, true} {
		event := eventWith(video("<VAST/>"))
		if legacy {
			event.RequestWrapper.Regs = &openrtb2.Regs{Ext: json.RawMessage(`{"gdpr":0}`)}
		}
		require.True(t, creativePrivacyAllowed(event.RequestWrapper))
	}
	require.False(t, creativePrivacyAllowed(nil))
}
