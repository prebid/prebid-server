package injector

import (
	"errors"
	"strings"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/exchange/entities"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

var reqWrapper = &openrtb_ext.RequestWrapper{
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
			Lmt: ptrutil.ToPtr(int8(1)),
		},
		User: &openrtb2.User{Consent: "1", Ext: []byte(`{"consent":"2" }`)},
		Ext:  []byte(`{"prebid":{"channel": {"name":"test1"},"macros":{"CUSTOMMACR1":"value1"}}}`),
	},
}

func TestInjectTracker(t *testing.T) {
	b := macros.NewProvider(reqWrapper)
	b.PopulateBidMacros(&entities.PbsOrtbBid{
		Bid: &openrtb2.Bid{
			ID: "bid123",
		},
	}, "testSeat")
	ti := NewTrackerInjector(
		macros.NewStringIndexBasedReplacer(),
		b,
		VASTEvents{
			Errors:                 []string{"http://errortracker.com?macro1=##PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-APPBUNDLE##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro6=##PBS-LIMITADTRACKING##&macro7=##PBS-GDPRCONSENT##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1CUST1##&macro10=##PBS-MACRO-CUSTOMMACR1CUST2##"},
			Impressions:            []string{"http://impressiontracker.com?macro1=##PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-APPBUNDLE##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro6=##PBS-LIMITADTRACKING##&macro7=##PBS-GDPRCONSENT##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1CUST1##&macro10=##PBS-MACRO-CUSTOMMACR1CUST2##"},
			VideoClicks:            []string{"http://videoclicktracker.com?macro1=##PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-APPBUNDLE##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro6=##PBS-LIMITADTRACKING##&macro7=##PBS-GDPRCONSENT##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1CUST1##&macro10=##PBS-MACRO-CUSTOMMACR1CUST2##"},
			NonLinearClickTracking: []string{"http://nonlinearclicktracker.com?macro1=##PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-APPBUNDLE##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro6=##PBS-LIMITADTRACKING##&macro7=##PBS-GDPRCONSENT##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1CUST1##&macro10=##PBS-MACRO-CUSTOMMACR1CUST2##"},
			CompanionClickThrough:  []string{"http://companionclicktracker.com?macro1=##PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-APPBUNDLE##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro6=##PBS-LIMITADTRACKING##&macro7=##PBS-GDPRCONSENT##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1CUST1##&macro10=##PBS-MACRO-CUSTOMMACR1CUST2##"},
			TrackingEvents:         map[string][]string{"firstQuartile": {"http://eventracker1.com?macro1=##PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-APPBUNDLE##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro6=##PBS-LIMITADTRACKING##&macro7=##PBS-GDPRCONSENT##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1CUST1##&macro10=##PBS-MACRO-CUSTOMMACR1CUST2##"}},
		},
	)
	type args struct {
		vastXML string
		NURL    string
	}
	tests := []struct {
		name      string
		args      args
		want      string
		wantError error
	}{
		{
			name: "Empty vastXML and NURL present",
			args: args{
				vastXML: "",
				NURL:    "www.nurl.com",
			},
			want:      `<VAST version="3.0"><Ad><Wrapper><AdSystem>prebid.org wrapper</AdSystem><VASTAdTagURI><![CDATA[www.nurl.com]]></VASTAdTagURI><Creatives></Creatives></Wrapper></Ad></VAST>`,
			wantError: nil,
		},
		{
			name: "Empty vastXML and empty NURL",
			args: args{
				vastXML: "",
				NURL:    "",
			},
			want:      "",
			wantError: errors.New("invalid Vast XML"),
		},
		{
			name: "No Inline/Wrapper tag present",
			args: args{
				vastXML: `<VAST version="4.0" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns="http://www.iab.com/VAST"><Ad id="20001" sequence="1" conditionalAd="false"></Ad></VAST>`,
				NURL:    "",
			},
			want:      `<VAST version="4.0" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns="http://www.iab.com/VAST"><Ad id="20001" sequence="1" conditionalAd="false"></Ad></VAST>`,
			wantError: errors.New("invalid VastXML, inline/wrapper tag not found"),
		},
		{
			name: "Invalid Vast XML, parsing error",
			args: args{
				vastXML: `<VAST version="4.0" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns="http://www.iab.com/VAST"><Ad id="20001" sequence="1" conditionalAd="false"><InLine><AdSystem version="4.0">iabtechlab</AdSystem><Error>http://example.com/error<Impression id="Impression-ID">http://example.com/track/impression</Impression><Pricing model="cpm" currency="USD"><![CDATA[ 25.00 ]]></Pricing><AdTitle>Inline Simple Ad</AdTitle><AdVerifications></AdVerifications><Advertiser>IAB Sample Company</Advertiser><Category authority="http://www.iabtechlab.com/categoryauthority">AD CONTENT description category</Category><Creatives><Creative id="5480" sequence="1" adId="2447226"><UniversalAdId idRegistry="Ad-ID" idValue="8465">8465</UniversalAdId><Linear><Duration>00:00:16</Duration><MediaFiles><MediaFile id="5241" delivery="progressive" type="video/mp4" bitrate="2000" width="1280" height="720" minBitrate="1500" maxBitrate="2500" scalable="1" maintainAspectRatio="1" codec="H.264"><![CDATA[https://iab-publicfiles.s3.amazonaws.com/vast/VAST-4.0-Short-Intro.mp4]]></MediaFile><MediaFile id="5244" delivery="progressive" type="video/mp4" bitrate="1000" width="854" height="480" minBitrate="700" maxBitrate="1500" scalable="1" maintainAspectRatio="1" codec="H.264"><![CDATA[https://iab-publicfiles.s3.amazonaws.com/vast/VAST-4.0-Short-Intro-mid-resolution.mp4]]></MediaFile><MediaFile id="5246" delivery="progressive" type="video/mp4" bitrate="600" width="640" height="360" minBitrate="500" maxBitrate="700" scalable="1" maintainAspectRatio="1" codec="H.264"><![CDATA[https://iab-publicfiles.s3.amazonaws.com/vast/VAST-4.0-Short-Intro-low-resolution.mp4]]></MediaFile></MediaFiles></Linear></Creative></Creatives></InLine></Ad></VAST>`,
				NURL:    "",
			},
			want:      ``,
			wantError: errors.New("XML processing error: xml: end tag </InLine> does not match start tag <Error>"),
		},
		{
			name: "Inline Linear vastXML, no existing event tracker",
			args: args{
				vastXML: `<VAST version="4.0" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns="http://www.iab.com/VAST"><Ad id="20001" sequence="1" conditionalAd="false"><InLine><AdSystem version="4.0">iabtechlab</AdSystem><Error>http://example.com/error</Error><Impression id="Impression-ID">http://example.com/track/impression</Impression><Pricing model="cpm" currency="USD"><![CDATA[ 25.00 ]]></Pricing><AdTitle>Inline Simple Ad</AdTitle><AdVerifications></AdVerifications><Advertiser>IAB Sample Company</Advertiser><Category authority="http://www.iabtechlab.com/categoryauthority">AD CONTENT description category</Category><Creatives><Creative id="5480" sequence="1" adId="2447226"><UniversalAdId idRegistry="Ad-ID" idValue="8465">8465</UniversalAdId><Linear><Duration>00:00:16</Duration><MediaFiles><MediaFile id="5241" delivery="progressive" type="video/mp4" bitrate="2000" width="1280" height="720" minBitrate="1500" maxBitrate="2500" scalable="1" maintainAspectRatio="1" codec="H.264"><![CDATA[https://iab-publicfiles.s3.amazonaws.com/vast/VAST-4.0-Short-Intro.mp4]]></MediaFile><MediaFile id="5244" delivery="progressive" type="video/mp4" bitrate="1000" width="854" height="480" minBitrate="700" maxBitrate="1500" scalable="1" maintainAspectRatio="1" codec="H.264"><![CDATA[https://iab-publicfiles.s3.amazonaws.com/vast/VAST-4.0-Short-Intro-mid-resolution.mp4]]></MediaFile><MediaFile id="5246" delivery="progressive" type="video/mp4" bitrate="600" width="640" height="360" minBitrate="500" maxBitrate="700" scalable="1" maintainAspectRatio="1" codec="H.264"><![CDATA[https://iab-publicfiles.s3.amazonaws.com/vast/VAST-4.0-Short-Intro-low-resolution.mp4]]></MediaFile></MediaFiles></Linear></Creative></Creatives></InLine></Ad></VAST>`,
				NURL:    "",
			},
			want: `<VAST version="4.0" xmlns:_xmlns="xmlns" _xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns="http://www.iab.com/VAST"><Ad id="20001" sequence="1" conditionalAd="false"><InLine><AdSystem version="4.0"><![CDATA[iabtechlab]]></AdSystem><Error><![CDATA[http://example.com/error]]></Error><Error><![CDATA[http://errortracker.com?macro1=bid123&macro2=testbundle&macro3=testbundle&macro4=publishertestdomain&macro5=pageurltest&macro6=testpublisherID&macro6=1&macro7=1&macro8=1&macro9=&macro10=]]></Error><Impression id="Impression-ID"><![CDATA[http://example.com/track/impression]]></Impression><Impression><![CDATA[http://impressiontracker.com?macro1=bid123&macro2=testbundle&macro3=testbundle&macro4=publishertestdomain&macro5=pageurltest&macro6=testpublisherID&macro6=1&macro7=1&macro8=1&macro9=&macro10=]]></Impression><Pricing model="cpm" currency="USD"><![CDATA[25.00]]></Pricing><AdTitle><![CDATA[Inline Simple Ad]]></AdTitle><AdVerifications></AdVerifications><Advertiser><![CDATA[IAB Sample Company]]></Advertiser><Category authority="http://www.iabtechlab.com/categoryauthority"><![CDATA[AD CONTENT description category]]></Category><Creatives><Creative id="5480" sequence="1" adId="2447226"><UniversalAdId idRegistry="Ad-ID" idValue="8465"><![CDATA[8465]]></UniversalAdId><Linear><Duration><![CDATA[00:00:16]]></Duration><MediaFiles><MediaFile id="5241" delivery="progressive" type="video/mp4" bitrate="2000" width="1280" height="720" minBitrate="1500" maxBitrate="2500" scalable="1" maintainAspectRatio="1" codec="H.264"><![CDATA[https://iab-publicfiles.s3.amazonaws.com/vast/VAST-4.0-Short-Intro.mp4]]></MediaFile><MediaFile id="5244" delivery="progressive" type="video/mp4" bitrate="1000" width="854" height="480" minBitrate="700" maxBitrate="1500" scalable="1" maintainAspectRatio="1" codec="H.264"><![CDATA[https://iab-publicfiles.s3.amazonaws.com/vast/VAST-4.0-Short-Intro-mid-resolution.mp4]]></MediaFile><MediaFile id="5246" delivery="progressive" type="video/mp4" bitrate="600" width="640" height="360" minBitrate="500" maxBitrate="700" scalable="1" maintainAspectRatio="1" codec="H.264"><![CDATA[https://iab-publicfiles.s3.amazonaws.com/vast/VAST-4.0-Short-Intro-low-resolution.mp4]]></MediaFile></MediaFiles><VideoClicks><ClickTracking><![CDATA[http://videoclicktracker.com?macro1=bid123&macro2=testbundle&macro3=testbundle&macro4=publishertestdomain&macro5=pageurltest&macro6=testpublisherID&macro6=1&macro7=1&macro8=1&macro9=&macro10=]]></ClickTracking></VideoClicks><TrackingEvents><Tracking event="firstQuartile"><![CDATA[http://eventracker1.com?macro1=bid123&macro2=testbundle&macro3=testbundle&macro4=publishertestdomain&macro5=pageurltest&macro6=testpublisherID&macro6=1&macro7=1&macro8=1&macro9=&macro10=]]></Tracking></TrackingEvents></Linear></Creative></Creatives></InLine></Ad></VAST>`,
		},
		{
			name: "Non Linear vastXML, no existing event tracker",
			args: args{
				NURL:    "",
				vastXML: `<VAST version="4.0" xmlns="http://www.iab.com/VAST"><Ad id="20005" sequence="1" conditionalAd="false"><InLine><AdSystem version="4.0">iabtechlab</AdSystem><Extensions><Extension type="iab-Count"><total_available><![CDATA[ 2 ]]></total_available></Extension></Extensions><Pricing model="cpm" currency="USD"><![CDATA[ 25.00 ]]></Pricing><AdTitle><![CDATA[VAST 4.0 Pilot - Scenario 5]]></AdTitle><Creatives><Creative id="5480" sequence="1" adId="2447226"><UniversalAdId idRegistry="Ad-ID" idValue="8465">8465</UniversalAdId><NonLinearAds><NonLinear><StaticResource creativeType="image/png"><![CDATA[http://mms.businesswire.com/media/20150623005446/en/473787/21/iab_tech_lab.jpg]]></StaticResource></NonLinear></NonLinearAds></Creative></Creatives><Description><![CDATA[VAST 4.0 sample tag for Non Linear ad (i.e Overlay ad). Change the StaticResources to have a tag with your own content. Change NonLinear tag's parameters accordingly to view desired results.]]></Description></InLine></Ad></VAST>`,
			},
			want: `<VAST version="4.0" xmlns="http://www.iab.com/VAST"><Ad id="20005" sequence="1" conditionalAd="false"><InLine><AdSystem version="4.0"><![CDATA[iabtechlab]]></AdSystem><Extensions><Extension type="iab-Count"><total_available><![CDATA[2]]></total_available></Extension></Extensions><Pricing model="cpm" currency="USD"><![CDATA[25.00]]></Pricing><AdTitle><![CDATA[VAST 4.0 Pilot - Scenario 5]]></AdTitle><Creatives><Creative id="5480" sequence="1" adId="2447226"><UniversalAdId idRegistry="Ad-ID" idValue="8465"><![CDATA[8465]]></UniversalAdId><NonLinearAds><NonLinear><StaticResource creativeType="image/png"><![CDATA[http://mms.businesswire.com/media/20150623005446/en/473787/21/iab_tech_lab.jpg]]></StaticResource><NonLinearClickTracking><![CDATA[http://nonlinearclicktracker.com?macro1=bid123&macro2=testbundle&macro3=testbundle&macro4=publishertestdomain&macro5=pageurltest&macro6=testpublisherID&macro6=1&macro7=1&macro8=1&macro9=&macro10=]]></NonLinearClickTracking></NonLinear><TrackingEvents><Tracking event="firstQuartile"><![CDATA[http://eventracker1.com?macro1=bid123&macro2=testbundle&macro3=testbundle&macro4=publishertestdomain&macro5=pageurltest&macro6=testpublisherID&macro6=1&macro7=1&macro8=1&macro9=&macro10=]]></Tracking></TrackingEvents></NonLinearAds></Creative></Creatives><Description><![CDATA[VAST 4.0 sample tag for Non Linear ad (i.e Overlay ad). Change the StaticResources to have a tag with your own content. Change NonLinear tag's parameters accordingly to view desired results.]]></Description><Impression><![CDATA[http://impressiontracker.com?macro1=bid123&macro2=testbundle&macro3=testbundle&macro4=publishertestdomain&macro5=pageurltest&macro6=testpublisherID&macro6=1&macro7=1&macro8=1&macro9=&macro10=]]></Impression><Error><![CDATA[http://errortracker.com?macro1=bid123&macro2=testbundle&macro3=testbundle&macro4=publishertestdomain&macro5=pageurltest&macro6=testpublisherID&macro6=1&macro7=1&macro8=1&macro9=&macro10=]]></Error></InLine></Ad></VAST>`,
		},
		{
			name: "Wrapper Liner vastXML",
			args: args{
				NURL:    "",
				vastXML: `<VAST version="4.0" xmlns="http://www.iab.com/VAST"><Ad id="20011" sequence="1" conditionalAd="false"><Wrapper followAdditionalWrappers="0" allowMultipleAds="1" fallbackOnNoAd="0"><AdSystem version="4.0">iabtechlab</AdSystem><Error>http://example.com/error</Error><Impression id="Impression-ID">http://example.com/track/impression</Impression><Creatives><Creative id="5480" sequence="1" adId="2447226"><CompanionAds><Companion id="1232" width="100" height="150" assetWidth="250" assetHeight="200" expandedWidth="350" expandedHeight="250" apiFramework="VPAID" adSlotID="3214" pxratio="1400"><StaticResource creativeType="image/png"><![CDATA[https://www.iab.com/wp-content/uploads/2014/09/iab-tech-lab-6-644x290.png]]></StaticResource><CompanionClickThrough><![CDATA[https://iabtechlab.com]]></CompanionClickThrough></Companion></CompanionAds></Creative></Creatives><VASTAdTagURI><![CDATA[https://raw.githubusercontent.com/InteractiveAdvertisingBureau/VAST_Samples/master/VAST%204.0%20Samples/Inline_Companion_Tag-test.xml]]></VASTAdTagURI></Wrapper></Ad></VAST>`,
			},
			want: `<VAST version="4.0" xmlns="http://www.iab.com/VAST"><Ad id="20011" sequence="1" conditionalAd="false"><Wrapper followAdditionalWrappers="0" allowMultipleAds="1" fallbackOnNoAd="0"><AdSystem version="4.0"><![CDATA[iabtechlab]]></AdSystem><Error><![CDATA[http://example.com/error]]></Error><Error><![CDATA[http://errortracker.com?macro1=bid123&macro2=testbundle&macro3=testbundle&macro4=publishertestdomain&macro5=pageurltest&macro6=testpublisherID&macro6=1&macro7=1&macro8=1&macro9=&macro10=]]></Error><Impression id="Impression-ID"><![CDATA[http://example.com/track/impression]]></Impression><Impression><![CDATA[http://impressiontracker.com?macro1=bid123&macro2=testbundle&macro3=testbundle&macro4=publishertestdomain&macro5=pageurltest&macro6=testpublisherID&macro6=1&macro7=1&macro8=1&macro9=&macro10=]]></Impression><Creatives><Creative id="5480" sequence="1" adId="2447226"><CompanionAds><Companion id="1232" width="100" height="150" assetWidth="250" assetHeight="200" expandedWidth="350" expandedHeight="250" apiFramework="VPAID" adSlotID="3214" pxratio="1400"><StaticResource creativeType="image/png"><![CDATA[https://www.iab.com/wp-content/uploads/2014/09/iab-tech-lab-6-644x290.png]]></StaticResource><CompanionClickThrough><![CDATA[https://iabtechlab.com]]></CompanionClickThrough><CompanionClickThrough><![CDATA[http://companionclicktracker.com?macro1=bid123&macro2=testbundle&macro3=testbundle&macro4=publishertestdomain&macro5=pageurltest&macro6=testpublisherID&macro6=1&macro7=1&macro8=1&macro9=&macro10=]]></CompanionClickThrough></Companion></CompanionAds></Creative></Creatives><VASTAdTagURI><![CDATA[https://raw.githubusercontent.com/InteractiveAdvertisingBureau/VAST_Samples/master/VAST%204.0%20Samples/Inline_Companion_Tag-test.xml]]></VASTAdTagURI></Wrapper></Ad></VAST>`,
		},
		{
			name: "Wapper companion vastXML",
			args: args{
				NURL:    "",
				vastXML: `<VAST version="4.2" xmlns="http://www.iab.com/VAST"><Ad id="20011" sequence="1" ><Wrapper followAdditionalWrappers="0" allowMultipleAds="1" fallbackOnNoAd="0"><AdSystem version="4.0">iabtechlab</AdSystem><Error><![CDATA[https://example.com/error]]></Error><Impression id="Impression-ID"><![CDATA[https://example.com/track/impression]]></Impression><Creatives><Creative id="5480" sequence="1" adId="2447226"><CompanionAds><Companion id="1232" width="100" height="150" assetWidth="250" assetHeight="200" expandedWidth="350" expandedHeight="250" apiFramework="SIMID" adSlotId="3214" pxratio="1400"><StaticResource creativeType="image/png"><![CDATA[https://www.iab.com/wp-content/uploads/2014/09/iab-tech-lab-6-644x290.png]]></StaticResource><CompanionClickThrough><![CDATA[https://iabtechlab.com]]></CompanionClickThrough></Companion></CompanionAds></Creative></Creatives><VASTAdTagURI><![CDATA[https://raw.githubusercontent.com/InteractiveAdvertisingBureau/VAST_Samples/master/VAST%204.2%20Samples/Inline_Companion_Tag-test.xml]]></VASTAdTagURI></Wrapper></Ad></VAST>`,
			},
			want: `<VAST version="4.2" xmlns="http://www.iab.com/VAST"><Ad id="20011" sequence="1"><Wrapper followAdditionalWrappers="0" allowMultipleAds="1" fallbackOnNoAd="0"><AdSystem version="4.0"><![CDATA[iabtechlab]]></AdSystem><Error><![CDATA[https://example.com/error]]></Error><Error><![CDATA[http://errortracker.com?macro1=bid123&macro2=testbundle&macro3=testbundle&macro4=publishertestdomain&macro5=pageurltest&macro6=testpublisherID&macro6=1&macro7=1&macro8=1&macro9=&macro10=]]></Error><Impression id="Impression-ID"><![CDATA[https://example.com/track/impression]]></Impression><Impression><![CDATA[http://impressiontracker.com?macro1=bid123&macro2=testbundle&macro3=testbundle&macro4=publishertestdomain&macro5=pageurltest&macro6=testpublisherID&macro6=1&macro7=1&macro8=1&macro9=&macro10=]]></Impression><Creatives><Creative id="5480" sequence="1" adId="2447226"><CompanionAds><Companion id="1232" width="100" height="150" assetWidth="250" assetHeight="200" expandedWidth="350" expandedHeight="250" apiFramework="SIMID" adSlotId="3214" pxratio="1400"><StaticResource creativeType="image/png"><![CDATA[https://www.iab.com/wp-content/uploads/2014/09/iab-tech-lab-6-644x290.png]]></StaticResource><CompanionClickThrough><![CDATA[https://iabtechlab.com]]></CompanionClickThrough><CompanionClickThrough><![CDATA[http://companionclicktracker.com?macro1=bid123&macro2=testbundle&macro3=testbundle&macro4=publishertestdomain&macro5=pageurltest&macro6=testpublisherID&macro6=1&macro7=1&macro8=1&macro9=&macro10=]]></CompanionClickThrough></Companion></CompanionAds></Creative></Creatives><VASTAdTagURI><![CDATA[https://raw.githubusercontent.com/InteractiveAdvertisingBureau/VAST_Samples/master/VAST%204.2%20Samples/Inline_Companion_Tag-test.xml]]></VASTAdTagURI></Wrapper></Ad></VAST>`,
		},
		{
			name: "Wapper no companion vastXML",
			args: args{
				NURL:    "",
				vastXML: `<VAST version="4.2" xmlns="http://www.iab.com/VAST"><Ad id="20011" sequence="1" ><Wrapper followAdditionalWrappers="0" allowMultipleAds="1" fallbackOnNoAd="0"><AdSystem version="4.0">iabtechlab</AdSystem><Error><![CDATA[https://example.com/error]]></Error><Impression id="Impression-ID"><![CDATA[https://example.com/track/impression]]></Impression><Creatives><Creative id="5480" sequence="1" adId="2447226"><CompanionAds></CompanionAds></Creative></Creatives><VASTAdTagURI><![CDATA[https://raw.githubusercontent.com/InteractiveAdvertisingBureau/VAST_Samples/master/VAST%204.2%20Samples/Inline_Companion_Tag-test.xml]]></VASTAdTagURI></Wrapper></Ad></VAST>`,
			},
			want: `<VAST version="4.2" xmlns="http://www.iab.com/VAST"><Ad id="20011" sequence="1"><Wrapper followAdditionalWrappers="0" allowMultipleAds="1" fallbackOnNoAd="0"><AdSystem version="4.0"><![CDATA[iabtechlab]]></AdSystem><Error><![CDATA[https://example.com/error]]></Error><Error><![CDATA[http://errortracker.com?macro1=bid123&macro2=testbundle&macro3=testbundle&macro4=publishertestdomain&macro5=pageurltest&macro6=testpublisherID&macro6=1&macro7=1&macro8=1&macro9=&macro10=]]></Error><Impression id="Impression-ID"><![CDATA[https://example.com/track/impression]]></Impression><Impression><![CDATA[http://impressiontracker.com?macro1=bid123&macro2=testbundle&macro3=testbundle&macro4=publishertestdomain&macro5=pageurltest&macro6=testpublisherID&macro6=1&macro7=1&macro8=1&macro9=&macro10=]]></Impression><Creatives><Creative id="5480" sequence="1" adId="2447226"><CompanionAds><Companion><CompanionClickThrough><![CDATA[http://companionclicktracker.com?macro1=bid123&macro2=testbundle&macro3=testbundle&macro4=publishertestdomain&macro5=pageurltest&macro6=testpublisherID&macro6=1&macro7=1&macro8=1&macro9=&macro10=]]></CompanionClickThrough></Companion></CompanionAds></Creative></Creatives><VASTAdTagURI><![CDATA[https://raw.githubusercontent.com/InteractiveAdvertisingBureau/VAST_Samples/master/VAST%204.2%20Samples/Inline_Companion_Tag-test.xml]]></VASTAdTagURI></Wrapper></Ad></VAST>`,
		},
		{
			name: "Inline Non Linear empty",
			args: args{
				NURL:    "",
				vastXML: `<VAST version="4.2" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns="http://www.iab.com/VAST"><Ad id="20001" ><InLine><AdSystem version="1">iabtechlab</AdSystem><Pricing model="cpm" currency="USD"><![CDATA[ 25.00 ]]></Pricing><AdServingId>a532d16d-4d7f-4440-bd29-2ec0e693fc80</AdServingId><AdTitle>iabtechlab video ad</AdTitle><Creatives><Creative id="5480" sequence="1" adId="2447226"><NonLinearAds></NonLinearAds></Creative></Creatives></InLine></Ad></VAST>`,
			},
			want: `<VAST version="4.2" xmlns:_xmlns="xmlns" _xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns="http://www.iab.com/VAST"><Ad id="20001"><InLine><AdSystem version="1"><![CDATA[iabtechlab]]></AdSystem><Pricing model="cpm" currency="USD"><![CDATA[25.00]]></Pricing><AdServingId><![CDATA[a532d16d-4d7f-4440-bd29-2ec0e693fc80]]></AdServingId><AdTitle><![CDATA[iabtechlab video ad]]></AdTitle><Creatives><Creative id="5480" sequence="1" adId="2447226"><NonLinearAds><TrackingEvents><Tracking event="firstQuartile"><![CDATA[http://eventracker1.com?macro1=bid123&macro2=testbundle&macro3=testbundle&macro4=publishertestdomain&macro5=pageurltest&macro6=testpublisherID&macro6=1&macro7=1&macro8=1&macro9=&macro10=]]></Tracking></TrackingEvents></NonLinearAds></Creative></Creatives><Impression><![CDATA[http://impressiontracker.com?macro1=bid123&macro2=testbundle&macro3=testbundle&macro4=publishertestdomain&macro5=pageurltest&macro6=testpublisherID&macro6=1&macro7=1&macro8=1&macro9=&macro10=]]></Impression><Error><![CDATA[http://errortracker.com?macro1=bid123&macro2=testbundle&macro3=testbundle&macro4=publishertestdomain&macro5=pageurltest&macro6=testpublisherID&macro6=1&macro7=1&macro8=1&macro9=&macro10=]]></Error></InLine></Ad></VAST>`,
		},
		{
			name: "Wrapper linear and non linear",
			args: args{
				NURL:    "",
				vastXML: `<?xml version="1.0" encoding="UTF-8"?><VAST version="3.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="../../vast/vast3_draft.xsd"><Ad id="1" sequence="1"><Wrapper><AdSystem version="1.0">Test Ad Server</AdSystem><VASTAdTagURI><![CDATA[http://localhost/test/resources/vast/inlines/test_vast_inline_with-linear-ad.xml]]></VASTAdTagURI><Creatives><Creative><NonLinearAds></NonLinearAds></Creative><Creative><Linear><TrackingEvents><Tracking event="start"><![CDATA[http://example.com/start?d=[CACHEBUSTER]]]></Tracking><Tracking event="start"><![CDATA[http://example.com/start2?d=[CACHEBUSTER]]]></Tracking><Tracking event="firstQuartile"><![CDATA[http://example.com/q2?d=[CACHEBUSTER]]]></Tracking><Tracking event="midpoint"><![CDATA[http://example.com/q3?d=[CACHEBUSTER]]]></Tracking><Tracking event="thirdQuartile"><![CDATA[http://example.com/q4?d=[CACHEBUSTER]]]></Tracking><Tracking event="complete"><![CDATA[http://example.com/complete?d=[CACHEBUSTER]]]></Tracking></TrackingEvents><VideoClicks><ClickTracking id="video_click"><![CDATA[http://example.com/linear-video-click]]></ClickTracking><ClickTracking id="video_click"><![CDATA[http://example.com/linear-video-click2]]></ClickTracking><ClickTracking id="video_click"><![CDATA[http://example.com/linear-video-click3]]></ClickTracking><ClickTracking id="post_video_click"><![CDATA[http://example.com/linear-post-video-click]]></ClickTracking><ClickTracking id="post_video_click"><![CDATA[http://example.com/linear-post-video-click2]]></ClickTracking></VideoClicks></Linear></Creative></Creatives></Wrapper></Ad></VAST>`,
			},
			want: `<?xml version="1.0" encoding="UTF-8"?><VAST version="3.0" xmlns:_xmlns="xmlns" _xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsi="xsi" xsi:noNamespaceSchemaLocation="../../vast/vast3_draft.xsd"><Ad id="1" sequence="1"><Wrapper><AdSystem version="1.0"><![CDATA[Test Ad Server]]></AdSystem><VASTAdTagURI><![CDATA[http://localhost/test/resources/vast/inlines/test_vast_inline_with-linear-ad.xml]]></VASTAdTagURI><Creatives><Creative><NonLinearAds><TrackingEvents><Tracking event="firstQuartile"><![CDATA[http://eventracker1.com?macro1=bid123&macro2=testbundle&macro3=testbundle&macro4=publishertestdomain&macro5=pageurltest&macro6=testpublisherID&macro6=1&macro7=1&macro8=1&macro9=&macro10=]]></Tracking></TrackingEvents><NonLinear><NonLinearClickTracking><![CDATA[http://nonlinearclicktracker.com?macro1=bid123&macro2=testbundle&macro3=testbundle&macro4=publishertestdomain&macro5=pageurltest&macro6=testpublisherID&macro6=1&macro7=1&macro8=1&macro9=&macro10=]]></NonLinearClickTracking></NonLinear></NonLinearAds></Creative><Creative><Linear><TrackingEvents><Tracking event="firstQuartile"><![CDATA[http://eventracker1.com?macro1=bid123&macro2=testbundle&macro3=testbundle&macro4=publishertestdomain&macro5=pageurltest&macro6=testpublisherID&macro6=1&macro7=1&macro8=1&macro9=&macro10=]]></Tracking><Tracking event="start"><![CDATA[http://example.com/start?d=[CACHEBUSTER]]]></Tracking><Tracking event="start"><![CDATA[http://example.com/start2?d=[CACHEBUSTER]]]></Tracking><Tracking event="firstQuartile"><![CDATA[http://example.com/q2?d=[CACHEBUSTER]]]></Tracking><Tracking event="midpoint"><![CDATA[http://example.com/q3?d=[CACHEBUSTER]]]></Tracking><Tracking event="thirdQuartile"><![CDATA[http://example.com/q4?d=[CACHEBUSTER]]]></Tracking><Tracking event="complete"><![CDATA[http://example.com/complete?d=[CACHEBUSTER]]]></Tracking></TrackingEvents><VideoClicks><ClickTracking><![CDATA[http://videoclicktracker.com?macro1=bid123&macro2=testbundle&macro3=testbundle&macro4=publishertestdomain&macro5=pageurltest&macro6=testpublisherID&macro6=1&macro7=1&macro8=1&macro9=&macro10=]]></ClickTracking><ClickTracking id="video_click"><![CDATA[http://example.com/linear-video-click]]></ClickTracking><ClickTracking id="video_click"><![CDATA[http://example.com/linear-video-click2]]></ClickTracking><ClickTracking id="video_click"><![CDATA[http://example.com/linear-video-click3]]></ClickTracking><ClickTracking id="post_video_click"><![CDATA[http://example.com/linear-post-video-click]]></ClickTracking><ClickTracking id="post_video_click"><![CDATA[http://example.com/linear-post-video-click2]]></ClickTracking></VideoClicks></Linear></Creative></Creatives><Impression><![CDATA[http://impressiontracker.com?macro1=bid123&macro2=testbundle&macro3=testbundle&macro4=publishertestdomain&macro5=pageurltest&macro6=testpublisherID&macro6=1&macro7=1&macro8=1&macro9=&macro10=]]></Impression><Error><![CDATA[http://errortracker.com?macro1=bid123&macro2=testbundle&macro3=testbundle&macro4=publishertestdomain&macro5=pageurltest&macro6=testpublisherID&macro6=1&macro7=1&macro8=1&macro9=&macro10=]]></Error></Wrapper></Ad></VAST>`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ti.InjectTracker(tt.args.vastXML, tt.args.NURL)
			assert.Equal(t, tt.want, got, tt.name)
			if tt.wantError != nil {
				assert.EqualError(t, err, tt.wantError.Error())
			}
		})
	}
}

