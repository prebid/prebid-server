package datablocks

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewDatablocksSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("datablocks", temp, adapters.SyncTypeRedirect)
}
