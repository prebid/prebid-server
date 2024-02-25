package dsa

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/exchange/entities"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	validBehalf := strings.Repeat("a", 100)
	invalidBehalf := strings.Repeat("a", 101)
	validPaid := strings.Repeat("a", 100)
	invalidPaid := strings.Repeat("a", 101)

	tests := []struct {
		name        string
		giveRequest *openrtb_ext.RequestWrapper
		giveBid     *entities.PbsOrtbBid
		wantError   bool
	}{
		{
			name:        "nil",
			giveRequest: nil,
			giveBid:     nil,
			wantError:   false,
		},
		{
			name:        "request_nil_bid_not_nil",
			giveRequest: nil,
			giveBid: &entities.PbsOrtbBid{
				Bid: &openrtb2.Bid{
					Ext: json.RawMessage(`{"dsa":{"behalf":"` + validBehalf + `","paid":"` + validPaid + `","adrender":1}}`),
				},
			},
			wantError:   false,
		},
		{
			name: "not_required_and_bid_is_nil",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						Ext: json.RawMessage(`{"dsa": {"dsarequired": 0}}`),
					},
				},
			},
			giveBid:   nil,
			wantError: false,
		},
		{
			name: "not_required_and_bid_dsa_is_valid",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						Ext: json.RawMessage(`{"dsa": {"dsarequired": 0}}`),
					},
				},
			},
			giveBid: &entities.PbsOrtbBid{
				Bid: &openrtb2.Bid{
					Ext: json.RawMessage(`{"dsa":{"behalf":"` + validBehalf + `","paid":"` + validPaid + `","adrender":1}}`),
				},
			},
			wantError: false,
		},
		{
			name: "required_and_bid_is_nil",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						Ext: json.RawMessage(`{"dsa": {"dsarequired": 2}}`),
					},
				},
			},
			giveBid:   nil,
			wantError: true,
		},
		{
			name: "required_and_bid_dsa_has_invalid_behalf",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						Ext: json.RawMessage(`{"dsa": {"dsarequired": 2}}`),
					},
				},
			},
			giveBid: &entities.PbsOrtbBid{
				Bid: &openrtb2.Bid{
					Ext: json.RawMessage(`{"dsa":{"behalf":"` + invalidBehalf + `"}}`),
				},
			},
			wantError: true,
		},
		{
			name: "required_and_bid_dsa_has_invalid_paid",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						Ext: json.RawMessage(`{"dsa": {"dsarequired": 2}}`),
					},
				},
			},
			giveBid: &entities.PbsOrtbBid{
				Bid: &openrtb2.Bid{
					Ext: json.RawMessage(`{"dsa":{"paid":"` + invalidPaid + `"}}`),
				},
			},
			wantError: true,
		},
		{
			name: "required_and_neither_will_render",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						Ext: json.RawMessage(`{"dsa": {"dsarequired": 2,"pubrender":0}}`),
					},
				},
			},
			giveBid: &entities.PbsOrtbBid{
				Bid: &openrtb2.Bid{
					Ext: json.RawMessage(`{"dsa":{"adrender":0}}`),
				},
			},
			wantError: true,
		},
		{
			name: "required_and_both_will_render",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						Ext: json.RawMessage(`{"dsa": {"dsarequired": 2,"pubrender":2}}`),
					},
				},
			},
			giveBid: &entities.PbsOrtbBid{
				Bid: &openrtb2.Bid{
					Ext: json.RawMessage(`{"dsa":{"adrender":1}}`),
				},
			},
			wantError: true,
		},
		{
			name: "required_and_bid_dsa_is_valid",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						Ext: json.RawMessage(`{"dsa": {"dsarequired": 2}}`),
					},
				},
			},
			giveBid: &entities.PbsOrtbBid{
				Bid: &openrtb2.Bid{
					Ext: json.RawMessage(`{"dsa":{"behalf":"` + validBehalf + `","paid":"` + validPaid + `","adrender":1}}`),
				},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.giveRequest, tt.giveBid)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestDSARequired(t *testing.T) {
	tests := []struct {
		name         string
		giveReqDSA   *openrtb_ext.ExtRegsDSA
		wantRequired bool
	}{
		{
			name:         "nil",
			giveReqDSA:   nil,
			wantRequired: false,
		},
		{
			name: "not_required_0",
			giveReqDSA: &openrtb_ext.ExtRegsDSA{
				Required: 0,
			},
			wantRequired: false,
		},
		{
			name: "not_required_1",
			giveReqDSA: &openrtb_ext.ExtRegsDSA{
				Required: 0,
			},
			wantRequired: false,
		},
		{
			name: "required_2",
			giveReqDSA: &openrtb_ext.ExtRegsDSA{
				Required: 2,
			},
			wantRequired: true,
		},
		{
			name: "required_3",
			giveReqDSA: &openrtb_ext.ExtRegsDSA{
				Required: 3,
			},
			wantRequired: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			required := dsaRequired(tt.giveReqDSA)
			assert.Equal(t, tt.wantRequired, required)
		})
	}
}

