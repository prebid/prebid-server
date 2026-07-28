package models

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

func TestErrorWrap(t *testing.T) {
	type args struct {
		cErr error
		nErr error
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "current error as nil",
			args: args{
				cErr: nil,
				nErr: fmt.Errorf("error found for %d", 1234),
			},
			wantErr: true,
		},
		{
			name: "wrap error",
			args: args{
				cErr: fmt.Errorf("current error found for %d", 1234),
				nErr: fmt.Errorf("new error found for %d", 1234),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ErrorWrap(tt.args.cErr, tt.args.nErr); (err != nil) != tt.wantErr {
				t.Errorf("ErrorWrap() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetCreativeType(t *testing.T) {
	type args struct {
		bid    *openrtb2.Bid
		bidExt *BidExt
		impCtx *ImpCtx
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "bid.ext.prebid.type absent",
			args: args{
				bid: &openrtb2.Bid{},
				bidExt: &BidExt{
					ExtBid: openrtb_ext.ExtBid{
						Prebid: &openrtb_ext.ExtBidPrebid{},
					},
				},
				impCtx: &ImpCtx{},
			},
			want: "",
		},
		{
			name: "bid.ext.prebid.type empty",
			args: args{
				bid: &openrtb2.Bid{},
				bidExt: &BidExt{
					ExtBid: openrtb_ext.ExtBid{
						Prebid: &openrtb_ext.ExtBidPrebid{
							Type: "",
						},
					},
				},
				impCtx: &ImpCtx{},
			},
			want: "",
		},
		{
			name: "bid.ext.prebid.type is banner",
			args: args{
				bid: &openrtb2.Bid{},
				bidExt: &BidExt{
					ExtBid: openrtb_ext.ExtBid{
						Prebid: &openrtb_ext.ExtBidPrebid{
							Type: Banner,
						},
					},
				},
				impCtx: &ImpCtx{},
			},
			want: Banner,
		},
		{
			name: "bid.ext.prebid.type is video",
			args: args{
				bid: &openrtb2.Bid{},
				bidExt: &BidExt{
					ExtBid: openrtb_ext.ExtBid{
						Prebid: &openrtb_ext.ExtBidPrebid{
							Type: Video,
						},
					},
				},
				impCtx: &ImpCtx{},
			},
			want: Video,
		},
		{
			name: "Banner Adm has json assets keyword",
			args: args{
				bid: &openrtb2.Bid{
					AdM: `u003cscript src='mraid.js'></script><script>document.createElement('IMG').src="https://tlx.3lift.com/s2s/notify?px=1&pr=${AUCTION_PRICE}&ts=1686039380&aid=45036723730776858540010&ec=2711_67927_9991141&n=GFABCP4AhSIAwCSAwQwMTNimAMAoAOfqByoAwA%3D";window.tl_auction_response_559942={"settings":{"viewability":{},"additional_data":{"pr":"${AUCTION_PRICE}","bc":"AAABiI_HQr-J9SWLTbGinTR6NNuHz29x102WBw==","aid":"45036723730776858540010","bmid":"2711","biid":"7295","sid":"67927","brid":"82983","adid":"9991141","crid":"737729","ts":"1686039380","bcud":"1240","ss":"20"},"template_id":210,"payable_event":1,"billable_event":1,"billable_pixel":"https:\/\/tlx.3lift.com\/s2s\/notify?px=1&pr=${AUCTION_PRICE}&ts=1686039380&aid=45036723730776858540010&ec=2711_67927_9991141&n=GpIGaCP%3D&b=1","adchoices_url":"https:\/\/optout.aboutads.info\/","format_id":10,"render_options_bm":0,"cta":"Learn more"},"assets":[{"asset_id":0,"cta":"Learn more","banner_width":300,"banner_height":600,"banner_markup":"<script type='text\/javascript' src='https:\/\/ads.as.criteo.com\/delivery\/r\/ajs.php?z=AAABiI_HQr-J9SWLTbGinTR6NNuHz29x102WBw==&u=%7Cgud5gNZYq-lw&ct0={clickurl_enc}'><\/script>"}]};</script><script src="https://ib.3lift.com/ttj?inv_code=HK01_Android_Opening_InListBox_3_336x280&tid=210" data-auction-response-id="559942" data-ss-id="20"></script>`,
				},
				bidExt: &BidExt{
					ExtBid: openrtb_ext.ExtBid{
						Prebid: &openrtb_ext.ExtBidPrebid{},
					},
				},
				impCtx: &ImpCtx{},
			},
			want: Banner,
		},
		{
			name: "Empty Bid Adm",
			args: args{
				bid: &openrtb2.Bid{
					AdM: "",
				},
				bidExt: &BidExt{},
				impCtx: &ImpCtx{},
			},
			want: "",
		},
		{
			name: "VAST Ad",
			args: args{
				bid: &openrtb2.Bid{
					AdM: "<VAST version='3.0'><Ad id='601364'><InLine><AdSystem>Acudeo Compatible</AdSystem><AdTitle>VAST 2.0 Instream Test 1</AdTitle><Description>VAST 2.0 Instream Test 1</Description><Impression>http://172.16.4.213/AdServer/AdDisplayTrackerServlet</Impression><Impression>https://dsptracker.com/{PSPM}</Impression><Error>http://172.16.4.213/track</Error><Error>https://Errortrack.com</Error><Creatives><Creative AdID='601364'><Linear skipoffset='20%'><Duration>00:00:04</Duration><VideoClicks><ClickTracking>http://172.16.4.213/track</ClickTracking><ClickThrough>https://www.pubmatic.com</ClickThrough></VideoClicks><MediaFiles><MediaFile delivery='progressive' type='video/mp4' bitrate='500' width='400' height='300' scalable='true' maintainAspectRatio='true'>https://stagingnyc.pubmatic.com:8443/video/Shashank/mediaFileHost/media/mp4-sample-1.mp4]</MediaFile><MediaFile delivery='progressive' type='video/mp4' bitrate='500' width='400' height='300' scalable='true' maintainAspectRatio='true'>https://stagingnyc.pubmatic.com:8443/video/Shashank/mediaFileHost/media/mp4-sample-2.mp4]</MediaFile></MediaFiles></Linear></Creative></Creatives></InLine></Ad></VAST>",
				},
				bidExt: &BidExt{},
				impCtx: &ImpCtx{},
			},
			want: Video,
		},
		{
			name: "VAST Ad xml",
			args: args{
				bid: &openrtb2.Bid{
					AdM: "<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"yes\"?><VAST version=\"4.0\"><Ad id=\"97517771\"><Wrapper><AdSystem version=\"4.0\">adnxs</AdSystem><VASTAdTagURI><![CDATA[http://sin3-ib.adnxs.com/ab?an_audit=0&test=1&referrer=http%3A%2F%2Fprebid.org%2Fexamples%2Fvideo%2Fserver%2Fjwplayer%2Fpbs-ve-jwplayer-hosted.html&e=wqT_3QKUC6CUBQAAAwDWAAUBCObImfIFEN_CrZCNxam9JhjNy6T75_2f4igqNgkAAAECCBRAEQEHNAAAFEAZAAAAwB6FPUAhERIAKREJADERG6gw6dGnBjjtSEDtSEgCUMuBwC5YnPFbYABozbp1eMu4BYABAYoBA1VTRJIBAQbwUpgBAaABAagBAbABALgBA8ABA8gBAtABANgBAOABAfABAIoCO3VmKCdhJywgMjUyOTg4NSwgMTU4MTY3MTUyNik7dWYoJ3InLCA5NzUxNzc3MSwgLh4A9A4BkgLRAiE0RUQzNndpMi1Md0tFTXVCd0M0WUFDQ2M4VnN3QURnQVFBUkk3VWhRNmRHbkJsZ0FZUF9fX184UGFBQndBWGdCZ0FFQmlBRUJrQUVCbUFFQm9BRUJxQUVEc0FFQXVRSHpyV3FrQUFBVVFNRUI4NjFxcEFBQUZFREpBVlBTU2JZNVlPY18yUUVBQUFBQUFBRHdQLUFCQVBVQkFBQUFBSmdDQUtBQ0FMVUNBQUFBQUwwQ0FBQUFBTUFDQWNnQ0FkQUNBZGdDQWVBQ0FPZ0NBUGdDQUlBREFaZ0RBYWdEdHZpOENyb0RDVk5KVGpNNk5EZ3pOdUFEbEJ1SUJBQ1FCQUNZQkFIQkJBQUFBQQmDCHlRUQkJAQEYTmdFQVBFRQELCQEgQ0lCZVFscVFVCQ8YQUR3UDdFRg0NAQEsLpoCiQEheXc3WjlnNlUBJG5QRmJJQVFvQUQVUFRVUURvSlUwbE9Nem8wT0RNMlFKUWJTEYAMUEFfVREMDEFBQVcdDABZHQwAYR0MAGMdDPBSZUFBLsICP2h0dHA6Ly9wcmViaWQub3JnL2Rldi1kb2NzL3Nob3ctdmlkZW8td2l0aC1hLWRmcC12aWRlby10YWcuaHRtbNgCAOACrZhI6gJMaHQ-SgAgZXhhbXBsZXMvBUVcL3NlcnZlci9qd3BsYXllci9wYnMtdmUtERAYLWhvc3RlZAVXNPICEQoGQURWX0lEEgcySbsFFAhDUEcFFBg1NzU5MzY0ARQIBUNQARM0CDIxOTY5OTc08gINCggBPBhGUkVREgEwBRAcUkVNX1VTRVIFEAAMCSAYQ09ERRIA8gEPAVcRDxALCgdDUBUOEAkKBUlPAWAEAPIBGgRJTxUaOBMKD0NVU1RPTV9NT0RFTA0kCBoKFjIWABxMRUFGX05BTQVqCB4KGjYdAAhBU1QBPhBJRklFRAFiHA0KCFNQTElUAU3wgQEwgAMAiAMBkAMAmAMUoAMBqgMAwAPgqAHIAwDYAwDgAwDoAwD4AwOABACSBAkvb3BlbnJ0YjKYBACiBA0xODIuNzQuMzkuMjUwqAQAsgQOCAAQBBiABSDoAjAAOAS4BADABADIBADSBA45MzI1I1NJTjM6NDgzNtoEAggA4AQA8ASBgSCIBQGYBQCgBf8RAbABqgUkZDc0MzQ3ZDUtYzY3Mi00NTM5LWIxNDEtOWVjMWMzMzJiZTI2wAUAyQWJ4hTwP9IFCQkJDHgAANgFAeAFAfAFw5UL-gUECAAQAJAGAZgGALgGAMEGCSUo8D_QBvUv2gYWChAJERkBUBAAGADgBgTyBgIIAIAHAYgHAKAHQA..&s=dcc685e3549971224cbd8615ff729bcb19107ec0&pp=${AUCTION_PRICE}]]></VASTAdTagURI><Impression><![CDATA[http://ib.adnxs.com/nop]]></Impression><Creatives><Creative adID=\"97517771\"><Linear></Linear></Creative></Creatives></Wrapper></Ad></VAST>",
				},
				bidExt: &BidExt{},
				impCtx: &ImpCtx{},
			},
			want: Video,
		},
		{
			name: "Banner Ad",
			args: args{
				bid: &openrtb2.Bid{
					AdM: "<span class=\"PubAPIAd\"  id=\"4E733404-CC2E-48A2-BC83-4DD5F38FE9BB\"><script type=\"text/javascript\"> document.writeln('<iframe width=\"300\" scrolling=\"no\" height=\"250\" frameborder=\"0\" name=\"iframe0\" allowtransparency=\"true\" marginheight=\"0\" marginwidth=\"0\" vspace=\"0\" hspace=\"0\" src=\"https://ads.pubmatic.com/AdTag/300x250.png\"></iframe>');</script><iframe width=\"0\" scrolling=\"no\" height=\"0\" frameborder=\"0\" src=\"https://st.pubmatic.com/AdServer/AdDisplayTrackerServlet?pubId=5890\" style=\"position:absolute;top:-15000px;left:-15000px\" vspace=\"0\" hspace=\"0\" marginwidth=\"0\" marginheight=\"0\" allowtransparency=\"true\" name=\"pbeacon\"></iframe></span> <!-- PubMatic Ad Ends --><div style=\"position:absolute;left:0px;top:0px;visibility:hidden;\">",
				},
				bidExt: &BidExt{},
				impCtx: &ImpCtx{},
			},
			want: Banner,
		},
		{
			name: "Native Adm with `native` Object",
			args: args{
				bid: &openrtb2.Bid{
					AdM: `{"native":{"ver":1.2,"link":{"url":"https://dummyimage.com/1x1/000000/fff.jpg&text=420x420+Creative.jpg","clicktrackers":["http://image3.pubmatic.com/AdServer/layer?a={PUBMATIC_SECOND_PRICE}&ucrid=9335447642416814892&t=FNOZW09VkdSTVM0eU5BPT09JmlkPTAmY2lkPTIyNzcyJnhwcj0xLjAwMDAwMCZmcD00JnBwPTIuMzcxMiZ0cD0yJnBlPTAuMDA9=","http://image3.pubmatic.com/AdServer/layer?a={PUBMATIC_SECOND_PRICE}&ucrid=9335447642416814892&t=FNOZW09VkdSTVM0eU5BPT09JmlkPTAmY2lkPTIyNzcyJnhwcj0xLjAwMDAwMCZmcD00JnBwPTIuMzcxMiZ0cD0yJnBlPTAuMDA9="]},"eventtrackers":[{"event":1,"method":1,"url":"http://image3.pubmatic.com/AdServer/layer?a={PUBMATIC_SECOND_PRICE}&ucrid=9335447642416814892&t=FNOZW09VkdSTVM0eU5BPT09JmlkPTAmY2lkPTIyNzcyJnhwcj0xLjAwMDAwMCZmcD00JnBwPTIuMzcxMiZ0cD0yJnBlPTAuMDA9="}]}}`,
				},
				bidExt: &BidExt{},
				impCtx: &ImpCtx{
					Native: &openrtb2.Native{},
				},
			},
			want: Native,
		},
		{
			name: "Native Adm with `native` and `assets` Object",
			args: args{
				bid: &openrtb2.Bid{
					AdM: "{\"native\":{\"assets\":[{\"id\":1,\"required\":0,\"title\":{\"text\":\"Lexus - Luxury vehicles company\"}},{\"id\":2,\"img\":{\"h\":150,\"url\":\"https://stagingnyc.pubmatic.com:8443//sdk/lexus_logo.png\",\"w\":150},\"required\":0},{\"id\":3,\"img\":{\"h\":428,\"url\":\"https://stagingnyc.pubmatic.com:8443//sdk/28f48244cafa0363b03899f267453fe7%20copy.png\",\"w\":214},\"required\":0},{\"data\":{\"value\":\"Goto PubMatic\"},\"id\":4,\"required\":0},{\"data\":{\"value\":\"Lexus - Luxury vehicles company\"},\"id\":5,\"required\":0},{\"data\":{\"value\":\"4\"},\"id\":6,\"required\":0}],\"imptrackers\":[\"http://phtrack.pubmatic.com/?ts=1496043362&r=84137f17-eefd-4f06-8380-09138dc616e6&i=c35b1240-a0b3-4708-afca-54be95283c61&a=130917&t=9756&au=10002949&p=&c=10014299&o=10002476&wl=10009731&ty=1\"],\"link\":{\"clicktrackers\":[\"http://ct.pubmatic.com/track?ts=1496043362&r=84137f17-eefd-4f06-8380-09138dc616e6&i=c35b1240-a0b3-4708-afca-54be95283c61&a=130917&t=9756&au=10002949&p=&c=10014299&o=10002476&wl=10009731&ty=3&url=\"],\"url\":\"http://www.lexus.com/\"},\"ver\":1}}",
				},
				bidExt: &BidExt{},
				impCtx: &ImpCtx{
					Native: &openrtb2.Native{},
				},
			},
			want: Native,
		},
		{
			name: "Native Adm with `native` Object but Native is missing in impCtx",
			args: args{
				bid: &openrtb2.Bid{
					AdM: `{"native":{"ver":1.2,"link":{"url":"https://dummyimage.com/1x1/000000/fff.jpg&text=420x420+Creative.jpg","clicktrackers":["http://image3.pubmatic.com/AdServer/layer?a={PUBMATIC_SECOND_PRICE}&ucrid=9335447642416814892&t=FNOZW09VkdSTVM0eU5BPT09JmlkPTAmY2lkPTIyNzcyJnhwcj0xLjAwMDAwMCZmcD00JnBwPTIuMzcxMiZ0cD0yJnBlPTAuMDA9=","http://image3.pubmatic.com/AdServer/layer?a={PUBMATIC_SECOND_PRICE}&ucrid=9335447642416814892&t=FNOZW09VkdSTVM0eU5BPT09JmlkPTAmY2lkPTIyNzcyJnhwcj0xLjAwMDAwMCZmcD00JnBwPTIuMzcxMiZ0cD0yJnBlPTAuMDA9="]},"eventtrackers":[{"event":1,"method":1,"url":"http://image3.pubmatic.com/AdServer/layer?a={PUBMATIC_SECOND_PRICE}&ucrid=9335447642416814892&t=FNOZW09VkdSTVM0eU5BPT09JmlkPTAmY2lkPTIyNzcyJnhwcj0xLjAwMDAwMCZmcD00JnBwPTIuMzcxMiZ0cD0yJnBlPTAuMDA9="}]}}`,
				},
				bidExt: &BidExt{},
				impCtx: &ImpCtx{},
			},
			want: Banner,
		},
		{
			name: "Video Adm \t",
			args: args{
				bid: &openrtb2.Bid{
					AdM: "<VAST\t></VAST>",
				},
				bidExt: &BidExt{},
				impCtx: &ImpCtx{},
			},
			want: Video,
		},
		{
			name: "Video Adm \r",
			args: args{
				bid: &openrtb2.Bid{
					AdM: "<VAST\r></VAST>",
				},
				bidExt: &BidExt{},
				impCtx: &ImpCtx{},
			},
			want: Video,
		},
		{
			name: "Video AdM \n",
			args: args{
				bid: &openrtb2.Bid{
					AdM: "<VAST\n></VAST>",
				},
				bidExt: &BidExt{},
				impCtx: &ImpCtx{},
			},
			want: Video,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creativeType := GetCreativeType(tt.args.bid, tt.args.bidExt, tt.args.impCtx)
			assert.Equal(t, tt.want, creativeType, tt.name)
		})
	}
}

func TestGetAdFormat(t *testing.T) {
	type args struct {
		bid    *openrtb2.Bid
		bidExt *BidExt
		impCtx *ImpCtx
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "no bid object",
			args: args{
				bid:    nil,
				bidExt: &BidExt{},
				impCtx: &ImpCtx{},
			},
			want: "",
		},
		{
			name: "no bidExt object",
			args: args{
				bid:    nil,
				bidExt: &BidExt{},
				impCtx: &ImpCtx{},
			},
			want: "",
		},
		{
			name: "no impctx object",
			args: args{
				bid:    nil,
				bidExt: &BidExt{},
				impCtx: &ImpCtx{},
			},
			want: "",
		},
		{
			name: "Default bid and banner present",
			args: args{
				bid: &openrtb2.Bid{
					DealID: "",
					Price:  0,
				},
				bidExt: &BidExt{},
				impCtx: &ImpCtx{
					IsBanner: true,
				},
			},
			want: Banner,
		},
		{
			name: "Default bid and video present",
			args: args{
				bid: &openrtb2.Bid{
					DealID: "",
					Price:  0,
				},
				bidExt: &BidExt{},
				impCtx: &ImpCtx{
					Video: &openrtb2.Video{},
				},
			},
			want: Video,
		},
		{
			name: "Default bid and native present",
			args: args{
				bid: &openrtb2.Bid{
					DealID: "",
					Price:  0,
				},
				bidExt: &BidExt{},
				impCtx: &ImpCtx{
					Native: &openrtb2.Native{},
				},
			},
			want: Native,
		},
		{
			name: "Default bid and banner and video and native present",
			args: args{
				bid: &openrtb2.Bid{
					DealID: "",
					Price:  0,
				},
				bidExt: &BidExt{},
				impCtx: &ImpCtx{
					IsBanner: true,
					Native:   &openrtb2.Native{},
					Video:    &openrtb2.Video{},
				},
			},
			want: Banner,
		},
		{
			name: "Default bid and none of banner/video/native present",
			args: args{
				bid: &openrtb2.Bid{
					DealID: "",
					Price:  0,
				},
				bidExt: &BidExt{},
				impCtx: &ImpCtx{},
			},
			want: "",
		},
		{
			name: "Empty Bid Adm",
			args: args{
				bid: &openrtb2.Bid{
					Price: 10,
					AdM:   "",
				},
				bidExt: &BidExt{},
				impCtx: &ImpCtx{},
			},
			want: "",
		},
		{
			name: "VAST Ad",
			args: args{
				bid: &openrtb2.Bid{
					DealID: "dl",
					AdM:    "<VAST version='3.0'><Ad id='601364'><InLine><AdSystem>Acudeo Compatible</AdSystem><AdTitle>VAST 2.0 Instream Test 1</AdTitle><Description>VAST 2.0 Instream Test 1</Description><Impression>http://172.16.4.213/AdServer/AdDisplayTrackerServlet</Impression><Impression>https://dsptracker.com/{PSPM}</Impression><Error>http://172.16.4.213/track</Error><Error>https://Errortrack.com</Error><Creatives><Creative AdID='601364'><Linear skipoffset='20%'><Duration>00:00:04</Duration><VideoClicks><ClickTracking>http://172.16.4.213/track</ClickTracking><ClickThrough>https://www.pubmatic.com</ClickThrough></VideoClicks><MediaFiles><MediaFile delivery='progressive' type='video/mp4' bitrate='500' width='400' height='300' scalable='true' maintainAspectRatio='true'>https://stagingnyc.pubmatic.com:8443/video/Shashank/mediaFileHost/media/mp4-sample-1.mp4]</MediaFile><MediaFile delivery='progressive' type='video/mp4' bitrate='500' width='400' height='300' scalable='true' maintainAspectRatio='true'>https://stagingnyc.pubmatic.com:8443/video/Shashank/mediaFileHost/media/mp4-sample-2.mp4]</MediaFile></MediaFiles></Linear></Creative></Creatives></InLine></Ad></VAST>",
				},
				bidExt: &BidExt{},
				impCtx: &ImpCtx{},
			},
			want: Video,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creativeType := GetAdFormat(tt.args.bid, tt.args.bidExt, tt.args.impCtx)
			assert.Equal(t, tt.want, creativeType, tt.name)
		})
	}
}

