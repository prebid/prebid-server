package http

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/version"
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

func createHttpSender(httpClient *http.Client, endpoint config.AnalyticsHttpEndpoint) (httpSender, error) {
	_, err := url.Parse(endpoint.Url)
	if err != nil {
		return nil, err
	}

	httpTimeout, err := time.ParseDuration(endpoint.Timeout)
	if err != nil {
		return nil, err
	}

	return func(payload []byte) error {
		// add http timeout
		ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
		defer cancel()

		var requestBody []byte
		var err error

		if endpoint.Gzip {
			// we compress with gzip if enabled
			requestBody, err = compressToGZIP(payload)
			if err != nil {
				glog.Errorf("[HttpAnalytics] Compressing request failed %v", err)
				return err
			}
		} else {
			requestBody = payload
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.Url, bytes.NewBuffer(requestBody))
		if err != nil {
			glog.Errorf("[HttpAnalytics] Creating request failed %v", err)
			return err
		}

		// Set default headers
		req.Header.Set("X-Prebid", version.BuildXPrebidHeader(version.Ver))
		req.Header.Set("Content-Type", "application/json")
		if endpoint.Gzip {
			req.Header.Set("Content-Encoding", "gzip")
		}

		// Set additional headers for config
		for k, v := range endpoint.AdditionalHeaders {
			req.Header.Set(k, v)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			glog.Errorf("[HttpAnalytics] Sending request failed %v", err)
			return err
		}

		if resp.StatusCode != http.StatusOK {
			glog.Errorf("[HttpAnalytics] Wrong code received %d instead of %d", resp.StatusCode, http.StatusOK)
			return fmt.Errorf("wrong code received %d instead of %d", resp.StatusCode, http.StatusOK)
		}
		return nil
	}, nil
}
