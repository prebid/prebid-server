package vastbidder

import (
	"errors"
	"fmt"
	"sort"
	"testing"

	"github.com/beevik/etree"
	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestVASTTagResponseHandler_vastTagToBidderResponse(t *testing.T) {
	type args struct {
		internalRequest *openrtb2.BidRequest
		externalRequest *adapters.RequestData
		response        *adapters.ResponseData
		vastTag         *openrtb_ext.ExtImpVASTBidderTag
	}
	type want struct {
		bidderResponse *adapters.BidderResponse
		err            []error
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: `InlinePricingNode`,
			args: args{
				internalRequest: &openrtb2.BidRequest{
					ID: `request_id_1`,
					Imp: []openrtb2.Imp{
						{
							ID: `imp_id_1`,
						},
					},
				},
				externalRequest: &adapters.RequestData{
					Params: &adapters.BidRequestParams{
						ImpIndex: 0,
					},
				},
				response: &adapters.ResponseData{
					Body: []byte(`<VAST version="2.0"> <Ad id="1"> <InLine> <Creatives> <Creative sequence="1"> <Linear> <MediaFiles> <MediaFile><![CDATA[ad.mp4]]></MediaFile> </MediaFiles> </Linear> </Creative> </Creatives> <Extensions> <Extension type="LR-Pricing"> <Price model="CPM" currency="USD"><![CDATA[0.05]]></Price> </Extension> </Extensions> </InLine> </Ad> </VAST>`),
				},
				vastTag: &openrtb_ext.ExtImpVASTBidderTag{
					TagID:    "101",
					Duration: 15,
				},
			},
			want: want{
				bidderResponse: &adapters.BidderResponse{
					Bids: []*adapters.TypedBid{
						{
							Bid: &openrtb2.Bid{
								ID:    `1234`,
								ImpID: `imp_id_1`,
								Price: 0.05,
								AdM:   `<VAST version="2.0"> <Ad id="1"> <InLine> <Creatives> <Creative sequence="1"> <Linear> <MediaFiles> <MediaFile><![CDATA[ad.mp4]]></MediaFile> </MediaFiles> </Linear> </Creative> </Creatives> <Extensions> <Extension type="LR-Pricing"> <Price model="CPM" currency="USD"><![CDATA[0.05]]></Price> </Extension> </Extensions> </InLine> </Ad> </VAST>`,
								CrID:  "cr_1234",
							},
							BidType: openrtb_ext.BidTypeVideo,
							BidVideo: &openrtb_ext.ExtBidPrebidVideo{
								VASTTagID: "101",
								Duration:  15,
							},
						},
					},
					Currency: `USD`,
				},
			},
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewVASTTagResponseHandler()
			GetRandomID = func() string {
				return `1234`
			}
			handler.VASTTag = tt.args.vastTag

			bidderResponse, err := handler.vastTagToBidderResponse(tt.args.internalRequest, tt.args.externalRequest, tt.args.response)
			assert.Equal(t, tt.want.bidderResponse, bidderResponse)
			assert.Equal(t, tt.want.err, err)
		})
	}
}

