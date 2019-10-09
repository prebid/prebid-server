package engagebdr

import (
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
	"text/template"
)

func NewEngageBDRSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("engagebdr", 62, temp, adapters.SyncTypeIframe)
}
