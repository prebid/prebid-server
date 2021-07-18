package bidmyadz

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestNewBidmyadzSyncer(t *testing.T) {
	syncURL := "https://test.com/c0f68227d14ed938c6c49f3967cbe9bc?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&ccpa={{.USPrivacy}}"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)
	syncer := NewBidmyadzSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal:  "0",
			Consent: "allGdpr",
		},
		CCPA: ccpa.Policy{
			Consent: "1---",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "https://test.com/c0f68227d14ed938c6c49f3967cbe9bc?gdpr=0&gdpr_consent=allGdpr&ccpa=1---", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.Equal(t, false, syncInfo.SupportCORS)
}
