package lifestreet

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewLifestreetSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("lifestreet", 67, temp, adapters.SyncTypeRedirect)
}
