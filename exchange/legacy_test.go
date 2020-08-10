package exchange

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/buger/jsonparser"
	"github.com/evanphx/json-patch"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/currencies"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
	"github.com/prebid/prebid-server/usersync"
)

func TestSiteVideo(t *testing.T) {
	ortbRequest := &openrtb.BidRequest{
		ID:   "request-id",
		TMax: 1000,
		Site: &openrtb.Site{
			Page:   "http://www.site.com",
			Domain: "site.com",
			Publisher: &openrtb.Publisher{
				ID: "b1c81a38-1415-42b7-8238-0d2d64016c27",
			},
		},
		Source: &openrtb.Source{
			TID: "transaction-id",
		},
		User: &openrtb.User{
			ID:       "host-id",
			BuyerUID: "bidder-id",
		},
		Test: 1,
		Imp: []openrtb.Imp{{
			ID: "imp-id",
			Video: &openrtb.Video{
				MIMEs:       []string{"video/mp4"},
				MinDuration: 20,
				MaxDuration: 40,
				Protocols:   []openrtb.Protocol{openrtb.ProtocolVAST10},
				StartDelay:  openrtb.StartDelayGenericMidRoll.Ptr(),
			},
			Banner: &openrtb.Banner{
				Format: []openrtb.Format{{
					W: 300,
					H: 250,
				}},
			},
			Ext: json.RawMessage(`{"bidder":{"cp":512379,"ct":486653,"cf":"300x250"}}`),
		}},
	}

	mockAdapter := mockLegacyAdapter{}

	exchangeBidder := adaptLegacyAdapter(&mockAdapter)
	currencyConverter := currencies.NewRateConverter(&http.Client{}, "", time.Duration(0))
	_, errs := exchangeBidder.requestBid(context.Background(), ortbRequest, openrtb_ext.BidderRubicon, 1.0, currencyConverter.Rates(), &adapters.ExtraRequestInfo{})
	if len(errs) > 0 {
		t.Errorf("Unexpected error requesting bids: %v", errs)
	}

	if mockAdapter.gotRequest == nil {
		t.Fatalf("Mock adapter never received a request.")
	}

	if mockAdapter.gotBidder == nil {
		t.Fatalf("Mock adapter never received a bidder.")
	}

	assertEquivalentRequests(t, ortbRequest, mockAdapter.gotRequest)

	if mockAdapter.gotBidder.BidderCode != string(openrtb_ext.BidderRubicon) {
		t.Errorf("Wrong bidder code. Expected %s, got %s", string(openrtb_ext.BidderRubicon), mockAdapter.gotBidder.BidderCode)
	}
	assertEquivalentBidder(t, ortbRequest, mockAdapter.gotBidder)
}

func TestAppBanner(t *testing.T) {
	ortbRequest := newAppOrtbRequest()
	ortbRequest.TMax = 1000
	ortbRequest.User = &openrtb.User{
		ID:       "host-id",
		BuyerUID: "bidder-id",
	}
	ortbRequest.Test = 1

	mockAdapter := mockLegacyAdapter{}

	exchangeBidder := adaptLegacyAdapter(&mockAdapter)
	currencyConverter := currencies.NewRateConverter(&http.Client{}, "", time.Duration(0))
	_, errs := exchangeBidder.requestBid(context.Background(), ortbRequest, openrtb_ext.BidderRubicon, 1.0, currencyConverter.Rates(), &adapters.ExtraRequestInfo{})
	if len(errs) > 0 {
		t.Errorf("Unexpected error requesting bids: %v", errs)
	}

	if mockAdapter.gotRequest == nil {
		t.Fatalf("Mock adapter never received a request.")
	}

	if mockAdapter.gotBidder == nil {
		t.Fatalf("Mock adapter never received a bidder.")
	}
	if mockAdapter.gotBidder.BidderCode != string(openrtb_ext.BidderRubicon) {
		t.Errorf("Wrong bidder code. Expected %s, got %s", string(openrtb_ext.BidderRubicon), mockAdapter.gotBidder.BidderCode)
	}

	assertEquivalentRequests(t, ortbRequest, mockAdapter.gotRequest)
	assertEquivalentBidder(t, ortbRequest, mockAdapter.gotBidder)
}

