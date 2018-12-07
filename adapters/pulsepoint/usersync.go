package pulsepoint

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewPulsepointSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("pulsepoint", 81, temp, adapters.SyncTypeRedirect)
}