func TestAddClickTrackingEvent(t *testing.T) {
	tests := []struct {
		name         string
		addParentTag bool
		expected     string
	}{
		{
			name:         "With Parent Tag",
			addParentTag: true,
			expected:     "<VideoClicks><ClickTracking><![CDATA[http://videoclicktracker.com?macro1=bid123&macro2=testbundle&macro3=testbundle&macro4=publishertestdomain&macro5=pageurltest&macro6=testpublisherID&macro6=1&macro7=1&macro8=1&macro9=&macro10=]]></ClickTracking></VideoClicks>",
		},
		{
			name:         "Without Parent Tag",
			addParentTag: false,
			expected:     "<ClickTracking><![CDATA[http://videoclicktracker.com?macro1=bid123&macro2=testbundle&macro3=testbundle&macro4=publishertestdomain&macro5=pageurltest&macro6=testpublisherID&macro6=1&macro7=1&macro8=1&macro9=&macro10=]]></ClickTracking>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var outputXML strings.Builder
			b := macros.NewProvider(reqWrapper)
			b.PopulateBidMacros(&entities.PbsOrtbBid{
				Bid: &openrtb2.Bid{
					ID: "bid123",
				},
			}, "testSeat")
			ti := NewTrackerInjector(
				macros.NewStringIndexBasedReplacer(),
				b,
				VASTEvents{
					VideoClicks: []string{"http://videoclicktracker.com?macro1=##PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-APPBUNDLE##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro6=##PBS-LIMITADTRACKING##&macro7=##PBS-GDPRCONSENT##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1CUST1##&macro10=##PBS-MACRO-CUSTOMMACR1CUST2##"},
				},
			)
			ti.addClickTrackingEvent(&outputXML, "testCreativeId", tt.addParentTag)
			assert.Equal(t, tt.expected, outputXML.String(), tt.name)
		})
	}
}

