package gamma

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestGammaSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("//hb.gammaplatform.com/sync?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&redirectUri=http%3A%2F%2Flocalhost%2F%2Fsetuid%3Fbidder%3Dgamma%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID"))
	syncer := NewGammaSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo("0", "")
	assert.NoError(t, err)
	assert.Equal(t, "//hb.gammaplatform.com/sync?gdpr=0&gdpr_consent=&redirectUri=http%3A%2F%2Flocalhost%2F%2Fsetuid%3Fbidder%3Dgamma%26gdpr%3D0%26gdpr_consent%3D%26uid%3D%24UID", syncInfo.URL)
	assert.Equal(t, "iframe", syncInfo.Type)
	assert.EqualValues(t, 0, syncer.GDPRVendorID())
	assert.False(t, syncInfo.SupportCORS)
}
