package usersync

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEncoderDecoder(t *testing.T) {
	encoder := Base64Encoder{}
	decoder := Base64Decoder{}

	testCases := []struct {
		name            string
		givenRequest    *http.Request
		givenHttpCookie *http.Cookie
		givenCookie     *Cookie
		givenDecoder    Decoder
		expectedCookie  *Cookie
	}{
		{
			name: "simple-cookie",
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
		{
			name:        "empty-cookie",
			givenCookie: &Cookie{},
			expectedCookie: &Cookie{
				uids:   map[string]UIDEntry{},
				optOut: false,
			},
		},
		{
			name:        "nil-cookie",
			givenCookie: nil,
			expectedCookie: &Cookie{
				uids:   map[string]UIDEntry{},
				optOut: false,
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			encodedCookie, err := encoder.Encode(test.givenCookie)
			assert.NoError(t, err)
			decodedCookie := decoder.Decode(encodedCookie)

			assert.Equal(t, test.expectedCookie.uids, decodedCookie.uids)
			assert.Equal(t, test.expectedCookie.optOut, decodedCookie.optOut)
		})
	}
}

func TestEncoder(t *testing.T) {
	encoder := Base64Encoder{}

	testCases := []struct {
		name                  string
		givenCookie           *Cookie
		expectedEncodedCookie string
	}{
		{
			name: "simple-cookie",
			givenCookie: &Cookie{
				uids: map[string]UIDEntry{
					"adnxs": {
						UID:     "UID",
						Expires: time.Time{},
					},
				},
				optOut: false,
			},
			expectedEncodedCookie: "eyJ0ZW1wVUlEcyI6eyJhZG54cyI6eyJ1aWQiOiJVSUQiLCJleHBpcmVzIjoiMDAwMS0wMS0wMVQwMDowMDowMFoifX19",
		},
		{
			name:                  "empty-cookie",
			givenCookie:           &Cookie{},
			expectedEncodedCookie: "e30=",
		},
		{
			name:                  "nil-cookie",
			givenCookie:           nil,
			expectedEncodedCookie: "bnVsbA==",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			encodedCookie, err := encoder.Encode(test.givenCookie)
			assert.NoError(t, err)

			assert.Equal(t, test.expectedEncodedCookie, encodedCookie)
		})
	}
}

func TestDecoder(t *testing.T) {
	decoder := Base64Decoder{}

	testCases := []struct {
		name               string
		givenEncodedCookie string
		expectedCookie     *Cookie
	}{
		{
			name:               "simple-encoded-cookie",
			givenEncodedCookie: "eyJ0ZW1wVUlEcyI6eyJhZG54cyI6eyJ1aWQiOiJVSUQiLCJleHBpcmVzIjoiMDAwMS0wMS0wMVQwMDowMDowMFoifX19",
			expectedCookie: &Cookie{
				uids: map[string]UIDEntry{
					"adnxs": {
						UID: "UID",
					},
				},
				optOut: false,
			},
		},
		{
			name:               "nil-encoded-cookie",
			givenEncodedCookie: "",
			expectedCookie:     NewCookie(),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			decodedCookie := decoder.Decode(test.givenEncodedCookie)
			assert.Equal(t, test.expectedCookie, decodedCookie)
		})
	}
}
