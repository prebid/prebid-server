package account

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/prebid/prebid-server/v4/fetcher"
	fetchersource "github.com/prebid/prebid-server/v4/fetcher/source"
	"github.com/prebid/prebid-server/v4/logger"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

// NewFileSource loads raw account JSON from the accounts subdirectory under directory.
func NewFileSource(directory string) (fetcher.Source[string], error) {
	if _, err := os.Stat(directory); err != nil {
		return nil, err
	}
	return fetchersource.NewFileSource(filepath.Join(directory, "accounts"))
}

// NewHTTPSource fetches raw account JSON from the configured by-id account endpoint.
func NewHTTPSource(client *http.Client, endpoint string, useRfcCompliantBuilder bool) (fetcher.Source[string], error) {
	logger.Infof("Making Fetchers 2.0 account HTTP source for endpoint %v", endpoint)
	return fetchersource.NewHTTPSource(client, endpoint, accountRequestBuilder(useRfcCompliantBuilder), decodeAccountResponse)
}

func accountRequestBuilder(useRfcCompliantBuilder bool) fetchersource.RequestBuilder[string] {
	return func(ctx context.Context, endpoint *url.URL, accountID string) (*http.Request, error) {
		q := endpoint.Query()
		if useRfcCompliantBuilder {
			q.Add("account-id", accountID)
		} else {
			q.Set("account-ids", `["`+accountID+`"]`)
		}
		endpoint.RawQuery = q.Encode()
		return http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	}
}

func decodeAccountResponse(accountID string, body []byte) (json.RawMessage, bool, error) {
	var responseData accountsResponseContract
	if err := jsonutil.UnmarshalValid(body, &responseData); err != nil {
		return nil, false, fmt.Errorf("failed to parse account response: %w", err)
	}
	raw, found := responseData.Accounts[accountID]
	if !found || len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil, false, nil
	}
	return raw, true, nil
}

type accountsResponseContract struct {
	Accounts map[string]json.RawMessage `json:"accounts"`
}
