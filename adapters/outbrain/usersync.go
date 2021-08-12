package outbrain

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewOutbrainSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("outbrain", temp, adapters.SyncTypeRedirect)
}
