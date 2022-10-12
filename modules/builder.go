// Package modules, the content of the file generated automatically; DO NOT EDIT.
package modules

import (
	"github.com/prebid/prebid-server/modules/foobar"
)

// builders returns mapping between module name and its hook builder
// module name is chosen based on module directory name
func builders() map[string]HookBuilderFn {
	return map[string]HookBuilderFn{
		"foobar": foobar.Builder,
	}
}
