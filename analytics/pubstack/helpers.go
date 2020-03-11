package pubstack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/prebid/prebid-server/analytics"
	"net/http"
)

type RequestType string

var HEALTH = "/v1/health"

func testEndpoint(c *http.Client, endpoint string) (error) {
	r, err := c.Get(fmt.Sprintf("%s%s", endpoint, HEALTH))
	if err != nil {
		return err
	}

	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("receive code %d instead of %d", r.StatusCode, http.StatusOK)
	}
	return nil
}

func sendPayloadToTarget(c *http.Client, payload []byte, target string) error {
	resp, err := c.Post(target, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("wrong code received %d instead of %d", resp.StatusCode, http.StatusOK)
	}
	return nil
}


func jsonifyAuctionObject(ao *analytics.AuctionObject, scope string) ([]byte, error) {
	type alias analytics.AuctionObject
	b, err := json.Marshal(&struct {
		Scope string `json:"scope"`
		*alias
	}{
		Scope: scope,
		alias: (*alias)(ao),
	})

	if err == nil {
		return b, nil
	} else {
		return []byte(""), fmt.Errorf("transactional logs error: auction object badly formed %v", err)
	}
}

