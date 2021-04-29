package criteo

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestCriteoSyncer(t *testing.T) {
	url := "//prebidserver/getuid?https%3A%2F%2Fprebidserver%2Fpbs%2Fv1%2Fsetuid%3Fbidder%3Dcriteo%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID"
	temp := template.Must(template.New("sync-template").Parse(url))
	syncer := NewCriteoSyncer(temp)
	info, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal: "1",
		},
	})
	assert.NoError(t, err)
	assert.EqualValues(t, adapters.SyncTypeRedirect, info.Type)
	assert.Equal(t, false, info.SupportCORS)
	assert.Equal(t, "//prebidserver/getuid?https%3A%2F%2Fprebidserver%2Fpbs%2Fv1%2Fsetuid%3Fbidder%3Dcriteo%26gdpr%3D1%26gdpr_consent%3D%26uid%3D%24UID", info.URL)
}
