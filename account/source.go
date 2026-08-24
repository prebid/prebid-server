package account

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/prebid/prebid-server/v4/logger"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
	"golang.org/x/net/context/ctxhttp"
)

var errAccountNotFound = errors.New("account not found")

// Source fetches raw, unmerged account JSON for Fetchers 2.0.
type Source interface {
	Fetch(ctx context.Context, accountID string) (json.RawMessage, error)
}

// BulkSource is an optional account source capability used by refresh=preload.
type BulkSource interface {
	Source
	FetchAll(ctx context.Context) (map[string]json.RawMessage, error)
}

// EmptySource is used when account fetching is not configured.
type EmptySource struct{}

func (EmptySource) Fetch(ctx context.Context, accountID string) (json.RawMessage, error) {
	return nil, errAccountNotFound
}

// NewFileSource loads raw account JSON from the accounts subdirectory under directory.
func NewFileSource(directory string) (Source, error) {
	if _, err := os.Stat(directory); err != nil {
		return nil, err
	}

	accountsDir := filepath.Join(directory, "accounts")
	entries, err := os.ReadDir(accountsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fileSource{accounts: map[string]json.RawMessage{}}, nil
		}
		return nil, err
	}

	accounts := make(map[string]json.RawMessage)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(accountsDir, entry.Name()))
		if err != nil {
			return nil, err
		}
		accounts[strings.TrimSuffix(entry.Name(), ".json")] = json.RawMessage(data)
	}
	return fileSource{accounts: accounts}, nil
}

type fileSource struct {
	accounts map[string]json.RawMessage
}

func (s fileSource) Fetch(ctx context.Context, accountID string) (json.RawMessage, error) {
	if accountID == "" {
		return nil, errors.New("Cannot look up an empty accountID")
	}
	raw, ok := s.accounts[accountID]
	if !ok {
		return nil, errAccountNotFound
	}
	return raw, nil
}

func (s fileSource) FetchAll(ctx context.Context) (map[string]json.RawMessage, error) {
	out := make(map[string]json.RawMessage, len(s.accounts))
	for id, raw := range s.accounts {
		out[id] = raw
	}
	return out, nil
}

// NewHTTPSource fetches raw account JSON from the configured by-id account endpoint.
func NewHTTPSource(client *http.Client, endpoint string, useRfcCompliantBuilder bool) (Source, error) {
	endpointURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf(`invalid account HTTP endpoint "%s": %w`, endpoint, err)
	}
	logger.Infof("Making Fetchers 2.0 account HTTP source for endpoint %v", endpoint)
	return httpSource{
		client:                 client,
		endpointURL:            endpointURL,
		useRfcCompliantBuilder: useRfcCompliantBuilder,
	}, nil
}

type httpSource struct {
	client                 *http.Client
	endpointURL            *url.URL
	useRfcCompliantBuilder bool
}

func (s httpSource) Fetch(ctx context.Context, accountID string) (json.RawMessage, error) {
	accounts, err := s.fetch(ctx, []string{accountID})
	if err != nil {
		return nil, err
	}
	raw, ok := accounts[accountID]
	if !ok {
		return nil, errAccountNotFound
	}
	return raw, nil
}

func (s httpSource) fetch(ctx context.Context, accountIDs []string) (map[string]json.RawMessage, error) {
	if len(accountIDs) == 0 {
		return nil, nil
	}

	u := *s.endpointURL
	q := u.Query()
	if s.useRfcCompliantBuilder {
		for _, id := range accountIDs {
			q.Add("account-id", id)
		}
	} else {
		q.Set("account-ids", `["`+strings.Join(accountIDs, `","`)+`"]`)
	}
	u.RawQuery = q.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching accounts %v via http: build request failed with %w", accountIDs, err)
	}
	httpResp, err := ctxhttp.Do(ctx, s.client, httpReq)
	if err != nil {
		return nil, fmt.Errorf("error fetching accounts %v via http: %w", accountIDs, err)
	}
	defer httpResp.Body.Close()

	respBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("error fetching accounts %v via http: error reading response: %w", accountIDs, err)
	}
	if httpResp.StatusCode == http.StatusNotFound {
		return nil, errAccountNotFound
	}
	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error fetching accounts %v via http: unexpected response status %d", accountIDs, httpResp.StatusCode)
	}

	var responseData accountsResponseContract
	if err = jsonutil.UnmarshalValid(respBytes, &responseData); err != nil {
		return nil, fmt.Errorf("error fetching accounts %v via http: failed to parse response: %w", accountIDs, err)
	}
	removeNullAccounts(responseData.Accounts)
	return responseData.Accounts, nil
}

func removeNullAccounts(accounts map[string]json.RawMessage) {
	for id, raw := range accounts {
		if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
			delete(accounts, id)
		}
	}
}

type accountsResponseContract struct {
	Accounts map[string]json.RawMessage `json:"accounts"`
}
