package smaato

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewSmaatoSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("smaato", 76, temp, adapters.SyncTypeIframe)
}
