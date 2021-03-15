package krushmedia

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestKrushmediaSyncer(t *testing.T) {
	syncURL := "https://cs.krushmedia.com/4e4abdd5ecc661643458a730b1aa927d.gif?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redir=http%3A%2F%2Flocalhost%3A8000%2Fsetuid%3Fbidder%3Dkrushmedia%26uid%3D%5BUID%5D"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)
	syncer := NewKrushmediaSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal:  "0",
			Consent: "allGdpr",
		},
		CCPA: ccpa.Policy{
			Consent: "1-YY",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "https://cs.krushmedia.com/4e4abdd5ecc661643458a730b1aa927d.gif?gdpr=0&gdpr_consent=allGdpr&us_privacy=1-YY&redir=http%3A%2F%2Flocalhost%3A8000%2Fsetuid%3Fbidder%3Dkrushmedia%26uid%3D%5BUID%5D", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.Equal(t, false, syncInfo.SupportCORS)
}
