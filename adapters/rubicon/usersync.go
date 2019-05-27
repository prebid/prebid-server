package rubicon

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewRubiconSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("rubicon", 52, temp, adapters.SyncTypeRedirect)
}
