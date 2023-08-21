package injector

import (
	"fmt"
	"testing"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
)

var textXML = `<VAST version="4.2"
	xmlns="http://www.iab.com/VAST">
	<Ad id="20004" >
		<InLine>
			<AdSystem version="1">iabtechlab</AdSystem>
			<Error>
				<![CDATA[https://example.com/error]]>
			</Error>
			<Impression id="Impression-ID">
				<![CDATA[https://example.com/track/impression]]>
			</Impression>
			<Pricing model="cpm" currency="USD">
				<![CDATA[ 25.00 ]]>
			</Pricing>
			<AdServingId>a532d16d-4d7f-4440-bd29-2ec0e693fc80</AdServingId>
			<AdTitle>
				<![CDATA[VAST 4.0 Pilot - Scenario 5]]>
			</AdTitle>
			<Creatives>
				<Creative id="5481" sequence="1" adId="2447226">
					<Linear>
						<TrackingEvents>
							<Tracking event="start" >
								<![CDATA[https://example.com/tracking/start]]>
							</Tracking>
						</TrackingEvents>
						<VideoClicks>
							<ClickThrough id="blog">
								<![CDATA[https://iabtechlab.com]]>
							</ClickThrough>
						</VideoClicks>
					</Linear>
					<UniversalAdId idRegistry="Ad-ID" >8466</UniversalAdId>
				</Creative>
				<Creative id="5480" sequence="1" adId="2447226">
					<NonLinearAds>
						<NonLinear width="350" height="350">
							<StaticResource creativeType="image/png">
								<![CDATA[https://mms.businesswire.com/media/20150623005446/en/473787/21/iab_tech_lab.jpg]]>
							</StaticResource>
							<NonLinearClickThrough>
								<![CDATA[https://iabtechlab.com]]>
							</NonLinearClickThrough>
							<NonLinearClickTracking>
								<![CDATA[https://example.com/tracking/clickTracking]]>
							</NonLinearClickTracking>
						</NonLinear>
					</NonLinearAds>
					<UniversalAdId idRegistry="Ad-ID">8465</UniversalAdId>
				</Creative>
				<Creative id="5480" sequence="1" adId="2447226">
					<CompanionAds>
						<Companion id="1232" width="100" height="150" assetWidth="250" assetHeight="200" expandedWidth="350" expandedHeight="250" adSlotId="3214" pxratio="1400">
							<StaticResource creativeType="image/png">
								<![CDATA[https://www.iab.com/wp-content/uploads/2014/09/iab-tech-lab-6-644x290.png]]>
							</StaticResource>
							<CompanionClickThrough>
								<![CDATA[https://iabtechlab.com]]>
							</CompanionClickThrough>
						</Companion>
					</CompanionAds>
					<UniversalAdId idRegistry="Ad-ID" >8465</UniversalAdId>
				</Creative>
			</Creatives>
			<Description>
				<![CDATA[This is sample companion ad tag with Linear ad tag. This tag while showing video ad on the player, will show a companion ad beside the player where it can be fitted. At most 3 companion ads can be placed. Modify accordingly to see your own content. ]]>
			</Description>
		</InLine>
	</Ad>
</VAST>`

func TestTrackerInjector(t *testing.T) {

	//	pro.PopulateBidMacros(nil, "seat")
	builder := NewTrackerInjector(rep, pro, evnt)
	fmt.Println(builder.Build(textXML, ""))
}

var lmts int8 = 1
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
			Lmt: &lmts,
		},
		User: &openrtb2.User{Ext: []byte(`{"consent":"yes" }`)},
		Ext:  []byte(`{"prebid":{"channel": {"name":"test1"},"macros":{"CUSTOMMACR1":"value1","CUSTOMMACR2":"value2","CUSTOMMACR3":"value3"}}}`),
	},
}

var (
	evnt VASTEvents
	rep  macros.Replacer
	pro  *macros.MacroProvider
)

