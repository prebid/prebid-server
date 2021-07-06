package salunamedia

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewSaLunamediaSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("salunamedia", temp, adapters.SyncTypeRedirect)
}
