package openrtb_ext

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"
)

func TestConvertUpTo26(t *testing.T) {
	testCases := []struct {
		description     string
		givenRequest    openrtb2.BidRequest
		expectedRequest openrtb2.BidRequest
		expectedErr     string
	}{
		{
			description: "Malformed",
			givenRequest: openrtb2.BidRequest{
				Ext: json.RawMessage(`malformed`),
			},
			expectedErr: "req.ext is invalid: expect { or n, but found m",
		},
		{
			description: "2.4 -> 2.6",
			givenRequest: openrtb2.BidRequest{
				ID:   "anyID",
				Imp:  []openrtb2.Imp{{Ext: json.RawMessage(`{"prebid":{"is_rewarded_inventory":1}}`)}},
				Ext:  json.RawMessage(`{"schain":{"complete":1,"nodes":[],"ver":"2"}}`),
				Regs: &openrtb2.Regs{Ext: json.RawMessage(`{"gdpr":1,"us_privacy":"3"}`)},
				User: &openrtb2.User{Ext: json.RawMessage(`{"consent":"1","eids":[{"source":"42"}]}`)},
			},
			expectedRequest: openrtb2.BidRequest{
				ID:     "anyID",
				Imp:    []openrtb2.Imp{{Rwdd: 1}},
				Source: &openrtb2.Source{SChain: &openrtb2.SupplyChain{Complete: 1, Nodes: []openrtb2.SupplyChainNode{}, Ver: "2"}},
				Regs:   &openrtb2.Regs{GDPR: openrtb2.Int8Ptr(1), USPrivacy: "3"},
				User:   &openrtb2.User{Consent: "1", EIDs: []openrtb2.EID{{Source: "42"}}},
			},
		},
		{
			description: "2.4 -> 2.6 + Other Ext Fields",
			givenRequest: openrtb2.BidRequest{
				ID:   "anyID",
				Imp:  []openrtb2.Imp{{Ext: json.RawMessage(`{"prebid":{"is_rewarded_inventory":1},"other":"otherImp"}`)}},
				Ext:  json.RawMessage(`{"schain":{"complete":1,"nodes":[],"ver":"2"},"other":"otherExt"}`),
				Regs: &openrtb2.Regs{Ext: json.RawMessage(`{"gdpr":1,"other":"otherRegs","us_privacy":"3"}`)},
				User: &openrtb2.User{Ext: json.RawMessage(`{"consent":"1","eids":[{"source":"42"}],"other":"otherUser"}`)},
			},
			expectedRequest: openrtb2.BidRequest{
				ID:     "anyID",
				Imp:    []openrtb2.Imp{{Rwdd: 1, Ext: json.RawMessage(`{"other":"otherImp"}`)}},
				Ext:    json.RawMessage(`{"other":"otherExt"}`),
				Source: &openrtb2.Source{SChain: &openrtb2.SupplyChain{Complete: 1, Nodes: []openrtb2.SupplyChainNode{}, Ver: "2"}},
				Regs:   &openrtb2.Regs{GDPR: openrtb2.Int8Ptr(1), USPrivacy: "3", Ext: json.RawMessage(`{"other":"otherRegs"}`)},
				User:   &openrtb2.User{Consent: "1", EIDs: []openrtb2.EID{{Source: "42"}}, Ext: json.RawMessage(`{"other":"otherUser"}`)},
			},
		},
		{
			description: "2.5 -> 2.6",
			givenRequest: openrtb2.BidRequest{
				ID:     "anyID",
				Imp:    []openrtb2.Imp{{Ext: json.RawMessage(`{"prebid":{"is_rewarded_inventory":1}}`)}},
				Source: &openrtb2.Source{Ext: json.RawMessage(`{"schain":{"complete":1,"nodes":[],"ver":"2"}}`)},
				Regs:   &openrtb2.Regs{Ext: json.RawMessage(`{"gdpr":1,"us_privacy":"3"}`)},
				User:   &openrtb2.User{Ext: json.RawMessage(`{"consent":"1","eids":[{"source":"42"}]}`)},
			},
			expectedRequest: openrtb2.BidRequest{
				ID:     "anyID",
				Imp:    []openrtb2.Imp{{Rwdd: 1}},
				Source: &openrtb2.Source{SChain: &openrtb2.SupplyChain{Complete: 1, Nodes: []openrtb2.SupplyChainNode{}, Ver: "2"}},
				Regs:   &openrtb2.Regs{GDPR: openrtb2.Int8Ptr(1), USPrivacy: "3"},
				User:   &openrtb2.User{Consent: "1", EIDs: []openrtb2.EID{{Source: "42"}}},
			},
		},
		{
			description: "2.5 -> 2.6 + Other Ext Fields",
			givenRequest: openrtb2.BidRequest{
				ID:     "anyID",
				Imp:    []openrtb2.Imp{{Ext: json.RawMessage(`{"prebid":{"is_rewarded_inventory":1},"other":"otherImp"}`)}},
				Ext:    json.RawMessage(`{"other":"otherExt"}`),
				Source: &openrtb2.Source{Ext: json.RawMessage(`{"schain":{"complete":1,"nodes":[],"ver":"2"},"other":"otherSource"}`)},
				Regs:   &openrtb2.Regs{Ext: json.RawMessage(`{"gdpr":1,"us_privacy":"3","other":"otherRegs"}`)},
				User:   &openrtb2.User{Ext: json.RawMessage(`{"consent":"1","eids":[{"source":"42"}],"other":"otherUser"}`)},
			},
			expectedRequest: openrtb2.BidRequest{
				ID:     "anyID",
				Imp:    []openrtb2.Imp{{Rwdd: 1, Ext: json.RawMessage(`{"other":"otherImp"}`)}},
				Ext:    json.RawMessage(`{"other":"otherExt"}`),
				Source: &openrtb2.Source{SChain: &openrtb2.SupplyChain{Complete: 1, Nodes: []openrtb2.SupplyChainNode{}, Ver: "2"}, Ext: json.RawMessage(`{"other":"otherSource"}`)},
				Regs:   &openrtb2.Regs{GDPR: openrtb2.Int8Ptr(1), USPrivacy: "3", Ext: json.RawMessage(`{"other":"otherRegs"}`)},
				User:   &openrtb2.User{Consent: "1", EIDs: []openrtb2.EID{{Source: "42"}}, Ext: json.RawMessage(`{"other":"otherUser"}`)},
			},
		},
	}

	for _, test := range testCases {
		w := &RequestWrapper{BidRequest: &test.givenRequest}
		err := ConvertUpTo26(w)
		if len(test.expectedErr) > 0 {
			assert.EqualError(t, err, test.expectedErr, test.description)
		} else {
			assert.NoError(t, w.RebuildRequest(), test.description)
			assert.Equal(t, test.expectedRequest, *w.BidRequest, test.description)
		}
	}
}

