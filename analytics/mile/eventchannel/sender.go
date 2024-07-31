package eventchannel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/v2/analytics/mile/helpers"
)

type Sender = func(payload []*helpers.PageViewRecord) error

func NewHttpSender(client *http.Client, endpoint string) Sender {
	return func(payload []*helpers.PageViewRecord) error {

		data, err := json.Marshal(payload)
		if err != nil {
			glog.Error(err)
			return err
		}

		req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(data))
		if err != nil {
			glog.Error(err)
			return err
		}

		req.Header.Set("Content-Type", "application/json")
		//req.Header.Set("Content-Encoding", "gzip")

		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusNoContent {
			glog.Errorf("[mile] Wrong code received %d instead of %d", resp.StatusCode, http.StatusOK)
			return fmt.Errorf("wrong code received %d instead of %d", resp.StatusCode, http.StatusOK)
		}
		return nil
	}
}

func BuildEndpointSender(client *http.Client, baseUrl string, module string) Sender {
	endpoint, err := url.Parse(baseUrl)
	if err != nil {
		glog.Error(err)
	}
	endpoint.Path = path.Join(endpoint.Path, "intake", module)
	return NewHttpSender(client, endpoint.String())
}