func TestAddImpressionTrackingEvent(t *testing.T) {
	tests := []struct {
		name         string
		addParentTag bool
		expected     string
	}{
		{
			name:         "Add impression tag",
			addParentTag: true,
			expected:     "<Impression><![CDATA[http://impressiontracker.com?macro1=bid123&macro2=testbundle&macro3=testbundle&macro4=publishertestdomain&macro5=pageurltest&macro6=testpublisherID&macro6=1&macro7=1&macro8=1&macro9=&macro10=]]></Impression>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var outputXML strings.Builder
			b := macros.NewProvider(reqWrapper)
			b.PopulateBidMacros(&entities.PbsOrtbBid{
				Bid: &openrtb2.Bid{
					ID: "bid123",
				},
			}, "testSeat")
			ti := NewTrackerInjector(
				macros.NewStringIndexBasedReplacer(),
				b,
				VASTEvents{
					Impressions: []string{"http://impressiontracker.com?macro1=##PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-APPBUNDLE##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro6=##PBS-LIMITADTRACKING##&macro7=##PBS-GDPRCONSENT##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1CUST1##&macro10=##PBS-MACRO-CUSTOMMACR1CUST2##"},
				},
			)
			ti.addImpressionTrackingEvent(&outputXML)
			assert.Equal(t, tt.expected, outputXML.String(), tt.name)
		})
	}
}

