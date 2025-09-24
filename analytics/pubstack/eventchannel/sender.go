package eventchannel

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"

	"github.com/prebid/prebid-server/v3/logger"
)

type Sender = func(payload []byte) error

func NewHttpSender(client *http.Client, endpoint string) Sender {
	return func(payload []byte) error {
		req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
		if err != nil {
			logger.Error(err)
			return err
		}

		req.Header.Set("Content-Type", "application/octet-stream")
		req.Header.Set("Content-Encoding", "gzip")

		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer func() {
			if _, err := io.Copy(io.Discard, resp.Body); err != nil {
				logger.Error(fmt.Sprintf("[pubstack] Draining sender response body failed: %v", err))
			}
			resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusOK {
			logger.Error(fmt.Sprintf("[pubstack] Wrong code received %d instead of %d", resp.StatusCode, http.StatusOK))
			return fmt.Errorf("wrong code received %d instead of %d", resp.StatusCode, http.StatusOK)
		}
		return nil
	}
}

func BuildEndpointSender(client *http.Client, baseUrl string, module string) Sender {
	endpoint, err := url.Parse(baseUrl)
	if err != nil {
		logger.Error(err)
	}
	endpoint.Path = path.Join(endpoint.Path, "intake", module)
	return NewHttpSender(client, endpoint.String())
}
