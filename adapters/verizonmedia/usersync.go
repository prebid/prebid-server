package verizonmedia

import (
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
	"text/template"
)

func NewVerizonMediaSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("verizonmedia", 25, temp, adapters.SyncTypeRedirect)
}
