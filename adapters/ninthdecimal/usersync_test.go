package ninthdecimal

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestNinthdecimalSyncer(t *testing.T) {
	syncURL := "https://rtb.ninthdecimal.com/xp/user-sync?acctid={aid}&&redirect=localhost/setuid?bidder=ninthdecimal&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&uid=$UID"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewNinthdecimalSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal:  "1",
			Consent: "BOPVK28OVJoTBABABAENBs-AAAAhuAKAANAAoACwAGgAPAAxAB0AHgAQAAiABOADkA",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "https://rtb.ninthdecimal.com/xp/user-sync?acctid={aid}&&redirect=localhost/setuid?bidder=ninthdecimal&gdpr=1&gdpr_consent=BOPVK28OVJoTBABABAENBs-AAAAhuAKAANAAoACwAGgAPAAxAB0AHgAQAAiABOADkA&uid=$UID", syncInfo.URL)
	assert.Equal(t, "iframe", syncInfo.Type)
	assert.EqualValues(t, 61, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
