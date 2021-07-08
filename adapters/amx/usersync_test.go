package amx

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/stretchr/testify/assert"
)

func TestAMXSyncer(t *testing.T) {
	syncURL := "http://pbs.amxrtb.com/cchain/0?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&cb=localhost%2Fsetuid%3Fbidder%3Damx%26uid%3D"
	syncURLTemplate := template.Must(template.New("sync-template").Parse(syncURL))

	syncer := NewAMXSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{})

	assert.NoError(t, err)
	assert.Equal(t, "http://pbs.amxrtb.com/cchain/0?gdpr=&gdpr_consent=&cb=localhost%2Fsetuid%3Fbidder%3Damx%26uid%3D", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.Equal(t, false, syncInfo.SupportCORS)
}
