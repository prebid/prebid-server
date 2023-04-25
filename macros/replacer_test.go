package macros

import (
	"regexp"
	"strings"
	"testing"

	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/exchange/entities"
	"github.com/prebid/prebid-server/openrtb_ext"
)

var lmt int8 = 10
var testURL string = "http://tracker.com?macro1=##PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-APPBUNDLE##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro6=##PBS-LIMITADTRACKING##&macro7=##PBS-GDPRCONSENT##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1CUST1##&macro10=##PBS-MACRO-CUSTOMMACR1CUST2##"
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

func BenchmarkStringIndexCachedBasedReplacer(b *testing.B) {

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

func BenchmarkGolangReplacer(b *testing.B) {

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
		StringReplacer(testURL, macroProvider)
	}
}

func StringReplacer(url string, mp *macroProvider) string {
	keyValue := []string{}
	for key, value := range mp.GetAllMacro() {
		keyValue = append(keyValue, "##"+key+"##")
		keyValue = append(keyValue, value)
	}

	rplcr := strings.NewReplacer(keyValue...)
	output := rplcr.Replace(url)
	r := regexp.MustCompile(`##(.*?)##`)
	return r.ReplaceAllString(output, "")
}

// ^BenchmarkStringIndexCachedBasedReplacer$ github.com/prebid/prebid-server/macros

// goos: darwin
// goarch: amd64
// pkg: github.com/prebid/prebid-server/macros
// cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
// BenchmarkStringIndexCachedBasedReplacer-12    	  351223	      3060 ns/op	    2522 B/op	      11 allocs/op
// PASS
// ok  	github.com/prebid/prebid-server/macros	2.464s

// ^BenchmarkGolangReplacer$ github.com/prebid/prebid-server/macros

// goos: darwin
// goarch: amd64
// pkg: github.com/prebid/prebid-server/macros
// cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
// BenchmarkGolangReplacer-12    	   63328	     17184 ns/op	   16375 B/op	     120 allocs/op
// PASS
// ok  	github.com/prebid/prebid-server/macros	1.485s
