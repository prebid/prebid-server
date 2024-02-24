package dsa

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestWrite(t *testing.T) {
	requestDSAJSON := json.RawMessage(`{"dsa":{"datatopub":1,"dsarequired":2,"pubrender":1,"transparency":[{"domain":"example1.com","dsaparams":[1,2,3]}]}}`)
	defaultDSAJSON := json.RawMessage(`{"dsa":{"datatopub":2,"dsarequired":3,"pubrender":2,"transparency":[{"domain":"example2.com","dsaparams":[4,5,6]}]}}`)
	defaultDSA := &openrtb_ext.ExtRegsDSA{
		DataToPub: 2,
		Required:  3,
		PubRender: 2,
		Transparency: []openrtb_ext.ExtBidDSATransparency{
			{
				Domain: "example2.com",
				Params: []int{4, 5, 6},
			},
		},
	}

	tests := []struct {
		name          string
		giveConfig    *config.AccountDSA
		giveGDPR      bool
		giveRequest   *openrtb_ext.RequestWrapper
		expectRequest *openrtb_ext.RequestWrapper
	}{
		{
			name: "request_nil",
		},
		{
			name:       "config_nil",
			giveConfig: nil,
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						Ext: nil,
					},
				},
			},
			expectRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						Ext: nil,
					},
				},
			},
		},
		{
			name: "config_default_nil",
			giveConfig: &config.AccountDSA{
				Default: nil,
			},
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						Ext: nil,
					},
				},
			},
			expectRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						Ext: nil,
					},
				},
			},
		},
		{
			name: "request_dsa_present",
			giveConfig: &config.AccountDSA{
				Default: defaultDSA,
			},
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						Ext: requestDSAJSON,
					},
				},
			},
			expectRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						Ext: requestDSAJSON,
					},
				},
			},
		},
		{
			name: "config_default_present_with_gdpr_only_set_and_gdpr_in_scope",
			giveConfig: &config.AccountDSA{
				Default:  defaultDSA,
				GDPROnly: true,
			},
			giveGDPR: true,
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{},
			},
			expectRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						Ext: defaultDSAJSON,
					},
				},
			},
		},
		{
			name: "config_default_present_with_gdpr_only_set_and_gdpr_not_in_scope",
			giveConfig: &config.AccountDSA{
				Default:  defaultDSA,
				GDPROnly: true,
			},
			giveGDPR: false,
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{},
			},
			expectRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{},
			},
		},
		{
			name: "config_default_present_with_gdpr_only_not_set_and_gdpr_in_scope",
			giveConfig: &config.AccountDSA{
				Default:  defaultDSA,
				GDPROnly: false,
			},
			giveGDPR: true,
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{},
			},
			expectRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						Ext: defaultDSAJSON,
					},
				},
			},
		},
		{
			name: "config_default_present_with_gdpr_only_not_set_and_gdpr_not_in_scope",
			giveConfig: &config.AccountDSA{
				Default:  defaultDSA,
				GDPROnly: false,
			},
			giveGDPR: false,
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{},
			},
			expectRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						Ext: defaultDSAJSON,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := DSAWriter{
				Config:      tt.giveConfig,
				GDPRInScope: tt.giveGDPR,
			}
			err := writer.Write(tt.giveRequest)

			if tt.giveRequest != nil {
				tt.giveRequest.RebuildRequest()
				assert.Equal(t, tt.expectRequest.BidRequest, tt.giveRequest.BidRequest)
			} else {
				assert.Nil(t, tt.giveRequest)
			}
			assert.Nil(t, err)
		})
	}
}
