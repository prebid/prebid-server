package cache

import (
	"encoding/json"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"
)

// Entry represents cache data, with such characteristics as:
//
//   - a positive entry carries Eids
//   - a negative entry has Negative=true (id known-unresolvable)
//   - an in-progress marker has InProgress=true (a resolution call is in flight)
type Entry struct {
	Eids       []openrtb2.EID `json:"eids,omitempty"`
	AbTestUUID string         `json:"abTestUuid,omitempty"`
	Tc         *int64         `json:"tc,omitempty"`
	Negative   bool           `json:"negative,omitempty"`
	InProgress bool           `json:"inProgress,omitempty"`
	Exp        int64          `json:"exp"` // ms
}

// decodeValid returns nil for an entry at or past Exp. Checking the absolute expiry keeps reads
// correct however coarsely a backend rounds its own per-entry TTL.
func decodeValid(value []byte) *Entry {
	if len(value) == 0 {
		return nil
	}
	var entry Entry
	if err := json.Unmarshal(value, &entry); err != nil {
		return nil
	}
	if entry.Exp <= time.Now().UnixMilli() {
		return nil
	}
	return &entry
}
