package schain

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"
)

func TestSChainWriter(t *testing.T) {

	const seller1SChain string = `"schain":{"complete":1,"nodes":[{"asi":"directseller1.com","sid":"00001","rid":"BidRequest1","hp":1}],"ver":"1.0"}`
	const seller2SChain string = `"schain":{"complete":2,"nodes":[{"asi":"directseller2.com","sid":"00002","rid":"BidRequest2","hp":2}],"ver":"2.0"}`
	const seller3SChain string = `"schain":{"complete":3,"nodes":[{"asi":"directseller3.com","sid":"00003","rid":"BidRequest3","hp":3}],"ver":"3.0"}`
	const sellerWildCardSChain string = `"schain":{"complete":1,"nodes":[{"asi":"wildcard1.com","sid":"wildcard1","rid":"WildcardReq1","hp":1}],"ver":"1.0"}`
	const seller1Node string = `{"asi":"directseller1.com","sid":"00001","rid":"BidRequest1","hp":1}`

	tests := []struct {
		description    string
		giveRequest    *openrtb_ext.RequestWrapper
		giveBidder     string
		giveHostSChain *openrtb2.SupplyChainNode
		wantRequest    *openrtb_ext.RequestWrapper
		wantError      bool
	}{
		{
			description: "nil source, nil ext.prebid.schains and empty host schain",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext:    nil,
					Source: nil,
				},
			},

			giveBidder:     "appnexus",
			giveHostSChain: nil,
			wantRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext:    nil,
					Source: nil,
				},
			},
		},
		{
			description: "Use source schain -- no bidder schain or wildcard schain in nil ext.prebid.schains - so source.schain is set and unmodified",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{}`),
					Source: &openrtb2.Source{
						SChain: &openrtb2.SupplyChain{
							Ver: "1.1",
						},
					},
				},
			},
			giveBidder: "appnexus",
			wantRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{}`),
					Source: &openrtb2.Source{
						SChain: &openrtb2.SupplyChain{
							Ver: "1.1",
						},
					},
				},
			},
		},
		{
			description: "Use source schain -- no bidder schain or wildcard schain in not nil ext.prebid.schains - so source.schain is set and unmodified",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{"prebid":{"schains":[{"bidders":["appnexus"],` + seller1SChain + `}]}}`),
					Source: &openrtb2.Source{
						SChain: &openrtb2.SupplyChain{
							Ver: "1.1",
						},
						Ext: json.RawMessage(`{"some":"data"}`),
					},
				},
			},
			giveBidder: "rubicon",
			wantRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{"prebid":{"schains":[{"bidders":["appnexus"],` + seller1SChain + `}]}}`),
					Source: &openrtb2.Source{
						SChain: &openrtb2.SupplyChain{
							Ver: "1.1",
						},
						Ext: json.RawMessage(`{"some":"data"}`),
					},
				},
			},
		},
		{
			description: "Use schain for bidder in ext.prebid.schains; ensure other source field values are retained.",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{"prebid":{"schains":[{"bidders":["appnexus"],` + seller1SChain + `}]}}`),
					Source: &openrtb2.Source{
						FD:     openrtb2.Int8Ptr(1),
						TID:    "tid data",
						PChain: "pchain data",
						Ext:    json.RawMessage(`{"some":"data"}`),
					},
				},
			},
			giveBidder: "appnexus",
			wantRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{"prebid":{"schains":[{"bidders":["appnexus"],` + seller1SChain + `}]}}`),
					Source: &openrtb2.Source{
						FD:     openrtb2.Int8Ptr(1),
						TID:    "tid data",
						PChain: "pchain data",
						SChain: &openrtb2.SupplyChain{
							Complete: 1,
							Ver:      "1.0",
							Ext:      nil,
							Nodes: []openrtb2.SupplyChainNode{
								{
									ASI: "directseller1.com",
									SID: "00001",
									RID: "BidRequest1",
									HP:  openrtb2.Int8Ptr(1),
									Ext: nil,
								},
							},
						},
						Ext: json.RawMessage(`{"some":"data"}`),
					},
				},
			},
		},
		{
			description: "Use schain for bidder in ext.prebid.schains, nil req.source ",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext:    json.RawMessage(`{"prebid":{"schains":[{"bidders":["appnexus"],` + seller1SChain + `}]}}`),
					Source: nil,
				},
			},
			giveBidder: "appnexus",
			wantRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{"prebid":{"schains":[{"bidders":["appnexus"],` + seller1SChain + `}]}}`),
					Source: &openrtb2.Source{
						SChain: &openrtb2.SupplyChain{
							Complete: 1,
							Ver:      "1.0",
							Ext:      nil,
							Nodes: []openrtb2.SupplyChainNode{
								{
									ASI: "directseller1.com",
									SID: "00001",
									RID: "BidRequest1",
									HP:  openrtb2.Int8Ptr(1),
									Ext: nil,
								},
							},
						},
						Ext: nil,
					},
				},
			},
		},
		{
			description: "Use wildcard schain in ext.prebid.schains.",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{"prebid":{"schains":[{"bidders":["*"],` + sellerWildCardSChain + `}]}}`),
					Source: &openrtb2.Source{
						Ext: nil,
					},
				},
			},
			giveBidder: "appnexus",
			wantRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{"prebid":{"schains":[{"bidders":["*"],` + sellerWildCardSChain + `}]}}`),
					Source: &openrtb2.Source{
						SChain: &openrtb2.SupplyChain{
							Complete: 1,
							Ver:      "1.0",
							Ext:      nil,
							Nodes: []openrtb2.SupplyChainNode{
								{
									ASI: "wildcard1.com",
									SID: "wildcard1",
									RID: "WildcardReq1",
									HP:  openrtb2.Int8Ptr(1),
									Ext: nil,
								},
							},
						},
						Ext: nil,
					},
				},
			},
		},
		{
			description: "Use schain for bidder in ext.prebid.schains instead of wildcard.",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{"prebid":{"schains":[{"bidders":["appnexus"],` + seller1SChain + `},{"bidders":["*"],` + sellerWildCardSChain + `}]}}`),
					Source: &openrtb2.Source{
						Ext: nil,
					},
				},
			},
			giveBidder: "appnexus",
			wantRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{"prebid":{"schains":[{"bidders":["appnexus"],` + seller1SChain + `},{"bidders":["*"],` + sellerWildCardSChain + `}]}}`),
					Source: &openrtb2.Source{
						SChain: &openrtb2.SupplyChain{
							Complete: 1,
							Ver:      "1.0",
							Ext:      nil,
							Nodes: []openrtb2.SupplyChainNode{
								{
									ASI: "directseller1.com",
									SID: "00001",
									RID: "BidRequest1",
									HP:  openrtb2.Int8Ptr(1),
									Ext: nil,
								},
							},
						},
						Ext: nil,
					},
				},
			},
		},
		{
			description: "Use source schain -- multiple (two) bidder schains in ext.prebid.schains.",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{"prebid":{"schains":[{"bidders":["appnexus"],` + seller1SChain + `},{"bidders":["appnexus"],` + seller2SChain + `}]}}`),
					Source: &openrtb2.Source{
						Ext: json.RawMessage(`{` + seller3SChain + `}`),
					},
				},
			},
			giveBidder: "appnexus",
			wantRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{"prebid":{"schains":[{"bidders":["appnexus"],` + seller1SChain + `},{"bidders":["appnexus"],` + seller2SChain + `}]}}`),
					Source: &openrtb2.Source{
						Ext: json.RawMessage(`{` + seller3SChain + `}`),
					},
				},
			},
			wantError: true,
		},
		{
			description: "Schain in request, host schain defined, source.ext for bidder request should update with appended host schain",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext:    json.RawMessage(`{"prebid":{"schains":[{"bidders":["testbidder"],"schain":{"complete":1,"nodes":[` + seller1Node + `],"ver":"1.0"}}]}}`),
					Source: nil,
				},
			},
			giveBidder: "testbidder",
			giveHostSChain: &openrtb2.SupplyChainNode{
				ASI: "pbshostcompany.com", SID: "00001", RID: "BidRequest", HP: openrtb2.Int8Ptr(1),
			},
			wantRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{"prebid":{"schains":[{"bidders":["testbidder"],"schain":{"complete":1,"nodes":[` + seller1Node + `],"ver":"1.0"}}]}}`),
					Source: &openrtb2.Source{
						SChain: &openrtb2.SupplyChain{
							Complete: 1,
							Ver:      "1.0",
							Ext:      nil,
							Nodes: []openrtb2.SupplyChainNode{
								{
									ASI: "directseller1.com",
									SID: "00001",
									RID: "BidRequest1",
									HP:  openrtb2.Int8Ptr(1),
									Ext: nil,
								},
								{
									ASI: "pbshostcompany.com",
									SID: "00001",
									RID: "BidRequest",
									HP:  openrtb2.Int8Ptr(1),
									Ext: nil,
								},
							},
						},
						Ext: nil,
					},
				},
			},
		},
		{
			description: "No Schain in request, host schain defined, source.ext for bidder request should have just the host schain",
			giveRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext:    nil,
					Source: nil,
				},
			},
			giveBidder: "testbidder",
			giveHostSChain: &openrtb2.SupplyChainNode{
				ASI: "pbshostcompany.com", SID: "00001", RID: "BidRequest", HP: openrtb2.Int8Ptr(1),
			},
			wantRequest: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: nil,
					Source: &openrtb2.Source{
						SChain: &openrtb2.SupplyChain{
							Ver: "1.0",
							Ext: nil,
							Nodes: []openrtb2.SupplyChainNode{
								{
									ASI: "pbshostcompany.com",
									SID: "00001",
									RID: "BidRequest",
									HP:  openrtb2.Int8Ptr(1),
									Ext: nil,
								},
							},
						},
						Ext: nil,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			// unmarshal ext to get schains object needed to initialize writer
			var reqExt *openrtb_ext.ExtRequest
			if tt.giveRequest.Ext != nil {
				reqExt = &openrtb_ext.ExtRequest{}
				err := jsonutil.UnmarshalValid(tt.giveRequest.Ext, reqExt)
				if err != nil {
					t.Error("Unable to unmarshal request.ext")
				}
			}

			writer, err := NewSChainWriter(reqExt, tt.giveHostSChain)

			if tt.wantError {
				assert.NotNil(t, err)
				assert.Nil(t, writer)
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, writer)

				writer.Write(tt.giveRequest, tt.giveBidder)

				assert.Equal(t, tt.wantRequest, tt.giveRequest, tt.description)
			}
		})
	}
}