func TestGetReqDSA(t *testing.T) {
	tests := []struct {
		name        string
		giveRequest *openrtb_ext.RequestWrapper
		expectedDSA *openrtb_ext.ExtRegsDSA
	}{
		{
			name:        "req_is_nil",
			giveRequest: nil,
			expectedDSA: nil,
		},
		{
			name: "bidrequest_is_nil",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: nil,
			},
			expectedDSA: nil,
		},
		{
			name: "req.regs_is_nil",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: nil,
				},
			},
			expectedDSA: nil,
		},
		{
			name: "req.regs.ext_is_nil",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						Ext: nil,
					},
				},
			},
			expectedDSA: nil,
		},
		{
			name: "req.regs.ext_is_empty",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						Ext: json.RawMessage(`{}`),
					},
				},
			},
			expectedDSA: nil,
		},
		{
			name: "req.regs.ext_dsa_is_populated",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						Ext: json.RawMessage(`{"dsa": {"dsarequired": 2}}`),
					},
				},
			},
			expectedDSA: &openrtb_ext.ExtRegsDSA{
				Required: 2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dsa := getReqDSA(tt.giveRequest)
			assert.Equal(t, tt.expectedDSA, dsa)
		})
	}
}

func TestGetBidDSA(t *testing.T) {
	tests := []struct {
		name        string
		bid         *entities.PbsOrtbBid
		expectedDSA *openrtb_ext.ExtBidDSA
	}{
		{
			name:        "bid_is_nil",
			bid:         nil,
			expectedDSA: nil,
		},
		{
			name: "bid.bid_is_nil",
			bid: &entities.PbsOrtbBid{
				Bid: nil,
			},
			expectedDSA: nil,
		},
		{
			name: "bid.bid.ext_is_nil",
			bid: &entities.PbsOrtbBid{
				Bid: &openrtb2.Bid{
					Ext: nil,
				},
			},
			expectedDSA: nil,
		},
		{
			name: "bid.bid.ext_is_empty",
			bid: &entities.PbsOrtbBid{
				Bid: &openrtb2.Bid{
					Ext: json.RawMessage(`{}`),
				},
			},
			expectedDSA: nil,
		},
		{
			name: "bid.bid.ext.dsa_is_populated",
			bid: &entities.PbsOrtbBid{
				Bid: &openrtb2.Bid{
					Ext: json.RawMessage(`{"dsa": {"behalf":"test1","paid":"test2","adrender":1}}`),
				},
			},
			expectedDSA: &openrtb_ext.ExtBidDSA{
				Behalf:   "test1",
				Paid:     "test2",
				AdRender: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dsa := getBidDSA(tt.bid)
			assert.Equal(t, tt.expectedDSA, dsa)
		})
	}
}
