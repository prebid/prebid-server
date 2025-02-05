package avocet

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAvocet, config.Adapter{
		Endpoint: "https://bid.staging.avct.cloud/ortb/bid/5e722ee9bd6df11d063a8013"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "avocettest", bidder)
}

func TestAvocetAdapter_MakeRequests(t *testing.T) {
	type fields struct {
		Endpoint string
	}
	type args struct {
		request *openrtb2.BidRequest
		reqInfo *adapters.ExtraRequestInfo
	}
	type reqData []*adapters.RequestData
	tests := []struct {
		name     string
		fields   fields
		args     args
		want     []*adapters.RequestData
		wantErrs []error
	}{
		{
			name:   "return nil if zero imps",
			fields: fields{Endpoint: "https://bid.avct.cloud"},
			args: args{
				&openrtb2.BidRequest{},
				nil,
			},
			want:     nil,
			wantErrs: nil,
		},
		{
			name:   "makes POST request with JSON content",
			fields: fields{Endpoint: "https://bid.avct.cloud"},
			args: args{
				&openrtb2.BidRequest{Imp: []openrtb2.Imp{{}}},
				nil,
			},
			want: reqData{
				&adapters.RequestData{
					Method: http.MethodPost,
					Uri:    "https://bid.avct.cloud",
					Body:   []byte(`{"id":"","imp":[{"id":""}]}`),
					Headers: map[string][]string{
						"Accept":       {"application/json"},
						"Content-Type": {"application/json;charset=utf-8"},
					},
					ImpIDs: []string{""},
				},
			},
			wantErrs: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AvocetAdapter{
				Endpoint: tt.fields.Endpoint,
			}
			got, got1 := a.MakeRequests(tt.args.request, tt.args.reqInfo)
			if len(got) != len(tt.want) {
				t.Errorf("AvocetAdapter.MakeRequests() got %v requests, wanted %v requests", len(got), len(tt.want))
			}
			if len(got) == len(tt.want) {
				for i := range tt.want {
					if !reflect.DeepEqual(got[i], tt.want[i]) {
						t.Errorf("AvocetAdapter.MakeRequests() got = %v, want %v", got[i], tt.want[i])
					}
				}
			}
			if !reflect.DeepEqual(got1, tt.wantErrs) {
				t.Errorf("AvocetAdapter.MakeRequests() got1 = %v, want %v", got1, tt.wantErrs)
			}
		})
	}
}

func TestAvocetAdapter_MakeBids(t *testing.T) {
	type fields struct {
		Endpoint string
	}
	type args struct {
		internalRequest *openrtb2.BidRequest
		externalRequest *adapters.RequestData
		response        *adapters.ResponseData
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *adapters.BidderResponse
		errs   []error
	}{
		{
			name:   "204 No Content indicates no bids",
			fields: fields{Endpoint: "https://bid.avct.cloud"},
			args: args{
				nil,
				nil,
				&adapters.ResponseData{StatusCode: http.StatusNoContent},
			},
			want: nil,
			errs: nil,
		},
		{
			name:   "Non-200 return error",
			fields: fields{Endpoint: "https://bid.avct.cloud"},
			args: args{
				nil,
				nil,
				&adapters.ResponseData{StatusCode: http.StatusBadRequest, Body: []byte("message")},
			},
			want: nil,
			errs: []error{&errortypes.BadServerResponse{Message: "received status code: 400 error: message"}},
		},
		{
			name:   "200 response containing banner bids",
			fields: fields{Endpoint: "https://bid.avct.cloud"},
			args: args{
				nil,
				nil,
				&adapters.ResponseData{StatusCode: http.StatusOK, Body: validBannerBidResponseBody},
			},
			want: &adapters.BidderResponse{
				Currency: "USD",
				Bids: []*adapters.TypedBid{
					{
						Bid:     &validBannerBid,
						BidType: openrtb_ext.BidTypeBanner,
					},
				},
			},
			errs: nil,
		},
		{
			name:   "200 response containing video bids",
			fields: fields{Endpoint: "https://bid.avct.cloud"},
			args: args{
				nil,
				nil,
				&adapters.ResponseData{StatusCode: http.StatusOK, Body: validVideoBidResponseBody},
			},
			want: &adapters.BidderResponse{
				Currency: "USD",
				Bids: []*adapters.TypedBid{
					{
						Bid:     &validVideoBid,
						BidType: openrtb_ext.BidTypeVideo,
						BidVideo: &openrtb_ext.ExtBidPrebidVideo{
							Duration: 30,
						},
					},
				},
			},
			errs: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AvocetAdapter{
				Endpoint: tt.fields.Endpoint,
			}
			got, got1 := a.MakeBids(tt.args.internalRequest, tt.args.externalRequest, tt.args.response)
			if !reflect.DeepEqual(got, tt.want) {
				gotb, _ := json.Marshal(got)
				wantb, _ := json.Marshal(tt.want)
				t.Errorf("AvocetAdapter.MakeBids() got = %s, want %s", string(gotb), string(wantb))
			}
			if !reflect.DeepEqual(got1, tt.errs) {
				t.Errorf("AvocetAdapter.MakeBids() got1 = %v, want %v", got1, tt.errs)
			}
		})
	}
}

