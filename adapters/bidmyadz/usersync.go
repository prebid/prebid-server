package bidmyadz

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewBidmyadzSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("bidmyadz", temp, adapters.SyncTypeRedirect)
}
