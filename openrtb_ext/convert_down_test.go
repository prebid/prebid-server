package openrtb_ext

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/stretchr/testify/assert"
)

func TestConvertDownTo25(t *testing.T) {
	testCases := []struct {
		description     string
		givenRequest    openrtb2.BidRequest
		expectedRequest openrtb2.BidRequest
		expectedErr     string
	}{
		{
			description: "2.6 -> 2.5",
			givenRequest: openrtb2.BidRequest{
				ID:     "anyID",
				Imp:    []openrtb2.Imp{{Rwdd: 1}},
				Source: &openrtb2.Source{SChain: &openrtb2.SupplyChain{Complete: 1, Nodes: []openrtb2.SupplyChainNode{}, Ver: "2"}},
				Regs:   &openrtb2.Regs{GDPR: openrtb2.Int8Ptr(1), USPrivacy: "3"},
				User:   &openrtb2.User{Consent: "1", EIDs: []openrtb2.EID{{Source: "42"}}},
			},
			expectedRequest: openrtb2.BidRequest{
				ID:     "anyID",
				Imp:    []openrtb2.Imp{{Ext: json.RawMessage(`{"prebid":{"is_rewarded_inventory":1}}`)}},
				Source: &openrtb2.Source{Ext: json.RawMessage(`{"schain":{"complete":1,"nodes":[],"ver":"2"}}`)},
				Regs:   &openrtb2.Regs{Ext: json.RawMessage(`{"gdpr":1,"us_privacy":"3"}`)},
				User:   &openrtb2.User{Ext: json.RawMessage(`{"consent":"1","eids":[{"source":"42"}]}`)},
			},
		},
		{
			description: "2.6 -> 2.5 + Other Ext Fields",
			givenRequest: openrtb2.BidRequest{
				ID:     "anyID",
				Imp:    []openrtb2.Imp{{Rwdd: 1, Ext: json.RawMessage(`{"other":"otherImp"}`)}},
				Ext:    json.RawMessage(`{"other":"otherExt"}`),
				Source: &openrtb2.Source{SChain: &openrtb2.SupplyChain{Complete: 1, Nodes: []openrtb2.SupplyChainNode{}, Ver: "2"}, Ext: json.RawMessage(`{"other":"otherSource"}`)},
				Regs:   &openrtb2.Regs{GDPR: openrtb2.Int8Ptr(1), USPrivacy: "3", Ext: json.RawMessage(`{"other":"otherRegs"}`)},
				User:   &openrtb2.User{Consent: "1", EIDs: []openrtb2.EID{{Source: "42"}}, Ext: json.RawMessage(`{"other":"otherUser"}`)},
			},
			expectedRequest: openrtb2.BidRequest{
				ID:     "anyID",
				Imp:    []openrtb2.Imp{{Ext: json.RawMessage(`{"other":"otherImp","prebid":{"is_rewarded_inventory":1}}`)}},
				Ext:    json.RawMessage(`{"other":"otherExt"}`),
				Source: &openrtb2.Source{Ext: json.RawMessage(`{"other":"otherSource","schain":{"complete":1,"nodes":[],"ver":"2"}}`)},
				Regs:   &openrtb2.Regs{Ext: json.RawMessage(`{"gdpr":1,"other":"otherRegs","us_privacy":"3"}`)},
				User:   &openrtb2.User{Ext: json.RawMessage(`{"consent":"1","eids":[{"source":"42"}],"other":"otherUser"}`)},
			},
		},
		{
			description: "Malformed - SChain",
			givenRequest: openrtb2.BidRequest{
				ID:     "anyID",
				Source: &openrtb2.Source{SChain: &openrtb2.SupplyChain{Complete: 1, Nodes: []openrtb2.SupplyChainNode{}, Ver: "2"}, Ext: json.RawMessage(`malformed`)},
			},
			expectedErr: "invalid character 'm' looking for beginning of value",
		},
		{
			description: "Malformed - GDPR",
			givenRequest: openrtb2.BidRequest{
				ID:   "anyID",
				Regs: &openrtb2.Regs{GDPR: openrtb2.Int8Ptr(1), Ext: json.RawMessage(`malformed`)},
			},
			expectedErr: "invalid character 'm' looking for beginning of value",
		},
		{
			description: "Malformed - Consent",
			givenRequest: openrtb2.BidRequest{
				ID:   "anyID",
				User: &openrtb2.User{Consent: "1", Ext: json.RawMessage(`malformed`)},
			},
			expectedErr: "invalid character 'm' looking for beginning of value",
		},
		{
			description: "Malformed - USPrivacy",
			givenRequest: openrtb2.BidRequest{
				ID:   "anyID",
				Regs: &openrtb2.Regs{USPrivacy: "3", Ext: json.RawMessage(`malformed`)},
			},
			expectedErr: "invalid character 'm' looking for beginning of value",
		},
		{
			description: "Malformed - EID",
			givenRequest: openrtb2.BidRequest{
				ID:   "anyID",
				User: &openrtb2.User{EIDs: []openrtb2.EID{{Source: "42"}}, Ext: json.RawMessage(`malformed`)},
			},
			expectedErr: "invalid character 'm' looking for beginning of value",
		},
		{
			description: "Malformed - Imp",
			givenRequest: openrtb2.BidRequest{
				ID:  "anyID",
				Imp: []openrtb2.Imp{{Rwdd: 1, Ext: json.RawMessage(`malformed`)}},
			},
			expectedErr: "invalid character 'm' looking for beginning of value",
		},
	}

	for _, test := range testCases {
		w := &RequestWrapper{BidRequest: &test.givenRequest}
		err := ConvertDownTo25(w)
		if len(test.expectedErr) > 0 {
			assert.EqualError(t, err, test.expectedErr, test.description)
		} else {
			assert.NoError(t, w.RebuildRequest(), test.description)
			assert.Equal(t, test.expectedRequest, *w.BidRequest, test.description)
		}
	}
}

