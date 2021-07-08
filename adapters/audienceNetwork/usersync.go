package audienceNetwork

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewFacebookSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("audienceNetwork", temp, adapters.SyncTypeRedirect)
}
