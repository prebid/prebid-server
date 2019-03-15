package onemobile

import (
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
	"text/template"
)

func NewOneMobileSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("onemobile", 25, temp, adapters.SyncTypeRedirect)
}
