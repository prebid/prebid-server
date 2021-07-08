package smartadserver

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestSmartadserverSyncer(t *testing.T) {
	syncURL := "//ssbsync.smartadserver.com/getuid?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&url=localhost%2Fsetuid%3Fbidder%3Dsmartadserver%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%5Bsas_uid%5D%22"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewSmartadserverSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal:  "1",
			Consent: "COyASAoOyASAoAfAAAENAfCAAAAAAAAAAAAAAAAAAAAA",
		},
		CCPA: ccpa.Policy{
			Consent: "1YNN",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "//ssbsync.smartadserver.com/getuid?gdpr=1&gdpr_consent=COyASAoOyASAoAfAAAENAfCAAAAAAAAAAAAAAAAAAAAA&us_privacy=1YNN&url=localhost%2Fsetuid%3Fbidder%3Dsmartadserver%26gdpr%3D1%26gdpr_consent%3DCOyASAoOyASAoAfAAAENAfCAAAAAAAAAAAAAAAAAAAAA%26uid%3D%5Bsas_uid%5D%22", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.Equal(t, false, syncInfo.SupportCORS)
}
