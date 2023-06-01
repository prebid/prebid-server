package usersync

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDecoder(t *testing.T) {
	encoder := EncoderV1{}
	decoder := DecodeV1{}

	testCases := []struct {
		name            string
		givenRequest    *http.Request
		givenHttpCookie *http.Cookie
		givenCookie     *Cookie
		givenDecoder    Decoder
		expectedCookie  *Cookie
	}{
		{
			name: "Basic Decoder",
			givenCookie: &Cookie{
				uids: map[string]UIDEntry{
					"adnxs": {
						UID:     "UID",
						Expires: time.Time{},
					},
				},
				optOut: false,
			},
			expectedCookie: &Cookie{
				uids: map[string]UIDEntry{
					"adnxs": {
						UID: "UID",
					},
				},
				optOut: false,
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			encodedCookie := encoder.Encode(test.givenCookie)
			decodedCookie := decoder.Decode(encodedCookie)

			assert.Equal(t, test.expectedCookie.uids, decodedCookie.uids)
			assert.Equal(t, test.expectedCookie.optOut, decodedCookie.optOut)
		})
	}
}
