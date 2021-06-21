package info

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/stretchr/testify/assert"
)

func TestPrepareBiddersResponse(t *testing.T) {
	testCases := []struct {
		description  string
		givenBidders config.BidderInfos
		givenAliases map[string]string
		expected     string
	}{
		{
			description:  "None",
			givenBidders: config.BidderInfos{},
			givenAliases: nil,
			expected:     `[]`,
		},
		{
			description:  "Core Bidders Only - One",
			givenBidders: config.BidderInfos{"a": {}},
			givenAliases: nil,
			expected:     `["a"]`,
		},
		{
			description:  "Core Bidders Only - Many",
			givenBidders: config.BidderInfos{"a": {}, "b": {}},
			givenAliases: nil,
			expected:     `["a","b"]`,
		},
		{
			description:  "Core Bidders Only - Many Sorted",
			givenBidders: config.BidderInfos{"z": {}, "a": {}},
			givenAliases: nil,
			expected:     `["a","z"]`,
		},
		{
			description:  "With Aliases - One",
			givenBidders: config.BidderInfos{"a": {}},
			givenAliases: map[string]string{"b": "b"},
			expected:     `["a","b"]`,
		},
		{
			description:  "With Aliases - Many",
			givenBidders: config.BidderInfos{"a": {}},
			givenAliases: map[string]string{"b": "b", "c": "c"},
			expected:     `["a","b","c"]`,
		},
		{
			description:  "With Aliases - Sorted",
			givenBidders: config.BidderInfos{"z": {}},
			givenAliases: map[string]string{"a": "a"},
			expected:     `["a","z"]`,
		},
	}

	for _, test := range testCases {
		result, err := prepareBiddersResponse(test.givenBidders, test.givenAliases)

		assert.NoError(t, err, test.description)
		assert.Equal(t, []byte(test.expected), result, test.description)
	}
}

func TestBiddersHandler(t *testing.T) {
	bidders := config.BidderInfos{"a": {}}
	aliases := map[string]string{"b": "b"}

	handler := NewBiddersEndpoint(bidders, aliases)

	responseRecorder := httptest.NewRecorder()
	handler(responseRecorder, nil, nil)

	result := responseRecorder.Result()
	assert.Equal(t, result.StatusCode, http.StatusOK)

	resultBody, _ := ioutil.ReadAll(result.Body)
	assert.Equal(t, []byte(`["a","b"]`), resultBody)

	resultHeaders := result.Header
	assert.Equal(t, http.Header{"Content-Type": []string{"application/json"}}, resultHeaders)
}
