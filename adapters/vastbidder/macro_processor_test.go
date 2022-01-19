package vastbidder

import (
	"encoding/json"
	"net/url"
	"testing"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/stretchr/testify/assert"
)

func getBidRequest(requestJSON string) *openrtb2.BidRequest {
	bidRequest := &openrtb2.BidRequest{}
	json.Unmarshal([]byte(requestJSON), bidRequest)
	return bidRequest
}
func TestMacroProcessor_ProcessString(t *testing.T) {
	testMacroValues := map[string]string{
		MacroPubID:                     `pubID`,
		MacroTagID:                     `tagid value`,
		MacroTagID + macroEscapeSuffix: `tagid+value`,
		MacroTagID + macroEscapeSuffix + macroEscapeSuffix: `tagid%2Bvalue`,
	}

	sampleBidRequest := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{
			{TagID: testMacroValues[MacroTagID]},
		},
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				ID: testMacroValues[MacroPubID],
			},
		},
	}

	type fields struct {
		bidRequest *openrtb2.BidRequest
	}
	tests := []struct {
		name     string
		in       string
		expected string
	}{
		{
			name:     "Empty Input",
			in:       "",
			expected: "",
		},
		{
			name:     "No Macro Replacement",
			in:       "Hello Test No Macro",
			expected: "Hello Test No Macro",
		},
		{
			name:     "Start Macro",
			in:       GetMacroKey(MacroTagID) + "HELLO",
			expected: testMacroValues[MacroTagID] + "HELLO",
		},
		{
			name:     "End Macro",
			in:       "HELLO" + GetMacroKey(MacroTagID),
			expected: "HELLO" + testMacroValues[MacroTagID],
		},
		{
			name:     "Start-End Macro",
			in:       GetMacroKey(MacroTagID) + "HELLO" + GetMacroKey(MacroTagID),
			expected: testMacroValues[MacroTagID] + "HELLO" + testMacroValues[MacroTagID],
		},
		{
			name:     "Half Start Macro",
			in:       macroPrefix + GetMacroKey(MacroTagID) + "HELLO",
			expected: macroPrefix + testMacroValues[MacroTagID] + "HELLO",
		},
		{
			name:     "Half End Macro",
			in:       "HELLO" + GetMacroKey(MacroTagID) + macroSuffix,
			expected: "HELLO" + testMacroValues[MacroTagID] + macroSuffix,
		},
		{
			name:     "Concatenated Macro",
			in:       GetMacroKey(MacroTagID) + GetMacroKey(MacroTagID) + "HELLO",
			expected: testMacroValues[MacroTagID] + testMacroValues[MacroTagID] + "HELLO",
		},
		{
			name:     "Incomplete Concatenation Macro",
			in:       GetMacroKey(MacroTagID) + macroSuffix + "LINKHELLO",
			expected: testMacroValues[MacroTagID] + macroSuffix + "LINKHELLO",
		},
		{
			name:     "Concatenation with Suffix Macro",
			in:       GetMacroKey(MacroTagID) + macroPrefix + GetMacroKey(MacroTagID) + "HELLO",
			expected: testMacroValues[MacroTagID] + macroPrefix + testMacroValues[MacroTagID] + "HELLO",
		},
		{
			name:     "Unknown Macro",
			in:       GetMacroKey(`UNKNOWN`) + `ABC`,
			expected: GetMacroKey(`UNKNOWN`) + `ABC`,
		},
		{
			name:     "Incomplete macro suffix",
			in:       "START" + macroSuffix,
			expected: "START" + macroSuffix,
		},
		{
			name:     "Incomplete Start and End",
			in:       string(macroPrefix[0]) + GetMacroKey(MacroTagID) + " Value " + GetMacroKey(MacroTagID) + string(macroSuffix[0]),
			expected: string(macroPrefix[0]) + testMacroValues[MacroTagID] + " Value " + testMacroValues[MacroTagID] + string(macroSuffix[0]),
		},
		{
			name:     "Special Character",
			in:       macroPrefix + MacroTagID + `\n` + macroSuffix + "Sample \"" + GetMacroKey(MacroTagID) + "\" Data",
			expected: macroPrefix + MacroTagID + `\n` + macroSuffix + "Sample \"" + testMacroValues[MacroTagID] + "\" Data",
		},
		{
			name:     "Empty Value",
			in:       GetMacroKey(MacroTimeout) + "Hello",
			expected: "Hello",
		},
		{
			name:     "EscapingMacræo",
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

			gotResponse := mp.ProcessString(tt.in)
			assert.Equal(t, tt.expected, gotResponse)
		})
	}
}

