package bidtheatre

import (
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"strings"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderBidtheatre, config.Adapter{
		Endpoint: "http://any.url"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "bidtheatretest", bidder)
}

func TestGetBidTypes(t *testing.T) {
	mockBid := openrtb2.Bid{
		ID:     "mock-bid-id",
		ImpID:  "mock-imp-id",
		Price:  1.23,
		AdID:   "mock-ad-id",
		CrID:   "mock-cr-id",
		DealID: "mock-deal-id",
		W:      980,
		H:      240,
		Ext:    []byte(`{"prebid": {"type": "banner"}}`),
		BURL:   "https://example.com/win-notify",
		Cat:    []string{"IAB1"},
	}

	actualBidTypeValue, _ := getMediaTypeForBid(mockBid)

	if actualBidTypeValue != openrtb_ext.BidTypeBanner {
		t.Errorf("Expected Bid Type value was: %v, actual value is: %v", openrtb_ext.BidTypeBanner, actualBidTypeValue)
	}

	mockBid.Ext = []byte(`{"prebid": {"type": "video"}}`)

	actualBidTypeValue, _ = getMediaTypeForBid(mockBid)

	if actualBidTypeValue != openrtb_ext.BidTypeVideo {
		t.Errorf("Expected Bid Type value was: %v, actual value is: %v", openrtb_ext.BidTypeVideo, actualBidTypeValue)
	}

}

func TestReplaceMacros(t *testing.T) {
	mockBid := openrtb2.Bid{
		ID:     "mock-bid-id",
		ImpID:  "mock-imp-id",
		Price:  1.23,
		AdID:   "mock-ad-id",
		CrID:   "mock-cr-id",
		DealID: "mock-deal-id",
		W:      980,
		H:      240,
		Ext:    []byte(`{"prebid": {"type": "banner"}}`),
		BURL:   "https://example.com/win-notify",
		Cat:    []string{"IAB1"},
		AdM:    "<script type=\"text/javascript\">\n    var uri = 'https://adsby.bidtheatre.com/imp?z=27025&a=1915538&so=1&ex=36&eb=3672319&xs=940698616&wp=${AUCTION_PRICE}&su=unknown&es=prebid.org&tag=unspec_980_300&kuid=eab9340e-8731-4027-9ada-57b554c75501&dealId=&mp=&ma=eyJjZCI6ZmFsc2UsInN0IjoxLCJtbGF0Ijo1OS4zMjkzLCJhZGMiOi0xLCJtb3JnIjoidGVsaWEgbmV0d29yayBzZXJ2aWNlcyIsIm1sc2NvcmUiOjAuMDczMTkwMTc0OTk2ODUyODcsIm16aXAiOiIxMTEyMCIsImJpcCI6IjgxLjIyNy44Mi4yOCIsImFnaWQiOjM1NjI3MDIsIm1sbW9kZWwiOiJtYXN0ZXJfbWxfY2xrXzU0MyIsInVhIjoiY3VybFwvNy44Ny4wIiwiYnJyZSI6ImFiIiwibWxvbiI6MTguMDY4NiwibXJlZ2lvbiI6ImFiIiwiZHQiOjgsImJyY28iOiJzd2UiLCJtY2l0eSI6InN0b2NraG9sbSIsImJyY2kiOiJzdG9ja2hvbG0iLCJwYWdldXJsIjoicHJlYmlkLm9yZyIsImltcGlkIjoieDM2X2FzeC1iLXMyXzc4MTc1MTk2NTcxMDMzNjUyMDciLCJtY291bnRyeSI6InN3ZSIsInRzIjoxNzMyMTEyNzMxNjgyfQ%3D%3D&usersync=1&cd=0&impId=x36_asx-b-s2_7817519657103365207&gdpr=0&gdpr_consent=&cb0=&rnd=' + new String (Math.random()).substring (2, 11);\n    document.write('<sc'+'ript type=\"text/javascript\" src=\"'+uri+'\" charset=\"ISO-8859-1\"></sc'+'ript>');\n</script>",
		NURL:   "https://adsby.bidtheatre.com/video?z=27025;a=1922926;ex=36;es=prebid.org;eb=3672319;xs=940698616;so=1;tag=unspec_640_360;kuid=1d10dda6-740d-4386-94a0-7042b2ad2a66;wp=${AUCTION_PRICE};su=unknown;iab=vast2;dealId=;ma=eyJjZCI6ZmFsc2UsInN0IjozLCJtbGF0Ijo1OS4zMjkzLCJhZGMiOi0xLCJtb3JnIjoidGVsaWEgbmV0d29yayBzZXJ2aWNlcyIsIm1sc2NvcmUiOjkuNTY5ODA3NDQzNzY3Nzg2RS00LCJtemlwIjoiMTExMjAiLCJiaXAiOiI4MS4yMjcuODIuMjgiLCJhZ2lkIjozNTYyNzAyLCJtbG1vZGVsIjoibWFzdGVyX21sX2Nsa181NDMiLCJ1YSI6ImN1cmxcLzcuODcuMCIsImJycmUiOiJhYiIsIm1sb24iOjE4LjA2ODYsIm1yZWdpb24iOiJhYiIsImR0Ijo4LCJicmNvIjoic3dlIiwibWNpdHkiOiJzdG9ja2hvbG0iLCJicmNpIjoic3RvY2tob2xtIiwicGFnZXVybCI6InByZWJpZC5vcmciLCJpbXBpZCI6IngzNl9hc3gtYi1zMV8yNTY5OTI0ODYzMjY2ODA4OTM2IiwibWNvdW50cnkiOiJzd2UiLCJ0cyI6MTczMjA5NjgyNjg5OH0%3D;cd=0;cb0=;impId=x36_asx-b-s1_2569924863266808936;gdpr=0;gdpr_consent=",
	}

	resolveMacros(&mockBid)

	if !strings.Contains(mockBid.AdM, "&wp=1.23&") {
		t.Errorf("AdM ${AUCTION_PRICE} not correctly replaced")
	}

	if strings.Contains(mockBid.AdM, "${AUCTION_PRICE}") {
		t.Errorf("AdM ${AUCTION_PRICE} not correctly replaced")
	}

	if !strings.Contains(mockBid.NURL, ";wp=1.23;") {
		t.Errorf("NURL ${AUCTION_PRICE} not correctly replaced")
	}

	if strings.Contains(mockBid.NURL, "${AUCTION_PRICE}") {
		t.Errorf("NURL ${AUCTION_PRICE} not correctly replaced")
	}

}
