package gamoshi

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewGamoshiSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("gamoshi", 0, temp, adapters.SyncTypeRedirect)
}
