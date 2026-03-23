package jsonutil

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIntStringUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    json.RawMessage
		expectError bool
		want        string
	}{
		{
			name:        "null",
			jsonData:    []byte(`{"item_id": null}`),
			want:        "",
			expectError: true,
		},
		{
			name:     "string",
			jsonData: []byte(`{"item_id": "30"}`),
			want:     "30",
		},
		{
			name:     "int",
			jsonData: []byte(`{"item_id": 30}`),
			want:     "30",
		},
		{
			name:        "error",
			jsonData:    []byte(`{"item_id": []`),
			want:        "",
			expectError: true,
		},
	}

	type Item struct {
		ItemId IntString `json:"item_id"`
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var item Item
			err := UnmarshalValid(test.jsonData, &item)
			assert.Equal(t, string(test.want), string(item.ItemId))

			if test.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
