package amx

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

// NewAMXSyncer produces an AMX RTB usersyncer
func NewAMXSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("amx", temp, adapters.SyncTypeRedirect)
}
