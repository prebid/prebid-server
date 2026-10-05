package tracing

import (
	"encoding/json"
	"io"
	"time"
)

// PacketType identifies which point of the auction lifecycle a Packet was collected at.
type PacketType string

const (
	PacketTypeIncomingBidRequest  PacketType = "incoming_bid_request"
	PacketTypeOutgoingBidRequest  PacketType = "outgoing_bid_request"
	PacketTypeIncomingBidResponse PacketType = "incoming_bid_response"
	PacketTypeAuctionResponse     PacketType = "auction_response"
)

// Packet is a single collected trace event.
type Packet struct {
	PartnerID string          `json:"partner_id"`
	Type      PacketType      `json:"type"`
	Timestamp time.Time       `json:"timestamp"`
	Bidder    string          `json:"bidder,omitempty"`
	Payload   json.RawMessage `json:"payload"`
}

// newPacket builds a Packet, marshaling value as its payload.
func newPacket(partnerID string, packetType PacketType, bidder string, timestamp time.Time, value any) (Packet, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return Packet{}, err
	}

	return Packet{
		PartnerID: partnerID,
		Type:      packetType,
		Timestamp: timestamp,
		Bidder:    bidder,
		Payload:   payload,
	}, nil
}

// emit writes packet to w as a single line of JSON.
func emit(w io.Writer, packet Packet) error {
	data, err := json.Marshal(packet)
	if err != nil {
		return err
	}

	data = append(data, '\n')
	_, err = w.Write(data)
	return err
}
