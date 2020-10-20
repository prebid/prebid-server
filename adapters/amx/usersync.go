package amx

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

// NewAmxSyncer produces an AMX RTB usersyncer
func NewAmxSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("amx", 737, temp, adapters.SyncTypeRedirect)
}
