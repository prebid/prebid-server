package adapters

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestGetUsersyncInfo(t *testing.T) {
	privacyPolicies := privacy.Policies{
		GDPR: gdpr.Policy{
			Signal:  "A",
			Consent: "B",
		},
		CCPA: ccpa.Policy{
			Consent: "C",
		},
	}

	syncURL := "{{.GDPR}}{{.GDPRConsent}}{{.USPrivacy}}"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)
	syncer := Syncer{
		urlTemplate: syncURLTemplate,
	}

	syncInfo, err := syncer.GetUsersyncInfo(privacyPolicies)

	assert.NoError(t, err)
	assert.Equal(t, "ABC", syncInfo.URL)
}
