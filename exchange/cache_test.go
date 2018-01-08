package exchange

import (
	"context"
	"encoding/json"
	"github.com/evanphx/json-patch"
	"github.com/mxmCherry/openrtb"
	"testing"
)

func TestBidSerialization(t *testing.T) {
	bids := []*openrtb.Bid{{
		ID: "foo",
	}, {
		ID: "bar",
	}}
	expectedJson := []json.RawMessage{
		json.RawMessage(`{"id":"foo","impid":"","price":0}`),
		json.RawMessage(`{"id":"bar","impid":"","price":0}`),
	}
	mockClient := &mockCacheClient{
		mockReturn: []string{"0", "1"},
	}

	ids := cacheBids(context.Background(), mockClient, bids)
	assertJSONMatch(t, expectedJson, mockClient.capturedRequest)
	assertStringsMatch(t, mockClient.mockReturn, ids)
}

func assertJSONMatch(t *testing.T, expected []json.RawMessage, actual []json.RawMessage) {
	t.Helper()
	if len(expected) != len(actual) {
		t.Errorf("Mismatched lengths. Expected %d, actual %d", len(expected), len(actual))
	}
	for i := 0; i < len(expected); i++ {
		if !jsonpatch.Equal(actual[i], expected[i]) {
			t.Errorf("Wrong JSON at index %d. Expected %s, got %s", i, string(expected[i]), string(actual[i]))
		}
	}
}

func assertStringsMatch(t *testing.T, expected []string, actual []string) {
	t.Helper()
	if len(expected) != len(actual) {
		t.Errorf("Mismatched lengths. Expected %d, actual %d", len(expected), len(actual))
	}
	for i := 0; i < len(expected); i++ {
		if actual[i] != expected[i] {
			t.Errorf("String mismatch at index %d. Expected %s, got %s", i, string(expected[i]), string(actual[i]))
		}
	}
}

type mockCacheClient struct {
	capturedRequest []json.RawMessage
	mockReturn      []string
}

func (c *mockCacheClient) PutJson(ctx context.Context, values []json.RawMessage) []string {
	c.capturedRequest = values
	return c.mockReturn
}