func TestGetSizeForPlatform(t *testing.T) {
	type args struct {
		width, height int64
		platform      string
	}
	tests := []struct {
		name string
		args args
		size string
	}{
		{
			name: "in-app platform",
			args: args{
				width:    100,
				height:   10,
				platform: PLATFORM_APP,
			},
			size: "100x10",
		},
		{
			name: "video platform",
			args: args{
				width:    100,
				height:   10,
				platform: PLATFORM_VIDEO,
			},
			size: "100x10",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := GetSizeForPlatform(tt.args.width, tt.args.height, tt.args.platform)
			assert.Equal(t, tt.size, size, tt.name)
		})
	}
}

func TestGenerateSlotName(t *testing.T) {
	type args struct {
		h     int64
		w     int64
		kgp   string
		tagid string
		div   string
		src   string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "_AU_",
			args: args{
				h:     100,
				w:     200,
				kgp:   "_AU_",
				tagid: "/15671365/Test_Adunit",
				div:   "Div1",
				src:   "test.com",
			},
			want: "/15671365/Test_Adunit",
		},
		{
			name: "_RE_",
			args: args{
				h:     100,
				w:     200,
				kgp:   "_RE_",
				tagid: "/15671365/Test_Adunit",
				div:   "Div1",
				src:   "test.com",
			},
			want: "/15671365/Test_Adunit",
		},
		{
			name: "_DIV_",
			args: args{
				h:     100,
				w:     200,
				kgp:   "_DIV_",
				tagid: "/15671365/Test_Adunit",
				div:   "Div1",
				src:   "test.com",
			},
			want: "Div1",
		},
		{
			name: "_AU_",
			args: args{
				h:     100,
				w:     200,
				kgp:   "_AU_",
				tagid: "/15671365/Test_Adunit",
				div:   "Div1",
				src:   "test.com",
			},
			want: "/15671365/Test_Adunit",
		},
		{
			name: "_AU_@_W_x_H_",
			args: args{
				h:     100,
				w:     200,
				kgp:   "_AU_@_W_x_H_",
				tagid: "/15671365/Test_Adunit",
				div:   "Div1",
				src:   "test.com",
			},
			want: "/15671365/Test_Adunit@200x100",
		},
		{
			name: "_RE_@_W_x_H_",
			args: args{
				h:     100,
				w:     200,
				kgp:   "_RE_@_W_x_H_",
				tagid: "/15671365/Test_Adunit",
				div:   "Div1",
				src:   "test.com",
			},
			want: "/15671365/Test_Adunit@200x100",
		},
		{
			name: "_DIV_@_W_x_H_",
			args: args{
				h:     100,
				w:     200,
				kgp:   "_DIV_@_W_x_H_",
				tagid: "/15671365/Test_Adunit",
				div:   "Div1",
				src:   "test.com",
			},
			want: "Div1@200x100",
		},
		{
			name: "_W_x_H_@_W_x_H_",
			args: args{
				h:     100,
				w:     200,
				kgp:   "_W_x_H_@_W_x_H_",
				tagid: "/15671365/Test_Adunit",
				div:   "Div1",
				src:   "test.com",
			},
			want: "200x100@200x100",
		},
		{
			name: "_AU_@_DIV_@_W_x_H_",
			args: args{
				h:     100,
				w:     200,
				kgp:   "_AU_@_DIV_@_W_x_H_",
				tagid: "/15671365/Test_Adunit",
				div:   "Div1",
				src:   "test.com",
			},
			want: "/15671365/Test_Adunit@Div1@200x100",
		},
		{
			name: "_AU_@_SRC_@_VASTTAG_",
			args: args{
				h:     100,
				w:     200,
				kgp:   "_AU_@_SRC_@_VASTTAG_",
				tagid: "/15671365/Test_Adunit",
				div:   "Div1",
				src:   "test.com",
			},
			want: "/15671365/Test_Adunit@test.com@",
		},
		{
			name: "empty_kgp",
			args: args{
				h:     100,
				w:     200,
				kgp:   "",
				tagid: "/15671365/Test_Adunit",
				div:   "Div1",
				src:   "test.com",
			},
			want: "",
		},
		{
			name: "random_kgp",
			args: args{
				h:     100,
				w:     200,
				kgp:   "fjkdfhk",
				tagid: "/15671365/Test_Adunit",
				div:   "Div1",
				src:   "test.com",
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateSlotName(tt.args.h, tt.args.w, tt.args.kgp, tt.args.tagid, tt.args.div, tt.args.src)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetRevenueShare(t *testing.T) {
	tests := []struct {
		name          string
		partnerConfig map[string]string
		revshare      float64
	}{
		{
			name:          "Empty partnerConfig",
			partnerConfig: make(map[string]string),
			revshare:      0,
		},
		{
			name: "partnerConfig without rev_share",
			partnerConfig: map[string]string{
				"anykey": "anyval",
			},
			revshare: 0,
		},
		{
			name: "partnerConfig with invalid rev_share",
			partnerConfig: map[string]string{
				REVSHARE: "invalid",
			},
			revshare: 0,
		},
		{
			name: "partnerConfig with valid rev_share",
			partnerConfig: map[string]string{
				REVSHARE: "10",
			},
			revshare: 10,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			revshare := GetRevenueShare(tt.partnerConfig)
			assert.Equal(t, tt.revshare, revshare, tt.name)
		})
	}
}

func TestGetNetEcpm(t *testing.T) {
	type args struct {
		price, revShare float64
	}
	tests := []struct {
		name    string
		args    args
		netecpm float64
	}{
		{
			name: "revshare is 0",
			args: args{
				revShare: 0,
				price:    10,
			},
			netecpm: 10,
		},
		{
			name: "revshare is int",
			args: args{
				revShare: 2,
				price:    2,
			},
			netecpm: 1.96,
		},
		{
			name: "revshare is float",
			args: args{
				revShare: 3.338,
				price:    100,
			},
			netecpm: 96.66,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			netecpm := GetNetEcpm(tt.args.price, tt.args.revShare)
			assert.Equal(t, tt.netecpm, netecpm, tt.name)
		})
	}
}

func TestGetGrossEcpm(t *testing.T) {

	tests := []struct {
		name      string
		price     float64
		grossecpm float64
	}{
		{
			name:      "grossecpm ceiling",
			price:     18.998,
			grossecpm: 19,
		},
		{
			name:      "grossecpm floor",
			price:     18.901,
			grossecpm: 18.90,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grossecpm := GetGrossEcpm(tt.price)
			assert.Equal(t, tt.grossecpm, grossecpm, tt.name)
		})
	}
}

