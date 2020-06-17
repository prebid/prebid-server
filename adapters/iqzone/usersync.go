package iqzone

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewIQZoneSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("iqzone", 0, temp, adapters.SyncTypeIframe)
}
