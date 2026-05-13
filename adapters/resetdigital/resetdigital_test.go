package resetdigital

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {

	bidder, buildErr := Builder(openrtb_ext.BidderResetDigital, config.Adapter{
		Endpoint: "https://test.com"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "resetdigitaltest", bidder)
}

func TestGetLiveRampEIDs(t *testing.T) {
	testCases := []struct {
		name         string
		user         *openrtb2.User
		expectedJSON string
	}{
		{
			name: "no user",
		},
		{
			name: "no eids",
			user: &openrtb2.User{},
		},
		{
			name: "openrtb 2.6 user eids",
			user: &openrtb2.User{
				EIDs: []openrtb2.EID{
					{
						Source: liveRampEIDSource,
						UIDs: []openrtb2.UID{
							{ID: "envelope-26"},
						},
					},
				},
			},
			expectedJSON: `[{"source":"liveramp.com","uids":[{"id":"envelope-26"}]}]`,
		},
		{
			name: "openrtb 2.5 user ext eids",
			user: &openrtb2.User{
				Ext: json.RawMessage(`{"eids":[{"source":"liveramp.com","uids":[{"id":"envelope-25","atype":1,"ext":{"rtiPartner":"idl","stype":"ppuid"}}]}]}`),
			},
			expectedJSON: `[{"source":"liveramp.com","uids":[{"id":"envelope-25","atype":1,"ext":{"rtiPartner":"idl","stype":"ppuid"}}]}]`,
		},
		{
			name: "only liveramp eids",
			user: &openrtb2.User{
				EIDs: []openrtb2.EID{
					{
						Source: "sharedid.org",
						UIDs: []openrtb2.UID{
							{ID: "shared-id"},
						},
					},
					{
						Source: liveRampEIDSource,
						UIDs: []openrtb2.UID{
							{ID: "liveramp-id"},
						},
					},
				},
			},
			expectedJSON: `[{"source":"liveramp.com","uids":[{"id":"liveramp-id"}]}]`,
		},
		{
			name: "malformed user ext",
			user: &openrtb2.User{
				Ext: json.RawMessage(`malformed`),
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			eids := getLiveRampEIDs(test.user)

			if test.expectedJSON == "" {
				assert.Empty(t, eids)
				return
			}

			actualJSON, err := json.Marshal(eids)
			assert.NoError(t, err)
			assert.JSONEq(t, test.expectedJSON, string(actualJSON))
		})
	}
}