func TestConvertUpEnsureExt(t *testing.T) {
	testCases := []struct {
		description  string
		givenRequest openrtb2.BidRequest
		expectedErr  string
	}{
		{
			description:  "Empty",
			givenRequest: openrtb2.BidRequest{},
		},
		{
			description:  "Ext",
			givenRequest: openrtb2.BidRequest{Ext: json.RawMessage("malformed")},
			expectedErr:  "req.ext is invalid: expect { or n, but found m",
		},
		{
			description:  "Source.Ext",
			givenRequest: openrtb2.BidRequest{Source: &openrtb2.Source{Ext: json.RawMessage("malformed")}},
			expectedErr:  "req.source.ext is invalid: expect { or n, but found m",
		},
		{
			description:  "Regs.Ext",
			givenRequest: openrtb2.BidRequest{Regs: &openrtb2.Regs{Ext: json.RawMessage("malformed")}},
			expectedErr:  "req.regs.ext is invalid: expect { or n, but found m",
		},
		{
			description:  "User.Ext",
			givenRequest: openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage("malformed")}},
			expectedErr:  "req.user.ext is invalid: expect { or n, but found m",
		},
		{
			description:  "Imp.Ext",
			givenRequest: openrtb2.BidRequest{Imp: []openrtb2.Imp{{Ext: json.RawMessage("malformed")}}},
			expectedErr:  "imp[0].imp.ext is invalid: expect { or n, but found m",
		},
	}

	for _, test := range testCases {
		w := &RequestWrapper{BidRequest: &test.givenRequest}
		err := convertUpEnsureExt(w)
		if len(test.expectedErr) > 0 {
			assert.EqualError(t, err, test.expectedErr, test.description)
		} else {
			assert.NoError(t, err, test.description)
		}
	}
}

