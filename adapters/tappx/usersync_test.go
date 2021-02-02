package tappx

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestTappxSyncer(t *testing.T) {
	syncURL := "//ssp.api.tappx.com/cs/usersync.php?gdpr_optin={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&type=iframe&ruid=localhost%2Fsetuid%3Fbidder%3Dtappx%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%7B%7BTPPXUID%7D%7D"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewTappxSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal:  "1",
			Consent: "CPAQoFAPAQoFaAeABBESBECoAPLAAHLAAAiQHKtd_X_fb39j-_59_9t0eY1f9_7_v20zjgeds-8Nyd_X_L8X42M7vF36pq4KuR4Eu3LBIQFlHOHcTUmw6IkVqTPsak2Mr7NKJ7PEinMbe2dYGHtfn9VTuZKYr97s___z__-__v__75f_r-3_3_vp9V-2-_egcqASYal8BFmJY4Ek0aVQogQhXEh0AIAKKEYWiawgJXBTsrgI_QQMAEBqAjAiBBiCjFkEAAAAASURASAHggEQBEAgABACpAQgAI0AAWAEgYBAAKAaFABFAEIEhBEYFRymBARItFBPJGAJRd7GGEIZRQAUCj-AAAAA.YAAAAAAAAAAA",
		},
		CCPA: ccpa.Policy{
			Consent: "1YNN",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "//ssp.api.tappx.com/cs/usersync.php?gdpr_optin=1&gdpr_consent=CPAQoFAPAQoFaAeABBESBECoAPLAAHLAAAiQHKtd_X_fb39j-_59_9t0eY1f9_7_v20zjgeds-8Nyd_X_L8X42M7vF36pq4KuR4Eu3LBIQFlHOHcTUmw6IkVqTPsak2Mr7NKJ7PEinMbe2dYGHtfn9VTuZKYr97s___z__-__v__75f_r-3_3_vp9V-2-_egcqASYal8BFmJY4Ek0aVQogQhXEh0AIAKKEYWiawgJXBTsrgI_QQMAEBqAjAiBBiCjFkEAAAAASURASAHggEQBEAgABACpAQgAI0AAWAEgYBAAKAaFABFAEIEhBEYFRymBARItFBPJGAJRd7GGEIZRQAUCj-AAAAA.YAAAAAAAAAAA&us_privacy=1YNN&type=iframe&ruid=localhost%2Fsetuid%3Fbidder%3Dtappx%26gdpr%3D1%26gdpr_consent%3DCPAQoFAPAQoFaAeABBESBECoAPLAAHLAAAiQHKtd_X_fb39j-_59_9t0eY1f9_7_v20zjgeds-8Nyd_X_L8X42M7vF36pq4KuR4Eu3LBIQFlHOHcTUmw6IkVqTPsak2Mr7NKJ7PEinMbe2dYGHtfn9VTuZKYr97s___z__-__v__75f_r-3_3_vp9V-2-_egcqASYal8BFmJY4Ek0aVQogQhXEh0AIAKKEYWiawgJXBTsrgI_QQMAEBqAjAiBBiCjFkEAAAAASURASAHggEQBEAgABACpAQgAI0AAWAEgYBAAKAaFABFAEIEhBEYFRymBARItFBPJGAJRd7GGEIZRQAUCj-AAAAA.YAAAAAAAAAAA%26uid%3D%7B%7BTPPXUID%7D%7D", syncInfo.URL)
	assert.Equal(t, "iframe", syncInfo.Type)
	assert.EqualValues(t, 628, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}