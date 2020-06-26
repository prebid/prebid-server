package avocet

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewAvocetSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("avocet", 63, temp, adapters.SyncTypeRedirect)
}