func TestMoveSupplyChainFrom26To25(t *testing.T) {
	var (
		schain1     = &openrtb2.SupplyChain{Complete: 1, Nodes: []openrtb2.SupplyChainNode{}, Ver: "1"}
		schain1Json = json.RawMessage(`{"schain":{"complete":1,"nodes":[],"ver":"1"}}`)
		schain2Json = json.RawMessage(`{"schain":{"complete":1,"nodes":[],"ver":"2"}}`)
	)

	testCases := []struct {
		description     string
		givenRequest    openrtb2.BidRequest
		expectedRequest openrtb2.BidRequest
		expectedErr     string
	}{
		{
			description:     "Not Present - Source",
			givenRequest:    openrtb2.BidRequest{},
			expectedRequest: openrtb2.BidRequest{},
		},
		{
			description:     "Not Present - Source Schain",
			givenRequest:    openrtb2.BidRequest{Source: &openrtb2.Source{}},
			expectedRequest: openrtb2.BidRequest{Source: &openrtb2.Source{}},
		},
		{
			description:     "2.6 Migrated To 2.5",
			givenRequest:    openrtb2.BidRequest{Source: &openrtb2.Source{SChain: schain1}},
			expectedRequest: openrtb2.BidRequest{Source: &openrtb2.Source{Ext: schain1Json}},
		},
		{
			description:     "2.5 Overwritten",
			givenRequest:    openrtb2.BidRequest{Source: &openrtb2.Source{SChain: schain1, Ext: schain2Json}},
			expectedRequest: openrtb2.BidRequest{Source: &openrtb2.Source{Ext: schain1Json}},
		},
		{
			description:  "Malformed",
			givenRequest: openrtb2.BidRequest{Source: &openrtb2.Source{SChain: schain1, Ext: json.RawMessage(`malformed`)}},
			expectedErr:  "invalid character 'm' looking for beginning of value",
		},
	}

	for _, test := range testCases {
		w := &RequestWrapper{BidRequest: &test.givenRequest}
		err := moveSupplyChainFrom26To25(w)

		if len(test.expectedErr) > 0 {
			assert.EqualError(t, err, test.expectedErr, test.description)
		} else {
			assert.NoError(t, w.RebuildRequest(), test.description)
			assert.Equal(t, test.expectedRequest, *w.BidRequest, test.description)
		}
	}
}

