package ortb2blocking

import (
	"encoding/json"

	"github.com/prebid/prebid-server/modules/moduledeps"
)

func Builder(_ json.RawMessage, _ moduledeps.ModuleDeps) (interface{}, error) {
	return Module{}, nil
}

type Module struct{}
