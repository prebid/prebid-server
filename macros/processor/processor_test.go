package processor

import (
	"fmt"
	"testing"

	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
)

var lmt int8 = 10
var testURL string = "http://tracker.com?macro1=##PBS_BIDID##&macro2=##PBS_APPBUNDLE##&macro3=##PBS_APPBUNDLE##&macro4=##PBS_PUBDOMAIN##&macro5=##PBS_PAGEURL##&macro6=##PBS_ACCOUNTID##&macro6=##PBS_LIMITADTRACKING##&macro7=##PBS_GDPRCONSENT##&macro8=##PBS_GDPRCONSENT##&macro9=##PBS_MACRO_CUST1##&macro10=##PBS_MACRO_CUST2##"
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
			Bundle: "testBundle",
			Publisher: &openrtb2.Publisher{
				Domain: "publishertestdomain",
				ID:     "testpublisherID",
			},
		},
		Device: &openrtb2.Device{
			Lmt: &lmt,
		},
		User: &openrtb2.User{Ext: []byte(`{"consent":"yes" }`)},
		Ext:  []byte(`{"prebid":{"macros":{"CUSTOMMACR1":"value1","CUSTOMMACR2":"value2","CUSTOMMACR3":"value3"}}}`),
	},
}

var bid *openrtb2.Bid = &openrtb2.Bid{ID: "bidId123"}

func BenchmarkStringIndexCachedBasedProcessor(b *testing.B) {

	processor := NewProcessor()
	for n := 0; n < b.N; n++ {
		macroProvider := NewProvider(req)
		macroProvider.SetContext(bid, nil, "test")
		_, err := processor.Replace(testURL, macroProvider)
		if err != nil {
			fmt.Println(err)
		}
	}
}
