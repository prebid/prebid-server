package nanointeractive

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewNanoInteractiveSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("nanointeractive", temp, adapters.SyncTypeRedirect)
}
