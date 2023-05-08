package gpp

import (
	"testing"

	gpplib "github.com/prebid/go-gpp"
	gppConstants "github.com/prebid/go-gpp/constants"
	"github.com/stretchr/testify/assert"
)

func TestIsSIDInList(t *testing.T) {
	type testInput struct {
		gppSIDs []int8
		sid     gppConstants.SectionID
	}
	testCases := []struct {
		desc     string
		in       testInput
		expected bool
	}{
		{
			desc: "nil gppSID array, expect false",
			in: testInput{
				gppSIDs: nil,
				sid:     gppConstants.SectionTCFEU2,
			},
			expected: false,
		},
		{
			desc: "empty gppSID array, expect false",
			in: testInput{
				gppSIDs: []int8{},
				sid:     gppConstants.SectionTCFEU2,
			},
			expected: false,
		},
		{
			desc: "SID not found in gppSID array, expect false",
			in: testInput{
				gppSIDs: []int8{int8(8), int8(9)},
				sid:     gppConstants.SectionTCFEU2,
			},
			expected: false,
		},
		{
			desc: "SID found in gppSID array, expect true",
			in: testInput{
				gppSIDs: []int8{int8(2)},
				sid:     gppConstants.SectionTCFEU2,
			},
			expected: true,
		},
	}
	for _, tc := range testCases {
		// run
		out := IsSIDInList(tc.in.gppSIDs, tc.in.sid)
		// assertions
		assert.Equal(t, tc.expected, out, tc.desc)
	}
}

func TestIndexOfSID(t *testing.T) {
	type testInput struct {
		gpp gpplib.GppContainer
		sid gppConstants.SectionID
	}
	testCases := []struct {
		desc     string
		in       testInput
		expected int
	}{
		{
			desc: "Empty SectionTypes array, expect -1 out",
			in: testInput{
				gpp: gpplib.GppContainer{},
				sid: gppConstants.SectionTCFEU2,
			},
			expected: -1,
		},
		{
			desc: "SID not found in SectionTypes array, expect -1 out",
			in: testInput{
				gpp: gpplib.GppContainer{Version: 1, SectionTypes: []gppConstants.SectionID{gppConstants.SectionUSPV1}},
				sid: gppConstants.SectionTCFEU2,
			},
			expected: -1,
		},
		{
			desc: "SID matches an element in SectionTypes array, expect index 1 out",
			in: testInput{
				gpp: gpplib.GppContainer{Version: 1, SectionTypes: []gppConstants.SectionID{gppConstants.SectionUSPV1, gppConstants.SectionTCFEU2}},
				sid: gppConstants.SectionTCFEU2,
			},
			expected: 1,
		},
	}
	for _, tc := range testCases {
		// run
		out := IndexOfSID(tc.in.gpp, tc.in.sid)
		// assertions
		assert.Equal(t, tc.expected, out, tc.desc)
	}
}