func TestMoveSupplyChainFrom24To25(t *testing.T) {
	var (
		schain1 = json.RawMessage(`{"schain":{"complete":1,"nodes":[],"ver":"1"}}`)
		schain2 = json.RawMessage(`{"schain":{"complete":1,"nodes":[],"ver":"2"}}`)
	)

	testCases := []struct {
		description     string
		givenRequest    openrtb2.BidRequest
		expectedRequest openrtb2.BidRequest
	}{
		{
			description:     "Not Present",
			givenRequest:    openrtb2.BidRequest{},
			expectedRequest: openrtb2.BidRequest{},
		},
		{
			description:     "2.4 Migrated To 2.5 - Source Doesn't Exist",
			givenRequest:    openrtb2.BidRequest{Ext: schain1},
			expectedRequest: openrtb2.BidRequest{Source: &openrtb2.Source{Ext: schain1}},
		},
		{
			description:     "2.4 Migrated To 2.5 - Source Exists",
			givenRequest:    openrtb2.BidRequest{Source: &openrtb2.Source{}, Ext: schain1},
			expectedRequest: openrtb2.BidRequest{Source: &openrtb2.Source{Ext: schain1}},
		},
		{
			description:     "2.4 Dropped",
			givenRequest:    openrtb2.BidRequest{Source: &openrtb2.Source{Ext: schain1}, Ext: schain2},
			expectedRequest: openrtb2.BidRequest{Source: &openrtb2.Source{Ext: schain1}},
		},
		{
			description:     "2.5 Left Alone",
			givenRequest:    openrtb2.BidRequest{Source: &openrtb2.Source{Ext: schain1}},
			expectedRequest: openrtb2.BidRequest{Source: &openrtb2.Source{Ext: schain1}},
		},
	}

	for _, test := range testCases {
		w := &RequestWrapper{BidRequest: &test.givenRequest}
		moveSupplyChainFrom24To25(w)
		assert.NoError(t, w.RebuildRequest(), test.description)
		assert.Equal(t, test.expectedRequest, *w.BidRequest, test.description)
	}
}

func TestConvertSupplyChainFrom25To26(t *testing.T) {
	var (
		schain1     = &openrtb2.SupplyChain{Complete: 1, Nodes: []openrtb2.SupplyChainNode{}, Ver: "1"}
		schain1Json = json.RawMessage(`{"schain":{"complete":1,"nodes":[],"ver":"1"}}`)
		schain2Json = json.RawMessage(`{"schain":{"complete":1,"nodes":[],"ver":"2"}}`)
	)

	testCases := []struct {
		description     string
		givenRequest    openrtb2.BidRequest
		expectedRequest openrtb2.BidRequest
	}{
		{
			description:     "Not Present",
			givenRequest:    openrtb2.BidRequest{},
			expectedRequest: openrtb2.BidRequest{},
		},
		{
			description:     "2.5 Migrated To 2.6",
			givenRequest:    openrtb2.BidRequest{Source: &openrtb2.Source{Ext: schain1Json}},
			expectedRequest: openrtb2.BidRequest{Source: &openrtb2.Source{SChain: schain1}},
		},
		{
			description:     "2.5 Dropped",
			givenRequest:    openrtb2.BidRequest{Source: &openrtb2.Source{SChain: schain1, Ext: schain2Json}},
			expectedRequest: openrtb2.BidRequest{Source: &openrtb2.Source{SChain: schain1}},
		},
		{
			description:     "2.6 Left Alone",
			givenRequest:    openrtb2.BidRequest{Source: &openrtb2.Source{SChain: schain1}},
			expectedRequest: openrtb2.BidRequest{Source: &openrtb2.Source{SChain: schain1}},
		},
	}

	for _, test := range testCases {
		w := &RequestWrapper{BidRequest: &test.givenRequest}
		moveSupplyChainFrom25To26(w)
		assert.NoError(t, w.RebuildRequest(), test.description)
		assert.Equal(t, test.expectedRequest, *w.BidRequest, test.description)
	}
}

