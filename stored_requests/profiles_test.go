package stored_requests

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- TestMergeProfiles_EmptyProfiles ---

func TestMergeProfiles_EmptyProfiles(t *testing.T) {
	base := []byte(`{"site":{"id":"base"}}`)
	result, errs := MergeProfiles(base, []string{}, map[string]json.RawMessage{})
	assert.Empty(t, errs)
	assert.JSONEq(t, `{"site":{"id":"base"}}`, string(result))
}

// --- TestMergeProfiles_SingleProfile ---

func TestMergeProfiles_SingleProfile(t *testing.T) {
	base := []byte(`{"site":{"id":"base"}}`)
	profiles := map[string]json.RawMessage{
		"p1": json.RawMessage(`{"device":{"os":"android"}}`),
	}
	result, errs := MergeProfiles(base, []string{"p1"}, profiles)
	assert.Empty(t, errs)

	var merged map[string]interface{}
	require.NoError(t, json.Unmarshal(result, &merged))

	site, _ := merged["site"].(map[string]interface{})
	assert.Equal(t, "base", site["id"])

	device, _ := merged["device"].(map[string]interface{})
	assert.Equal(t, "android", device["os"])
}

// --- TestMergeProfiles_MultipleProfiles_OrderMatters ---

func TestMergeProfiles_MultipleProfiles_OrderMatters(t *testing.T) {
	profileA := json.RawMessage(`{"device":{"os":"android"}}`)
	profileB := json.RawMessage(`{"device":{"os":"ios"}}`)

	base := []byte(`{}`)

	t.Run("A then B — B wins", func(t *testing.T) {
		profiles := map[string]json.RawMessage{
			"A": profileA,
			"B": profileB,
		}
		result, errs := MergeProfiles(base, []string{"A", "B"}, profiles)
		assert.Empty(t, errs)

		var merged map[string]interface{}
		require.NoError(t, json.Unmarshal(result, &merged))
		device, _ := merged["device"].(map[string]interface{})
		assert.Equal(t, "ios", device["os"])
	})

	t.Run("B then A — A wins", func(t *testing.T) {
		profiles := map[string]json.RawMessage{
			"A": profileA,
			"B": profileB,
		}
		result, errs := MergeProfiles(base, []string{"B", "A"}, profiles)
		assert.Empty(t, errs)

		var merged map[string]interface{}
		require.NoError(t, json.Unmarshal(result, &merged))
		device, _ := merged["device"].(map[string]interface{})
		assert.Equal(t, "android", device["os"])
	})
}

// --- TestMergeProfiles_MissingProfileSkipped ---

func TestMergeProfiles_MissingProfileSkipped(t *testing.T) {
	base := []byte(`{}`)
	profiles := map[string]json.RawMessage{
		"exists": json.RawMessage(`{"site":{"id":"found"}}`),
		// "missing" is intentionally absent
	}
	result, errs := MergeProfiles(base, []string{"exists", "missing"}, profiles)
	// missing profile is silently skipped — no fatal error
	assert.Empty(t, errs)

	var merged map[string]interface{}
	require.NoError(t, json.Unmarshal(result, &merged))
	site, _ := merged["site"].(map[string]interface{})
	assert.Equal(t, "found", site["id"])
}

// --- TestMergeProfiles_InvalidJSONProfile ---

func TestMergeProfiles_InvalidJSONProfile(t *testing.T) {
	base := []byte(`{"site":{"id":"base"}}`)
	profiles := map[string]json.RawMessage{
		"bad": json.RawMessage(`{invalid json`),
	}
	result, errs := MergeProfiles(base, []string{"bad"}, profiles)
	// Should return an error; base JSON should be returned unchanged
	assert.NotEmpty(t, errs)
	assert.JSONEq(t, `{"site":{"id":"base"}}`, string(result))
}

// --- TestMergeProfiles_DeepMerge ---

func TestMergeProfiles_DeepMerge(t *testing.T) {
	base := []byte(`{"site":{"content":{"genre":"drama","id":1}}}`)
	profiles := map[string]json.RawMessage{
		"p1": json.RawMessage(`{"site":{"content":{"genre":"comedy"}}}`),
	}
	result, errs := MergeProfiles(base, []string{"p1"}, profiles)
	assert.Empty(t, errs)

	var merged map[string]interface{}
	require.NoError(t, json.Unmarshal(result, &merged))

	site, _ := merged["site"].(map[string]interface{})
	content, _ := site["content"].(map[string]interface{})
	// genre should be overwritten
	assert.Equal(t, "comedy", content["genre"])
	// id should survive (deep merge, not replace)
	assert.EqualValues(t, float64(1), content["id"])
}

// --- TestNoopProfileFetcher ---

func TestNoopProfileFetcher(t *testing.T) {
	f := NoopProfileFetcher{}
	result, errs := f.FetchProfiles(context.Background(), "acc", []string{"a", "b"})
	assert.Empty(t, errs)
	assert.NotNil(t, result)
	// Noop returns an empty map (no data, but no error either)
	assert.Empty(t, result)
}
