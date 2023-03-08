package processor

import (
	"fmt"
	"testing"

	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/config"
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

func BenchmarkTemplateBasedProcessor(b *testing.B) {

	processor := NewProcessor(config.MacroProcessorConfig{ProcessorType: config.TemplateBasedProcessor})
	for n := 0; n < b.N; n++ {
		macroProvider := NewProvider(req)
		macroProvider.SetContext(bid, nil, "test")
		_, err := processor.Replace(testURL, macroProvider)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func BenchmarkStringIndexCachedBasedProcessor(b *testing.B) {

	processor := NewProcessor(config.MacroProcessorConfig{ProcessorType: config.StringBasedProcessor})
	for n := 0; n < b.N; n++ {
		macroProvider := NewProvider(req)
		macroProvider.SetContext(bid, nil, "test")
		_, err := processor.Replace(testURL, macroProvider)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func BenchmarkStringIndexBasedProcessor(b *testing.B) {

	processor := NewProcessor(config.MacroProcessorConfig{ProcessorType: 3})
	for n := 0; n < b.N; n++ {
		macroProvider := NewProvider(req)
		macroProvider.SetContext(bid, nil, "test")
		_, err := processor.Replace(testURL, macroProvider)
		if err != nil {
			fmt.Println(err)
		}
	}
}

// Bechmark with macro provider created only once
// Running tool: /usr/local/go/bin/go test -benchmem -run=^$ -coverprofile=/var/folders/z5/cj4zjtv15wn8yt53qzh57lnr0000gp/T/vscode-go3xzhae/go-code-cover -bench . github.com/prebid/prebid-server/macros/processor

// goos: darwin
// goarch: amd64
// pkg: github.com/prebid/prebid-server/macros/processor
// cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
// BenchmarkTemplateBasedProcessor-12             	  242168	      4742 ns/op	    2002 B/op	      31 allocs/op
// BenchmarkStringIndexCachedBasedProcessor-12    	 1407223	       747.8 ns/op	     640 B/op	       4 allocs/op
// BenchmarkStringIndexBasedProcessor-12          	 1000000	      1075 ns/op	     688 B/op	       4 allocs/op
// PASS
// coverage: 85.9% of statements
// ok  	github.com/prebid/prebid-server/macros/processor	4.386s
//
//
//
//
//
// Bechmark with macro provider created on every run
// goos: darwin
// goarch: amd64
// pkg: github.com/prebid/prebid-server/macros/processor
// cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
// BenchmarkTemplateBasedProcessor-12             	  217635	      4613 ns/op	    2346 B/op	      34 allocs/op
// BenchmarkStringIndexCachedBasedProcessor-12    	 1000000	      1035 ns/op	     984 B/op	       7 allocs/op
// BenchmarkStringIndexBasedProcessor-12          	  833914	      1390 ns/op	    1032 B/op	       7 allocs/op
// PASS
// coverage: 85.9% of statements
// ok  	github.com/prebid/prebid-server/macros/processor	3.549s
