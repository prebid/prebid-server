package forwarder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/prebid-server/analytics/pubstack/model"
)

type Forwarder struct {
	client http.Client
	url    string
}

func NewForwarder(intake string) *Forwarder {
	fmt.Printf("[PBSTCK]: preparing to forward to %s", intake)
	return &Forwarder{
		client: http.Client{},
		url:    intake,
	}
}

func (f *Forwarder) Feed(auctions []model.Auction) error {

	payload, err := json.Marshal(auctions)
	if err != nil {
		return err
	}

	resp, err := f.client.Post(f.url, "text/plain", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	if resp.StatusCode != 204 {
		return fmt.Errorf("Error while sending auctions")
	}

	return nil
}
