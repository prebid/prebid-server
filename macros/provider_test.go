package macros

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/exchange/entities"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
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
		args args
		want map[string]string
	}{
		{
			name: "No request level macros present",
			args: args{
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{},
				},
			},
			want: map[string]string{MacroKeyBidID: "", MacroKeyAppBundle: "", MacroKeyDomain: "", MacroKeyPubDomain: "", MacroKeyPageURL: "", MacroKeyAccountID: "", MacroKeyLmtTracking: "", MacroKeyConsent: "", MacroKeyBidder: "", MacroKeyIntegration: "", MacroKeyVastCRTID: "", MacroKeyAuctionID: "", MacroKeyChannel: "", MacroKeyEventType: "", MacroKeyVastEvent: ""},
		},
		{
			name: " AUCTIONID, AppBundle, PageURL present key",
			args: args{
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID: "123",
						App: &openrtb2.App{
							Bundle: "testBundle",
						},
						Site: &openrtb2.Site{
							Page: "testPage",
						},
					},
				},
			},
			want: map[string]string{MacroKeyBidID: "", MacroKeyAppBundle: "testBundle", MacroKeyDomain: "", MacroKeyPubDomain: "", MacroKeyPageURL: "testPage", MacroKeyAccountID: "", MacroKeyLmtTracking: "", MacroKeyConsent: "", MacroKeyBidder: "", MacroKeyIntegration: "", MacroKeyVastCRTID: "", MacroKeyAuctionID: "123", MacroKeyChannel: "", MacroKeyEventType: "", MacroKeyVastEvent: ""},
		},
		{
			name: " AppDomain, PubDomain, present key",
			args: args{
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						App: &openrtb2.App{
							Domain: "testDomain",
						},
						Site: &openrtb2.Site{
							Publisher: &openrtb2.Publisher{
								Domain: "pubDomain",
							},
						},
					},
				},
			},
			want: map[string]string{MacroKeyBidID: "", MacroKeyAppBundle: "", MacroKeyDomain: "testDomain", MacroKeyPubDomain: "pubDomain", MacroKeyPageURL: "", MacroKeyAccountID: "", MacroKeyLmtTracking: "", MacroKeyConsent: "", MacroKeyBidder: "", MacroKeyIntegration: "", MacroKeyVastCRTID: "", MacroKeyAuctionID: "", MacroKeyChannel: "", MacroKeyEventType: "", MacroKeyVastEvent: ""},
		},
		{
			name: " Integration, Consent present key",
			args: args{
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						User: &openrtb2.User{Consent: "1", Ext: []byte(`{"consent":"2" }`)},
						Ext:  []byte(`{"prebid":{"integration":"testIntegration"}}`),
					},
				},
			},
			want: map[string]string{MacroKeyBidID: "", MacroKeyAppBundle: "", MacroKeyDomain: "", MacroKeyPubDomain: "", MacroKeyPageURL: "", MacroKeyAccountID: "", MacroKeyLmtTracking: "", MacroKeyConsent: "1", MacroKeyBidder: "", MacroKeyIntegration: "testIntegration", MacroKeyVastCRTID: "", MacroKeyAuctionID: "", MacroKeyChannel: "", MacroKeyEventType: "", MacroKeyVastEvent: ""},
		},
		{
			name: " PBS-CHANNEL, LIMITADTRACKING present key",
			args: args{
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						Device: &openrtb2.Device{
							Lmt: &lmt,
						},
						Ext: []byte(`{"prebid":{"channel": {"name":"test1"}}}`),
					},
				},
			},
			want: map[string]string{MacroKeyBidID: "", MacroKeyAppBundle: "", MacroKeyDomain: "", MacroKeyPubDomain: "", MacroKeyPageURL: "", MacroKeyAccountID: "", MacroKeyLmtTracking: "10", MacroKeyConsent: "", MacroKeyBidder: "", MacroKeyIntegration: "", MacroKeyVastCRTID: "", MacroKeyAuctionID: "", MacroKeyChannel: "test1", MacroKeyEventType: "", MacroKeyVastEvent: ""},
		},
		{
			name: " custom macros present key",
			args: args{
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						Ext: []byte(`{"prebid":{"macros":{"CUSTOMMACR1":"value1"}}}`),
					},
				},
			},
			want: map[string]string{"PBS-MACRO-CUSTOMMACR1": "value1", MacroKeyBidID: "", MacroKeyAppBundle: "", MacroKeyDomain: "", MacroKeyPubDomain: "", MacroKeyPageURL: "", MacroKeyAccountID: "", MacroKeyLmtTracking: "", MacroKeyConsent: "", MacroKeyBidder: "", MacroKeyIntegration: "", MacroKeyVastCRTID: "", MacroKeyAuctionID: "", MacroKeyChannel: "", MacroKeyEventType: "", MacroKeyVastEvent: ""},
		},
		{
			name: " All request macros present key",
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
						User: &openrtb2.User{Consent: "1", Ext: []byte(`{"consent":"2" }`)},
						Ext:  []byte(`{"prebid":{"channel": {"name":"test1"},"macros":{"CUSTOMMACR1":"value1"}}}`),
					},
				},
			},
			want: map[string]string{"PBS-MACRO-CUSTOMMACR1": "value1", MacroKeyBidID: "", MacroKeyAppBundle: "testbundle", MacroKeyDomain: "testdomain", MacroKeyPubDomain: "publishertestdomain", MacroKeyPageURL: "pageurltest", MacroKeyAccountID: "testpublisherID", MacroKeyLmtTracking: "10", MacroKeyConsent: "1", MacroKeyBidder: "", MacroKeyIntegration: "", MacroKeyVastCRTID: "", MacroKeyAuctionID: "123", MacroKeyChannel: "test1", MacroKeyEventType: "", MacroKeyVastEvent: ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &MacroProvider{
				macros: map[string]string{},
			}
			b.populateRequestMacros(tt.args.reqWrapper)
			output := map[string]string{}
			for key := range tt.want {
				output[key] = b.GetMacro(key)
			}
			assert.Equal(t, tt.want, output, tt.name)
		})
	}
}

