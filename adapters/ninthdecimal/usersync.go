package ninthdecimal

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewNinthDecimalSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("ninthdecimal", 0, temp, adapters.SyncTypeIframe)
}
