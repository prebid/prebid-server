package advangelists

import (
	"testing"
	"text/template"

<<<<<<< HEAD
=======
	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/gdpr"
>>>>>>> fb386190f4491648bb1e8d1b0345a333be1c0393
	"github.com/stretchr/testify/assert"
)

func TestAdvangelistsSyncer(t *testing.T) {
<<<<<<< HEAD
	temp := template.Must(template.New("sync-template").Parse("https://nep.advangelists.com/xp/user-sync?acctid={aid}&&redirect=localhost/setuid?bidder=advangelists&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&uid=$UID"))
	syncer := NewAdvangelistsSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo("1", "BOPVK28OVJoTBABABAENBs-AAAAhuAKAANAAoACwAGgAPAAxAB0AHgAQAAiABOADkA")
=======
	syncURL := "https://nep.advangelists.com/xp/user-sync?acctid={aid}&&redirect=localhost/setuid?bidder=advangelists&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&uid=$UID"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewAdvangelistsSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal:  "1",
			Consent: "BOPVK28OVJoTBABABAENBs-AAAAhuAKAANAAoACwAGgAPAAxAB0AHgAQAAiABOADkA",
		},
	})

>>>>>>> fb386190f4491648bb1e8d1b0345a333be1c0393
	assert.NoError(t, err)
	assert.Equal(t, "https://nep.advangelists.com/xp/user-sync?acctid={aid}&&redirect=localhost/setuid?bidder=advangelists&gdpr=1&gdpr_consent=BOPVK28OVJoTBABABAENBs-AAAAhuAKAANAAoACwAGgAPAAxAB0AHgAQAAiABOADkA&uid=$UID", syncInfo.URL)
	assert.Equal(t, "iframe", syncInfo.Type)
	assert.EqualValues(t, 61, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
