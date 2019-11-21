package triplelift_native

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestTripleliftSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("//eb2.3lift.com/getuid?gdpr={{.GDPR}}&cmp_cs={{.GDPRConsent}}&redir=https%3A%2F%2Feb2.3lift.com%2Fsetuid%3Fbidder%3Dtriplelift%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID"))
	syncer := NewTripleliftSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo("", "")
	assert.NoError(t, err)
	assert.Equal(t, "//eb2.3lift.com/getuid?gdpr=&cmp_cs=&redir=https%3A%2F%2Feb2.3lift.com%2Fsetuid%3Fbidder%3Dtriplelift%26gdpr%3D%26gdpr_consent%3D%26uid%3D%24UID", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, 28, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