func TestAddErrorTrackingEvent(t *testing.T) {
	tests := []struct {
		name         string
		addParentTag bool
		expected     string
	}{
		{
			name:         "Add impression tag",
			addParentTag: true,
			expected:     "<Error><![CDATA[http://errortracker.com?macro1=bid123&macro2=testbundle&macro3=testbundle&macro4=publishertestdomain&macro5=pageurltest&macro6=testpublisherID&macro6=1&macro7=1&macro8=1&macro9=&macro10=]]></Error>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var outputXML strings.Builder
			b := macros.NewProvider(reqWrapper)
			b.PopulateBidMacros(&entities.PbsOrtbBid{
				Bid: &openrtb2.Bid{
					ID: "bid123",
				},
			}, "testSeat")
			ti := NewTrackerInjector(
				macros.NewStringIndexBasedReplacer(),
				b,
				VASTEvents{
					Errors: []string{"http://errortracker.com?macro1=##PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-APPBUNDLE##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro6=##PBS-LIMITADTRACKING##&macro7=##PBS-GDPRCONSENT##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1CUST1##&macro10=##PBS-MACRO-CUSTOMMACR1CUST2##"},
				},
			)
			ti.addErrorTrackingEvent(&outputXML)
			assert.Equal(t, tt.expected, outputXML.String(), tt.name)
		})
	}
}

