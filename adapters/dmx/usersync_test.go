package dmx

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"

	"github.com/stretchr/testify/assert"
)

func TestDmxSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("https://dmx.districtm.io/s/v1/img/s/10007"))
	syncer := NewDmxSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{})
	assert.NoError(t, err)
	assert.Equal(t, "https://dmx.districtm.io/s/v1/img/s/10007", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.Equal(t, false, syncInfo.SupportCORS)
}