func TestMoveGDPRFrom25To26(t *testing.T) {
	testCases := []struct {
		description     string
		givenRequest    openrtb2.BidRequest
		expectedRequest openrtb2.BidRequest
	}{
		{
			description:     "Not Present",
			givenRequest:    openrtb2.BidRequest{},
			expectedRequest: openrtb2.BidRequest{},
		},
		{
			description:     "2.5 Migrated To 2.6",
			givenRequest:    openrtb2.BidRequest{Regs: &openrtb2.Regs{Ext: json.RawMessage(`{"gdpr":0}`)}},
			expectedRequest: openrtb2.BidRequest{Regs: &openrtb2.Regs{GDPR: openrtb2.Int8Ptr(0)}},
		},
		{
			description:     "2.5 Dropped",
			givenRequest:    openrtb2.BidRequest{Regs: &openrtb2.Regs{GDPR: openrtb2.Int8Ptr(0), Ext: json.RawMessage(`{"gdpr":1}`)}},
			expectedRequest: openrtb2.BidRequest{Regs: &openrtb2.Regs{GDPR: openrtb2.Int8Ptr(0)}},
		},
		{
			description:     "2.6 Left Alone",
			givenRequest:    openrtb2.BidRequest{Regs: &openrtb2.Regs{GDPR: openrtb2.Int8Ptr(0)}},
			expectedRequest: openrtb2.BidRequest{Regs: &openrtb2.Regs{GDPR: openrtb2.Int8Ptr(0)}},
		},
	}

	for _, test := range testCases {
		w := &RequestWrapper{BidRequest: &test.givenRequest}
		moveGDPRFrom25To26(w)
		assert.NoError(t, w.RebuildRequest(), test.description)
		assert.Equal(t, test.expectedRequest, *w.BidRequest, test.description)
	}
}

