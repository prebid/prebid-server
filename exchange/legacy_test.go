package exchange

import (
	"context"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
	"testing"
	"reflect"
	"github.com/buger/jsonparser"
	"github.com/evanphx/json-patch"
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
			ID: "host-id",
			BuyerUID: "bidder-id",
		},
		Test: 1,
		Imp: []openrtb.Imp{{
			ID: "imp-id",
			Video: &openrtb.Video{
				MIMEs: []string{"video/mp4"},
				MinDuration: 20,
				MaxDuration: 40,
				Protocols: []openrtb.Protocol{openrtb.ProtocolVAST10},
				StartDelay: openrtb.StartDelayGenericMidRoll.Ptr(),
			},
			Banner: &openrtb.Banner{
				Format: []openrtb.Format{{
					W: 300,
					H: 250,
				}},
			},
			Ext: openrtb.RawJSON(`{"bidder":{"cp":512379,"ct":486653,"cf":"300x250"}}`),
		}},
	}

	mockAdapter := mockLegacyAdapter{}

	exchangeBidder := adaptLegacyAdapter(&mockAdapter)
	_, errs := exchangeBidder.requestBid(context.Background(), ortbRequest, nil, openrtb_ext.BidderRubicon)
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
	ortbRequest := &openrtb.BidRequest{
		ID:   "request-id",
		TMax: 1000,
		App: &openrtb.App{
			Publisher: &openrtb.Publisher{
				ID: "b1c81a38-1415-42b7-8238-0d2d64016c27",
			},
		},
		Source: &openrtb.Source{
			TID: "transaction-id",
		},
		User: &openrtb.User{
			ID: "host-id",
			BuyerUID: "bidder-id",
		},
		Test: 1,
		Imp: []openrtb.Imp{{
			ID: "imp-id",
			Banner: &openrtb.Banner{
				Format: []openrtb.Format{{
					W: 300,
					H: 250,
				}},
			},
			Ext: openrtb.RawJSON(`{"bidder":{"cp":512379,"ct":486653,"cf":"300x250"}}`),
		}},
	}

	mockAdapter := mockLegacyAdapter{}

	exchangeBidder := adaptLegacyAdapter(&mockAdapter)
	_, errs := exchangeBidder.requestBid(context.Background(), ortbRequest, nil, openrtb_ext.BidderRubicon)
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
		t.Errorf("tmax did not translate. OpenRTB: %s, Legacy: %s.", expectedSecure, legacy.Secure)
	}
	// TODO: Secure

	if req.TMax != legacy.TimeoutMillis {
		t.Errorf("tmax did not translate. OpenRTB: %s, Legacy: %s.", req.TMax, legacy.TimeoutMillis)
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
		if id, _, _ := legacy.Cookie.GetUID("rubicon"); id != req.User.BuyerUID {
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
		t.Errorf("imp[%d].instl did not translate. OpenRTB %s, legacy %s", index, imp.Instl, legacy.Instl)
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
			t.Errorf("imp[%d].video.mimes did not translate. OpenRTB %d, legacy %d", index, imp.Video.MIMEs, legacy.Video.Mimes)
		}
		if len(imp.Video.Protocols) != len()
		if !reflect.DeepEqual(imp.Video.Protocols, legacy.Video.Protocols) {
			t.Errorf("imp[%d].video.protocols did not translate. OpenRTB %d, legacy %d", index, imp.Video.Protocols, legacy.Video.Protocols)
		}
		if int8(imp.Video.PlaybackMethod[0]) != legacy.Video.PlaybackMethod {
			t.Errorf("imp[%d].video.playbackmethod[0] did not translate. OpenRTB %d, legacy %d", index, int8(imp.Video.PlaybackMethod[0]), legacy.Video.PlaybackMethod)
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
	return "someBidder"
}

func (a *mockLegacyAdapter) FamilyName() string {
	return "someFamily"
}

func (a *mockLegacyAdapter) SkipNoCookies() bool {
	return false
}

func (a *mockLegacyAdapter) GetUsersyncInfo() *pbs.UsersyncInfo {
	return nil
}

func (a *mockLegacyAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	a.gotRequest = req
	a.gotBidder = bidder
	return a.returnedBids, a.returnedError
}
