package nobid

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewNoBidSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("nobid", 816, temp, adapters.SyncTypeRedirect)
}