func TestBidTransforms(t *testing.T) {
	bidAdjustment := 0.3
	initialBidPrice := 0.5
	legalBid := &pbs.PBSBid{
		BidID:             "bid-1",
		AdUnitCode:        "adunit-1",
		Creative_id:       "creative-1",
		CreativeMediaType: "banner",
		Price:             initialBidPrice,
		NURL:              "nurl",
		Adm:               "ad-markup",
		Width:             10,
		Height:            20,
		DealId:            "some-deal",
	}
	mockAdapter := mockLegacyAdapter{
		returnedBids: pbs.PBSBidSlice{
			legalBid,
			&pbs.PBSBid{
				CreativeMediaType: "unsupported",
			},
		},
	}

	exchangeBidder := adaptLegacyAdapter(&mockAdapter)
	currencyConverter := currencies.NewRateConverter(&http.Client{}, "", time.Duration(0))
	seatBid, errs := exchangeBidder.requestBid(context.Background(), newAppOrtbRequest(), openrtb_ext.BidderRubicon, bidAdjustment, currencyConverter.Rates(), &adapters.ExtraRequestInfo{})
	if len(errs) != 1 {
		t.Fatalf("Bad error count. Expected 1, got %d", len(errs))
	}
	if errs[0].Error() != "invalid BidType: unsupported" {
		t.Errorf("Unexpected error message. Got %s", errs[0].Error())
	}

	if len(seatBid.bids) != 1 {
		t.Fatalf("Bad bid count. Expected 1, got %d", len(seatBid.bids))
	}
	theBid := seatBid.bids[0]
	if theBid.bidType != openrtb_ext.BidTypeBanner {
		t.Errorf("Bad BidType. Expected banner, got %s", theBid.bidType)
	}
	if theBid.bid.ID != legalBid.BidID {
		t.Errorf("Bad id. Expected %s, got %s", legalBid.NURL, theBid.bid.NURL)
	}
	if theBid.bid.ImpID != legalBid.AdUnitCode {
		t.Errorf("Bad impid. Expected %s, got %s", legalBid.AdUnitCode, theBid.bid.ImpID)
	}
	if theBid.bid.CrID != legalBid.Creative_id {
		t.Errorf("Bad creativeid. Expected %s, got %s", legalBid.Creative_id, theBid.bid.CrID)
	}
	if theBid.bid.Price != initialBidPrice*bidAdjustment {
		t.Errorf("Bad price. Expected %f, got %f", initialBidPrice*bidAdjustment, theBid.bid.Price)
	}
	if theBid.bid.NURL != legalBid.NURL {
		t.Errorf("Bad NURL. Expected %s, got %s", legalBid.NURL, theBid.bid.NURL)
	}
	if theBid.bid.AdM != legalBid.Adm {
		t.Errorf("Bad adm. Expected %s, got %s", legalBid.Adm, theBid.bid.AdM)
	}
	if theBid.bid.W != legalBid.Width {
		t.Errorf("Bad adm. Expected %d, got %d", legalBid.Width, theBid.bid.W)
	}
	if theBid.bid.H != legalBid.Height {
		t.Errorf("Bad adm. Expected %d, got %d", legalBid.Height, theBid.bid.H)
	}
	if theBid.bid.DealID != legalBid.DealId {
		t.Errorf("Bad dealid. Expected %s, got %s", legalBid.DealId, theBid.bid.DealID)
	}
}

func TestInsecureImps(t *testing.T) {
	insecure := int8(0)
	bidReq := &openrtb.BidRequest{
		Imp: []openrtb.Imp{{
			Secure: &insecure,
		}, {
			Secure: &insecure,
		}},
	}
	isSecure, err := toSecure(bidReq)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if isSecure != 0 {
		t.Errorf("Final request should be insecure. Got %d", isSecure)
	}
}

func TestSecureImps(t *testing.T) {
	secure := int8(1)
	bidReq := &openrtb.BidRequest{
		Imp: []openrtb.Imp{{
			Secure: &secure,
		}, {
			Secure: &secure,
		}},
	}
	isSecure, err := toSecure(bidReq)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if isSecure != 1 {
		t.Errorf("Final request should be secure. Got %d", isSecure)
	}
}

