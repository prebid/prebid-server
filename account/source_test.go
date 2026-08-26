package account

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

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

	raw, err := source.Fetch(context.Background(), "pub-1")
	require.NoError(t, err)
	assert.JSONEq(t, `{"id":"pub-1"}`, string(raw))

	_, err = source.Fetch(context.Background(), "missing")
	assert.ErrorIs(t, err, errAccountNotFound)

	bulk := source.(BulkSource)
	accounts, err := bulk.FetchAll(context.Background())
	require.NoError(t, err)
	assert.Len(t, accounts, 1)
	assert.JSONEq(t, `{"id":"pub-1"}`, string(accounts["pub-1"]))
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

	raw, err := source.Fetch(context.Background(), "pub-1")
	require.NoError(t, err)
	assert.Equal(t, []string{"pub-1"}, seen)
	assert.JSONEq(t, `{"id":"pub-1"}`, string(raw))
}

func TestHTTPSourceFetchNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	source, err := NewHTTPSource(server.Client(), server.URL, true)
	require.NoError(t, err)

	_, err = source.Fetch(context.Background(), "missing")
	assert.ErrorIs(t, err, errAccountNotFound)
}

type errorSource struct {
	err error
}

func (s errorSource) Fetch(ctx context.Context, accountID string) (json.RawMessage, error) {
	return nil, s.err
}
