package zeroclickfraud

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewZeroClickFraudSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("zeroclickfraud", temp, adapters.SyncTypeIframe)
}