//TestGetDurationInSeconds ...
// hh:mm:ss.mmm => 3:40:43.5 => 3 hours, 40 minutes, 43 seconds and 5 milliseconds
// => 3*60*60 + 40*60 + 43 + 5*0.001 => 10800 + 2400 + 43 + 0.005 => 13243.005
func TestGetDurationInSeconds(t *testing.T) {
	type args struct {
		creativeTag string // ad element
	}
	type want struct {
		duration int // seconds  (will converted from string with format as  HH:MM:SS.mmm)
		err      error
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		// duration validation tests
		{name: "duration 00:00:25 (= 25 seconds)", want: want{duration: 25}, args: args{creativeTag: `<Creative sequence="1"> <Linear> <Duration>00:00:25</Duration> </Linear> </Creative>`}},
		{name: "duration 00:00:-25 (= -25 seconds)", want: want{err: errors.New("Invalid Duration")}, args: args{creativeTag: `<Creative sequence="1"> <Linear> <Duration>00:00:-25</Duration> </Linear> </Creative>`}},
		{name: "duration 00:00:30.999 (= 30.990 seconds (int -> 30 seconds))", want: want{duration: 30}, args: args{creativeTag: `<Creative sequence="1"> <Linear> <Duration>00:00:30.999</Duration> </Linear> </Creative>`}},
		{name: "duration 00:01:08 (1 min 8 seconds = 68 seconds)", want: want{duration: 68}, args: args{creativeTag: `<Creative sequence="1"> <Linear> <Duration>00:01:08</Duration> </Linear> </Creative>`}},
		{name: "duration 02:13:12 (2 hrs 13 min  12 seconds) = 7992 seconds)", want: want{duration: 7992}, args: args{creativeTag: `<Creative sequence="1"> <Linear> <Duration>02:13:12</Duration> </Linear> </Creative>`}},
		{name: "duration 3:40:43.5 (3 hrs 40 min  43 seconds 5 ms) = 6043.005 seconds (int -> 6043 seconds))", want: want{duration: 13243}, args: args{creativeTag: `<Creative sequence="1"> <Linear> <Duration>3:40:43.5</Duration> </Linear> </Creative>`}},
		{name: "duration 00:00:25.0005458 (0 hrs 0 min  25 seconds 0005458 ms) - invalid max ms is 999", want: want{err: errors.New("Invalid Duration")}, args: args{creativeTag: `<Creative sequence="1"> <Linear> <Duration>00:00:25.0005458</Duration> </Linear> </Creative>`}},
		{name: "invalid duration 3:13:900 (3 hrs 13 min  900 seconds) = Invalid seconds )", want: want{err: errors.New("Invalid Duration")}, args: args{creativeTag: `<Creative sequence="1"> <Linear> <Duration>3:13:900</Duration> </Linear> </Creative>`}},
		{name: "invalid duration 3:13:34:44 (3 hrs 13 min 34 seconds :44=invalid) = ?? )", want: want{err: errors.New("Invalid Duration")}, args: args{creativeTag: `<Creative sequence="1"> <Linear> <Duration>3:13:34:44</Duration> </Linear> </Creative>`}},
		{name: "duration = 0:0:45.038 , with milliseconds duration (0 hrs 0 min 45 seconds and 038 millseconds) = 45.038 seconds (int -> 45 seconds) )", want: want{duration: 45}, args: args{creativeTag: `<Creative sequence="1"> <Linear> <Duration>0:0:45.038</Duration> </Linear> </InLine> </Creative>`}},
		{name: "duration = 0:0:48.50  = 48.050 seconds (int -> 48 seconds))", want: want{duration: 48}, args: args{creativeTag: `<Creative sequence="1"> <Linear> <Duration>0:0:48.50</Duration> </Linear> </InLine> </Creative>`}},
		{name: "duration = 0:0:28.59  = 28.059 seconds  (int -> 28 seconds))", want: want{duration: 28}, args: args{creativeTag: `<Creative sequence="1"> <Linear> <Duration>0:0:28.59</Duration> </Linear> </InLine> </Creative>`}},
		{name: "duration = 56 (ambiguity w.r.t. HH:MM:SS.mmm format)", want: want{err: errors.New("Invalid Duration")}, args: args{creativeTag: `<Creative sequence="1"> <Linear> <Duration>56</Duration> </Linear> </Creative>`}},
		{name: "duration = :56 (ambiguity w.r.t. HH:MM:SS.mmm format)", want: want{err: errors.New("Invalid Duration")}, args: args{creativeTag: `<Creative sequence="1"> <Linear> <Duration>:56</Duration> </Linear> </Creative>`}},
		{name: "duration = :56: (ambiguity w.r.t. HH:MM:SS.mmm format)", want: want{err: errors.New("Invalid Duration")}, args: args{creativeTag: `<Creative sequence="1"> <Linear> <Duration>:56:</Duration> </Linear> </Creative>`}},
		{name: "duration = ::56 (ambiguity w.r.t. HH:MM:SS.mmm format)", want: want{err: errors.New("Invalid Duration")}, args: args{creativeTag: `<Creative sequence="1"> <Linear> <Duration>::56</Duration> </Linear> </Creative>`}},
		{name: "duration = 56.445 (ambiguity w.r.t. HH:MM:SS.mmm format)", want: want{err: errors.New("Invalid Duration")}, args: args{creativeTag: `<Creative sequence="1"> <Linear> <Duration>56.445</Duration> </Linear> </Creative>`}},
		{name: "duration = a:b:c.d (no numbers)", want: want{err: errors.New("Invalid Duration")}, args: args{creativeTag: `<Creative sequence="1"> <Linear> <Duration>a:b:c.d</Duration> </Linear> </Creative>`}},

		// tag validations tests
		{name: "Linear Creative no duration", want: want{err: errors.New("Invalid Duration")}, args: args{creativeTag: `<Creative><Linear><Linear></Creative>`}},
		{name: "Companion Creative no duration", want: want{err: errors.New("Invalid Duration")}, args: args{creativeTag: `<Creative><CompanionAds></CompanionAds></Creative>`}},
		{name: "Non-Linear Creative no duration", want: want{err: errors.New("Invalid Duration")}, args: args{creativeTag: `<Creative><NonLinearAds></NonLinearAds></Creative>`}},
		{name: "Invalid Creative tag", want: want{err: errors.New("Invalid Creative")}, args: args{creativeTag: `<Ad></Ad>`}},
		{name: "Nil Creative tag", want: want{err: errors.New("Invalid Creative")}, args: args{creativeTag: ""}},

		// multiple linear tags in creative
		{name: "Multiple Linear Ads within Creative", want: want{duration: 25}, args: args{creativeTag: `<Creative><Linear><Duration>0:0:25<Duration></Linear><Linear><Duration>0:0:30<Duration></Linear></Creative>`}},
		// Case sensitivity check - passing DURATION (vast is case-sensitive as per https://vastvalidator.iabtechlab.com/dash)
		{name: "<DURATION> all caps", want: want{err: errors.New("Invalid Duration")}, args: args{creativeTag: `<Creative><Linear><DURATION>0:0:10</Duration></Linear></Creative>`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := etree.NewDocument()
			doc.ReadFromString(tt.args.creativeTag)
			dur, err := getDuration(doc.FindElement("./Creative"))
			assert.Equal(t, tt.want.duration, dur)
			assert.Equal(t, tt.want.err, err)
			// if error expects 0 value for duration
			if nil != err {
				assert.Equal(t, 0, dur)
			}
		})
	}
}

