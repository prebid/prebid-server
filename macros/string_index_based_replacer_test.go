package macros

import (
	"testing"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/exchange/entities"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestStringIndexBasedReplace(t *testing.T) {

	type args struct {
		url              string
		getMacroProvider func() *macroProvider
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "success",
			args: args{
				url: "http://tracker.com?macro1=##PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-DOMAIN##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro7=##PBS-LIMITADTRACKING##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1##&macro10=##PBS-BIDDER##&macro11=##PBS-INTEGRATION##&macro12=##PBS-VASTCRTID##&macro15=##PBS-AUCTIONID##&macro16=##PBS-CHANNEL##&macro17=##PBS-EVENTTYPE##&macro18=##PBS-VASTEVENT##",
				getMacroProvider: func() *macroProvider {
					macroProvider := NewProvider(req)
					macroProvider.PopulateBidMacros(&entities.PbsOrtbBid{Bid: bid}, "test")
					macroProvider.PopulateEventMacros("123", "vast", "firstQuartile")
					return macroProvider
				},
			},
			want:    "http://tracker.com?macro1=bidId123&macro2=testbundle&macro3=testdomain&macro4=publishertestdomain&macro5=pageurltest&macro6=testpublisherID&macro7=10&macro8=yes&macro9=value1&macro10=test&macro11=&macro12=123&macro15=123&macro16=test1&macro17=vast&macro18=firstQuartile",
			wantErr: false,
		},
		{
			name: "url does not have macro",
			args: args{
				url: "http://tracker.com",
				getMacroProvider: func() *macroProvider {
					macroProvider := NewProvider(req)
					macroProvider.PopulateBidMacros(&entities.PbsOrtbBid{Bid: bid}, "test")
					macroProvider.PopulateEventMacros("123", "vast", "firstQuartile")
					return macroProvider
				},
			},
			want:    "http://tracker.com",
			wantErr: false,
		},
		{
			name: "macro not found",
			args: args{
				url: "http://tracker.com?macro1=##PBS-test1##",
				getMacroProvider: func() *macroProvider {
					macroProvider := NewProvider(&openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{}})
					macroProvider.PopulateBidMacros(&entities.PbsOrtbBid{Bid: bid}, "test")
					macroProvider.PopulateEventMacros("123", "vast", "firstQuartile")
					return macroProvider
				},
			},
			want:    "http://tracker.com?macro1=",
			wantErr: false,
		},
		{
			name: "tracker url is empty",
			args: args{
				url: "",
				getMacroProvider: func() *macroProvider {
					macroProvider := NewProvider(&openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{}})
					macroProvider.PopulateBidMacros(&entities.PbsOrtbBid{Bid: bid}, "test")
					macroProvider.PopulateEventMacros("123", "vast", "firstQuartile")
					return macroProvider
				},
			},
			want:    "",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			replacer := NewStringIndexBasedReplacer()
			got, err := replacer.Replace(tt.args.url, tt.args.getMacroProvider())
			if tt.wantErr {
				assert.Error(t, err, tt.name)
			} else {
				assert.NoError(t, err, tt.name)
				assert.Equal(t, tt.want, got, tt.name)
			}
		})
	}
}

var lmt int8 = 10
var benchmarkURL = []string{
	"http://tracker1.com?macro1=##PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-APPBUNDLE##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro6=##PBS-LIMITADTRACKING##&macro7=##PBS-GDPRCONSENT##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1CUST1##&macro10=##PBS-MACRO-CUSTOMMACR1CUST2##",
	"http://google.com?macro1=##PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-APPBUNDLE##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro6=##PBS-LIMITADTRACKING##&macro7=##PBS-GDPRCONSENT##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1CUST1##&macro10=##PBS-MACRO-CUSTOMMACR1CUST2##",
	"http://pubmatic.com?macro1=##PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-APPBUNDLE##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro6=##PBS-LIMITADTRACKING##&macro7=##PBS-GDPRCONSENT##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1CUST1##&macro10=##PBS-MACRO-CUSTOMMACR1CUST2##",
	"http://testbidder.com?macro1=##PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-APPBUNDLE##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro6=##PBS-LIMITADTRACKING##&macro7=##PBS-GDPRCONSENT##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1CUST1##&macro10=##PBS-MACRO-CUSTOMMACR1CUST2##",
	"http://dummybidder.com?macro1=##PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-APPBUNDLE##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro6=##PBS-LIMITADTRACKING##&macro7=##PBS-GDPRCONSENT##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1CUST1##&macro10=##PBS-MACRO-CUSTOMMACR1CUST2##",
}

var req *openrtb_ext.RequestWrapper = &openrtb_ext.RequestWrapper{
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
}

var bid *openrtb2.Bid = &openrtb2.Bid{ID: "bidId123", CID: "campaign_1", CrID: "creative_1"}

func BenchmarkStringIndexBasedReplacer(b *testing.B) {
	replacer := NewStringIndexBasedReplacer()
	for n := 0; n < b.N; n++ {
		for _, url := range benchmarkURL {
			macroProvider := NewProvider(req)
			macroProvider.PopulateBidMacros(&entities.PbsOrtbBid{Bid: bid}, "test")
			macroProvider.PopulateEventMacros("123", "vast", "firstQuartile")
			_, err := replacer.Replace(url, macroProvider)
			if err != nil {
				b.Errorf("Fail to replace macro in tracker")
			}
		}
	}
}
