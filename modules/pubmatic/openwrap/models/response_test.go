package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBidExtTrackersUnmarshal(t *testing.T) {
	tests := []struct {
		name    string
		rawExt  string
		want    []ExtBidTracker
		wantErr bool
	}{
		{
			name: "parses_ad_attribute_trackers",
			rawExt: `{
				"trackers": [{
					"event": "ad_attribute",
					"url": "https://t.pubmatic.com?bidid=789&ad_attribute={ADATTRIBUTE}"
				}]
			}`,
			want: []ExtBidTracker{
				{
					Event: "ad_attribute",
					URL:   "https://t.pubmatic.com?bidid=789&ad_attribute={ADATTRIBUTE}",
				},
			},
		},
		{
			name:   "omits_trackers_when_absent",
			rawExt: `{"crtype":"banner"}`,
			want:   nil,
		},
		{
			name:   "empty_trackers_array_unmarshals_to_empty_slice",
			rawExt: `{"trackers":[]}`,
			want:   []ExtBidTracker{},
		},
		{
			name: "passes_malformed_tracker_entries_as_is",
			rawExt: `{
				"trackers": [
					{"event": "ad_attribute"},
					{"url": "https://t.pubmatic.com"},
					{"event": "ad_attribute", "url": "https://t.pubmatic.com?bidid=789&ad_attribute={ADATTRIBUTE}"}
				]
			}`,
			want: []ExtBidTracker{
				{Event: "ad_attribute"},
				{URL: "https://t.pubmatic.com"},
				{
					Event: "ad_attribute",
					URL:   "https://t.pubmatic.com?bidid=789&ad_attribute={ADATTRIBUTE}",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bidExt BidExt
			err := json.Unmarshal([]byte(tt.rawExt), &bidExt)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, bidExt.Trackers)
		})
	}
}

func TestBidExtTrackersMarshalOmitempty(t *testing.T) {
	tests := []struct {
		name   string
		bidExt BidExt
		want   string
	}{
		{
			name: "includes_trackers_when_present",
			bidExt: BidExt{
				CreativeType: "video",
				Trackers: []ExtBidTracker{
					{
						Event: "ad_attribute",
						URL:   "https://t.pubmatic.com?bidid=789&ad_attribute={ADATTRIBUTE}",
					},
				},
			},
			want: `{"crtype":"video","trackers":[{"event":"ad_attribute","url":"https://t.pubmatic.com?bidid=789&ad_attribute={ADATTRIBUTE}"}]}`,
		},
		{
			name: "omits_trackers_when_nil",
			bidExt: BidExt{
				CreativeType: "banner",
			},
			want: `{"crtype":"banner"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.bidExt)
			require.NoError(t, err)
			assert.JSONEq(t, tt.want, string(got))
		})
	}
}
