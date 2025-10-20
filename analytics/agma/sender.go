package agma

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/logger"
	"github.com/prebid/prebid-server/v3/version"
)

func compressToGZIP(requestBody []byte) ([]byte, error) {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	_, err := w.Write([]byte(requestBody))
	if err != nil {
		_ = w.Close()
		return nil, err
	}
	err = w.Close()
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func createHttpSender(httpClient *http.Client, endpoint config.AgmaAnalyticsHttpEndpoint) (httpSender, error) {
	_, err := url.Parse(endpoint.Url)
	if err != nil {
		return nil, err
	}

	httpTimeout, err := time.ParseDuration(endpoint.Timeout)
	if err != nil {
		return nil, err
	}

	return func(payload []byte) error {
		ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
		defer cancel()

		var requestBody []byte
		var err error

		if endpoint.Gzip {
			requestBody, err = compressToGZIP(payload)
			if err != nil {
				logger.Error("[agmaAnalytics] Compressing request failed %v", err)
				return err
			}
		} else {
			requestBody = payload
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.Url, bytes.NewBuffer(requestBody))
		if err != nil {
			logger.Error("[agmaAnalytics] Creating request failed %v", err)
			return err
		}

		req.Header.Set("X-Prebid", version.BuildXPrebidHeader(version.Ver))
		req.Header.Set("Content-Type", "application/json")
		if endpoint.Gzip {
			req.Header.Set("Content-Encoding", "gzip")
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			logger.Error("[agmaAnalytics] Sending request failed %v", err)
			return err
		}
		defer func() {
			if _, err := io.Copy(io.Discard, resp.Body); err != nil {
				logger.Error("[agmaAnalytics] Draining response body failed: %v", err)
			}
			resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusOK {
			logger.Error("[agmaAnalytics] Wrong code received %d instead of %d", resp.StatusCode, http.StatusOK)
			return fmt.Errorf("wrong code received %d instead of %d", resp.StatusCode, http.StatusOK)
		}
		return nil
	}, nil
}
