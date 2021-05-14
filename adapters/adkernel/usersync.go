package adkernel

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewAdkernelSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("adkernel", temp, adapters.SyncTypeRedirect)
}