func TestExtractDomain(t *testing.T) {
	type want struct {
		domain string
		err    bool
	}
	tests := []struct {
		name string
		url  string
		want want
	}{
		{
			name: "url without http prefix",
			url:  "google.com",
			want: want{
				domain: "google.com",
			},
		},
		{
			name: "url with http prefix",
			url:  "http://google.com",
			want: want{
				domain: "google.com",
			},
		},
		{
			name: "url with https prefix",
			url:  "https://google.com",
			want: want{
				domain: "google.com",
			},
		},
		{
			name: "invalid",
			url:  "https://google:com?a=1;b=2",
			want: want{
				domain: "",
				err:    true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			domain, err := ExtractDomain(tt.url)
			assert.Equal(t, tt.want.domain, domain, tt.name)
			assert.Equal(t, tt.want.err, err != nil, tt.name)
		})
	}
}

func TestGetBidLevelFloorsDetails(t *testing.T) {
	type args struct {
		bidExt             BidExt
		impCtx             ImpCtx
		currencyConversion func(from, to string, value float64) (float64, error)
	}
	type want struct {
		fv, frv float64
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "set_floor_values_from_bidExt",
			args: args{
				bidExt: BidExt{
					ExtBid: openrtb_ext.ExtBid{
						Prebid: &openrtb_ext.ExtBidPrebid{
							Floors: &openrtb_ext.ExtBidPrebidFloors{
								FloorRuleValue: 10,
								FloorValue:     5,
							},
						},
					},
				},
				impCtx: ImpCtx{
					BidFloor:    2.2,
					BidFloorCur: "EUR",
				},
			},
			want: want{
				fv:  5,
				frv: 10,
			},
		},
		{
			name: "frv_absent_in_bidExt",
			args: args{
				bidExt: BidExt{
					ExtBid: openrtb_ext.ExtBid{
						Prebid: &openrtb_ext.ExtBidPrebid{
							Floors: &openrtb_ext.ExtBidPrebidFloors{
								FloorValue: 5,
							},
						},
					},
				},
				impCtx: ImpCtx{
					BidFloor:    2.2,
					BidFloorCur: "EUR",
				},
			},
			want: want{
				fv:  5,
				frv: 5,
			},
		},
		{
			name: "fv_is_0_in_bidExt",
			args: args{
				bidExt: BidExt{
					ExtBid: openrtb_ext.ExtBid{
						Prebid: &openrtb_ext.ExtBidPrebid{
							Floors: &openrtb_ext.ExtBidPrebidFloors{
								FloorValue: 0,
							},
						},
					},
				},
				impCtx: ImpCtx{
					BidFloor:    2.2,
					BidFloorCur: "EUR",
				},
			},
			want: want{
				fv:  0,
				frv: 0,
			},
		},
		{
			name: "currency_conversion_for_floor_values_in_bidExt",
			args: args{
				bidExt: BidExt{
					ExtBid: openrtb_ext.ExtBid{
						Prebid: &openrtb_ext.ExtBidPrebid{
							Floors: &openrtb_ext.ExtBidPrebidFloors{
								FloorValue:    5,
								FloorCurrency: "EUR",
							},
						},
					},
				},
				impCtx: ImpCtx{
					BidFloor:    2.2,
					BidFloorCur: "EUR",
				},
				currencyConversion: func(from, to string, value float64) (float64, error) {
					return 10, nil
				},
			},
			want: want{
				fv:  10,
				frv: 10,
			},
		},
		{
			name: "floor_values_missing_in_bidExt_fallback_to_impctx",
			args: args{
				bidExt: BidExt{
					ExtBid: openrtb_ext.ExtBid{
						Prebid: &openrtb_ext.ExtBidPrebid{},
					},
				},
				impCtx: ImpCtx{
					BidFloor:    2.2,
					BidFloorCur: "USD",
				},
			},
			want: want{
				fv:  2.2,
				frv: 2.2,
			},
		},
		{
			name: "bidExt.Prebid_is_nil_fallback_to_impctx",
			args: args{
				bidExt: BidExt{
					ExtBid: openrtb_ext.ExtBid{
						Prebid: nil,
					},
				},
				impCtx: ImpCtx{
					BidFloor:    2.2,
					BidFloorCur: "USD",
				},
			},
			want: want{
				fv:  2.2,
				frv: 2.2,
			},
		},
		{
			name: "bidExt.Prebid.Floors_is_nil_fallback_to_impctx",
			args: args{
				bidExt: BidExt{
					ExtBid: openrtb_ext.ExtBid{
						Prebid: &openrtb_ext.ExtBidPrebid{
							Floors: nil,
						},
					},
				},
				impCtx: ImpCtx{
					BidFloor:    2.2,
					BidFloorCur: "USD",
				},
			},
			want: want{
				fv:  2.2,
				frv: 2.2,
			},
		},
		{
			name: "currency_conversion_for_floor_values_in_impctx",
			args: args{
				bidExt: BidExt{
					ExtBid: openrtb_ext.ExtBid{},
				},
				impCtx: ImpCtx{
					BidFloor:    5,
					BidFloorCur: "EUR",
				},
				currencyConversion: func(from, to string, value float64) (float64, error) {
					return 10, nil
				},
			},
			want: want{
				fv:  10,
				frv: 10,
			},
		},
		{
			name: "floor_values_not_set_in_both_bidExt_and_impctx",
			args: args{
				bidExt: BidExt{
					ExtBid: openrtb_ext.ExtBid{},
				},
				impCtx: ImpCtx{},
				currencyConversion: func(from, to string, value float64) (float64, error) {
					return 10, nil
				},
			},
			want: want{
				fv:  0,
				frv: 0,
			},
		},
		{
			name: "floor_values_set_from_bidExt_mbmfv_for_applovinmax",
			args: args{
				bidExt: BidExt{
					MultiBidMultiFloorValue: 5.0,
				},
			},
			want: want{
				fv:  5.0,
				frv: 5.0,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fv, frv := GetBidLevelFloorsDetails(tt.args.bidExt, tt.args.impCtx, tt.args.currencyConversion)
			assert.Equal(t, tt.want.fv, fv, tt.name)
			assert.Equal(t, tt.want.frv, frv, tt.name)
		})
	}
}

