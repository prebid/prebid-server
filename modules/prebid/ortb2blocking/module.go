package ortb2blocking

import (
	"encoding/json"
	"net/http"
)

func Builder(_ json.RawMessage, _ *http.Client) (interface{}, error) {
	return Module{}, nil
}

type Module struct{}
