package lockerdome

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewLockerDomeSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("lockerdome", 0, temp, adapters.SyncTypeRedirect)
}
