package zeroclickfraud

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewZeroclickfraudSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("zeroclickfraud", 0, temp, adapters.SyncTypeIframe)
}
