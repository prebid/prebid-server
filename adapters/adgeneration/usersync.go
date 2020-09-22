package adgeneration

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewAdgenerationSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("adgeneration", 0, temp, adapters.SyncTypeRedirect)
}
