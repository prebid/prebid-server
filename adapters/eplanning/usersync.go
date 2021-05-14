package eplanning

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewEPlanningSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("eplanning", temp, adapters.SyncTypeIframe)
}
