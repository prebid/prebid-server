package macros

import (
	"regexp"
	"strings"
	"testing"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/exchange/entities"
	"github.com/prebid/prebid-server/openrtb_ext"
)

var lmt int8 = 10
var testURL string = "http://tracker.com?macro1=##PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-APPBUNDLE##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro6=##PBS-LIMITADTRACKING##&macro7=##PBS-GDPRCONSENT##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1CUST1##&macro10=##PBS-MACRO-CUSTOMMACR1CUST2##"

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

func BenchmarkStringIndexCachedBasedReplacer(b *testing.B) {

	processor := NewReplacer()
	for n := 0; n < b.N; n++ {
		for _, url := range benchmarkURL {
			macroProvider := NewProvider(req)

			macroProvider.SetContext(MacroContext{
				Bid:            &entities.PbsOrtbBid{Bid: bid},
				Imp:            nil,
				Seat:           "test",
				VastCreativeID: "123",
				VastEventType:  "firstQuartile",
				EventElement:   "tracking",
			})
			_, err := processor.Replace(url, macroProvider)
			if err != nil {
				b.Errorf("Fail to replace macro in tracker")
			}
		}
	}
}

func BenchmarkGolangReplacer(b *testing.B) {

	for n := 0; n < b.N; n++ {
		for _, url := range benchmarkURL {
			macroProvider := NewProvider(req)

			macroProvider.SetContext(MacroContext{
				Bid:            &entities.PbsOrtbBid{Bid: bid},
				Imp:            nil,
				Seat:           "test",
				VastCreativeID: "123",
				VastEventType:  "firstQuartile",
				EventElement:   "tracking",
			})
			StringReplacer(url, macroProvider)
		}
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

// 1) ^BenchmarkStringIndexCachedBasedReplacer$ github.com/prebid/prebid-server/macros
// goos: darwin
// goarch: amd64
// pkg: github.com/prebid/prebid-server/macros
// cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
// BenchmarkStringIndexCachedBasedReplacer-12    	   65348	     17148 ns/op	   12617 B/op	      56 allocs/op
// PASS
// ok  	github.com/prebid/prebid-server/macros	2.446s

// 2) ^BenchmarkGolangReplacer$ github.com/prebid/prebid-server/macros
// goos: darwin
// goarch: amd64
// pkg: github.com/prebid/prebid-server/macros
// cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
// BenchmarkGolangReplacer-12    	   10926	     99165 ns/op	   81887 B/op	     603 allocs/op
// PASS
// ok  	github.com/prebid/prebid-server/macros	2.144s
