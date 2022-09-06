package vastbidder

import (
	"testing"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/stretchr/testify/assert"
)

func TestMacroProcessor_Process(t *testing.T) {
	bidRequestValues := map[string]string{
		MacroPubID: `pubID`,
		MacroTagID: `tagid value`,
	}

	testMacroValues := map[string]string{
		MacroPubID:                     `pubID`,
		MacroTagID:                     `tagid+value`, //default escaping
		MacroTagID + macroEscapeSuffix: `tagid+value`, //single escaping explicitly
		MacroTagID + macroEscapeSuffix + macroEscapeSuffix: `tagid%2Bvalue`,
	}

	sampleBidRequest := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{
			{TagID: bidRequestValues[MacroTagID]},
		},
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				ID: bidRequestValues[MacroPubID],
			},
		},
	}

	tests := []struct {
		name     string
		in       string
		expected string
	}{
		{
			name:     "EmptyInput",
			in:       "",
			expected: "",
		},
		{
			name:     "NoMacroReplacement",
			in:       "Hello Test No Macro",
			expected: "Hello Test No Macro",
		},
		{
			name:     "StartMacro",
			in:       GetMacroKey(MacroTagID) + "HELLO",
			expected: testMacroValues[MacroTagID] + "HELLO",
		},
		{
			name:     "EndMacro",
			in:       "HELLO" + GetMacroKey(MacroTagID),
			expected: "HELLO" + testMacroValues[MacroTagID],
		},
		{
			name:     "StartEndMacro",
			in:       GetMacroKey(MacroTagID) + "HELLO" + GetMacroKey(MacroTagID),
			expected: testMacroValues[MacroTagID] + "HELLO" + testMacroValues[MacroTagID],
		},
		{
			name:     "HalfStartMacro",
			in:       macroPrefix + GetMacroKey(MacroTagID) + "HELLO",
			expected: macroPrefix + testMacroValues[MacroTagID] + "HELLO",
		},
		{
			name:     "HalfEndMacro",
			in:       "HELLO" + GetMacroKey(MacroTagID) + macroSuffix,
			expected: "HELLO" + testMacroValues[MacroTagID] + macroSuffix,
		},
		{
			name:     "ConcatenatedMacro",
			in:       GetMacroKey(MacroTagID) + GetMacroKey(MacroTagID) + "HELLO",
			expected: testMacroValues[MacroTagID] + testMacroValues[MacroTagID] + "HELLO",
		},
		{
			name:     "IncompleteConcatenationMacro",
			in:       GetMacroKey(MacroTagID) + macroSuffix + "LINKHELLO",
			expected: testMacroValues[MacroTagID] + macroSuffix + "LINKHELLO",
		},
		{
			name:     "ConcatenationWithSuffixMacro",
			in:       GetMacroKey(MacroTagID) + macroPrefix + GetMacroKey(MacroTagID) + "HELLO",
			expected: testMacroValues[MacroTagID] + macroPrefix + testMacroValues[MacroTagID] + "HELLO",
		},
		{
			name:     "UnknownMacro",
			in:       GetMacroKey(`UNKNOWN`) + `ABC`,
			expected: GetMacroKey(`UNKNOWN`) + `ABC`,
		},
		{
			name:     "IncompleteMacroSuffix",
			in:       "START" + macroSuffix,
			expected: "START" + macroSuffix,
		},
		{
			name:     "IncompleteStartAndEnd",
			in:       string(macroPrefix[0]) + GetMacroKey(MacroTagID) + " Value " + GetMacroKey(MacroTagID) + string(macroSuffix[0]),
			expected: string(macroPrefix[0]) + testMacroValues[MacroTagID] + " Value " + testMacroValues[MacroTagID] + string(macroSuffix[0]),
		},
		{
			name:     "SpecialCharacter",
			in:       macroPrefix + MacroTagID + `\n` + macroSuffix + "Sample \"" + GetMacroKey(MacroTagID) + "\" Data",
			expected: macroPrefix + MacroTagID + `\n` + macroSuffix + "Sample \"" + testMacroValues[MacroTagID] + "\" Data",
		},
		{
			name:     "EmptyValue",
			in:       GetMacroKey(MacroTimeout) + "Hello",
			expected: "Hello",
		},
		{
			name:     "EscapingMacro",
			in:       GetMacroKey(MacroTagID),
			expected: testMacroValues[MacroTagID],
		},
		{
			name:     "SingleEscapingMacro",
			in:       GetMacroKey(MacroTagID + macroEscapeSuffix),
			expected: testMacroValues[MacroTagID+macroEscapeSuffix],
		},
		{
			name:     "DoubleEscapingMacro",
			in:       GetMacroKey(MacroTagID + macroEscapeSuffix + macroEscapeSuffix),
			expected: testMacroValues[MacroTagID+macroEscapeSuffix+macroEscapeSuffix],
		},

		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bidderMacro := NewBidderMacro()
			mapper := GetDefaultMapper()
			mp := NewMacroProcessor(bidderMacro, mapper)

			//Init Bidder Macro
			bidderMacro.InitBidRequest(sampleBidRequest)
			bidderMacro.LoadImpression(&sampleBidRequest.Imp[0])

			gotResponse := mp.Process(tt.in)
			assert.Equal(t, tt.expected, gotResponse)
		})
	}
}

