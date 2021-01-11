package foreshop

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

// NewForeshopSyncer returns an instance of the adapter syncer using the given template's URL
func NewForeshopSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("foreshop", 0, temp, adapters.SyncTypeRedirect)
}
