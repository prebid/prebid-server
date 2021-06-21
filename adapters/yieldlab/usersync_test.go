package yieldlab

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/gdpr"
)

func TestYieldlabSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("https://ad.yieldlab.net/mr?t=2&pid=9140838&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&redirectUri=http%3A%2F%2Flocalhost%2F%2Fsetuid%3Fbidder%3Dyieldlab%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%25%25YL_UID%25%25"))
	syncer := NewYieldlabSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal: "0",
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, "https://ad.yieldlab.net/mr?t=2&pid=9140838&gdpr=0&gdpr_consent=&redirectUri=http%3A%2F%2Flocalhost%2F%2Fsetuid%3Fbidder%3Dyieldlab%26gdpr%3D0%26gdpr_consent%3D%26uid%3D%25%25YL_UID%25%25", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.False(t, syncInfo.SupportCORS)
}
