package yieldlab

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewYieldlabSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("yieldlab", temp, adapters.SyncTypeRedirect)
}
