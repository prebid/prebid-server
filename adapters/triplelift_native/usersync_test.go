package triplelift

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestTripleliftSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("//eb2.3lift.com/sync?gdpr={{.GDPR}}&cmp_cs={{.GDPRConsent}}"))
	syncer := NewTripleliftSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo("", "")
	assert.NoError(t, err)
	assert.Equal(t, "//eb2.3lift.com/sync?gdpr=&cmp_cs=", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, 28, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
