package gamoshi

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewGamoshiSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("gamoshi", temp, adapters.SyncTypeRedirect)
}
