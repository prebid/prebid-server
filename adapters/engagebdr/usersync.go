package engagebdr

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewEngageBDRSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("engagebdr", temp, adapters.SyncTypeIframe)
}
