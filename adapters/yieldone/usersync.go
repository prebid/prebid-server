package yieldone

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewYieldoneSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("yieldone", temp, adapters.SyncTypeRedirect)
}