func Test_getFloorsDetails(t *testing.T) {
	type args struct {
		bidResponseExt openrtb_ext.ExtBidResponse
	}
	tests := []struct {
		name         string
		args         args
		floorDetails FloorsDetails
	}{
		{
			name:         "no_responseExt",
			args:         args{},
			floorDetails: FloorsDetails{},
		},
		{
			name: "empty_responseExt",
			args: args{
				bidResponseExt: openrtb_ext.ExtBidResponse{},
			},
			floorDetails: FloorsDetails{},
		},
		{
			name: "empty_prebid_in_responseExt",
			args: args{
				bidResponseExt: openrtb_ext.ExtBidResponse{
					Prebid: &openrtb_ext.ExtResponsePrebid{},
				},
			},
			floorDetails: FloorsDetails{},
		},
		{
			name: "empty_prebidfloors_in_responseExt",
			args: args{
				bidResponseExt: openrtb_ext.ExtBidResponse{
					Prebid: &openrtb_ext.ExtResponsePrebid{
						Floors: &openrtb_ext.PriceFloorRules{},
					},
				},
			},
			floorDetails: FloorsDetails{},
		},
		{
			name: "no_enforced_floors_data_in_responseExt",
			args: args{
				bidResponseExt: openrtb_ext.ExtBidResponse{
					Prebid: &openrtb_ext.ExtResponsePrebid{
						Floors: &openrtb_ext.PriceFloorRules{
							Data:               &openrtb_ext.PriceFloorData{},
							PriceFloorLocation: openrtb_ext.FetchLocation,
						},
					},
				},
			},
			floorDetails: FloorsDetails{
				Skipfloors:        nil,
				FloorType:         SoftFloor,
				FloorSource:       ptrutil.ToPtr(2),
				FloorModelVersion: "",
			},
		},
		{
			name: "no_modelsgroups_floors_data_in_responseExt",
			args: args{
				bidResponseExt: openrtb_ext.ExtBidResponse{
					Prebid: &openrtb_ext.ExtResponsePrebid{
						Floors: &openrtb_ext.PriceFloorRules{
							Data:               &openrtb_ext.PriceFloorData{},
							PriceFloorLocation: openrtb_ext.FetchLocation,
							Enforcement: &openrtb_ext.PriceFloorEnforcement{
								EnforcePBS: ptrutil.ToPtr(true),
							},
						},
					},
				},
			},
			floorDetails: FloorsDetails{
				Skipfloors:        nil,
				FloorType:         HardFloor,
				FloorSource:       ptrutil.ToPtr(2),
				FloorModelVersion: "",
			},
		},
		{
			name: "no_skipped_floors_data_in_responseExt",
			args: args{
				bidResponseExt: openrtb_ext.ExtBidResponse{
					Prebid: &openrtb_ext.ExtResponsePrebid{
						Floors: &openrtb_ext.PriceFloorRules{
							Data: &openrtb_ext.PriceFloorData{
								ModelGroups: []openrtb_ext.PriceFloorModelGroup{
									{
										ModelVersion: "version 1",
									},
								},
							},
							PriceFloorLocation: openrtb_ext.FetchLocation,
							Enforcement: &openrtb_ext.PriceFloorEnforcement{
								EnforcePBS: ptrutil.ToPtr(true),
							},
						},
					},
				},
			},
			floorDetails: FloorsDetails{
				Skipfloors:        nil,
				FloorType:         HardFloor,
				FloorSource:       ptrutil.ToPtr(2),
				FloorModelVersion: "version 1",
			},
		},
		{
			name: "all_floors_data_in_responseExt",
			args: args{
				bidResponseExt: openrtb_ext.ExtBidResponse{
					Prebid: &openrtb_ext.ExtResponsePrebid{
						Floors: &openrtb_ext.PriceFloorRules{
							Skipped: ptrutil.ToPtr(true),
							Data: &openrtb_ext.PriceFloorData{
								ModelGroups: []openrtb_ext.PriceFloorModelGroup{
									{
										ModelVersion: "version 1",
									},
								},
							},
							PriceFloorLocation: openrtb_ext.FetchLocation,
							Enforcement: &openrtb_ext.PriceFloorEnforcement{
								EnforcePBS: ptrutil.ToPtr(true),
							},
						},
					},
				},
			},
			floorDetails: FloorsDetails{
				Skipfloors:        ptrutil.ToPtr(1),
				FloorType:         HardFloor,
				FloorSource:       ptrutil.ToPtr(2),
				FloorModelVersion: "version 1",
			},
		},
		{
			name: "floor_provider_present",
			args: args{
				bidResponseExt: openrtb_ext.ExtBidResponse{
					Prebid: &openrtb_ext.ExtResponsePrebid{
						Floors: &openrtb_ext.PriceFloorRules{
							Skipped: ptrutil.ToPtr(true),
							Data: &openrtb_ext.PriceFloorData{
								ModelGroups: []openrtb_ext.PriceFloorModelGroup{
									{
										ModelVersion: "version 1",
									},
								},
								FloorProvider: "providerA",
							},
							PriceFloorLocation: openrtb_ext.FetchLocation,
							Enforcement: &openrtb_ext.PriceFloorEnforcement{
								EnforcePBS: ptrutil.ToPtr(true),
							},
						},
					},
				},
			},
			floorDetails: FloorsDetails{
				Skipfloors:        ptrutil.ToPtr(1),
				FloorType:         HardFloor,
				FloorSource:       ptrutil.ToPtr(2),
				FloorModelVersion: "version 1",
				FloorProvider:     "providerA",
			},
		},
		{
			name: "floor_fetch_status_absent_in_FloorSourceMap",
			args: args{
				bidResponseExt: openrtb_ext.ExtBidResponse{
					Prebid: &openrtb_ext.ExtResponsePrebid{
						Floors: &openrtb_ext.PriceFloorRules{
							Skipped:     ptrutil.ToPtr(true),
							FetchStatus: "invalid",
							Data: &openrtb_ext.PriceFloorData{
								ModelGroups: []openrtb_ext.PriceFloorModelGroup{
									{
										ModelVersion: "version 1",
									},
								},
								FloorProvider: "providerB",
							},
							PriceFloorLocation: openrtb_ext.FetchLocation,
							Enforcement: &openrtb_ext.PriceFloorEnforcement{
								EnforcePBS: ptrutil.ToPtr(true),
							},
						},
					},
				},
			},
			floorDetails: FloorsDetails{
				Skipfloors:        ptrutil.ToPtr(1),
				FloorType:         HardFloor,
				FloorSource:       ptrutil.ToPtr(2),
				FloorModelVersion: "version 1",
				FloorProvider:     "providerB",
			},
		},
		{
			name: "floor_fetch_status_present_in_FloorSourceMap",
			args: args{
				bidResponseExt: openrtb_ext.ExtBidResponse{
					Prebid: &openrtb_ext.ExtResponsePrebid{
						Floors: &openrtb_ext.PriceFloorRules{
							Skipped:     ptrutil.ToPtr(true),
							FetchStatus: openrtb_ext.FetchError,
							Data: &openrtb_ext.PriceFloorData{
								ModelGroups: []openrtb_ext.PriceFloorModelGroup{
									{
										ModelVersion: "version 1",
									},
								},
								FloorProvider: "providerC",
							},
							PriceFloorLocation: openrtb_ext.FetchLocation,
							Enforcement: &openrtb_ext.PriceFloorEnforcement{
								EnforcePBS: ptrutil.ToPtr(true),
							},
						},
					},
				},
			},
			floorDetails: FloorsDetails{
				Skipfloors:        ptrutil.ToPtr(1),
				FloorType:         HardFloor,
				FloorSource:       ptrutil.ToPtr(2),
				FloorModelVersion: "version 1",
				FloorProvider:     "providerC",
				FloorFetchStatus:  ptrutil.ToPtr(2),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			floorsDetails := GetFloorsDetails(tt.args.bidResponseExt)
			assert.Equal(t, tt.floorDetails, floorsDetails, tt.name)
		})
	}
}

