package somoaudience

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewSomoaudienceSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("somoaudience", 341, temp, adapters.SyncTypeRedirect)
}
