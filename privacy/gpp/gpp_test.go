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
			desc: "nil_gppSID_array,_expect_false",
			in: testInput{
				gppSIDs: nil,
				sid:     gppConstants.SectionTCFEU2,
			},
			expected: false,
		},
		{
			desc: "empty_gppSID_array, expect_false",
			in: testInput{
				gppSIDs: []int8{},
				sid:     gppConstants.SectionTCFEU2,
			},
			expected: false,
		},
		{
			desc: "SID_not_found_in_gppSID_array,_expect_false",
			in: testInput{
				gppSIDs: []int8{int8(8), int8(9)},
				sid:     gppConstants.SectionTCFEU2,
			},
			expected: false,
		},
		{
			desc: "SID_found_in_gppSID_array,_expect_true",
			in: testInput{
				gppSIDs: []int8{int8(2)},
				sid:     gppConstants.SectionTCFEU2,
			},
			expected: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) { assert.Equal(t, tc.expected, IsSIDInList(tc.in.gppSIDs, tc.in.sid)) })
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
			desc: "Empty_SectionTypes_array,_expect_-1_out",
			in: testInput{
				gpp: gpplib.GppContainer{},
				sid: gppConstants.SectionTCFEU2,
			},
			expected: -1,
		},
		{
			desc: "SID_not_found_in_SectionTypes_array,_expect_-1_out",
			in: testInput{
				gpp: gpplib.GppContainer{
					Version:      1,
					SectionTypes: []gppConstants.SectionID{gppConstants.SectionUSPV1},
				},
				sid: gppConstants.SectionTCFEU2,
			},
			expected: -1,
		},
		{
			desc: "SID_matches_an_element_in_SectionTypes_array,_expect_index_1_out",
			in: testInput{
				gpp: gpplib.GppContainer{
					Version: 1,
					SectionTypes: []gppConstants.SectionID{
						gppConstants.SectionUSPV1,
						gppConstants.SectionTCFEU2,
						gppConstants.SectionUSPCA,
					},
				},
				sid: gppConstants.SectionTCFEU2,
			},
			expected: 1,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) { assert.Equal(t, tc.expected, IndexOfSID(tc.in.gpp, tc.in.sid)) })
	}
}
