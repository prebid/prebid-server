package account

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/prebid/prebid-server/v4/fetcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileSourceFetchAndFetchAll(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "accounts"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "accounts", "pub-1.json"), []byte(`{"id":"pub-1"}`), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "accounts", "ignored.txt"), []byte(`{"id":"ignored"}`), 0644))

	source, err := NewFileSource(dir)
	require.NoError(t, err)

	raw, found, err := source.Fetch(context.Background(), "pub-1")
	require.NoError(t, err)
	require.True(t, found)
	assert.JSONEq(t, `{"id":"pub-1"}`, string(raw))

	_, found, err = source.Fetch(context.Background(), "missing")
	require.NoError(t, err)
	assert.False(t, found)

	bulk := source.(fetcher.BulkSource[string])
	accounts, err := bulk.FetchAll(context.Background())
	require.NoError(t, err)
	assert.Len(t, accounts, 1)
	assert.JSONEq(t, `{"id":"pub-1"}`, string(accounts["pub-1"]))
}

func TestFileSourceMissingAccountsDirectoryIsEmpty(t *testing.T) {
	source, err := NewFileSource(t.TempDir())
	require.NoError(t, err)

	accounts, err := source.(fetcher.BulkSource[string]).FetchAll(context.Background())
	require.NoError(t, err)
	assert.Empty(t, accounts)
}

func TestHTTPSourceFetch(t *testing.T) {
	var seen []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.URL.Query()["account-id"]
		fmt.Fprint(w, `{"accounts":{"pub-1":{"id":"pub-1"}}}`)
	}))
	defer server.Close()

	source, err := NewHTTPSource(server.Client(), server.URL, true)
	require.NoError(t, err)

	raw, found, err := source.Fetch(context.Background(), "pub-1")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, []string{"pub-1"}, seen)
	assert.JSONEq(t, `{"id":"pub-1"}`, string(raw))
}

func TestHTTPSourceFetchWithLegacyQuery(t *testing.T) {
	var seen string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.URL.Query().Get("account-ids")
		fmt.Fprint(w, `{"accounts":{"pub-1":{"id":"pub-1"}}}`)
	}))
	defer server.Close()

	source, err := NewHTTPSource(server.Client(), server.URL, false)
	require.NoError(t, err)

	_, found, err := source.Fetch(context.Background(), "pub-1")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, `["pub-1"]`, seen)
}

func TestHTTPSourceFetchNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	source, err := NewHTTPSource(server.Client(), server.URL, true)
	require.NoError(t, err)

	_, found, err := source.Fetch(context.Background(), "missing")
	require.NoError(t, err)
	assert.False(t, found)
}

func TestHTTPSourceFetchNullAccountIsNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"accounts":{"missing":null}}`)
	}))
	defer server.Close()

	source, err := NewHTTPSource(server.Client(), server.URL, true)
	require.NoError(t, err)

	_, found, err := source.Fetch(context.Background(), "missing")
	require.NoError(t, err)
	assert.False(t, found)
}

func TestHTTPSourceFetchOmittedAccountIsNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"accounts":{}}`)
	}))
	defer server.Close()

	source, err := NewHTTPSource(server.Client(), server.URL, true)
	require.NoError(t, err)

	_, found, err := source.Fetch(context.Background(), "missing")
	require.NoError(t, err)
	assert.False(t, found)
}

func TestHTTPSourceFetchMalformedResponseErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{`)
	}))
	defer server.Close()

	source, err := NewHTTPSource(server.Client(), server.URL, true)
	require.NoError(t, err)

	_, found, err := source.Fetch(context.Background(), "pub-1")
	require.ErrorContains(t, err, "failed to parse account response")
	assert.False(t, found)
}
