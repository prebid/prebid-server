package openrtb_ext

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/stretchr/testify/assert"
)

func TestConvertUpTo26(t *testing.T) {
	expectedRequest := openrtb2.BidRequest{
		ID:  "anyID",
		Ext: json.RawMessage(`{"other":"leftAloneExt"}`),
		Source: &openrtb2.Source{
			SChain: &openrtb2.SupplyChain{
				Complete: 1,
				Nodes:    []openrtb2.SupplyChainNode{},
				Ver:      "2",
			},
			Ext: json.RawMessage(`{"other":"leftAloneSource"}`),
		},
		Regs: &openrtb2.Regs{
			GDPR:      openrtb2.Int8Ptr(1),
			USPrivacy: "3",
			Ext:       json.RawMessage(`{"other":"leftAloneRegs"}`),
		},
		User: &openrtb2.User{
			Consent: "1",
			EIDs:    []openrtb2.EID{{Source: "42"}},
			Ext:     json.RawMessage(`{"other":"leftAloneUser"}`),
		},
	}

	testCases := []struct {
		description     string
		givenRequest    openrtb2.BidRequest
		expectedRequest openrtb2.BidRequest
		expectedErr     error
	}{
		{
			description: "Malformed",
			givenRequest: openrtb2.BidRequest{
				Ext: json.RawMessage(`malformed`),
			},
			expectedErr: errors.New("req.ext is invalid: invalid character 'm' looking for beginning of value"),
		},
		{
			description: "2.4 -> 2.6",
			givenRequest: openrtb2.BidRequest{
				ID:     "anyID",
				Ext:    json.RawMessage(`{"schain":{"complete":1,"nodes":[],"ver":"2"},"other":"leftAloneExt"}`),
				Source: &openrtb2.Source{Ext: json.RawMessage(`{"other":"leftAloneSource"}`)},
				Regs:   &openrtb2.Regs{Ext: json.RawMessage(`{"gdpr":1,"us_privacy":"3","other":"leftAloneRegs"}`)},
				User:   &openrtb2.User{Ext: json.RawMessage(`{"consent":"1","eids":[{"source":"42"}],"other":"leftAloneUser"}`)},
			},
			expectedErr: nil,
		},
		{
			description: "2.5 -> 2.6",
			givenRequest: openrtb2.BidRequest{
				ID:     "anyID",
				Ext:    json.RawMessage(`{"other":"leftAloneExt"}`),
				Source: &openrtb2.Source{Ext: json.RawMessage(`{"schain":{"complete":1,"nodes":[],"ver":"2"},"other":"leftAloneSource"}`)},
				Regs:   &openrtb2.Regs{Ext: json.RawMessage(`{"gdpr":1,"us_privacy":"3","other":"leftAloneRegs"}`)},
				User:   &openrtb2.User{Ext: json.RawMessage(`{"consent":"1","eids":[{"source":"42"}],"other":"leftAloneUser"}`)},
			},
			expectedErr: nil,
		},
	}

	for _, test := range testCases {
		w := &RequestWrapper{BidRequest: &test.givenRequest}
		err := ConvertUpTo26(w)
		assert.Equal(t, err, test.expectedErr, test.description)
		if test.expectedErr == nil {
			assert.NoError(t, w.RebuildRequest())
			assert.Equal(t, expectedRequest, *w.BidRequest)
		}
	}
}

func TestConvertUpEnsureExt(t *testing.T) {
	testCases := []struct {
		description  string
		givenRequest openrtb2.BidRequest
		expectedErr  error
	}{
		{
			description:  "Empty",
			givenRequest: openrtb2.BidRequest{},
			expectedErr:  nil,
		},
		{
			description:  "Ext",
			givenRequest: openrtb2.BidRequest{Ext: json.RawMessage("malformed")},
			expectedErr:  errors.New("req.ext is invalid: invalid character 'm' looking for beginning of value"),
		},
		{
			description:  "Source.Ext",
			givenRequest: openrtb2.BidRequest{Source: &openrtb2.Source{Ext: json.RawMessage("malformed")}},
			expectedErr:  errors.New("req.source.ext is invalid: invalid character 'm' looking for beginning of value"),
		},
		{
			description:  "Regs.Ext",
			givenRequest: openrtb2.BidRequest{Regs: &openrtb2.Regs{Ext: json.RawMessage("malformed")}},
			expectedErr:  errors.New("req.regs.ext is invalid: invalid character 'm' looking for beginning of value"),
		},
		{
			description:  "User.Ext",
			givenRequest: openrtb2.BidRequest{User: &openrtb2.User{Ext: json.RawMessage("malformed")}},
			expectedErr:  errors.New("req.user.ext is invalid: invalid character 'm' looking for beginning of value"),
		},
	}

	for _, test := range testCases {
		w := &RequestWrapper{BidRequest: &test.givenRequest}
		err := convertUpEnsureExt(w)
		assert.Equal(t, err, test.expectedErr, test.description)
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
			description:     "2.4 Migrated To 2.5 - Source Doesn't Exit",
			givenRequest:    openrtb2.BidRequest{Ext: schain1},
			expectedRequest: openrtb2.BidRequest{Source: &openrtb2.Source{Ext: schain1}},
		},
		{
			description:     "2.4 Migrated To 2.5 - Source Exits",
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
		assert.NoError(t, w.RebuildRequest())
		assert.Equal(t, test.expectedRequest, *w.BidRequest)
	}
}

func TestConvertSupplyChainFrom25To26(t *testing.T) {
	var (
		schain1     = &openrtb2.SupplyChain{Complete: 1, Nodes: []openrtb2.SupplyChainNode{}, Ver: "1"}
		schain1Json = json.RawMessage(`{"schain":{"complete":1,"nodes":[],"ver":"1"}}`)
		schain2Json = json.RawMessage(`{"schain":{"complete":1,"nodes":[],"ver":"2"}}`)
	)

	// schain 1, schain 2 constants
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
		assert.NoError(t, w.RebuildRequest())
		assert.Equal(t, test.expectedRequest, *w.BidRequest)
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
		assert.NoError(t, w.RebuildRequest())
		assert.Equal(t, test.expectedRequest, *w.BidRequest)
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
		assert.NoError(t, w.RebuildRequest())
		assert.Equal(t, test.expectedRequest, *w.BidRequest)
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
		assert.NoError(t, w.RebuildRequest())
		assert.Equal(t, test.expectedRequest, *w.BidRequest)
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
		assert.NoError(t, w.RebuildRequest())
		assert.Equal(t, test.expectedRequest, *w.BidRequest)
	}
}
