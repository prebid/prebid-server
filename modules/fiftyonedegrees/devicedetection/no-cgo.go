//go:build !cgo

package devicedetection

import (
	"encoding/json"

	"github.com/prebid/prebid-server/v3/modules/moduledeps"
)

func Builder(_ json.RawMessage, _ moduledeps.ModuleDeps) (interface{}, error) {
	panic("Do not enable the fiftyonedegrees module unless CGO is enabled")
}
