package ucfunnel

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewUcfunnelSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("ucfunnel", temp, adapters.SyncTypeRedirect)
}
