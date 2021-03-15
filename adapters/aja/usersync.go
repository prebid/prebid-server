package aja

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewAJASyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("aja", temp, adapters.SyncTypeRedirect)
}