func TestMixedSecureImps(t *testing.T) {
	insecure := int8(0)
	secure := int8(1)
	bidReq := &openrtb.BidRequest{
		Imp: []openrtb.Imp{{
			Secure: &insecure,
		}, {
			Secure: &secure,
		}},
	}
	_, err := toSecure(bidReq)
	if err == nil {
		t.Error("No error was received, but we should have gotten one.")
	}
}

func newAppOrtbRequest() *openrtb.BidRequest {
	return &openrtb.BidRequest{
		ID: "request-id",
		App: &openrtb.App{
			Publisher: &openrtb.Publisher{
				ID: "b1c81a38-1415-42b7-8238-0d2d64016c27",
			},
		},
		Source: &openrtb.Source{
			TID: "transaction-id",
		},
		Imp: []openrtb.Imp{{
			ID: "imp-id",
			Banner: &openrtb.Banner{
				Format: []openrtb.Format{{
					W: 300,
					H: 250,
				}},
			},
			Ext: json.RawMessage(`{"bidder":{"cp":512379,"ct":486653,"cf":"300x250"}}`),
		}},
	}
}

func TestErrorResponse(t *testing.T) {
	ortbRequest := &openrtb.BidRequest{
		ID: "request-id",
		App: &openrtb.App{
			Publisher: &openrtb.Publisher{
				ID: "b1c81a38-1415-42b7-8238-0d2d64016c27",
			},
		},
		Source: &openrtb.Source{
			TID: "transaction-id",
		},
		Imp: []openrtb.Imp{{
			ID: "imp-id",
			Banner: &openrtb.Banner{
				Format: []openrtb.Format{{
					W: 300,
					H: 250,
				}},
			},
			Ext: json.RawMessage(`{"bidder":{"cp":512379,"ct":486653,"cf":"300x250"}}`),
		}},
	}

	mockAdapter := mockLegacyAdapter{
		returnedError: errors.New("adapter failed"),
	}

	exchangeBidder := adaptLegacyAdapter(&mockAdapter)
	currencyConverter := currencies.NewRateConverter(&http.Client{}, "", time.Duration(0))
	_, errs := exchangeBidder.requestBid(context.Background(), ortbRequest, openrtb_ext.BidderRubicon, 1.0, currencyConverter.Rates(), &adapters.ExtraRequestInfo{})
	if len(errs) != 1 {
		t.Fatalf("Bad error count. Expected 1, got %d", len(errs))
	}
	if errs[0].Error() != "adapter failed" {
		t.Errorf("Unexpected error message. Got %s", errs[0].Error())
	}
}

func TestWithTargeting(t *testing.T) {
	ortbRequest := &openrtb.BidRequest{
		ID: "request-id",
		App: &openrtb.App{
			Publisher: &openrtb.Publisher{
				ID: "b1c81a38-1415-42b7-8238-0d2d64016c27",
			},
		},
		Source: &openrtb.Source{
			TID: "transaction-id",
		},
		Imp: []openrtb.Imp{{
			ID: "imp-id",
			Banner: &openrtb.Banner{
				Format: []openrtb.Format{{
					W: 300,
					H: 250,
				}},
			},
			Ext: json.RawMessage(`{"bidder": {"placementId": "1959066997713356_1959836684303054"}}`),
		}},
	}

	mockAdapter := mockLegacyAdapter{
		returnedBids: []*pbs.PBSBid{{
			CreativeMediaType: "banner",
		}},
	}
	exchangeBidder := adaptLegacyAdapter(&mockAdapter)
	currencyConverter := currencies.NewRateConverter(&http.Client{}, "", time.Duration(0))
	bid, errs := exchangeBidder.requestBid(context.Background(), ortbRequest, openrtb_ext.BidderFacebook, 1.0, currencyConverter.Rates(), &adapters.ExtraRequestInfo{})
	if len(errs) != 0 {
		t.Fatalf("This should not produce errors. Got %v", errs)
	}
	if len(bid.bids) != 1 {
		t.Fatalf("We should get one bid back.")
	}
	if bid.bids[0] == nil {
		t.Errorf("The returned bid should not be nil.")
	}
}

