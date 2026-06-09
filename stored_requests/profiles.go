package stored_requests

import (
	"context"
	"encoding/json"
	"fmt"

	jsonpatch "gopkg.in/evanphx/json-patch.v5"
)

// ProfileFetcher fetches named profile JSON fragments by account and profile name.
// Returns a map of profileName → raw JSON bytes.
// Missing profiles return an empty entry; errors are non-fatal (caller may skip with warning).
type ProfileFetcher interface {
	FetchProfiles(ctx context.Context, accountID string, profileIDs []string) (map[string]json.RawMessage, []error)
}

// NoopProfileFetcher returns empty results for all requests.
// Used as a default when profiles are not configured.
type NoopProfileFetcher struct{}

func (n NoopProfileFetcher) FetchProfiles(_ context.Context, _ string, profileIDs []string) (map[string]json.RawMessage, []error) {
	result := make(map[string]json.RawMessage, len(profileIDs))
	return result, nil
}

// MergeProfiles merges profile JSON fragments into a base request JSON in declaration order.
// Each profile is deep-merged (RFC 7396 JSON Merge Patch) into the running state of the request.
//
// Merge order: profiles are applied sequentially in the order given by profileIDs.
// Missing profiles (not found in profileData) are silently skipped — they still count
// toward the profile limit in the caller.
//
// Returns the merged JSON and any non-fatal errors encountered during merging.
func MergeProfiles(baseJSON []byte, profileIDs []string, profileData map[string]json.RawMessage) ([]byte, []error) {
	result := baseJSON
	var errs []error

	for _, id := range profileIDs {
		profileJSON, ok := profileData[id]
		if !ok || len(profileJSON) == 0 {
			// Missing or empty profile — silently skip.
			continue
		}
		merged, err := jsonpatch.MergePatch(result, profileJSON)
		if err != nil {
			errs = append(errs, fmt.Errorf("profile %q: merge failed: %w", id, err))
			continue
		}
		result = merged
	}

	return result, errs
}
