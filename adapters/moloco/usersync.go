package moloco

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

// NewMolocoSyncer ...
func NewMolocoSyncer(template *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("moloco", 807, template, adapters.SyncTypeRedirect)
}