func BenchmarkGetDuration(b *testing.B) {
	doc := etree.NewDocument()
	doc.ReadFromString(`<Creative sequence="1"> <Linear> <Duration>0:0:56.3</Duration> </Linear> </Creative>`)
	creative := doc.FindElement("/Creative")
	for n := 0; n < b.N; n++ {
		getDuration(creative)
	}
}

func TestGetCreativeId(t *testing.T) {
	type args struct {
		creativeTag string // ad element
	}
	type want struct {
		id string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{name: "creative tag with id", want: want{id: "233ff44"}, args: args{creativeTag: `<Creative id="233ff44"></Creative>`}},
		{name: "creative tag without id", want: want{id: ""}, args: args{creativeTag: `<Creative></Creative>`}},
		{name: "no creative tag", want: want{id: ""}, args: args{creativeTag: ""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := etree.NewDocument()
			doc.ReadFromString(tt.args.creativeTag)
			id := getCreativeID(doc.FindElement("./Creative"))
			assert.Equal(t, tt.want.id, id)
		})
	}
}

func BenchmarkGetCreativeID(b *testing.B) {
	doc := etree.NewDocument()
	doc.ReadFromString(`<Creative id="132324eerr">  </Creative>`)
	creative := doc.FindElement("/Creative")
	for n := 0; n < b.N; n++ {
		getCreativeID(creative)
	}
}

