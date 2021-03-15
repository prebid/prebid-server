package rubicon

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewRubiconSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("rubicon", temp, adapters.SyncTypeRedirect)
}
