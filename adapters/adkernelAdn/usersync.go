package adkernelAdn

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewAdkernelAdnSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("adkernelAdn", temp, adapters.SyncTypeRedirect)
}
