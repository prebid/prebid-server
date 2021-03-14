package avocet

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewAvocetSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("avocet", temp, adapters.SyncTypeRedirect)
}
