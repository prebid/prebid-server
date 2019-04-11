package brightroll

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewBrightrollSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("brightroll", 25, temp, adapters.SyncTypeRedirect)
}
