package dsa

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/exchange/entities"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		giveRequest *openrtb_ext.RequestWrapper
		giveBid     *entities.PbsOrtbBid
		wantValid   bool
	}{
		{
			name: "not_required",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						Ext: json.RawMessage(`{"dsa": {"dsarequired": 0}}`),
					},
				},
			},
			giveBid:   nil,
			wantValid: true,
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
			wantValid: false,
		},
		{
			name: "required_and_bid.bid_is_nil",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						Ext: json.RawMessage(`{"dsa": {"dsarequired": 2}}`),
					},
				},
			},
			giveBid:   &entities.PbsOrtbBid{},
			wantValid: false,
		},
		{
			name: "required_and_bid.ext.dsa_not_present",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						Ext: json.RawMessage(`{"dsa": {"dsarequired": 2}}`),
					},
				},
			},
			giveBid: &entities.PbsOrtbBid{
				Bid: &openrtb2.Bid{
					Ext: json.RawMessage(`{}`),
				},
			},
			wantValid: false,
		},
		{
			name: "required_and_bid.ext.dsa_present",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						Ext: json.RawMessage(`{"dsa": {"dsarequired": 2}}`),
					},
				},
			},
			giveBid: &entities.PbsOrtbBid{
				Bid: &openrtb2.Bid{
					Ext: json.RawMessage(`{"dsa": {}}`),
				},
			},
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := Validate(tt.giveRequest, tt.giveBid)
			assert.Equal(t, tt.wantValid, valid)
		})
	}
}

func TestDSARequired(t *testing.T) {
	tests := []struct {
		name         string
		giveRequest  *openrtb_ext.RequestWrapper
		wantRequired bool
	}{
		{
			name: "not_required_and_reg.ext.dsa_is_nil",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						Ext: json.RawMessage(`{}`),
					},
				},
			},
			wantRequired: false,
		},
		{
			name: "not_required_and_reg.ext.dsa_is_empty",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						Ext: json.RawMessage(`{"dsa": {}}`),
					},
				},
			},
			wantRequired: false,
		},
		{
			name: "required_and_reg.ext.dsa_is_0",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						Ext: json.RawMessage(`{"dsa": {"dsarequired": 0}}`),
					},
				},
			},
			wantRequired: false,
		},
		{
			name: "required_and_reg.ext.dsa_is_1",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						Ext: json.RawMessage(`{"dsa": {"dsarequired": 1}}`),
					},
				},
			},
			wantRequired: false,
		},
		{
			name: "required_and_reg.ext.dsa_is_2",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						Ext: json.RawMessage(`{"dsa": {"dsarequired": 2}}`),
					},
				},
			},
			wantRequired: true,
		},
		{
			name: "required_and_reg.ext.dsa_is_3",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						Ext: json.RawMessage(`{"dsa": {"dsarequired": 3}}`),
					},
				},
			},
			wantRequired: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			required := dsaRequired(tt.giveRequest)
			assert.Equal(t, tt.wantRequired, required)
		})
	}
}