func TestMacroProcessor_processKey(t *testing.T) {
	testMacroValues := map[string]string{
		MacroPubID:                     `pub id`,
		MacroPubID + macroEscapeSuffix: `pub+id`,
		MacroTagID:                     `tagid value`,
		MacroTagID + macroEscapeSuffix: `tagid+value`,
		MacroTagID + macroEscapeSuffix + macroEscapeSuffix: `tagid%2Bvalue`,
	}

	sampleBidRequest := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{
			{TagID: testMacroValues[MacroTagID]},
		},
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				ID: testMacroValues[MacroPubID],
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

func TestMacroProcessor_processURLValues(t *testing.T) {
	testMacroValues := map[string]string{
		MacroPubID:                     `pub id`,
		MacroPubID + macroEscapeSuffix: `pub+id`,
		MacroTagID:                     `tagid value`,
		MacroTagID + macroEscapeSuffix: `tagid+value`,
		MacroTagID + macroEscapeSuffix + macroEscapeSuffix: `tagid%2Bvalue`,
	}

	sampleBidRequest := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{
			{TagID: testMacroValues[MacroTagID]},
		},
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				ID: testMacroValues[MacroPubID],
			},
		},
	}
	type args struct {
		values url.Values
		flags  Flags
	}
	tests := []struct {
		name string
		args args
		want url.Values
	}{
		{
			name: `AllEmptyParamsRemovedEmptyParams`,
			args: args{
				values: url.Values{
					`k1`: []string{GetMacroKey(MacroPubName)},
					`k2`: []string{GetMacroKey(MacroPubName)},
					`k3`: []string{GetMacroKey(MacroPubName)},
				},
				flags: Flags{
					RemoveEmptyParam: true,
				},
			},
			want: url.Values{},
		},
		{
			name: `AllEmptyParamsKeepEmptyParams`,
			args: args{
				values: url.Values{
					`k1`: []string{GetMacroKey(MacroPubName)},
					`k2`: []string{GetMacroKey(MacroPubName)},
					`k3`: []string{GetMacroKey(MacroPubName)},
				},
				flags: Flags{
					RemoveEmptyParam: false,
				},
			},
			want: url.Values{
				`k1`: []string{""},
				`k2`: []string{""},
				`k3`: []string{""},
			},
		},
		{
			name: `MixedParamsRemoveEmptyParams`,
			args: args{
				values: url.Values{
					`k1`: []string{GetMacroKey(MacroPubID)},
					`k2`: []string{GetMacroKey(MacroPubName)},
					`k3`: []string{GetMacroKey(MacroTagID)},
				},
				flags: Flags{
					RemoveEmptyParam: true,
				},
			},
			want: url.Values{
				`k1`: []string{testMacroValues[MacroPubID]},
				`k3`: []string{testMacroValues[MacroTagID]},
			},
		},
		{
			name: `MixedParamsKeepEmptyParams`,
			args: args{
				values: url.Values{
					`k1`: []string{GetMacroKey(MacroPubID)},
					`k2`: []string{GetMacroKey(MacroPubName)},
					`k3`: []string{GetMacroKey(MacroTagID)},
					`k4`: []string{`UNKNOWN`},
					`k5`: []string{GetMacroKey(`UNKNOWN`)},
				},
				flags: Flags{
					RemoveEmptyParam: false,
				},
			},
			want: url.Values{
				`k1`: []string{testMacroValues[MacroPubID]},
				`k2`: []string{""},
				`k3`: []string{testMacroValues[MacroTagID]},
				`k4`: []string{`UNKNOWN`},
				`k5`: []string{GetMacroKey(`UNKNOWN`)},
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

			actual := mp.processURLValues(tt.args.values, tt.args.flags)

			actualValues, _ := url.ParseQuery(actual)
			assert.Equal(t, tt.want, actualValues)
		})
	}
}