func TestAddNonLinearClickTrackingEvent(t *testing.T) {
	tests := []struct {
		name         string
		addParentTag bool
		expected     string
	}{
		{
			name:         "With Parent Tag",
			addParentTag: true,
			expected:     "<NonLinear><NonLinearClickTracking><![CDATA[http://nonlinearclicktracker.com?macro1=bid123&macro2=testbundle&macro3=testbundle&macro4=publishertestdomain&macro5=pageurltest&macro6=testpublisherID&macro6=1&macro7=1&macro8=1&macro9=&macro10=]]></NonLinearClickTracking></NonLinear>",
		},
		{
			name:         "Without Parent Tag",
			addParentTag: false,
			expected:     "<NonLinearClickTracking><![CDATA[http://nonlinearclicktracker.com?macro1=bid123&macro2=testbundle&macro3=testbundle&macro4=publishertestdomain&macro5=pageurltest&macro6=testpublisherID&macro6=1&macro7=1&macro8=1&macro9=&macro10=]]></NonLinearClickTracking>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var outputXML strings.Builder
			b := macros.NewProvider(reqWrapper)
			b.PopulateBidMacros(&entities.PbsOrtbBid{
				Bid: &openrtb2.Bid{
					ID: "bid123",
				},
			}, "testSeat")
			ti := NewTrackerInjector(
				macros.NewStringIndexBasedReplacer(),
				b,
				VASTEvents{
					NonLinearClickTracking: []string{"http://nonlinearclicktracker.com?macro1=##PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-APPBUNDLE##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro6=##PBS-LIMITADTRACKING##&macro7=##PBS-GDPRCONSENT##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1CUST1##&macro10=##PBS-MACRO-CUSTOMMACR1CUST2##"},
				},
			)
			ti.addNonLinearClickTrackingEvent(&outputXML, "testCreativeId", tt.addParentTag)
			assert.Equal(t, tt.expected, outputXML.String(), tt.name)
		})
	}
}

