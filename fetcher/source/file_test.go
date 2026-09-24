package source

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileSourceFetchAndFetchAll(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "one.json"), []byte(`{"id":"one"}`), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ignored.txt"), []byte(`{"id":"ignored"}`), 0644))

	source, err := NewFileSource(dir)
	require.NoError(t, err)

	raw, found, err := source.Fetch(context.Background(), "one")
	require.NoError(t, err)
	require.True(t, found)
	assert.JSONEq(t, `{"id":"one"}`, string(raw))

	_, found, err = source.Fetch(context.Background(), "missing")
	require.NoError(t, err)
	assert.False(t, found)

	values, err := source.FetchAll(context.Background())
	require.NoError(t, err)
	assert.Len(t, values, 1)
	assert.JSONEq(t, `{"id":"one"}`, string(values["one"]))
}

func TestFileSourceMissingDirectoryIsEmpty(t *testing.T) {
	source, err := NewFileSource(filepath.Join(t.TempDir(), "missing"))
	require.NoError(t, err)

	_, found, err := source.Fetch(context.Background(), "missing")
	require.NoError(t, err)
	assert.False(t, found)
}

func TestFileSourceReturnsReadError(t *testing.T) {
	_, err := NewFileSource("invalid\x00path")

	require.Error(t, err)
}

func TestFileSourceFetchAllReturnsMapCopy(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "one.json"), []byte(`{"id":"one"}`), 0644))

	source, err := NewFileSource(dir)
	require.NoError(t, err)
	first, err := source.FetchAll(context.Background())
	require.NoError(t, err)
	delete(first, "one")

	second, err := source.FetchAll(context.Background())
	require.NoError(t, err)
	assert.Contains(t, second, "one")
}