func TestGetKGPSV(t *testing.T) {
	type args struct {
		bid        openrtb2.Bid
		bidExt     *BidExt
		bidderMeta PartnerData
		adformat   string
		tagId      string
		div        string
		source     string
	}
	tests := []struct {
		name  string
		args  args
		kgpv  string
		kgpsv string
	}{
		{
			name: "default bid not regex",
			args: args{
				bidderMeta: PartnerData{
					KGPV: "kgpv",
					KGP:  "_AU_@_W_x_H_",
				},
			},
			kgpv:  "kgpv",
			kgpsv: "kgpv",
		},
		{
			name: "default bid regex",
			args: args{
				bidderMeta: PartnerData{
					KGPV:    "kgpv",
					IsRegex: true,
					KGP:     "_AU_@_DIV_@_W_x_H_",
				},
			},
			kgpv:  "kgpv",
			kgpsv: "",
		},
		{
			name: "only kgpsv found in partnerData",
			args: args{
				bidderMeta: PartnerData{
					MatchedSlot: "kgpsv",
					IsRegex:     true,
					KGP:         "_AU_@_DIV_@_W_x_H_",
				},
			},
			kgpv:  "kgpsv",
			kgpsv: "kgpsv",
		},
		{
			name: "valid bid found in partnerData and regex true",
			args: args{
				bid: openrtb2.Bid{
					Price:  1,
					DealID: "deal",
					W:      250,
					H:      300,
				},
				bidderMeta: PartnerData{
					KGPV:        "kgpv",
					MatchedSlot: "kgpsv",
					IsRegex:     true,
					KGP:         "_AU_@_DIV_@_W_x_H_",
				},
			},
			kgpv:  "kgpv",
			kgpsv: "kgpsv",
		},
		{
			name: "valid bid and regex false",
			args: args{
				bid: openrtb2.Bid{
					Price:  1,
					DealID: "deal",
					W:      250,
					H:      300,
				},
				bidderMeta: PartnerData{
					KGPV:        "kgpv",
					MatchedSlot: "kgpsv",
					IsRegex:     false,
					KGP:         "_AU_@_W_x_H_",
				},
			},
			kgpv:  "kgpv",
			kgpsv: "kgpv",
		},
		{
			name: "KGPV not present in partnerData, regex false and adformat is video",
			args: args{
				bid: openrtb2.Bid{
					Price:  1,
					DealID: "deal",
					W:      250,
					H:      300,
				},
				adformat: Video,
				bidderMeta: PartnerData{
					KGP: "_AU_@_W_x_H_",
				},
				tagId: "adunit",
			},
			kgpv:  "adunit@0x0",
			kgpsv: "adunit@0x0",
		},
		{
			name: "KGPV not present in partnerData, regex false and adformat is banner",
			args: args{
				bid: openrtb2.Bid{
					Price:  1,
					DealID: "deal",
					W:      250,
					H:      300,
				},
				adformat: Banner,
				bidderMeta: PartnerData{
					KGP:         "_AU_@_W_x_H_",
					MatchedSlot: "matchedSlot",
				},
				tagId: "adunit",
			},
			kgpv:  "adunit@250x300",
			kgpsv: "adunit@250x300",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := GetKGPSV(tt.args.bid, tt.args.bidExt, tt.args.bidderMeta, tt.args.adformat, tt.args.tagId, tt.args.div, tt.args.source)

			// Ensure KGPV is not empty
			assert.NotEmpty(t, got, "KGPV should not be empty")
			assert.Equal(t, tt.kgpv, got, "GetKGPSV() got = %v, want %v", got, tt.kgpv)
			assert.Equal(t, tt.kgpsv, got1, "GetKGPSV() got1 = %v, want %v", got1, tt.kgpsv)
		})
	}
}

