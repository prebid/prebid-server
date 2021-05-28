package deepintent

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

// NewDeepintentSyncer returns deepintent syncer
func NewDeepintentSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("deepintent", temp, adapters.SyncTypeRedirect)
}
