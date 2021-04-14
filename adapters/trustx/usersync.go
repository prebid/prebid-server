package trustx

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewTrustXSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("trustx", temp, adapters.SyncTypeRedirect)
}