// assertEquivalentFields compares the OpenRTB request with the legacy request, using the mappings defined here:
// https://gist.github.com/dbemiller/68aa3387189fa17d3addfb9818dd4d97
func assertEquivalentRequests(t *testing.T, req *openrtb.BidRequest, legacy *pbs.PBSRequest) {
	if req.Site != nil {
		if req.Site.Publisher.ID != legacy.AccountID {
			t.Errorf("Account ID did not translate. OpenRTB: %s, Legacy: %s.", req.Site.Publisher.ID, legacy.AccountID)
		}
		if req.Site.Page != legacy.Url {
			t.Errorf("url did not translate. OpenRTB: %v, Legacy: %v.", req.Site.Page, legacy.Url)
		}
		if req.Site.Domain != legacy.Domain {
			t.Errorf("domain did not translate. OpenRTB: %v, Legacy: %v.", req.Site.Domain, legacy.Domain)
		}
	} else if req.App != nil {
		if req.App.Publisher.ID != legacy.AccountID {
			t.Errorf("Account ID did not translate. OpenRTB: %s, Legacy: %s.", req.Site.Publisher.ID, legacy.AccountID)
		}
	} else {
		t.Errorf("req.site and req.app are nil. This request was invalid.")
	}

	if req.Source.TID != legacy.Tid {
		t.Errorf("TID did not translate. OpenRTB: %s, Legacy: %s.", req.Source.TID, legacy.Tid)
	}

	expectedSecure := int8(0)
	if req.Imp[0].Secure != nil {
		expectedSecure = int8(*req.Imp[0].Secure)
	}

	if expectedSecure != legacy.Secure {
		t.Errorf("tmax did not translate. OpenRTB: %d, Legacy: %d.", expectedSecure, legacy.Secure)
	}
	// TODO: Secure

	if req.TMax != legacy.TimeoutMillis {
		t.Errorf("tmax did not translate. OpenRTB: %d, Legacy: %d.", req.TMax, legacy.TimeoutMillis)
	}

	if req.App != legacy.App {
		t.Errorf("app did not translate. OpenRTB: %v, Legacy: %v.", req.App, legacy.App)
	}
	if req.Device != legacy.Device {
		t.Errorf("device did not translate. OpenRTB: %v, Legacy: %v.", req.Device, legacy.Device)
	}
	if req.User != legacy.User {
		t.Errorf("user did not translate. OpenRTB: %v, Legacy: %v.", req.User, legacy.User)
	}
	if req.User != nil {
		if id, _, _ := legacy.Cookie.GetUID("someFamily"); id != req.User.BuyerUID {
			t.Errorf("bidder usersync did not translate. OpenRTB: %v, Legacy: %v.", req.User.BuyerUID, id)
		}
		if id, _, _ := legacy.Cookie.GetUID("adnxs"); id != req.User.ID {
			t.Errorf("user ID did not translate. OpenRTB: %v, Legacy: %v.", req.User.ID, id)
		}
	}
}

func assertEquivalentBidder(t *testing.T, req *openrtb.BidRequest, legacy *pbs.PBSBidder) {
	if len(req.Imp) != len(legacy.AdUnits) {
		t.Errorf("Wrong number of Imps. Expected %d, got %d", len(req.Imp), len(legacy.AdUnits))
		return
	}
	for i := 0; i < len(req.Imp); i++ {
		assertEquivalentImp(t, i, &req.Imp[i], &legacy.AdUnits[i])
	}
}