func TestAddCompanionClickThroughEvent(t *testing.T) {
	tests := []struct {
		name         string
		addParentTag bool
		expected     string
	}{
		{
			name:         "With Parent Tag",
			addParentTag: true,
			expected:     "<Companion><CompanionClickThrough><![CDATA[http://companionclicktracker.com?macro1=bid123&macro2=testbundle&macro3=testbundle&macro4=publishertestdomain&macro5=pageurltest&macro6=testpublisherID&macro6=1&macro7=1&macro8=1&macro9=&macro10=]]></CompanionClickThrough></Companion>",
		},
		{
			name:         "Without Parent Tag",
			addParentTag: false,
			expected:     "<CompanionClickThrough><![CDATA[http://companionclicktracker.com?macro1=bid123&macro2=testbundle&macro3=testbundle&macro4=publishertestdomain&macro5=pageurltest&macro6=testpublisherID&macro6=1&macro7=1&macro8=1&macro9=&macro10=]]></CompanionClickThrough>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var outputXML strings.Builder
			b := macros.NewProvider(reqWrapper)
			b.PopulateBidMacros(&entities.PbsOrtbBid{
				Bid: &openrtb2.Bid{
					ID: "bid123",
				},
			}, "testSeat")
			ti := NewTrackerInjector(
				macros.NewStringIndexBasedReplacer(),
				b,
				VASTEvents{
					CompanionClickThrough: []string{"http://companionclicktracker.com?macro1=##PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-APPBUNDLE##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro6=##PBS-LIMITADTRACKING##&macro7=##PBS-GDPRCONSENT##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1CUST1##&macro10=##PBS-MACRO-CUSTOMMACR1CUST2##"},
				},
			)
			ti.addCompanionClickThroughEvent(&outputXML, "testCreativeId", tt.addParentTag)
			assert.Equal(t, tt.expected, outputXML.String(), tt.name)
		})
	}
}

