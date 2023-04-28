package macros

import (
	"testing"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/exchange/entities"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestGetMacro(t *testing.T) {
	type args struct {
		key            string
		reqWrapper     *openrtb_ext.RequestWrapper
		seat           string
		vastEvent      string
		eventType      string
		vastCreativeID string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "get PBS-AUCTIONID key",
			args: args{
				key: MacroKeyAuctionID,
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID: "123",
					},
				},
			},
			want: "123",
		},
		{
			name: "get PBS-APPBUNDLE key",
			args: args{
				key: MacroKeyAppBundle,
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						App: &openrtb2.App{
							Bundle: "testbundle",
						},
					},
				},
			},
			want: "testbundle",
		},
		{
			name: "get PBS-DOMAIN key",
			args: args{
				key: MacroKeyDomain,
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						App: &openrtb2.App{
							Domain: "testDomain",
						},
					},
				},
			},
			want: "testDomain",
		},
		{
			name: "get PBS-PUBDOMAIN key",
			args: args{
				key: MacroKeyPubDomain,
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						App: &openrtb2.App{
							Publisher: &openrtb2.Publisher{
								Domain: "pubDomain",
							},
						},
					},
				},
			},
			want: "pubDomain",
		},
		{
			name: "get PBS-PAGEURL key",
			args: args{
				key: MacroKeyPageURL,
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						Site: &openrtb2.Site{
							Page: "pageurltest",
						},
					},
				},
			},
			want: "pageurltest",
		},
		{
			name: "get PBS-ACCOUNTID key",
			args: args{
				key: MacroKeyAccountID,
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						Site: &openrtb2.Site{
							Publisher: &openrtb2.Publisher{
								ID: "pubID",
							},
						},
					},
				},
			},
			want: "pubID",
		},
		{
			name: "get PBS-LIMITADTRACKING key",
			args: args{
				key: MacroKeyLmtTracking,
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						Device: &openrtb2.Device{
							Lmt: &lmt,
						},
					},
				},
			},
			want: "10",
		},
		{
			name: "get PBS-CONSENT key",
			args: args{
				key: MacroKeyConsent,
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						User: &openrtb2.User{Ext: []byte(`{"consent":"1" }`)},
					},
				},
			},
			want: "1",
		},
		{
			name: "get PBS-INTEGRATION key",
			args: args{
				key: MacroKeyIntegration,
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						Ext: []byte(`{"prebid":{"integration":"testIntegration","channel": {"name":"test1"},"macros":{"CUSTOMMACR1":"value1","CUSTOMMACR2":"value2","CUSTOMMACR3":"value3"}}}`),
					},
				},
			},
			want: "testIntegration",
		},
		{
			name: "get PBS-CHANNEL key",
			args: args{
				key: MacroKeyChannel,
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						Ext: []byte(`{"prebid":{"channel": {"name":"test1"}}}`),
					},
				},
			},
			want: "test1",
		},
		{
			name: "get PBS-EVENTTYPE key",
			args: args{
				key: MacroKeyEventType,
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{},
				},
				eventType: "win",
			},
			want: "win",
		},
		{
			name: "get PBS-ACCOUNTID key",
			args: args{
				key: MacroKeyVastCRTID,
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{},
				},
				vastCreativeID: "123",
			},
			want: "123",
		},
		{
			name: "get PBS-ACCOUNTID key",
			args: args{
				key: MacroKeyBidder,
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{},
				},
				seat: "pubmatic",
			},
			want: "pubmatic",
		},
		{
			name: "get PBS- key",
			args: args{
				key: MacroKeyVastEvent,
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{},
				},
				vastEvent: "firstQuartile",
			},
			want: "firstQuartile",
		},
		{
			name: "get PBS-MACRO-CUSTOMMACR3 key",
			args: args{
				key: "PBS-MACRO-CUSTOMMACR3",
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						Ext: []byte(`{"prebid":{"channel": {"name":"test1"},"macros":{"CUSTOMMACR3":"abcdefghijklmnopqrstuvwxyz01234567899876543210zyxwvutsrqponmlkjihgfedcbaabcdefghijklmnopqrstuvwxyz01234567899876543210zyxwvutsrqponmlkjihgfedcba"}}}`),
					},
				},
			},
			want: "abcdefghijklmnopqrstuvwxyz01234567899876543210zyxwvutsrqponmlkjihgfedcbaabcdefghijklmnopqrstuvwxyz01",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			macroProvider := NewProvider(tt.args.reqWrapper)
			macroProvider.PopulateBidMacros(&entities.PbsOrtbBid{Bid: bid}, tt.args.seat)
			macroProvider.PopulateEventMacros(tt.args.vastCreativeID, tt.args.eventType, tt.args.vastEvent)
			got := macroProvider.GetMacro(tt.args.key)
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}

