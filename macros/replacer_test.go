package macros

import (
	"testing"

	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/exchange/entities"
	"github.com/prebid/prebid-server/openrtb_ext"
)

var lmt int8 = 10
var testURL string = "http://tracker.com?macro1=#PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-APPBUNDLE##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro6=##PBS-LIMITADTRACKING##&macro7=##PBS-GDPRCONSENT##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1CUST1##&macro10=##PBS-MACRO-CUSTOMMACR1CUST2##"
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

func BenchmarkStringIndexCachedBasedProcessor(b *testing.B) {

	processor := NewReplacer()
	for n := 0; n < b.N; n++ {
		macroProvider := NewProvider(req)

		macroProvider.SetContext(MacroContext{
			Bid:            &entities.PbsOrtbBid{Bid: bid},
			Imp:            nil,
			Seat:           "test",
			VastCreativeID: "123",
			VastEventType:  "firstQuartile",
			EventElement:   "tracking",
		})
		_, err := processor.Replace(testURL, macroProvider)
		if err != nil {
			b.Errorf("Fail to replace macro in tracker")
		}
	}
}
