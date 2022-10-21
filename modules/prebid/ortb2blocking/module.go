package ortb2blocking

import (
	"encoding/json"
	"net/http"
)

func Builder(conf json.RawMessage, client *http.Client) (interface{}, error) {
	cfg, err := newConfig(conf)
	if err != nil {
		return nil, err
	}

	return Module{cfg}, nil
}

type Module struct {
	cfg Config
}
