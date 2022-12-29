package openrtb_ext

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAlternateBidderCodes_IsValidBidderCode(t *testing.T) {
	type fields struct {
		Enabled bool
		Bidders map[string]ExtAdapterAlternateBidderCodes
	}
	type args struct {
		bidder          string
		alternateBidder string
	}
	tests := []struct {
		name        string
		fields      fields
		args        args
		wantIsValid bool
		wantErr     error
	}{
		{
			name:        "alternateBidder is not set/blank (default non-extra bid case)",
			wantIsValid: true,
		},
		{
			name: "alternateBidder and bidder are same (default non-extra bid case with seat's alternateBidder explicitly set)",
			args: args{
				bidder:          "pubmatic",
				alternateBidder: "pubmatic",
			},
			wantIsValid: true,
		},
		{
			name: "account.alternatebiddercodes config not defined (default, reject bid)",
			args: args{
				bidder:          "pubmatic",
				alternateBidder: "groupm",
			},
			wantIsValid: false,
			wantErr:     errors.New(`alternateBidderCodes disabled for "pubmatic", rejecting bids for "groupm"`),
		},
		{
			name: "account.alternatebiddercodes config enabled but adapter config not defined",
			args: args{
				bidder:          "pubmatic",
				alternateBidder: "groupm",
			},
			fields:      fields{Enabled: true},
			wantIsValid: false,
			wantErr:     errors.New(`alternateBidderCodes not defined for adapter "pubmatic", rejecting bids for "groupm"`),
		},
		{
			name: "account.alternatebiddercodes config enabled but adapter config is not available",
			args: args{
				bidder:          "pubmatic",
				alternateBidder: "groupm",
			},
			fields: fields{
				Enabled: true,
				Bidders: map[string]ExtAdapterAlternateBidderCodes{
					"appnexus": {},
				},
			},
			wantIsValid: false,
			wantErr:     errors.New(`alternateBidderCodes not defined for adapter "pubmatic", rejecting bids for "groupm"`),
		},
		{
			name: "account.alternatebiddercodes config enabled but adapter config is disabled",
			args: args{
				bidder:          "pubmatic",
				alternateBidder: "groupm",
			},
			fields: fields{
				Enabled: true,
				Bidders: map[string]ExtAdapterAlternateBidderCodes{
					"pubmatic": {Enabled: false},
				},
			},
			wantIsValid: false,
			wantErr:     errors.New(`alternateBidderCodes disabled for "pubmatic", rejecting bids for "groupm"`),
		},
		{
			name: "account.alternatebiddercodes and adapter config enabled but adapter config does not have allowedBidderCodes defined",
			args: args{
				bidder:          "pubmatic",
				alternateBidder: "groupm",
			},
			fields: fields{
				Enabled: true,
				Bidders: map[string]ExtAdapterAlternateBidderCodes{
					"pubmatic": {Enabled: true},
				},
			},
			wantIsValid: true,
		},
		{
			name: "allowedBidderCodes is *",
			args: args{
				bidder:          "pubmatic",
				alternateBidder: "groupm",
			},
			fields: fields{
				Enabled: true,
				Bidders: map[string]ExtAdapterAlternateBidderCodes{
					"pubmatic": {
						Enabled:            true,
						AllowedBidderCodes: []string{"*"},
					},
				},
			},
			wantIsValid: true,
		},
		{
			name: "allowedBidderCodes is in the list",
			args: args{
				bidder:          "pubmatic",
				alternateBidder: "groupm",
			},
			fields: fields{
				Enabled: true,
				Bidders: map[string]ExtAdapterAlternateBidderCodes{
					"pubmatic": {
						Enabled:            true,
						AllowedBidderCodes: []string{"groupm"},
					},
				},
			},
			wantIsValid: true,
		},
		{
			name: "allowedBidderCodes is not in the list",
			args: args{
				bidder:          "pubmatic",
				alternateBidder: "groupm",
			},
			fields: fields{
				Enabled: true,
				Bidders: map[string]ExtAdapterAlternateBidderCodes{
					"pubmatic": {
						Enabled:            true,
						AllowedBidderCodes: []string{"xyz"},
					},
				},
			},
			wantIsValid: false,
			wantErr:     errors.New(`invalid biddercode "groupm" sent by adapter "pubmatic"`),
		},
		{
			name: "account.alternatebiddercodes and adapter config enabled but adapter config has allowedBidderCodes list empty",
			args: args{
				bidder:          "pubmatic",
				alternateBidder: "groupm",
			},
			fields: fields{
				Enabled: true,
				Bidders: map[string]ExtAdapterAlternateBidderCodes{
					"pubmatic": {Enabled: true, AllowedBidderCodes: []string{}},
				},
			},
			wantIsValid: false,
			wantErr:     errors.New(`invalid biddercode "groupm" sent by adapter "pubmatic"`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &ExtAlternateBidderCodes{
				Enabled: tt.fields.Enabled,
				Bidders: tt.fields.Bidders,
			}
			gotIsValid, gotErr := a.IsValidBidderCode(tt.args.bidder, tt.args.alternateBidder)
			assert.Equal(t, tt.wantIsValid, gotIsValid)
			assert.Equal(t, tt.wantErr, gotErr)
		})
	}
}