func TestMacroProcessor_processURLValuesEscapingKeys(t *testing.T) {
	testMacroImpValues := map[string]string{
		MacroPubID: `pub id`,
		MacroTagID: `tagid value`,
	}

	testMacroValues := map[string]string{
		MacroPubID:                     `pub+id`,
		MacroTagID:                     `tagid+value`,
		MacroTagID + macroEscapeSuffix: `tagid%2Bvalue`,
		MacroTagID + macroEscapeSuffix + macroEscapeSuffix: `tagid%252Bvalue`,
	}

	sampleBidRequest := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{
			{TagID: testMacroImpValues[MacroTagID]},
		},
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				ID: testMacroImpValues[MacroPubID],
			},
		},
	}
	type args struct {
		key   string
		value string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: `EmptyKeyValue`,
			args: args{},
			want: ``,
		},
		{
			name: `WithoutEscaping`,
			args: args{key: `k1`, value: GetMacroKey(MacroTagID)},
			want: `k1=` + testMacroValues[MacroTagID],
		},
		{
			name: `WithEscaping`,
			args: args{key: `k1`, value: GetMacroKey(MacroTagID + macroEscapeSuffix)},
			want: `k1=` + testMacroValues[MacroTagID+macroEscapeSuffix],
		},
		{
			name: `With2LevelEscaping`,
			args: args{key: `k1`, value: GetMacroKey(MacroTagID + macroEscapeSuffix + macroEscapeSuffix)},
			want: `k1=` + testMacroValues[MacroTagID+macroEscapeSuffix+macroEscapeSuffix],
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

			values := url.Values{}
			if len(tt.args.key) > 0 {
				values.Add(tt.args.key, tt.args.value)
			}

			actual := mp.processURLValues(values, Flags{})
			assert.Equal(t, tt.want, actual)
		})
	}
}

func TestMacroProcessor_ProcessURL(t *testing.T) {
	testMacroImpValues := map[string]string{
		MacroPubID:  `123`,
		MacroSiteID: `567`,
		MacroTagID:  `tagid value`,
	}

	sampleBidRequest := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{
			{TagID: testMacroImpValues[MacroTagID]},
		},
		Site: &openrtb2.Site{
			ID: testMacroImpValues[MacroSiteID],
			Publisher: &openrtb2.Publisher{
				ID: testMacroImpValues[MacroPubID],
			},
		},
	}

	type args struct {
		uri   string
		flags Flags
	}
	tests := []struct {
		name         string
		args         args
		wantResponse string
	}{
		{
			name: "EmptyURI",
			args: args{
				uri:   ``,
				flags: Flags{RemoveEmptyParam: true},
			},
			wantResponse: ``,
		},
		{
			name: "RemovedEmptyParams1",
			args: args{
				uri:   `http://xyz.domain.com/` + GetMacroKey(MacroPubID) + `/` + GetMacroKey(MacroSiteID) + `?tagID=` + GetMacroKey(MacroTagID) + `&notfound=` + GetMacroKey(MacroTimeout) + `&k1=v1&k2=v2`,
				flags: Flags{RemoveEmptyParam: true},
			},
			wantResponse: `http://xyz.domain.com/123/567?tagID=tagid+value&k1=v1&k2=v2`,
		},
		{
			name: "RemovedEmptyParams2",
			args: args{
				uri:   `http://xyz.domain.com/` + GetMacroKey(MacroPubID) + `/` + GetMacroKey(MacroSiteID) + `?tagID=` + GetMacroKey(MacroTagID+macroEscapeSuffix) + `&notfound=` + GetMacroKey(MacroTimeout) + `&k1=v1&k2=v2`,
				flags: Flags{RemoveEmptyParam: false},
			},
			wantResponse: `http://xyz.domain.com/123/567?tagID=tagid+value&notfound=&k1=v1&k2=v2`,
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

			gotResponse := mp.ProcessURL(tt.args.uri, tt.args.flags)
			assertURL(t, tt.wantResponse, gotResponse)
		})
	}
}

func assertURL(t *testing.T, expected, actual string) {
	actualURL, _ := url.Parse(actual)
	expectedURL, _ := url.Parse(expected)

	if nil == actualURL || nil == expectedURL {
		assert.True(t, (nil == actualURL) == (nil == expectedURL), `actual or expected url parsing failed`)
	} else {
		assert.Equal(t, expectedURL.Scheme, actualURL.Scheme)
		assert.Equal(t, expectedURL.Opaque, actualURL.Opaque)
		assert.Equal(t, expectedURL.User, actualURL.User)
		assert.Equal(t, expectedURL.Host, actualURL.Host)
		assert.Equal(t, expectedURL.Path, actualURL.Path)
		assert.Equal(t, expectedURL.RawPath, actualURL.RawPath)
		assert.Equal(t, expectedURL.ForceQuery, actualURL.ForceQuery)
		assert.Equal(t, expectedURL.Query(), actualURL.Query())
		assert.Equal(t, expectedURL.Fragment, actualURL.Fragment)
	}
}