func TestGetGrossEcpmFromNetEcpm(t *testing.T) {
	type args struct {
		netEcpm  float64
		revShare float64
	}
	tests := []struct {
		name string
		args args
		want float64
	}{
		{
			name: "When netcpm is 100 and revShare is 0",
			args: args{
				netEcpm:  100,
				revShare: 0,
			},
			want: 100,
		},
		{
			name: "When netcpm is 0 and revShare is 100",
			args: args{
				netEcpm:  0,
				revShare: 100,
			},
			want: 0,
		},
		{
			name: "When netcpm is 100 and revShare is 50",
			args: args{
				netEcpm:  100,
				revShare: 50,
			},
			want: 200,
		},
		{
			name: "When netcpm is 80 and revShare is 20",
			args: args{
				netEcpm:  80,
				revShare: 20,
			},
			want: 100,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetGrossEcpmFromNetEcpm(tt.args.netEcpm, tt.args.revShare); got != tt.want {
				t.Errorf("GetGrossEcpmFromNetEcpm() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToFixed(t *testing.T) {
	type args struct {
		num       float64
		precision int
	}
	tests := []struct {
		name string
		args args
		want float64
	}{
		{
			name: "Rounding of 0.1",
			args: args{
				num:       0.1,
				precision: 2,
			},
			want: 0.10,
		},
		{
			name: "Rounding of 0.1101",
			args: args{
				num:       0.1101,
				precision: 2,
			},
			want: 0.11,
		},
		{
			name: "Rounding of 0.10000000149011612",
			args: args{
				num:       0.10000000149011612,
				precision: 2,
			},
			want: 0.10,
		},
		{
			name: "Rounding of 0.10000000149011612",
			args: args{
				num:       0.10000000149011612,
				precision: 3,
			},
			want: 0.100,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToFixed(tt.args.num, tt.args.precision); got != tt.want {
				t.Errorf("toFixed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetIP(t *testing.T) {
	type args struct {
		in *http.Request
	}
	tests := []struct {
		name  string
		args  args
		setup func(in *http.Request)
		want  string
		want1 string
	}{
		{
			name: "priority_to_rlnclient",
			args: args{
				in: httptest.NewRequest("GET", "http://example.com", nil),
			},
			setup: func(in *http.Request) {
				in.Header = http.Header{
					"X-Forwarded-For":     []string{"1.1.1.1"},
					"X-Device-Ip":         []string{"1.0.0.1"},
					"Source_Ip":           []string{"0.1.1.0"},
					"X_Cluster_Client_Ip": []string{"0.1.0.0"},
					"Remote_Addr":         []string{"0.1.0.1"},
				}
				in.Header.Add(RlnClientIP, "0.0.0.0")
			},
			want:  "0.0.0.0",
			want1: RlnClientIP,
		},
		{
			name: "priority_to_XDeviceIp",
			args: args{
				in: httptest.NewRequest("GET", "http://example.com", nil),
			},
			setup: func(in *http.Request) {
				in.Header = http.Header{
					"X-Forwarded-For":     []string{"1.1.1.1"},
					"X-Device-Ip":         []string{"1.0.0.1"},
					"Source_Ip":           []string{"0.1.1.0"},
					"X_Cluster_Client_Ip": []string{"0.1.0.0"},
					"Remote_Addr":         []string{"0.1.0.1"},
				}
			},
			want:  "1.0.0.1",
			want1: XDeviceIP,
		},
		{
			name: "priority_to_source_ip",
			args: args{
				in: httptest.NewRequest("GET", "http://example.com", nil),
			},
			setup: func(in *http.Request) {
				in.Header = http.Header{
					"X-Forwarded-For":     []string{"1.1.1.1"},
					"X_Cluster_Client_Ip": []string{"0.1.0.0"},
					"Remote_Addr":         []string{"0.1.0.1"},
				}
				in.Header.Add(SourceIP, "0.1.1.0")
			},
			want: "0.1.1.0",
		},
		{
			name: "priority_to_clusterclient",
			args: args{
				in: httptest.NewRequest("GET", "http://example.com", nil),
			},
			setup: func(in *http.Request) {
				in.Header = http.Header{
					"X-Forwarded-For": []string{"1.1.1.1"},
					"Remote_Addr":     []string{"0.1.0.1"},
				}
				in.Header.Add(ClusterClientIP, "0.1.0.0")
			},
			want: "0.1.0.0",
		},
		{
			name: "priority_to_x-forwarded-for",
			args: args{
				in: httptest.NewRequest("GET", "http://example.com", nil),
			},
			setup: func(in *http.Request) {
				in.Header = http.Header{
					"X-Forwarded-For": []string{"1.1.1.1"},
					"Remote_Addr":     []string{"0.1.0.1"},
				}
			},
			want: "1.1.1.1",
		},
		{
			name: "only_remoteaddr",
			args: args{
				in: httptest.NewRequest("GET", "http://example.com", nil),
			},
			setup: func(in *http.Request) {
				in.Header = http.Header{}
				in.Header.Add(RemoteAddr, "1.1.1.1")
			},
			want: "1.1.1.1",
		},
		{
			name: "no_headers_but_root_level_Remote_Address_present",
			args: args{
				in: httptest.NewRequest("GET", "http://example.com", nil),
			},
			setup: func(in *http.Request) {
				in.Header = http.Header{}
				in.RemoteAddr = "1.1.1.1:8080"
			},
			want: "1.1.1.1",
		},
		{
			name: "no_headers_but_root_level_Remote_Address_present_without_port",
			args: args{
				in: httptest.NewRequest("GET", "http://example.com", nil),
			},
			setup: func(in *http.Request) {
				in.Header = http.Header{}
				in.RemoteAddr = "1.1.1.1"
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(tt.args.in)
			got := GetIP(tt.args.in)
			if got != tt.want {
				t.Errorf("GetIP() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetMultiFloors(t *testing.T) {
	tests := []struct {
		name  string
		rctx  *RequestCtx
		impID string
		want  []float64
	}{
		{
			name: "nil rctx.MultiFloors",
			rctx: &RequestCtx{
				MultiFloors: nil,
			},
			impID: "123",
			want:  nil,
		},
		{
			name: "impID not present in rctx.MultiFloors",
			rctx: &RequestCtx{
				MultiFloors: map[string]*MultiFloors{
					"123": {Tier1: 1.0, Tier2: 0.8, Tier3: 0.6},
				},
			},
			impID: "456",
			want:  nil,
		},
		{
			name: "impID present and multifloors nil",
			rctx: &RequestCtx{
				MultiFloors: map[string]*MultiFloors{
					"123": nil,
				},
			},
			impID: "123",
			want:  nil,
		},
		{
			name: "impID present with multifloors",
			rctx: &RequestCtx{
				MultiFloors: map[string]*MultiFloors{
					"123": {Tier1: 1.0, Tier2: 0.8, Tier3: 0.6},
				},
			},
			impID: "123",
			want:  []float64{1.0, 0.8, 0.6},
		},
		{
			name: "impID present with multifloors with tier4",
			rctx: &RequestCtx{
				MultiFloors: map[string]*MultiFloors{
					"123": {Tier1: 1.0, Tier2: 0.8, Tier3: 0.6, Tier4: 0.4},
				},
			},
			impID: "123",
			want:  []float64{1.0, 0.8, 0.6, 0.4},
		},
		{
			name: "impID present with multifloors with tier4 and tier5",
			rctx: &RequestCtx{
				MultiFloors: map[string]*MultiFloors{
					"123": {Tier1: 1.0, Tier2: 0.8, Tier3: 0.6, Tier4: 0.4, Tier5: 0.2},
				},
			},
			impID: "123",
			want:  []float64{1.0, 0.8, 0.6, 0.4, 0.2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetMultiFloors(tt.rctx.MultiFloors, tt.impID)
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}

func TestEdsStatusFromRequest(t *testing.T) {
	tests := []struct {
		name string
		req  *openrtb2.BidRequest
		want *int
	}{
		{
			name: "edsstatus present",
			req: &openrtb2.BidRequest{
				Ext: json.RawMessage(`{"wrapper":{"edsstatus":1}}`),
			},
			want: ptrutil.ToPtr(1),
		},
		{
			name: "edsstatus disabled",
			req: &openrtb2.BidRequest{
				Ext: json.RawMessage(`{"wrapper":{"edsstatus":0}}`),
			},
			want: ptrutil.ToPtr(0),
		},
		{
			name: "edsstatus unknown",
			req: &openrtb2.BidRequest{
				Ext: json.RawMessage(`{"wrapper":{"edsstatus":-1}}`),
			},
			want: ptrutil.ToPtr(-1),
		},
		{
			name: "wrapper without edsstatus",
			req: &openrtb2.BidRequest{
				Ext: json.RawMessage(`{"wrapper":{"profileid":5890}}`),
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, EdsStatusFromRequest(tt.req))
		})
	}
}

func TestGetRequestExtWrapperEdsStatus(t *testing.T) {
	got, err := GetRequestExtWrapper([]byte(`{"ext":{"wrapper":{"profileid":23432,"edsstatus":1}}}`), "ext", "wrapper")
	assert.NoError(t, err)
	assert.Equal(t, ptrutil.ToPtr(1), got.EdsStatus)
}

func TestResolveEdsStatus(t *testing.T) {
	wrapperWithEdsStatus := RequestExtWrapper{
		EdsStatus: ptrutil.ToPtr(1),
	}
	signalWithEdsStatus := &openrtb2.BidRequest{
		Ext: json.RawMessage(`{"wrapper":{"edsstatus":2}}`),
	}
	signalWithoutEdsStatus := &openrtb2.BidRequest{
		Ext: json.RawMessage(`{"wrapper":{"profileid":5890}}`),
	}

	tests := []struct {
		name           string
		fromSignalOnly bool
		wrapper        RequestExtWrapper
		signal         *openrtb2.BidRequest
		want           *int
	}{
		{
			name:           "non sdk uses wrapper edsstatus",
			fromSignalOnly: false,
			wrapper:        wrapperWithEdsStatus,
			signal:         signalWithEdsStatus,
			want:           ptrutil.ToPtr(1),
		},
		{
			name:           "sdk uses signal edsstatus",
			fromSignalOnly: true,
			wrapper:        wrapperWithEdsStatus,
			signal:         signalWithEdsStatus,
			want:           ptrutil.ToPtr(2),
		},
		{
			name:           "sdk ignores wrapper when signal lacks edsstatus",
			fromSignalOnly: true,
			wrapper:        wrapperWithEdsStatus,
			signal:         signalWithoutEdsStatus,
			want:           nil,
		},
		{
			name:           "sdk ignores wrapper when signal is nil",
			fromSignalOnly: true,
			wrapper:        wrapperWithEdsStatus,
			signal:         nil,
			want:           nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ResolveEdsStatus(tt.fromSignalOnly, tt.wrapper, tt.signal))
		})
	}
}