func TestPopulateRequestMacros(t *testing.T) {
	type fields struct {
		macros map[string]string
	}
	type args struct {
		reqWrapper *openrtb_ext.RequestWrapper
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   map[string]string
	}{
		{
			name: "Populate Request Level macros",
			fields: fields{
				macros: map[string]string{},
			},
			args: args{
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID: "123",
						Site: &openrtb2.Site{
							Domain: "testdomain",
							Publisher: &openrtb2.Publisher{
								Domain: "publishertestdomain",
								ID:     "testpublisherID",
							},
							Page: "pageurltest",
						},
						App: &openrtb2.App{
							Domain: "testdomain",
							Bundle: "testbundle",
							Publisher: &openrtb2.Publisher{
								Domain: "publishertestdomain",
								ID:     "testpublisherID",
							},
						},
						Device: &openrtb2.Device{
							Lmt: &lmt,
						},
						User: &openrtb2.User{Ext: []byte(`{"consent":"yes" }`)},
						Ext:  []byte(`{"prebid":{"channel": {"name":"test1"},"macros":{"CUSTOMMACR1":"value1","CUSTOMMACR2":"value2","CUSTOMMACR3":"value3"}}}`),
					},
				},
			},
			want: map[string]string{"PBS-ACCOUNTID": "testpublisherID", "PBS-APPBUNDLE": "testbundle", "PBS-AUCTIONID": "123", "PBS-CHANNEL": "test1", "PBS-DOMAIN": "testdomain", "PBS-GDPRCONSENT": "yes", "PBS-LIMITADTRACKING": "10", "PBS-MACRO-CUSTOMMACR1": "value1", "PBS-MACRO-CUSTOMMACR2": "value2", "PBS-MACRO-CUSTOMMACR3": "value3", "PBS-PAGEURL": "pageurltest", "PBS-PUBDOMAIN": "publishertestdomain"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &macroProvider{
				macros: tt.fields.macros,
			}
			b.populateRequestMacros(tt.args.reqWrapper)
			delete(b.macros, MacroKeyTimestamp)
			assert.Equal(t, tt.want, b.macros, tt.name)
		})
	}
}

func TestPopulateBidMacros(t *testing.T) {

	type args struct {
		bid  *entities.PbsOrtbBid
		seat string
	}
	tests := []struct {
		name string
		args args
		key  string
		want string
	}{
		{
			name: "Populate seat name",
			args: args{
				seat: "testBidder",
				bid: &entities.PbsOrtbBid{
					GeneratedBidID: "bid123",
				},
			},
			key:  MacroKeyBidder,
			want: "testBidder",
		},
		{
			name: "Populate generatedbid id",
			args: args{
				seat: "testBidder",
				bid: &entities.PbsOrtbBid{
					GeneratedBidID: "bid123",
				},
			},
			key:  MacroKeyBidID,
			want: "bid123",
		},
		{
			name: "Populate bid id",
			args: args{
				seat: "testBidder",
				bid: &entities.PbsOrtbBid{
					Bid: &openrtb2.Bid{
						ID: "bid123",
					},
				},
			},
			key:  MacroKeyBidID,
			want: "bid123",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &macroProvider{
				macros: map[string]string{},
			}
			b.PopulateBidMacros(tt.args.bid, tt.args.seat)
			assert.Equal(t, tt.want, b.macros[tt.key], tt.name)
		})
	}
}

func TestPopulateEventMacros(t *testing.T) {

	type args struct {
		vastCreativeID string
		eventType      string
		vastEvent      string
	}
	tests := []struct {
		name string
		args args
		key  string
		want string
	}{
		{
			name: "Populate creative Id",
			args: args{
				vastCreativeID: "crtID123",
			},
			key:  MacroKeyVastCRTID,
			want: "crtID123",
		},
		{
			name: "Populate eventType",
			args: args{
				eventType: "win",
			},
			key:  MacroKeyEventType,
			want: "win",
		},
		{
			name: "Populate vastEvent",
			args: args{
				vastEvent: "firstQuartile",
			},
			key:  MacroKeyVastEvent,
			want: "firstQuartile",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &macroProvider{
				macros: map[string]string{},
			}
			b.PopulateEventMacros(tt.args.vastCreativeID, tt.args.eventType, tt.args.vastEvent)
			assert.Equal(t, tt.want, b.macros[tt.key], tt.name)
		})
	}
}