func TestPopulateBidMacros(t *testing.T) {

	type args struct {
		bid  *entities.PbsOrtbBid
		seat string
	}
	tests := []struct {
		name      string
		args      args
		wantBidID string
		wantSeat  string
	}{
		{
			name: "Bid ID set, no generatedbid id, no seat",
			args: args{
				bid: &entities.PbsOrtbBid{
					Bid: &openrtb2.Bid{
						ID: "bid123",
					},
				},
			},
			wantBidID: "bid123",
			wantSeat:  "",
		},
		{
			name: "Bid ID set, no generatedbid id, seat set",
			args: args{
				bid: &entities.PbsOrtbBid{
					Bid: &openrtb2.Bid{
						ID: "bid123",
					},
				},
				seat: "testSeat",
			},
			wantBidID: "bid123",
			wantSeat:  "testSeat",
		},
		{
			name: "Bid ID set, generatedbid id set, no seat",
			args: args{
				bid: &entities.PbsOrtbBid{
					GeneratedBidID: "generatedbid123",
					Bid: &openrtb2.Bid{
						ID: "bid123",
					},
				},
			},
			wantBidID: "generatedbid123",
			wantSeat:  "",
		},
		{
			name: "Bid ID set, generatedbid id set, seat set",
			args: args{
				bid: &entities.PbsOrtbBid{
					GeneratedBidID: "generatedbid123",
					Bid: &openrtb2.Bid{
						ID: "bid123",
					},
				},
				seat: "testseat",
			},
			wantBidID: "generatedbid123",
			wantSeat:  "testseat",
		},
		{
			name: "Bid ID not set, generatedbid id set, seat set",
			args: args{
				seat: "test-seat",
				bid: &entities.PbsOrtbBid{
					GeneratedBidID: "generatedbid123",
					Bid:            &openrtb2.Bid{},
				},
			},
			wantBidID: "generatedbid123",
			wantSeat:  "test-seat",
		},
		{
			name: "Bid ID not set, generatedbid id not set, seat set",
			args: args{
				seat: "test-seat",
				bid:  &entities.PbsOrtbBid{},
			},
			wantBidID: "",
			wantSeat:  "test-seat",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &MacroProvider{
				macros: map[string]string{},
			}
			b.PopulateBidMacros(tt.args.bid, tt.args.seat)
			assert.Equal(t, tt.wantBidID, b.GetMacro(MacroKeyBidID), tt.name)
			assert.Equal(t, tt.wantSeat, b.GetMacro(MacroKeyBidder), tt.name)
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
		name               string
		args               args
		wantVastCreativeID string
		wantEventType      string
		wantVastEvent      string
	}{
		{
			name:               "creativeId not set, eventType not set, vastEvent not set",
			args:               args{},
			wantVastCreativeID: "",
			wantEventType:      "",
			wantVastEvent:      "",
		},
		{
			name: "creativeId set, eventType not set, vastEvent not set",
			args: args{
				vastCreativeID: "123",
			},
			wantVastCreativeID: "123",
			wantEventType:      "",
			wantVastEvent:      "",
		},
		{
			name: "creativeId not set, eventType  set, vastEvent not set",
			args: args{
				eventType: "win",
			},
			wantVastCreativeID: "",
			wantEventType:      "win",
			wantVastEvent:      "",
		},
		{
			name: "creativeId not set, eventType not set, vastEvent set",
			args: args{
				vastEvent: "firstQuartile",
			},
			wantVastCreativeID: "",
			wantEventType:      "",
			wantVastEvent:      "firstQuartile",
		},
		{
			name: "creativeId not set, eventType  set, vastEvent set",
			args: args{
				vastEvent: "firstQuartile",
				eventType: "win",
			},
			wantVastCreativeID: "",
			wantEventType:      "win",
			wantVastEvent:      "firstQuartile",
		},
		{
			name: "creativeId  set, eventType not set, vastEvent set",
			args: args{
				vastEvent:      "firstQuartile",
				vastCreativeID: "123",
			},
			wantVastCreativeID: "123",
			wantEventType:      "",
			wantVastEvent:      "firstQuartile",
		},
		{
			name: "creativeId set, eventType set, vastEvent not set",
			args: args{
				eventType:      "win",
				vastCreativeID: "123",
			},
			wantVastCreativeID: "123",
			wantEventType:      "win",
			wantVastEvent:      "",
		},
		{
			name: "creativeId set, eventType set, vastEvent set",
			args: args{
				vastEvent:      "firstQuartile",
				eventType:      "win",
				vastCreativeID: "123",
			},
			wantVastCreativeID: "123",
			wantEventType:      "win",
			wantVastEvent:      "firstQuartile",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &MacroProvider{
				macros: map[string]string{},
			}
			b.PopulateEventMacros(tt.args.vastCreativeID, tt.args.eventType, tt.args.vastEvent)
			assert.Equal(t, tt.wantVastCreativeID, b.GetMacro(MacroKeyVastCRTID), tt.name)
			assert.Equal(t, tt.wantVastEvent, b.GetMacro(MacroKeyVastEvent), tt.name)
			assert.Equal(t, tt.wantEventType, b.GetMacro(MacroKeyEventType), tt.name)
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