func init() {
	evnt = VASTEvents{
		Impressions: []string{"http://impression.com?macro1=##PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-DOMAIN##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro7=##PBS-LIMITADTRACKING##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1##&macro10=##PBS-BIDDER##&macro11=##PBS-INTEGRATION##&macro12=##PBS-VASTCRTID##&macro15=##PBS-AUCTIONID##&macro16=##PBS-CHANNEL##&macro17=##PBS-EVENTTYPE##&macro18=##PBS-VASTEVENT##"},
		Errors:      []string{"http://errors.com?macro1=##PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-DOMAIN##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro7=##PBS-LIMITADTRACKING##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1##&macro10=##PBS-BIDDER##&macro11=##PBS-INTEGRATION##&macro12=##PBS-VASTCRTID##&macro15=##PBS-AUCTIONID##&macro16=##PBS-CHANNEL##&macro17=##PBS-EVENTTYPE##&macro18=##PBS-VASTEVENT##"},

		VideoClicks:            []string{"http://videoClicks.com?macro1=##PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-DOMAIN##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro7=##PBS-LIMITADTRACKING##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1##&macro10=##PBS-BIDDER##&macro11=##PBS-INTEGRATION##&macro12=##PBS-VASTCRTID##&macro15=##PBS-AUCTIONID##&macro16=##PBS-CHANNEL##&macro17=##PBS-EVENTTYPE##&macro18=##PBS-VASTEVENT##"},
		NonLinearClickTracking: []string{"http://NonLinearClickTracking.com?macro1=##PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-DOMAIN##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro7=##PBS-LIMITADTRACKING##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1##&macro10=##PBS-BIDDER##&macro11=##PBS-INTEGRATION##&macro12=##PBS-VASTCRTID##&macro15=##PBS-AUCTIONID##&macro16=##PBS-CHANNEL##&macro17=##PBS-EVENTTYPE##&macro18=##PBS-VASTEVENT##"},
		CompanionClickThrough:  []string{"http://CompanionClickThrough.com?macro1=##PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-DOMAIN##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro7=##PBS-LIMITADTRACKING##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1##&macro10=##PBS-BIDDER##&macro11=##PBS-INTEGRATION##&macro12=##PBS-VASTCRTID##&macro15=##PBS-AUCTIONID##&macro16=##PBS-CHANNEL##&macro17=##PBS-EVENTTYPE##&macro18=##PBS-VASTEVENT##"},
		LinearTrackingEvents: map[string][]string{
			"firstQuartile": []string{"http://tracking.com?macro1=##PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-DOMAIN##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro7=##PBS-LIMITADTRACKING##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1##&macro10=##PBS-BIDDER##&macro11=##PBS-INTEGRATION##&macro12=##PBS-VASTCRTID##&macro15=##PBS-AUCTIONID##&macro16=##PBS-CHANNEL##&macro17=##PBS-EVENTTYPE##&macro18=##PBS-VASTEVENT##"},
			"midQuartile":   []string{"http://tracking.com?macro1=##PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-DOMAIN##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro7=##PBS-LIMITADTRACKING##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1##&macro10=##PBS-BIDDER##&macro11=##PBS-INTEGRATION##&macro12=##PBS-VASTCRTID##&macro15=##PBS-AUCTIONID##&macro16=##PBS-CHANNEL##&macro17=##PBS-EVENTTYPE##&macro18=##PBS-VASTEVENT##"},
			"end":           []string{"http://tracking.com?macro1=##PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-DOMAIN##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro7=##PBS-LIMITADTRACKING##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1##&macro10=##PBS-BIDDER##&macro11=##PBS-INTEGRATION##&macro12=##PBS-VASTCRTID##&macro15=##PBS-AUCTIONID##&macro16=##PBS-CHANNEL##&macro17=##PBS-EVENTTYPE##&macro18=##PBS-VASTEVENT##"},
		},
		NonLinearTrackingEvents: map[string][]string{
			"firstQuartile": []string{"http://tracking.com?macro1=##PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-DOMAIN##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro7=##PBS-LIMITADTRACKING##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1##&macro10=##PBS-BIDDER##&macro11=##PBS-INTEGRATION##&macro12=##PBS-VASTCRTID##&macro15=##PBS-AUCTIONID##&macro16=##PBS-CHANNEL##&macro17=##PBS-EVENTTYPE##&macro18=##PBS-VASTEVENT##"},
			"midQuartile":   []string{"http://tracking.com?macro1=##PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-DOMAIN##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro7=##PBS-LIMITADTRACKING##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1##&macro10=##PBS-BIDDER##&macro11=##PBS-INTEGRATION##&macro12=##PBS-VASTCRTID##&macro15=##PBS-AUCTIONID##&macro16=##PBS-CHANNEL##&macro17=##PBS-EVENTTYPE##&macro18=##PBS-VASTEVENT##"},
			"end":           []string{"http://tracking.com?macro1=##PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-DOMAIN##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro7=##PBS-LIMITADTRACKING##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1##&macro10=##PBS-BIDDER##&macro11=##PBS-INTEGRATION##&macro12=##PBS-VASTCRTID##&macro15=##PBS-AUCTIONID##&macro16=##PBS-CHANNEL##&macro17=##PBS-EVENTTYPE##&macro18=##PBS-VASTEVENT##"},
		},
	}

	rep = macros.NewStringIndexBasedReplacer()
	pro = macros.NewProvider(req)
}

// goos: darwin
// goarch: amd64
// pkg: github.com/prebid/prebid-server/exchange/injector/vaststruct
// cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
// BenchmarkStructUnmarshal-12    	    4402	    239360 ns/op	   72024 B/op	     829 allocs/op
// PASS
// ok  	github.com/prebid/prebid-server/exchange/injector/vaststruct	1.424s

func BenchmarkStructUnmarshal(b *testing.B) {
	//	pro.PopulateBidMacros(nil, "seat")
	for i := 0; i < b.N; i++ {
		builder := NewTrackerInjector(rep, pro, evnt)
		builder.Build(textXML, "")
	}
}
