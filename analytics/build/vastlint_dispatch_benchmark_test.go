package build

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/analytics"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/privacy"
)

func BenchmarkVastlintFullDispatch(b *testing.B) {
	for _, scenario := range []string{"disabled", "enabled_ineligible", "enabled_eligible_no_bid"} {
		for _, payload := range []int{0, 65536} {
			name := scenario + "/small_request"
			if payload > 0 {
				name = scenario + "/64KiB_user_ext"
			}
			b.Run(name, func(b *testing.B) {
				runner := New(&config.Analytics{Vastlint: config.VastlintAnalytics{Enabled: scenario != "disabled", Endpoint: "127.0.0.1:1", AllowInsecure: true, Accounts: []string{"host-account"}}})
				defer runner.Shutdown()
				req := getDefaultBidRequest()
				req.Ext = json.RawMessage(`{"prebid":{"targeting":{}}}`)
				if payload > 0 {
					req.User.Ext = json.RawMessage(`{"data":{"padding":"` + strings.Repeat("x", payload) + `"}}`)
				}
				account := "host-account"
				if scenario == "enabled_ineligible" {
					account = "unlisted-account"
				}
				ao := &analytics.AuctionObject{Status: 200, Account: &config.Account{ID: account}, FinalResponse: &openrtb2.BidResponse{ID: "auction"}, RequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: req}}
				ac := privacy.NewActivityControl(&config.AccountPrivacy{})
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					runner.LogAuctionObject(ao, ac)
				}
			})
		}
	}
}