func TestMoveConsentFrom25To26(t *testing.T) {
	testCases := []struct {
		description     string
		givenRequest    openrtb2.BidRequest
		expectedRequest openrtb2.BidRequest
	}{
		{
			description:     "Not Present",
			givenRequest:    openrtb2.BidRequest{},
			expectedRequest: openrtb2.BidRequest{},
		},
		{
			description:     "2.5 Migrated To 2.6",
			givenRequest:    openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"consent":"1"}`)}},
			expectedRequest: openrtb2.BidRequest{User: &openrtb2.User{Consent: "1"}},
		},
		{
			description:     "2.5 Dropped",
			givenRequest:    openrtb2.BidRequest{User: &openrtb2.User{Consent: "1", Ext: json.RawMessage(`{"consent":2}`)}},
			expectedRequest: openrtb2.BidRequest{User: &openrtb2.User{Consent: "1"}},
		},
		{
			description:     "2.6 Left Alone",
			givenRequest:    openrtb2.BidRequest{User: &openrtb2.User{Consent: "1"}},
			expectedRequest: openrtb2.BidRequest{User: &openrtb2.User{Consent: "1"}},
		},
	}

	for _, test := range testCases {
		w := &RequestWrapper{BidRequest: &test.givenRequest}
		moveConsentFrom25To26(w)
		assert.NoError(t, w.RebuildRequest(), test.description)
		assert.Equal(t, test.expectedRequest, *w.BidRequest, test.description)
	}
}

func TestMoveUSPrivacyFrom25To26(t *testing.T) {
	testCases := []struct {
		description     string
		givenRequest    openrtb2.BidRequest
		expectedRequest openrtb2.BidRequest
	}{
		{
			description:     "Not Present",
			givenRequest:    openrtb2.BidRequest{},
			expectedRequest: openrtb2.BidRequest{},
		},
		{
			description:     "2.5 Migrated To 2.6",
			givenRequest:    openrtb2.BidRequest{Regs: &openrtb2.Regs{Ext: json.RawMessage(`{"us_privacy":"1"}`)}},
			expectedRequest: openrtb2.BidRequest{Regs: &openrtb2.Regs{USPrivacy: "1"}},
		},
		{
			description:     "2.5 Dropped",
			givenRequest:    openrtb2.BidRequest{Regs: &openrtb2.Regs{USPrivacy: "1", Ext: json.RawMessage(`{"us_privacy":"2"}`)}},
			expectedRequest: openrtb2.BidRequest{Regs: &openrtb2.Regs{USPrivacy: "1"}},
		},
		{
			description:     "2.6 Left Alone",
			givenRequest:    openrtb2.BidRequest{Regs: &openrtb2.Regs{USPrivacy: "1"}},
			expectedRequest: openrtb2.BidRequest{Regs: &openrtb2.Regs{USPrivacy: "1"}},
		},
	}

	for _, test := range testCases {
		w := &RequestWrapper{BidRequest: &test.givenRequest}
		moveUSPrivacyFrom25To26(w)
		assert.NoError(t, w.RebuildRequest(), test.description)
		assert.Equal(t, test.expectedRequest, *w.BidRequest, test.description)
	}
}

func TestMoveEIDFrom25To26(t *testing.T) {
	var (
		eid1     = []openrtb2.EID{{Source: "1"}}
		eid1Json = json.RawMessage(`{"eids":[{"source":"1"}]}`)
		eid2Json = json.RawMessage(`{"eids":[{"source":"2"}]}`)
	)

	testCases := []struct {
		description     string
		givenRequest    openrtb2.BidRequest
		expectedRequest openrtb2.BidRequest
	}{
		{
			description:     "Not Present",
			givenRequest:    openrtb2.BidRequest{},
			expectedRequest: openrtb2.BidRequest{},
		},
		{
			description:     "2.5 Migrated To 2.6",
			givenRequest:    openrtb2.BidRequest{User: &openrtb2.User{Ext: eid1Json}},
			expectedRequest: openrtb2.BidRequest{User: &openrtb2.User{EIDs: eid1}},
		},
		{
			description:     "2.5 Dropped",
			givenRequest:    openrtb2.BidRequest{User: &openrtb2.User{EIDs: eid1, Ext: eid2Json}},
			expectedRequest: openrtb2.BidRequest{User: &openrtb2.User{EIDs: eid1}},
		},
		{
			description:     "2.6 Left Alone",
			givenRequest:    openrtb2.BidRequest{User: &openrtb2.User{EIDs: eid1}},
			expectedRequest: openrtb2.BidRequest{User: &openrtb2.User{EIDs: eid1}},
		},
	}

	for _, test := range testCases {
		w := &RequestWrapper{BidRequest: &test.givenRequest}
		moveEIDFrom25To26(w)
		assert.NoError(t, w.RebuildRequest(), test.description)
		assert.Equal(t, test.expectedRequest, *w.BidRequest, test.description)
	}
}

func TestMoveRewardedFromPrebidExtTo26(t *testing.T) {
	var (
		rwdd1Json = json.RawMessage(`{"prebid":{"is_rewarded_inventory":1}}`)
		rwdd2Json = json.RawMessage(`{"prebid":{"is_rewarded_inventory":2}}`)
	)

	testCases := []struct {
		description string
		givenImp    openrtb2.Imp
		expectedImp openrtb2.Imp
	}{
		{
			description: "Not Present - No Ext",
			givenImp:    openrtb2.Imp{},
			expectedImp: openrtb2.Imp{},
		},
		{
			description: "Not Present - Empty Ext",
			givenImp:    openrtb2.Imp{Ext: json.RawMessage(`{}`)},
			expectedImp: openrtb2.Imp{Ext: json.RawMessage(`{}`)},
		},
		{
			description: "Not Present - Null Prebid Ext",
			givenImp:    openrtb2.Imp{Ext: json.RawMessage(`{"prebid":null}`)},
			expectedImp: openrtb2.Imp{Ext: json.RawMessage(`{"prebid":null}`)},
		},
		{
			description: "Not Present - Empty Prebid Ext",
			givenImp:    openrtb2.Imp{Ext: json.RawMessage(`{"prebid":{}}`)},
			expectedImp: openrtb2.Imp{Ext: json.RawMessage(`{"prebid":{}}`)},
		},
		{
			description: "Prebid Ext Migrated To 2.6",
			givenImp:    openrtb2.Imp{Ext: rwdd1Json},
			expectedImp: openrtb2.Imp{Rwdd: 1},
		},
		{
			description: "2.5 Dropped",
			givenImp:    openrtb2.Imp{Rwdd: 1, Ext: rwdd2Json},
			expectedImp: openrtb2.Imp{Rwdd: 1},
		},
		{
			description: "2.6 Left Alone",
			givenImp:    openrtb2.Imp{Rwdd: 1},
			expectedImp: openrtb2.Imp{Rwdd: 1},
		},
	}

	for _, test := range testCases {
		w := &ImpWrapper{Imp: &test.givenImp}
		moveRewardedFromPrebidExtTo26(w)
		assert.NoError(t, w.RebuildImp(), test.description)
		assert.Equal(t, test.expectedImp, *w.Imp, test.description)
	}
}
