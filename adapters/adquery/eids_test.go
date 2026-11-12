package adquery

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeRequestsDoesNotModifyUserEids(t *testing.T) {
	request := &openrtb2.BidRequest{
		ID: "request-id",
		User: &openrtb2.User{
			EIDs: []openrtb2.EID{
				{Source: "pubcid.org", UIDs: []openrtb2.UID{{ID: "pubcid-b"}, {ID: ""}, {ID: "pubcid-a"}}},
				{Source: "adquery.io", UIDs: []openrtb2.UID{{ID: "qd_adquery-id-1234567"}}},
				{Source: "", UIDs: []openrtb2.UID{{ID: "no-source"}}},
			},
			Ext: json.RawMessage(`{"eids":[{"source":"criteo.com","uids":[{"id":"criteo-id"}]}]}`),
		},
		Imp: []openrtb2.Imp{{
			ID:  "1",
			Ext: json.RawMessage(`{"bidder":{"placementId":"placement","type":"banner"}}`),
		}},
	}
	before, err := json.Marshal(request)
	require.NoError(t, err)

	bidder := &adapter{endpoint: "https://bidder2.adquery.io/prebid/bid"}
	requests, errs := bidder.MakeRequests(request, nil)
	require.Empty(t, errs)
	require.Len(t, requests, 1)

	after, err := json.Marshal(request)
	require.NoError(t, err)
	assert.JSONEq(t, string(before), string(after))

	var body BidderRequest
	require.NoError(t, json.Unmarshal(requests[0].Body, &body))
	assert.Equal(t, []openrtb2.EID{
		{Source: "adquery.io", UIDs: []openrtb2.UID{{ID: "qd_adquery-id-1234567"}}},
		{Source: "pubcid.org", UIDs: []openrtb2.UID{{ID: "pubcid-a"}, {ID: "pubcid-b"}}},
	}, body.Eids)
}

func TestGetEidsDedupesExtWithDifferentWhitespace(t *testing.T) {
	user := &openrtb2.User{
		Ext: json.RawMessage(`{"eids":[
			{"source":"x.com","uids":[{"id":"a","ext":{"k":1}}],"ext":{"e":1}},
			{"source":"x.com","uids":[{"id":"a","ext":{ "k" : 1 }}],"ext":{ "e": 1 }}
		]}`),
	}

	eids, err := getEids(user)
	require.NoError(t, err)
	require.Len(t, eids, 1)
	assert.Equal(t, json.RawMessage(`{"e":1}`), eids[0].Ext)
	require.Len(t, eids[0].UIDs, 1)
	assert.Equal(t, json.RawMessage(`{"k":1}`), eids[0].UIDs[0].Ext)
}

func TestCompactJSONKeepsInvalidInput(t *testing.T) {
	assert.Equal(t, json.RawMessage(`{invalid`), compactJSON(json.RawMessage(`{invalid`)))
	assert.Empty(t, compactJSON(nil))
}
