package outbrain

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestSyncer(t *testing.T) {
	syncURL := "http://prebidtest.zemanta.com/usersync/prebidtest?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&cb=host%2Fsetuid%3Fbidder%3Dzemanta%26uid%3D__ZUID__"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewOutbrainSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal:  "A",
			Consent: "B",
		},
		CCPA: ccpa.Policy{
			Consent: "C",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "http://prebidtest.zemanta.com/usersync/prebidtest?gdpr=A&gdpr_consent=B&us_privacy=C&cb=host%2Fsetuid%3Fbidder%3Dzemanta%26uid%3D__ZUID__", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
}