func TestMoveGDPRFrom26To25(t *testing.T) {
	testCases := []struct {
		description     string
		givenRequest    openrtb2.BidRequest
		expectedRequest openrtb2.BidRequest
		expectedErr     string
	}{
		{
			description:     "Not Present - Regs",
			givenRequest:    openrtb2.BidRequest{},
			expectedRequest: openrtb2.BidRequest{},
		},
		{
			description:     "Not Present - Regs GDPR",
			givenRequest:    openrtb2.BidRequest{Regs: &openrtb2.Regs{}},
			expectedRequest: openrtb2.BidRequest{Regs: &openrtb2.Regs{}},
		},
		{
			description:     "2.6 Migrated To 2.5",
			givenRequest:    openrtb2.BidRequest{Regs: &openrtb2.Regs{GDPR: openrtb2.Int8Ptr(0)}},
			expectedRequest: openrtb2.BidRequest{Regs: &openrtb2.Regs{Ext: json.RawMessage(`{"gdpr":0}`)}},
		},
		{
			description:     "2.5 Overwritten",
			givenRequest:    openrtb2.BidRequest{Regs: &openrtb2.Regs{GDPR: openrtb2.Int8Ptr(0), Ext: json.RawMessage(`{"gdpr":1}`)}},
			expectedRequest: openrtb2.BidRequest{Regs: &openrtb2.Regs{Ext: json.RawMessage(`{"gdpr":0}`)}},
		},
		{
			description:  "Malformed",
			givenRequest: openrtb2.BidRequest{Regs: &openrtb2.Regs{GDPR: openrtb2.Int8Ptr(0), Ext: json.RawMessage(`malformed`)}},
			expectedErr:  "invalid character 'm' looking for beginning of value",
		},
	}

	for _, test := range testCases {
		w := &RequestWrapper{BidRequest: &test.givenRequest}
		err := moveGDPRFrom26To25(w)

		if len(test.expectedErr) > 0 {
			assert.EqualError(t, err, test.expectedErr, test.description)
		} else {
			assert.NoError(t, w.RebuildRequest(), test.description)
			assert.Equal(t, test.expectedRequest, *w.BidRequest, test.description)
		}
	}
}