func Test_getBidType(t *testing.T) {
	type args struct {
		bid openrtb2.Bid
		ext avocetBidExt
	}
	tests := []struct {
		name string
		args args
		want openrtb_ext.BidType
	}{
		{
			name: "VPAID 1.0",
			args: args{openrtb2.Bid{API: adcom1.APIVPAID10}, avocetBidExt{}},
			want: openrtb_ext.BidTypeVideo,
		},
		{
			name: "VPAID 2.0",
			args: args{openrtb2.Bid{API: adcom1.APIVPAID20}, avocetBidExt{}},
			want: openrtb_ext.BidTypeVideo,
		},
		{
			name: "other",
			args: args{openrtb2.Bid{}, avocetBidExt{}},
			want: openrtb_ext.BidTypeBanner,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getBidType(tt.args.bid, tt.args.ext); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getBidType() = %v, want %v", got, tt.want)
			}
		})
	}
}

var validBannerBid = openrtb2.Bid{
	AdM:      "<iframe src=\"http://ads.staging.avct.cloud/sv?pp=${AUCTION_PRICE}&uuid=0df2c449-6d85-4179-b5d5-37f2f91caa24&ty=h&crid=5b51e49634f2021f127ff7c9&tacid=5b51e4ed89654741306813a8&aid=749d36d7-c993-455f-aefd-ffd8a7e3ccf_0&accid=5b51dd1634f2021f127ff7c0&brid=5b51e20f34f2021f127ff7c4&ioid=5b51e22089654741306813a1&caid=5b51e2d689654741306813a4&it=1&iobsid=496e8cff35b2c0110029534d&ext_aid=749d36d7-c993-455f-aefd-ffd8a7e3ccf_0&bp=15.64434783&bt=1591874537316649768&h=250&w=300&vpr=0&vdp=0&domain=example.com&gco=54510b3b816269000061a0f7&stid=542d2c1615e3c013de53a6e2&glat=0&glong=0&bip4=3232238090&ext_siid=5ea89200c865f911007f1b0e&ext_pid=1&ext_sid=5ea84df8c865f911007f1ade&ext_plid=5ea9601ac865f911007f1b6a&optv=latest:latest&invsrc=5e722ee9bd6df11d063a8013&ug=0d&ca=0&biid=requestd-54644474bf-l7gx4|eu-central-1-staging&reg=eu-central-1&ck=1_5d99a849\" height=\"250\" width=\"300\" marginwidth=0 marginheight=0 hspace=0 vspace=0 frameborder=0 scrolling=\"no\"></iframe>",
	ADomain:  []string{"avocet.io"},
	CID:      "5b51e2d689654741306813a4",
	CrID:     "5b51e49634f2021f127ff7c9",
	H:        250,
	ID:       "bc708396-9202-437b-b726-08b9864cb8b8",
	ImpID:    "test-imp-id",
	IURL:     "https://cdn.staging.avocet.io/snapshots/5b51dd1634f2021f127ff7c0/5b51e49634f2021f127ff7c9.jpeg",
	Language: "en",
	Price:    15.64434783,
	W:        300,
}

