package onetag

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/stretchr/testify/assert"
)

func TestOneTagSyncer(t *testing.T) {
	syncURL := "https://onetag-sys.com/usync/?redir="
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{})

	assert.NoError(t, err)
	assert.Equal(t, "https://onetag-sys.com/usync/?redir=", syncInfo.URL)
	assert.Equal(t, "iframe", syncInfo.Type)
}