func TestAddTrackingEvent(t *testing.T) {
	tests := []struct {
		name         string
		addParentTag bool
		expected     string
	}{
		{
			name:         "With Parent Tag",
			addParentTag: true,
			expected:     "<TrackingEvents><Tracking event=\"firstQuartile\"><![CDATA[http://eventracker1.com?macro1=bid123&macro2=testbundle&macro3=testbundle&macro4=publishertestdomain&macro5=pageurltest&macro6=testpublisherID&macro6=1&macro7=1&macro8=1&macro9=&macro10=]]></Tracking></TrackingEvents>",
		},
		{
			name:         "Without Parent Tag",
			addParentTag: false,
			expected:     "<Tracking event=\"firstQuartile\"><![CDATA[http://eventracker1.com?macro1=bid123&macro2=testbundle&macro3=testbundle&macro4=publishertestdomain&macro5=pageurltest&macro6=testpublisherID&macro6=1&macro7=1&macro8=1&macro9=&macro10=]]></Tracking>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var outputXML strings.Builder
			b := macros.NewProvider(reqWrapper)
			b.PopulateBidMacros(&entities.PbsOrtbBid{
				Bid: &openrtb2.Bid{
					ID: "bid123",
				},
			}, "testSeat")
			ti := NewTrackerInjector(
				macros.NewStringIndexBasedReplacer(),
				b,
				VASTEvents{
					TrackingEvents: map[string][]string{"firstQuartile": {"http://eventracker1.com?macro1=##PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-APPBUNDLE##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro6=##PBS-LIMITADTRACKING##&macro7=##PBS-GDPRCONSENT##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1CUST1##&macro10=##PBS-MACRO-CUSTOMMACR1CUST2##"}},
				},
			)
			ti.addTrackingEvent(&outputXML, "testCreativeId", tt.addParentTag)
			assert.Equal(t, tt.expected, outputXML.String(), tt.name)
		})
	}
}