var validBannerBidResponseBody = []byte(`{
	"bidid": "dd87f80c-16a0-43c8-a673-b94b3ea4d417",
	"id": "test-request-id",
	"seatbid": [
		{
			"bid": [
				{
					"adm": "<iframe src=\"http://ads.staging.avct.cloud/sv?pp=${AUCTION_PRICE}&uuid=0df2c449-6d85-4179-b5d5-37f2f91caa24&ty=h&crid=5b51e49634f2021f127ff7c9&tacid=5b51e4ed89654741306813a8&aid=749d36d7-c993-455f-aefd-ffd8a7e3ccf_0&accid=5b51dd1634f2021f127ff7c0&brid=5b51e20f34f2021f127ff7c4&ioid=5b51e22089654741306813a1&caid=5b51e2d689654741306813a4&it=1&iobsid=496e8cff35b2c0110029534d&ext_aid=749d36d7-c993-455f-aefd-ffd8a7e3ccf_0&bp=15.64434783&bt=1591874537316649768&h=250&w=300&vpr=0&vdp=0&domain=example.com&gco=54510b3b816269000061a0f7&stid=542d2c1615e3c013de53a6e2&glat=0&glong=0&bip4=3232238090&ext_siid=5ea89200c865f911007f1b0e&ext_pid=1&ext_sid=5ea84df8c865f911007f1ade&ext_plid=5ea9601ac865f911007f1b6a&optv=latest:latest&invsrc=5e722ee9bd6df11d063a8013&ug=0d&ca=0&biid=requestd-54644474bf-l7gx4|eu-central-1-staging&reg=eu-central-1&ck=1_5d99a849\" height=\"250\" width=\"300\" marginwidth=0 marginheight=0 hspace=0 vspace=0 frameborder=0 scrolling=\"no\"></iframe>",
					"adomain": ["avocet.io"],
					"cid": "5b51e2d689654741306813a4",
					"crid": "5b51e49634f2021f127ff7c9",
					"h": 250,
					"id": "bc708396-9202-437b-b726-08b9864cb8b8",
					"impid": "test-imp-id",
					"iurl": "https://cdn.staging.avocet.io/snapshots/5b51dd1634f2021f127ff7c0/5b51e49634f2021f127ff7c9.jpeg",
					"language": "en",
					"price": 15.64434783,
					"w": 300
				}
			],
			"seat": "TEST_SEAT_ID"
		}
	]
}`)

var validVideoBid = openrtb2.Bid{
	AdM:      "<VAST version=\"3.0\"><Ad id=\"5ec530e32d57fe1100f17d87\"><Wrapper><AdSystem>Avocet</AdSystem><VASTAdTagURI><![CDATA[http://ads.staging.avct.cloud/vast?x=1&uuid=0df2c449-6d85-4179-b5d5-37f2f91caa24&ty=h&crid=5ec530e32d57fe1100f17d87&tacid=5ec531d32d57fe1100f17d89&aid=749d36d7-c993-455f-aefd-ffd8a7e3ccf_0&accid=5b51dd1634f2021f127ff7c0&brid=5b51e20f34f2021f127ff7c4&ioid=5b51e22089654741306813a1&caid=5b51e2d689654741306813a4&it=2&iobsid=496e8cff35b2c0110029534d&ext_aid=749d36d7-c993-455f-aefd-ffd8a7e3ccf_0&bp=15.64434783&bt=1591875033134290518&h=396&w=600&vpr=0&vdp=0&domain=example.com&gco=54510b3b816269000061a0f7&stid=542d2c1615e3c013de53a6e2&glat=0&glong=0&bip4=3232238090&ext_siid=5ea89200c865f911007f1b0e&ext_pid=1&ext_sid=5ea84df8c865f911007f1ade&ext_plid=5ea9601ac865f911007f1b6a&optv=latest:latest&invsrc=5e722ee9bd6df11d063a8013&ug=0d&ca=0&biid=requestd-54644474bf-l7gx4|eu-central-1-staging&reg=eu-central-1&pixel=1&ck=1_c343bf14]]></VASTAdTagURI><Impression><![CDATA[http://ads.staging.avct.cloud/sv?pp=${AUCTION_PRICE}&uuid=0df2c449-6d85-4179-b5d5-37f2f91caa24&ty=h&crid=5ec530e32d57fe1100f17d87&tacid=5ec531d32d57fe1100f17d89&aid=749d36d7-c993-455f-aefd-ffd8a7e3ccf_0&accid=5b51dd1634f2021f127ff7c0&brid=5b51e20f34f2021f127ff7c4&ioid=5b51e22089654741306813a1&caid=5b51e2d689654741306813a4&it=2&iobsid=496e8cff35b2c0110029534d&ext_aid=749d36d7-c993-455f-aefd-ffd8a7e3ccf_0&bp=15.64434783&bt=1591875033134290518&h=396&w=600&vpr=0&vdp=0&domain=example.com&gco=54510b3b816269000061a0f7&stid=542d2c1615e3c013de53a6e2&glat=0&glong=0&bip4=3232238090&ext_siid=5ea89200c865f911007f1b0e&ext_pid=1&ext_sid=5ea84df8c865f911007f1ade&ext_plid=5ea9601ac865f911007f1b6a&optv=latest:latest&invsrc=5e722ee9bd6df11d063a8013&ug=0d&ca=0&biid=requestd-54644474bf-l7gx4|eu-central-1-staging&reg=eu-central-1&pixel=1&ck=1_c343bf14]]></Impression><Creatives><Creative AdId=\"5ec530e32d57fe1100f17d87\"><Linear><TrackingEvents></TrackingEvents><VideoClicks></VideoClicks></Linear></Creative></Creatives></Wrapper></Ad></VAST>",
	ADomain:  []string{"avocet.io"},
	CID:      "5b51e2d689654741306813a4",
	CrID:     "5ec530e32d57fe1100f17d87",
	H:        396,
	ID:       "3d4c2d45-5a8c-43b8-9e15-4f48ac45204f",
	ImpID:    "dfp-ad--top-above-nav",
	IURL:     "https://cdn.staging.avocet.io/snapshots/5b51dd1634f2021f127ff7c0/5ec530e32d57fe1100f17d87.jpeg",
	Language: "en",
	Price:    15.64434783,
	W:        600,
	Ext:      []byte(`{"avocet":{"duration":30}}`),
}