func TestMoveConsentFrom26To25(t *testing.T) {
	testCases := []struct {
		description     string
		givenRequest    openrtb2.BidRequest
		expectedRequest openrtb2.BidRequest
		expectedErr     string
	}{
		{
			description:     "Not Present - User",
			givenRequest:    openrtb2.BidRequest{},
			expectedRequest: openrtb2.BidRequest{},
		},
		{
			description:     "Not Present - User Consent",
			givenRequest:    openrtb2.BidRequest{User: &openrtb2.User{}},
			expectedRequest: openrtb2.BidRequest{User: &openrtb2.User{}},
		},
		{
			description:     "2.6 Migrated To 2.5",
			givenRequest:    openrtb2.BidRequest{User: &openrtb2.User{Consent: "1"}},
			expectedRequest: openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"consent":"1"}`)}},
		},
		{
			description:     "2.5 Overwritten",
			givenRequest:    openrtb2.BidRequest{User: &openrtb2.User{Consent: "1", Ext: json.RawMessage(`{"consent":"2"}`)}},
			expectedRequest: openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage(`{"consent":"1"}`)}},
		},
		{
			description:  "Malformed",
			givenRequest: openrtb2.BidRequest{User: &openrtb2.User{Consent: "1", Ext: json.RawMessage(`malformed`)}},
			expectedErr:  "invalid character 'm' looking for beginning of value",
		},
	}

	for _, test := range testCases {
		w := &RequestWrapper{BidRequest: &test.givenRequest}
		err := moveConsentFrom26To25(w)

		if len(test.expectedErr) > 0 {
			assert.EqualError(t, err, test.expectedErr, test.description)
		} else {
			assert.NoError(t, w.RebuildRequest(), test.description)
			assert.Equal(t, test.expectedRequest, *w.BidRequest, test.description)
		}
	}
}

func TestMoveUSPrivacyFrom26To25(t *testing.T) {
	testCases := []struct {
		description     string
		givenRequest    openrtb2.BidRequest
		expectedRequest openrtb2.BidRequest
		expectedErr     string
	}{
		{
			description:     "Not Present - Regs",
			givenRequest:    openrtb2.BidRequest{},
			expectedRequest: openrtb2.BidRequest{},
		},
		{
			description:     "Not Present - Regs USPrivacy",
			givenRequest:    openrtb2.BidRequest{Regs: &openrtb2.Regs{}},
			expectedRequest: openrtb2.BidRequest{Regs: &openrtb2.Regs{}},
		},
		{
			description:     "2.6 Migrated To 2.5",
			givenRequest:    openrtb2.BidRequest{Regs: &openrtb2.Regs{USPrivacy: "1"}},
			expectedRequest: openrtb2.BidRequest{Regs: &openrtb2.Regs{Ext: json.RawMessage(`{"us_privacy":"1"}`)}},
		},
		{
			description:     "2.5 Overwritten",
			givenRequest:    openrtb2.BidRequest{Regs: &openrtb2.Regs{USPrivacy: "1", Ext: json.RawMessage(`{"us_privacy":"2"}`)}},
			expectedRequest: openrtb2.BidRequest{Regs: &openrtb2.Regs{Ext: json.RawMessage(`{"us_privacy":"1"}`)}},
		},
		{
			description:  "Malformed",
			givenRequest: openrtb2.BidRequest{Regs: &openrtb2.Regs{USPrivacy: "1", Ext: json.RawMessage(`malformed`)}},
			expectedErr:  "invalid character 'm' looking for beginning of value",
		},
	}

	for _, test := range testCases {
		w := &RequestWrapper{BidRequest: &test.givenRequest}
		err := moveUSPrivacyFrom26To25(w)

		if len(test.expectedErr) > 0 {
			assert.EqualError(t, err, test.expectedErr, test.description)
		} else {
			assert.NoError(t, w.RebuildRequest(), test.description)
			assert.Equal(t, test.expectedRequest, *w.BidRequest, test.description)
		}
	}
}

func TestMoveEIDFrom26To25(t *testing.T) {
	var (
		eid1     = []openrtb2.EID{{Source: "1"}}
		eid1Json = json.RawMessage(`{"eids":[{"source":"1"}]}`)
		eid2Json = json.RawMessage(`{"eids":[{"source":"2"}]}`)
	)

	testCases := []struct {
		description     string
		givenRequest    openrtb2.BidRequest
		expectedRequest openrtb2.BidRequest
		expectedErr     string
	}{
		{
			description:     "Not Present - User",
			givenRequest:    openrtb2.BidRequest{},
			expectedRequest: openrtb2.BidRequest{},
		},
		{
			description:     "Not Present - User EIDs",
			givenRequest:    openrtb2.BidRequest{User: &openrtb2.User{}},
			expectedRequest: openrtb2.BidRequest{User: &openrtb2.User{}},
		},
		{
			description:     "2.6 Migrated To 2.5",
			givenRequest:    openrtb2.BidRequest{User: &openrtb2.User{EIDs: eid1}},
			expectedRequest: openrtb2.BidRequest{User: &openrtb2.User{Ext: eid1Json}},
		},
		{
			description:     "2.6 Migrated To 2.5 - Empty",
			givenRequest:    openrtb2.BidRequest{User: &openrtb2.User{EIDs: []openrtb2.EID{}}},
			expectedRequest: openrtb2.BidRequest{User: &openrtb2.User{}},
		},
		{
			description:     "2.5 Overwritten",
			givenRequest:    openrtb2.BidRequest{User: &openrtb2.User{EIDs: eid1, Ext: eid2Json}},
			expectedRequest: openrtb2.BidRequest{User: &openrtb2.User{Ext: eid1Json}},
		},
		{
			description:  "Malformed",
			givenRequest: openrtb2.BidRequest{User: &openrtb2.User{EIDs: eid1, Ext: json.RawMessage(`malformed`)}},
			expectedErr:  "invalid character 'm' looking for beginning of value",
		},
	}

	for _, test := range testCases {
		w := &RequestWrapper{BidRequest: &test.givenRequest}
		err := moveEIDFrom26To25(w)

		if len(test.expectedErr) > 0 {
			assert.EqualError(t, err, test.expectedErr, test.description)
		} else {
			assert.NoError(t, w.RebuildRequest(), test.description)
			assert.Equal(t, test.expectedRequest, *w.BidRequest, test.description)
		}
	}
}

func TestMoveRewardedFrom26ToPrebidExt(t *testing.T) {
	testCases := []struct {
		description string
		givenImp    openrtb2.Imp
		expectedImp openrtb2.Imp
		expectedErr string
	}{
		{
			description: "Not Present",
			givenImp:    openrtb2.Imp{},
			expectedImp: openrtb2.Imp{},
		},
		{
			description: "2.6 Migrated To Prebid Ext",
			givenImp:    openrtb2.Imp{Rwdd: 1},
			expectedImp: openrtb2.Imp{Ext: json.RawMessage(`{"prebid":{"is_rewarded_inventory":1}}`)},
		},
		{
			description: "Prebid Ext Overwritten",
			givenImp:    openrtb2.Imp{Rwdd: 1, Ext: json.RawMessage(`{"prebid":{"is_rewarded_inventory":2}}`)},
			expectedImp: openrtb2.Imp{Ext: json.RawMessage(`{"prebid":{"is_rewarded_inventory":1}}`)},
		},
		{
			description: "Malformed",
			givenImp:    openrtb2.Imp{Rwdd: 1, Ext: json.RawMessage(`malformed`)},
			expectedErr: "invalid character 'm' looking for beginning of value",
		},
	}

	for _, test := range testCases {
		w := &ImpWrapper{Imp: &test.givenImp}
		err := moveRewardedFrom26ToPrebidExt(w)

		if len(test.expectedErr) > 0 {
			assert.EqualError(t, err, test.expectedErr, test.description)
		} else {
			assert.NoError(t, w.RebuildImp(), test.description)
			assert.Equal(t, test.expectedImp, *w.Imp, test.description)
		}
	}
}
