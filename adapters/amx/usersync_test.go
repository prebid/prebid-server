package amx

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/stretchr/testify/assert"
)

const (
	testSyncEndpoint = "http://pbs.amxrtb.com/cchain/0"
)

func TestDmxSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse(testSyncEndpoint))
	syncer := NewAmxSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{})
	assert.NoError(t, err)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.Equal(t, testSyncEndpoint, syncInfo.URL)
	assert.Equal(t, false, syncInfo.SupportCORS)
	assert.EqualValues(t, 737, syncer.GDPRVendorID())
}