var validVideoBidResponseBody = []byte(`{
	"bidid": "dd87f80c-16a0-43c8-a673-b94b3ea4d417",
	"id": "test-request-id",
	"seatbid": [
		{
			"bid": [
				{
					"adm": "<VAST version=\"3.0\"><Ad id=\"5ec530e32d57fe1100f17d87\"><Wrapper><AdSystem>Avocet</AdSystem><VASTAdTagURI><![CDATA[http://ads.staging.avct.cloud/vast?x=1&uuid=0df2c449-6d85-4179-b5d5-37f2f91caa24&ty=h&crid=5ec530e32d57fe1100f17d87&tacid=5ec531d32d57fe1100f17d89&aid=749d36d7-c993-455f-aefd-ffd8a7e3ccf_0&accid=5b51dd1634f2021f127ff7c0&brid=5b51e20f34f2021f127ff7c4&ioid=5b51e22089654741306813a1&caid=5b51e2d689654741306813a4&it=2&iobsid=496e8cff35b2c0110029534d&ext_aid=749d36d7-c993-455f-aefd-ffd8a7e3ccf_0&bp=15.64434783&bt=1591875033134290518&h=396&w=600&vpr=0&vdp=0&domain=example.com&gco=54510b3b816269000061a0f7&stid=542d2c1615e3c013de53a6e2&glat=0&glong=0&bip4=3232238090&ext_siid=5ea89200c865f911007f1b0e&ext_pid=1&ext_sid=5ea84df8c865f911007f1ade&ext_plid=5ea9601ac865f911007f1b6a&optv=latest:latest&invsrc=5e722ee9bd6df11d063a8013&ug=0d&ca=0&biid=requestd-54644474bf-l7gx4|eu-central-1-staging&reg=eu-central-1&pixel=1&ck=1_c343bf14]]></VASTAdTagURI><Impression><![CDATA[http://ads.staging.avct.cloud/sv?pp=${AUCTION_PRICE}&uuid=0df2c449-6d85-4179-b5d5-37f2f91caa24&ty=h&crid=5ec530e32d57fe1100f17d87&tacid=5ec531d32d57fe1100f17d89&aid=749d36d7-c993-455f-aefd-ffd8a7e3ccf_0&accid=5b51dd1634f2021f127ff7c0&brid=5b51e20f34f2021f127ff7c4&ioid=5b51e22089654741306813a1&caid=5b51e2d689654741306813a4&it=2&iobsid=496e8cff35b2c0110029534d&ext_aid=749d36d7-c993-455f-aefd-ffd8a7e3ccf_0&bp=15.64434783&bt=1591875033134290518&h=396&w=600&vpr=0&vdp=0&domain=example.com&gco=54510b3b816269000061a0f7&stid=542d2c1615e3c013de53a6e2&glat=0&glong=0&bip4=3232238090&ext_siid=5ea89200c865f911007f1b0e&ext_pid=1&ext_sid=5ea84df8c865f911007f1ade&ext_plid=5ea9601ac865f911007f1b6a&optv=latest:latest&invsrc=5e722ee9bd6df11d063a8013&ug=0d&ca=0&biid=requestd-54644474bf-l7gx4|eu-central-1-staging&reg=eu-central-1&pixel=1&ck=1_c343bf14]]></Impression><Creatives><Creative AdId=\"5ec530e32d57fe1100f17d87\"><Linear><TrackingEvents></TrackingEvents><VideoClicks></VideoClicks></Linear></Creative></Creatives></Wrapper></Ad></VAST>",
					"adomain": ["avocet.io"],
					"cid": "5b51e2d689654741306813a4",
					"crid": "5ec530e32d57fe1100f17d87",
					"h": 396,
					"id": "3d4c2d45-5a8c-43b8-9e15-4f48ac45204f",
					"impid": "dfp-ad--top-above-nav",
					"iurl": "https://cdn.staging.avocet.io/snapshots/5b51dd1634f2021f127ff7c0/5ec530e32d57fe1100f17d87.jpeg",
					"language": "en",
					"price": 15.64434783,
					"w": 600,
					"ext": {"avocet":{"duration":30}}
				}
			],
			"seat": "TEST_SEAT_ID"
		}
	]
}`)
