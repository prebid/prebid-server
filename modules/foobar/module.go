package foobar

import (
	"encoding/json"
	"net/http"

	"github.com/prebid/prebid-server/modules/foobar/config"
	"github.com/prebid/prebid-server/modules/foobar/hooks"
)

func Builder(conf json.RawMessage, client *http.Client) (map[string]interface{}, error) {
	cfg, err := config.New(conf)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"code1": hooks.NewValidateHeaderEntrypointHook(cfg),
		"code2": hooks.NewValidateQueryParamEntrypointHook(cfg),
		"code3": hooks.NewCheckBodyRawAuctionHook(client, cfg),
	}, nil
}
