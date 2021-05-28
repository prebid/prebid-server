package ttx

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func New33AcrossSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("33across", temp, adapters.SyncTypeIframe)
}
