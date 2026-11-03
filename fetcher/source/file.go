package source

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// FileSource loads JSON files from one directory and keys them by filename.
type FileSource struct {
	values map[string]json.RawMessage
}

// NewFileSource loads all .json files in directory. A missing directory is
// represented by an empty source.
func NewFileSource(directory string) (*FileSource, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &FileSource{values: map[string]json.RawMessage{}}, nil
		}
		return nil, err
	}

	values := make(map[string]json.RawMessage)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(directory, entry.Name()))
		if err != nil {
			return nil, err
		}
		values[strings.TrimSuffix(entry.Name(), ".json")] = json.RawMessage(data)
	}
	return &FileSource{values: values}, nil
}

func (s *FileSource) Fetch(_ context.Context, key string) (json.RawMessage, bool, error) {
	raw, found := s.values[key]
	return raw, found, nil
}

func (s *FileSource) FetchAll(context.Context) (map[string]json.RawMessage, error) {
	values := make(map[string]json.RawMessage, len(s.values))
	for key, raw := range s.values {
		values[key] = raw
	}
	return values, nil
}
