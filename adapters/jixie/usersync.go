package jixie

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewJixieSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("jixie", 0, temp, adapters.SyncTypeRedirect)
}