func TestGetAdvertisers(t *testing.T) {
	tt := []struct {
		name     string
		vastStr  string
		expected []string
	}{
		{
			name: "vast_4_with_advertiser",
			vastStr: `<VAST version="4.2" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns="http://www.iab.com/VAST">
  						<Ad id="20001" sequence="1" >
    						<InLine>
      							<Advertiser>www.iabtechlab.com</Advertiser>
    						</InLine>
  						</Ad>
					  </VAST>`,
			expected: []string{"www.iabtechlab.com"},
		},
		{
			name: "vast_4_without_advertiser",
			vastStr: `<VAST version="4.2" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns="http://www.iab.com/VAST">
  						<Ad id="20001" sequence="1" >
    						<InLine>
    						</InLine>
  						</Ad>
					  </VAST>`,
			expected: []string{},
		},
		{
			name: "vast_4_with_empty_advertiser",
			vastStr: `<VAST version="4.2" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns="http://www.iab.com/VAST">
  						<Ad id="20001" sequence="1" >
    						<InLine>
								<Advertiser></Advertiser>
    						</InLine>
  						</Ad>
					  </VAST>`,
			expected: []string{},
		},
		{
			name: "vast_2_with_single_advertiser",
			vastStr: `<VAST version="2.0" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns="http://www.iab.com/VAST">
  						<Ad id="20001" sequence="1" >
    						<InLine>
								<Extensions>
									<Extension type="advertiser">
										<Advertiser>google.com</Advertiser>
									</Extension>
								</Extensions>
    						</InLine>
  						</Ad>
					  </VAST>`,
			expected: []string{"google.com"},
		},
		{
			name: "vast_2_with_two_advertiser",
			vastStr: `<VAST version="2.0" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns="http://www.iab.com/VAST">
  						<Ad id="20001" sequence="1" >
    						<InLine>
								<Extensions>
									<Extension type="advertiser">
										<Advertiser>google.com</Advertiser>
									</Extension>
									<Extension type="advertiser">
										<Advertiser>facebook.com</Advertiser>
									</Extension>
								</Extensions>
    						</InLine>
  						</Ad>
					  </VAST>`,
			expected: []string{"google.com", "facebook.com"},
		},
		{
			name: "vast_2_with_no_advertiser",
			vastStr: `<VAST version="2.0" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns="http://www.iab.com/VAST">
  						<Ad id="20001" sequence="1" >
    						<InLine>
    						</InLine>
  						</Ad>
					  </VAST>`,
			expected: []string{},
		},
		{
			name: "vast_2_with_epmty_advertiser",
			vastStr: `<VAST version="2.0" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns="http://www.iab.com/VAST">
  						<Ad id="20001" sequence="1" >
    						<InLine>
								<Extensions>
									<Extension type="advertiser">
										<Advertiser></Advertiser>
									</Extension>
								</Extensions>
    						</InLine>
  						</Ad>
					  </VAST>`,
			expected: []string{},
		},
		{
			name: "vast_3_with_single_advertiser",
			vastStr: `<VAST version="3.1" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns="http://www.iab.com/VAST">
  						<Ad id="20001" sequence="1" >
    						<InLine>
								<Extensions>
									<Extension type="advertiser">
										<Advertiser>google.com</Advertiser>
									</Extension>
								</Extensions>
    						</InLine>
  						</Ad>
					  </VAST>`,
			expected: []string{"google.com"},
		},
		{
			name: "vast_3_with_two_advertiser",
			vastStr: `<VAST version="3.2" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns="http://www.iab.com/VAST">
  						<Ad id="20001" sequence="1" >
    						<InLine>
								<Extensions>
									<Extension type="advertiser">
										<Advertiser>google.com</Advertiser>
									</Extension>
									<Extension type="advertiser">
										<Advertiser>facebook.com</Advertiser>
									</Extension>
								</Extensions>
    						</InLine>
  						</Ad>
					  </VAST>`,
			expected: []string{"google.com", "facebook.com"},
		},
		{
			name: "vast_3_with_no_advertiser",
			vastStr: `<VAST version="3.0" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns="http://www.iab.com/VAST">
  						<Ad id="20001" sequence="1" >
    						<InLine>
    						</InLine>
  						</Ad>
					  </VAST>`,
			expected: []string{},
		},
		{
			name: "vast_3_with_epmty_advertiser",
			vastStr: `<VAST version="3.1" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns="http://www.iab.com/VAST">
  						<Ad id="20001" sequence="1" >
    						<InLine>
								<Extensions>
									<Extension type="advertiser">
										<Advertiser></Advertiser>
									</Extension>
								</Extensions>
    						</InLine>
  						</Ad>
					  </VAST>`,
			expected: []string{},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			doc := etree.NewDocument()
			if err := doc.ReadFromString(tc.vastStr); err != nil {
				t.Errorf("Failed to create etree doc from string %+v", err)
			}

			vastDoc := doc.FindElement("./VAST")
			vastVer := vastDoc.SelectAttrValue(`version`, `2.0`)

			ad := getAdElement(vastDoc)

			result := getAdvertisers(vastVer, ad)

			sort.Strings(result)
			sort.Strings(tc.expected)

			if !assert.Equal(t, len(tc.expected), len(result), fmt.Sprintf("Expected slice length - %+v \nResult slice length - %+v", len(tc.expected), len(result))) {
				return
			}

			for i, expected := range tc.expected {
				assert.Equal(t, expected, result[i], fmt.Sprintf("Element mismatch at position %d.\nExpected - %s\nActual - %s", i, expected, result[i]))
			}
		})
	}
}
