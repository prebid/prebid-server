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
			name: " Macro present, get PBS-APPBUNDLE key",
			args: args{
				key: MacroKeyAppBundle,
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID: "123",
						App: &openrtb2.App{
							Bundle: "test",
						},
					},
				},
			},
			want: "test",
		},
		{
			name: " Macro does present, get PBS-APPBUNDLE key",
			args: args{
				key: MacroKeyAppBundle,
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID: "123",
					},
				},
			},
			want: "",
		},
		{
			name: " Macro present, get PBS-PAGEURL key",
			args: args{
				key: MacroKeyAppBundle,
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID: "123",
						App: &openrtb2.App{
							Bundle: "test",
						},
					},
				},
			},
			want: "test",
		},
		{
			name: " Macro does present, get PBS-AUCTIONID key",
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
			name: " Macro not present, get PBS-AUCTIONID key",
			args: args{
				key: MacroKeyAuctionID,
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{},
				},
			},
			want: "",
		},
		{
			name: "Macro present, get PBS-DOMAIN key",
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
			name: "Macro present, get PBS-PUBDOMAIN key",
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
			name: "Macro present, get PBS-PAGEURL key",
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
			name: "Macro present, get PBS-ACCOUNTID key",
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
			name: "Macro present, get PBS-LIMITADTRACKING key",
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
			name: "Macro present, get PBS-CONSENT key",
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
			name: "Macro present, get PBS-INTEGRATION key",
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
			name: "Macro not present, get PBS-INTEGRATION key",
			args: args{
				key: MacroKeyIntegration,
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						Ext: []byte(`{"prebid":{"integration":"","channel": {"name":"test1"},"macros":{"CUSTOMMACR1":"value1","CUSTOMMACR2":"value2","CUSTOMMACR3":"value3"}}}`),
					},
				},
			},
			want: "",
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
			name: "Macro present, get PBS-EVENTTYPE key",
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
			name: "Macro does not exist, get PBS-EVENTTYPE key",
			args: args{
				key: MacroKeyEventType,
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{},
				},
				eventType: "",
			},
			want: "",
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
			name: "Macro Present, get PBS-BIDDER key",
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
			name: "Macro not present, get PBS-BIDDER key",
			args: args{
				key: MacroKeyBidder,
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{},
				},
				seat: "",
			},
			want: "",
		},
		{
			name: "Macro present, get PBS-VASTEVENT key",
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
			name: "Macro not present, get PBS-VASTEVENT key",
			args: args{
				key: MacroKeyVastEvent,
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{},
				},
				vastEvent: "",
			},
			want: "",
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
		{
			name: "Invalid Macro key",
			args: args{
				key: "PBS-NOEXISTENTKEY",
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						Ext: []byte(`{"prebid":{"channel": {"name":"test1"},"macros":{"CUSTOMMACR3":"a"}}}`),
					},
				},
			},
			want: "",
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
	type args struct {
		reqWrapper *openrtb_ext.RequestWrapper
	}
	tests := []struct {
		name string
		key  string
		args args
		want string
	}{
		{
			name: "populate PBS-AUCTIONID key",
			key:  MacroKeyAuctionID,
			args: args{
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID: "123",
					},
				},
			},
			want: "123",
		},
		{
			name: "populate PBS-APPBUNDLE key",
			key:  MacroKeyAppBundle,
			args: args{
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
			name: "populate App PBS-DOMAIN key",
			key:  MacroKeyDomain,
			args: args{
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
			name: "populate Site PBS-DOMAIN key",
			key:  MacroKeyDomain,
			args: args{
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						Site: &openrtb2.Site{
							Domain: "testDomain",
						},
					},
				},
			},
			want: "testDomain",
		},
		{
			name: "populate App PBS-PUBDOMAIN key",
			key:  MacroKeyPubDomain,
			args: args{
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
			name: "populate Site PBS-PUBDOMAIN key",
			key:  MacroKeyPubDomain,
			args: args{
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						Site: &openrtb2.Site{
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
			name: "populate PBS-PAGEURL key",
			key:  MacroKeyPageURL,
			args: args{
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
			name: "populate App PBS-ACCOUNTID key",
			key:  MacroKeyAccountID,
			args: args{
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						App: &openrtb2.App{
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
			name: "populate Site PBS-ACCOUNTID key",
			key:  MacroKeyAccountID,
			args: args{
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
			name: "populate PBS-LIMITADTRACKING key",
			key:  MacroKeyLmtTracking,
			args: args{
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
			name: "populate PBS-CONSENT key",
			key:  MacroKeyConsent,
			args: args{
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						User: &openrtb2.User{Ext: []byte(`{"consent":"1" }`)},
					},
				},
			},
			want: "1",
		},
		{
			name: "populate PBS-INTEGRATION key",
			key:  MacroKeyIntegration,
			args: args{
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						Ext: []byte(`{"prebid":{"integration":"testIntegration","channel": {"name":"test1"},"macros":{"CUSTOMMACR1":"value1","CUSTOMMACR2":"value2","CUSTOMMACR3":"value3"}}}`),
					},
				},
			},
			want: "testIntegration",
		},
		{
			name: "populate PBS-CHANNEL key",
			key:  MacroKeyChannel,
			args: args{
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						Ext: []byte(`{"prebid":{"channel": {"name":"test1"}}}`),
					},
				},
			},
			want: "test1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &macroProvider{
				macros: map[string]string{},
			}
			b.populateRequestMacros(tt.args.reqWrapper)
			assert.Equal(t, tt.want, b.GetMacro(tt.key), tt.name)
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
			assert.Equal(t, tt.want, b.GetMacro(tt.key), tt.name)
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
			assert.Equal(t, tt.want, b.GetMacro(tt.key), tt.name)
		})
	}
}

func TestTruncate(t *testing.T) {
	type args struct {
		text  string
		width uint
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "text is empty",
			args: args{
				text:  "",
				width: customMacroLength,
			},
			want: "",
		},
		{
			name: "width less than 100 chars",
			args: args{
				text:  "abcdef",
				width: customMacroLength,
			},
			want: "abcdef",
		},
		{
			name: "width exactly 100 chars",
			args: args{
				text:  "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuv",
				width: customMacroLength,
			},
			want: "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuv",
		},
		{
			name: "width greater than 100 chars",
			args: args{
				text:  "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz",
				width: customMacroLength,
			},
			want: "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuv",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.args.text, tt.args.width)
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}
