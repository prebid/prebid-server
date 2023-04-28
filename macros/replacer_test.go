package macros

import (
	"testing"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/exchange/entities"
	"github.com/prebid/prebid-server/openrtb_ext"
)

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
