//go:build !cgo

package devicedetection

import (
	"encoding/json"

	"github.com/prebid/prebid-server/v2/modules/moduledeps"
)

func Builder(rawConfig json.RawMessage, _ moduledeps.ModuleDeps) (interface{}, error) {
	panic("Not implemented when CGO_ENABLED=0")
}
