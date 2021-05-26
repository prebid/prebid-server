package aja

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy/ccpa"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestAJASyncer(t *testing.T) {
	syncURL := "https://ad.as.amanad.adtdp.com/v1/sync/ssp?ssp=4&gdpr={{.GDPR}}&us_privacy={{.USPrivacy}}&redir=localhost/setuid?bidder=aja&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&uid=%s"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewAJASyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal:  "1",
			Consent: "BOPVK28OVJoTBABABAENBs-AAAAhuAKAANAAoACwAGgAPAAxAB0AHgAQAAiABOADkA",
		},
		CCPA: ccpa.Policy{
			Consent: "C",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "https://ad.as.amanad.adtdp.com/v1/sync/ssp?ssp=4&gdpr=1&us_privacy=C&redir=localhost/setuid?bidder=aja&gdpr=1&gdpr_consent=BOPVK28OVJoTBABABAENBs-AAAAhuAKAANAAoACwAGgAPAAxAB0AHgAQAAiABOADkA&uid=%s", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.Equal(t, false, syncInfo.SupportCORS)
}
