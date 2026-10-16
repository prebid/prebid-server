package tracing

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type unmarshalable struct{}

func (unmarshalable) MarshalJSON() ([]byte, error) {
	return nil, errors.New("boom")
}

func TestNewPacket(t *testing.T) {
	ts := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	testCases := []struct {
		description   string
		partnerID     string
		packetType    PacketType
		bidder        string
		value         any
		expectedJSON  string
		expectedError string
	}{
		{
			description:  "builds a packet with a bidder set",
			partnerID:    "p1",
			packetType:   PacketTypeOutgoingBidRequest,
			bidder:       "appnexus",
			value:        map[string]string{"id": "req-1"},
			expectedJSON: `{"partner_id":"p1","type":"outgoing_bid_request","timestamp":"2026-01-01T12:00:00Z","bidder":"appnexus","payload":{"id":"req-1"}}`,
		},
		{
			description:  "builds a packet without a bidder",
			partnerID:    "p1",
			packetType:   PacketTypeIncomingBidRequest,
			bidder:       "",
			value:        map[string]string{"id": "req-1"},
			expectedJSON: `{"partner_id":"p1","type":"incoming_bid_request","timestamp":"2026-01-01T12:00:00Z","payload":{"id":"req-1"}}`,
		},
		{
			description:   "returns an error if the value cannot be marshaled",
			partnerID:     "p1",
			packetType:    PacketTypeAuctionResponse,
			value:         unmarshalable{},
			expectedError: "boom",
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			// Act
			packet, err := newPacket(test.partnerID, test.packetType, test.bidder, ts, test.value)

			// Assert
			if test.expectedError != "" {
				assert.ErrorContains(t, err, test.expectedError)
				return
			}
			assert.NoError(t, err)

			var buf bytes.Buffer
			assert.NoError(t, emit(&buf, packet))
			assert.JSONEq(t, test.expectedJSON, buf.String())
		})
	}
}

func TestEmitWriteError(t *testing.T) {
	// Arrange
	packet, err := newPacket("p1", PacketTypeAuctionResponse, "", testTime(), map[string]string{"a": "b"})
	assert.NoError(t, err)

	// Act
	err = emit(failingWriter{}, packet)

	// Assert
	assert.ErrorContains(t, err, "write failed")
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func testTime() time.Time {
	return time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
}
