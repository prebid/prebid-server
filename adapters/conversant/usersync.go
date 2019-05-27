package conversant

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewConversantSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("conversant", 24, temp, adapters.SyncTypeRedirect)
}