func TestMacroProcessor_processKey(t *testing.T) {
	bidRequestValues := map[string]string{
		MacroPubID: `1234`,
		MacroTagID: `tagid value`,
	}

	testMacroValues := map[string]string{
		MacroPubID:                     `1234`,
		MacroPubID + macroEscapeSuffix: `1234`,
		MacroTagID:                     `tagid+value`,
		MacroTagID + macroEscapeSuffix: `tagid+value`,
		MacroTagID + macroEscapeSuffix + macroEscapeSuffix: `tagid%2Bvalue`,
	}

	sampleBidRequest := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{
			{TagID: bidRequestValues[MacroTagID]},
		},
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				ID: bidRequestValues[MacroPubID],
			},
		},
	}
	type args struct {
		cache map[string]string
		key   string
	}
	type want struct {
		expected string
		ok       bool
		cache    map[string]string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: `emptyKey`,
			args: args{},
			want: want{
				expected: "",
				ok:       false,
				cache:    map[string]string{},
			},
		},
		{
			name: `cachedKeyFound`,
			args: args{
				cache: map[string]string{
					MacroPubID: testMacroValues[MacroPubID],
				},
				key: MacroPubID,
			},
			want: want{
				expected: testMacroValues[MacroPubID],
				ok:       true,
				cache: map[string]string{
					MacroPubID: testMacroValues[MacroPubID],
				},
			},
		},
		{
			name: `valueFound`,
			args: args{
				key: MacroTagID,
			},
			want: want{
				expected: testMacroValues[MacroTagID],
				ok:       true,
				cache:    map[string]string{},
			},
		},
		{
			name: `2TimesEscaping`,
			args: args{
				key: MacroTagID + macroEscapeSuffix + macroEscapeSuffix,
			},
			want: want{
				expected: testMacroValues[MacroTagID+macroEscapeSuffix+macroEscapeSuffix],
				ok:       true,
				cache:    map[string]string{},
			},
		},
		{
			name: `macroNotPresent`,
			args: args{
				key: `Unknown`,
			},
			want: want{
				expected: "",
				ok:       false,
				cache:    map[string]string{},
			},
		},
		{
			name: `macroNotPresentInEscaping`,
			args: args{
				key: `Unknown` + macroEscapeSuffix,
			},
			want: want{
				expected: "",
				ok:       false,
				cache:    map[string]string{},
			},
		},
		{
			name: `cachedKey`,
			args: args{
				key: MacroPubID,
			},
			want: want{
				expected: testMacroValues[MacroPubID],
				ok:       true,
				cache: map[string]string{
					MacroPubID: testMacroValues[MacroPubID],
				},
			},
		},
		{
			name: `cachedEscapingKey`,
			args: args{
				key: MacroPubID + macroEscapeSuffix,
			},
			want: want{
				expected: testMacroValues[MacroPubID+macroEscapeSuffix],
				ok:       true,
				cache: map[string]string{
					MacroPubID + macroEscapeSuffix: testMacroValues[MacroPubID+macroEscapeSuffix],
				},
			},
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bidderMacro := NewBidderMacro()
			mapper := GetDefaultMapper()
			mp := NewMacroProcessor(bidderMacro, mapper)

			//init bidder macro
			bidderMacro.InitBidRequest(sampleBidRequest)
			bidderMacro.LoadImpression(&sampleBidRequest.Imp[0])

			//init cache of macro processor
			if nil != tt.args.cache {
				mp.macroCache = tt.args.cache
			}

			actual, ok := mp.processKey(tt.args.key)
			assert.Equal(t, tt.want.expected, actual)
			assert.Equal(t, tt.want.ok, ok)
			assert.Equal(t, tt.want.cache, mp.macroCache)
		})
	}
}