func assertEquivalentImp(t *testing.T, index int, imp *openrtb.Imp, legacy *pbs.PBSAdUnit) {
	if imp.ID != legacy.BidID {
		t.Errorf("imp[%d].id did not translate. OpenRTB %s, legacy %s", index, imp.ID, legacy.BidID)
	}

	if imp.Instl != legacy.Instl {
		t.Errorf("imp[%d].instl did not translate. OpenRTB %d, legacy %d", index, imp.Instl, legacy.Instl)
	}

	if params, _, _, _ := jsonparser.Get(imp.Ext, "bidder"); !jsonpatch.Equal(params, legacy.Params) {
		t.Errorf("imp[%d].ext.bidder did not translate. OpenRTB %s, legacy %s", index, string(params), string(legacy.Params))
	}

	if imp.Banner != nil {
		if imp.Banner.TopFrame != legacy.TopFrame {
			t.Errorf("imp[%d].topframe did not translate. OpenRTB %d, legacy %d", index, imp.Banner.TopFrame, legacy.TopFrame)
		}
		if imp.Banner.Format[0].W != legacy.Sizes[0].W {
			t.Errorf("imp[%d].format[0].w did not translate. OpenRTB %d, legacy %d", index, imp.Banner.Format[0].W, legacy.Sizes[0].W)
		}
		if imp.Banner.Format[0].H != legacy.Sizes[0].H {
			t.Errorf("imp[%d].format[0].h did not translate. OpenRTB %d, legacy %d", index, imp.Banner.Format[0].H, legacy.Sizes[0].H)
		}
	}

	if imp.Video != nil {
		if !reflect.DeepEqual(imp.Video.MIMEs, legacy.Video.Mimes) {
			t.Errorf("imp[%d].video.mimes did not translate. OpenRTB %v, legacy %v", index, imp.Video.MIMEs, legacy.Video.Mimes)
		}
		if len(imp.Video.Protocols) != len(imp.Video.Protocols) {
			t.Errorf("len(imp[%d].video.protocols) did not match. OpenRTB %d, legacy %d", index, len(imp.Video.Protocols), len(imp.Video.Protocols))
			return
		}
		for i := 0; i < len(imp.Video.Protocols); i++ {
			if int8(imp.Video.Protocols[i]) != legacy.Video.Protocols[i] {
				t.Errorf("imp[%d].video.protocol[%d] did not match. OpenRTB %d, legacy %d", index, i, imp.Video.Protocols[i], imp.Video.Protocols[i])
			}
		}
		if len(imp.Video.PlaybackMethod) > 0 {
			if int8(imp.Video.PlaybackMethod[0]) != legacy.Video.PlaybackMethod {
				t.Errorf("imp[%d].video.playbackmethod[0] did not translate. OpenRTB %d, legacy %d", index, int8(imp.Video.PlaybackMethod[0]), legacy.Video.PlaybackMethod)
			}
		}
		if imp.Video.Skip == nil {
			if legacy.Video.Skippable != 0 {
				t.Errorf("imp[%d].video.skip did not translate. OpenRTB nil, legacy %d", index, legacy.Video.Skippable)
			}
		} else {
			if int(*imp.Video.Skip) != legacy.Video.Skippable {
				t.Errorf("imp[%d].video.skip did not translate. OpenRTB %d, legacy %d", index, *imp.Video.Skip, legacy.Video.Skippable)
			}
		}
		if imp.Video.StartDelay == nil {
			if legacy.Video.Startdelay != 0 {
				t.Errorf("imp[%d].video.startdelay did not translate. OpenRTB nil, legacy %d", index, legacy.Video.Startdelay)
			}
		} else {
			if int64(*imp.Video.StartDelay) != legacy.Video.Startdelay {
				t.Errorf("imp[%d].video.startdelay did not translate. OpenRTB %d, legacy %d", index, int64(*imp.Video.StartDelay), legacy.Video.Startdelay)
			}
		}
		if imp.Video.MaxDuration != legacy.Video.Maxduration {
			t.Errorf("imp[%d].video.maxduration did not translate. OpenRTB %d, legacy %d", index, imp.Video.MaxDuration, legacy.Video.Maxduration)
		}
		if imp.Video.MinDuration != legacy.Video.Minduration {
			t.Errorf("imp[%d].video.minduration did not translate. OpenRTB %d, legacy %d", index, imp.Video.MinDuration, legacy.Video.Minduration)
		}
	}
}

type mockLegacyAdapter struct {
	returnedBids  pbs.PBSBidSlice
	returnedError error
	gotRequest    *pbs.PBSRequest
	gotBidder     *pbs.PBSBidder
}

func (a *mockLegacyAdapter) Name() string {
	return "someFamily"
}

func (a *mockLegacyAdapter) SkipNoCookies() bool {
	return false
}

func (a *mockLegacyAdapter) GetUsersyncInfo() (*usersync.UsersyncInfo, error) {
	return nil, nil
}

func (a *mockLegacyAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	a.gotRequest = req
	a.gotBidder = bidder
	return a.returnedBids, a.returnedError
}
