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
			name: "dsa not required",
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
			name: "dsa required and bid is nil",
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
			name: "dsa required and bid.bid is nil",
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
			name: "dsa required and bid.ext.dsa not present",
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
			name: "dsa required and bid.ext.dsa present",
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
			name: "dsa not required, reg.ext.dsa is nil",
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
			name: "dsa not required, reg.ext.dsa is empty",
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
			name: "dsa required, reg.ext.dsa is 0",
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
			name: "dsa required, reg.ext.dsa is 1",
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
			name: "dsa required, reg.ext.dsa is 2",
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
			name: "dsa required, reg.ext.dsa is 3",
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
