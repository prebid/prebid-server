package valueimpression

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewValueImpressionSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("valueimpression", 0, temp, adapters.SyncTypeRedirect)
}
