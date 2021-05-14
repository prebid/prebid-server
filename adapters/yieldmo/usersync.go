package yieldmo

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewYieldmoSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("yieldmo", temp, adapters.SyncTypeRedirect)
}
