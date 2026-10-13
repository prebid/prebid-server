package floxis

import (
	"encoding/json"
	"net/url"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/macros"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/usersync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdapexBidderInfo(t *testing.T) {
	infos, err := config.LoadBidderInfoFromDisk("../../static/bidder-info")
	require.NoError(t, err)
	adapex, exists := infos["adapex"]
	require.True(t, exists)
	floxis := infos["floxis"]

	t.Run("inherited capabilities", func(t *testing.T) {
		assert.Equal(t, "floxis", adapex.AliasOf)
		assert.Equal(t, uint16(1609), adapex.GVLVendorID)
		assert.Equal(t, "prebid@floxis.tech", adapex.Maintainer.Email)
		assert.Equal(t, floxis.Capabilities, adapex.Capabilities)
		assert.Equal(t, floxis.OpenRTB, adapex.OpenRTB)
		assert.Equal(t, floxis.ModifyingVastXmlAllowed, adapex.ModifyingVastXmlAllowed)
		assert.Equal(t, "gzip", adapex.EndpointCompression)
	})

	t.Run("request endpoint and privacy", func(t *testing.T) {
		bidder, err := Builder(openrtb_ext.BidderName("adapex"), config.Adapter{Endpoint: adapex.Endpoint}, config.Server{})
		require.NoError(t, err)
		request := &openrtb2.BidRequest{
			ID:   "request",
			Imp:  []openrtb2.Imp{bannerImp(`{"bidder":{"seat":"a b&c","region":"eu","partner":"acme"}}`)},
			Regs: &openrtb2.Regs{GDPR: int8Ptr(1), GPP: "consent", GPPSID: []int8{2, 6}, USPrivacy: "1YNN", COPPA: 1},
			User: &openrtb2.User{Consent: "gdpr-consent", Ext: json.RawMessage(`{"eids":[{"source":"example.com","uids":[{"id":"uid"}]}]}`)},
		}
		before, err := json.Marshal(request)
		require.NoError(t, err)
		requests, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})
		require.Empty(t, errs)
		require.Len(t, requests, 1)
		assert.Equal(t, "https://hb.adapex.io/pbs?seat=a+b%26c", requests[0].Uri)
		assert.JSONEq(t, string(before), string(requests[0].Body))
		after, err := json.Marshal(request)
		require.NoError(t, err)
		assert.JSONEq(t, string(before), string(after))
	})

	t.Run("inherited params schema", func(t *testing.T) {
		validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
		require.NoError(t, err)
		name := openrtb_ext.BidderName("adapex")
		assert.Equal(t, validator.Schema(openrtb_ext.BidderFloxis), validator.Schema(name))
		for _, params := range validParams {
			assert.NoError(t, validator.Validate(name, json.RawMessage(params)), params)
		}
		for _, params := range invalidParams {
			assert.Error(t, validator.Validate(name, json.RawMessage(params)), params)
		}
	})

	t.Run("distinct user sync with privacy", func(t *testing.T) {
		syncers, errs := usersync.BuildSyncers(&config.Configuration{
			ExternalURL: "https://pbs.example.com",
			UserSync:    config.UserSync{RedirectURL: "{{.ExternalURL}}/setuid?bidder={{.SyncerKey}}&uid={{.UserMacro}}"},
		}, config.BidderInfos{"adapex": adapex, "floxis": floxis})
		require.Empty(t, errs)
		assert.Equal(t, "adapex", syncers["adapex"].Key())
		assert.Equal(t, "floxis", syncers["floxis"].Key())
		sync, err := syncers["adapex"].GetSync([]usersync.SyncType{usersync.SyncTypeRedirect}, macros.UserSyncPrivacy{
			GDPR: "1", GDPRConsent: "consent", GPP: "gpp", GPPSID: "2,6", USPrivacy: "1YNN",
		})
		require.NoError(t, err)
		endpoint, err := url.Parse(sync.URL)
		require.NoError(t, err)
		assert.Equal(t, "https", endpoint.Scheme)
		assert.Equal(t, "sync.adapex.io", endpoint.Host)
		assert.Equal(t, "/sync", endpoint.Path)
		query := endpoint.Query()
		assert.Equal(t, "1", query.Get("gdpr"))
		assert.Equal(t, "consent", query.Get("gdpr_consent"))
		assert.Equal(t, "gpp", query.Get("gpp"))
		assert.Equal(t, "2,6", query.Get("gpp_sid"))
		assert.Equal(t, "1YNN", query.Get("us_privacy"))
		assert.Equal(t, "https://pbs.example.com/setuid?bidder=adapex&uid=${USER_ID}", query.Get("dest"))
	})
}

func int8Ptr(value int8) *int8 {
	return &value
}
