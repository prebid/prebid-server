package zeroclickfraud

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewZeroClickFraudSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("zeroclickfraud", 0, temp, adapters.SyncTypeIframe)
}