func TestWriteTrackingEvent(t *testing.T) {
	tests := []struct {
		name        string
		urls        []string
		startTag    string
		endTag      string
		creativeId  string
		eventType   string
		vastEvent   string
		expectedXML string
	}{
		{
			name:        "Single URL",
			urls:        []string{"http://tracker.com"},
			startTag:    "<Tracking>",
			endTag:      "</Tracking>",
			creativeId:  "123",
			eventType:   "start",
			vastEvent:   "tracking",
			expectedXML: "<Tracking>http://tracker.com</Tracking>",
		},
		{
			name:        "Multiple URL",
			urls:        []string{"http://tracker1.com", "http://tracker2.com"},
			startTag:    "<Tracking>",
			endTag:      "</Tracking>",
			creativeId:  "123",
			eventType:   "start",
			vastEvent:   "tracking",
			expectedXML: "<Tracking>http://tracker1.com</Tracking><Tracking>http://tracker2.com</Tracking>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var outputXML strings.Builder
			b := macros.NewProvider(reqWrapper)
			b.PopulateBidMacros(&entities.PbsOrtbBid{
				Bid: &openrtb2.Bid{
					ID: "bid123",
				},
			}, "testSeat")
			ti := NewTrackerInjector(
				macros.NewStringIndexBasedReplacer(),
				b,
				VASTEvents{
					TrackingEvents: map[string][]string{"firstQuartile": {"http://eventracker1.com?macro1=##PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-APPBUNDLE##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro6=##PBS-LIMITADTRACKING##&macro7=##PBS-GDPRCONSENT##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1CUST1##&macro10=##PBS-MACRO-CUSTOMMACR1CUST2##"}},
				},
			)
			ti.writeTrackingEvent(tt.urls, &outputXML, tt.startTag, tt.endTag, tt.creativeId, tt.eventType, tt.vastEvent)
			assert.Equal(t, tt.expectedXML, outputXML.String(), tt.name)
		})
	}
}
