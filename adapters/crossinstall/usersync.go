package crossinstall

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

// NewCrossInstallSyncer ...
func NewCrossInstallSyncer(template *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("crossinstall", 807, template, adapters.SyncTypeRedirect)
}
