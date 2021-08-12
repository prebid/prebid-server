package smilewanted

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewSmileWantedSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("smilewanted", temp, adapters.SyncTypeRedirect)
}
