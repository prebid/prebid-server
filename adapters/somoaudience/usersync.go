package somoaudience

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewSomoaudienceSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("somoaudience", 341, temp, adapters.SyncTypeRedirect)
}
