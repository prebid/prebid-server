package appnexus

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prebid/prebid-server/cache/dummycache"
	"github.com/prebid/prebid-server/pbs"

	"fmt"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// TestPlacementIdImp makes sure we make the correct HTTP requests when given a placementId
// and a valid Banner Imp.
func TestPlacementIdImp(t *testing.T) {
	bidder := new(AppNexusAdapter)

	request := &openrtb.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb.Imp{{
			ID: "test-imp-id",
			Banner: &openrtb.Banner{
				Format: []openrtb.Format{{
					W: 300,
					H: 250,
				}, {
					W: 300,
					H: 600,
				}},
			},
			Ext: openrtb.RawJSON(`{"bidder": {
				"placementId": 10433394,
				"reserve": 20,
				"position": "below",
				"trafficSourceCode": "trafficSource",
				"keywords": [
					{"key": "foo", "value": ["bar","baz"]},
					{"key": "valueless"}
				]
			}}`),
		}},
	}

	reqs, errs := bidder.MakeRequests(request)
	if len(errs) > 0 {
		t.Errorf("Got unexpected errors while building HTTP requests: %v", errs)
	}
	if len(reqs) != 1 {
		t.Fatalf("Unexpected number of HTTP requests. Got %d. Expected %d", len(reqs), 1)
	}

	httpReq := reqs[0]
	if httpReq.Method != "POST" {
		t.Errorf("Expected a POST message. Got %s", httpReq.Method)
	}
	if httpReq.Uri != "http://ib.adnxs.com/openrtb2" {
		t.Errorf("Bad URI. Expected %s, got %s", "http://ib.adnxs.com/openrtb2", httpReq.Uri)
	}

	var apnRequest openrtb.BidRequest
	if err := json.Unmarshal(httpReq.Body, &apnRequest); err != nil {
		t.Fatalf("Failed to unmarshal HTTP request: %v", apnRequest)
	}

	if apnRequest.ID != request.ID {
		t.Errorf("Bad Request ID. Expected %s, Got %s", request.ID, apnRequest.ID)
	}
	if len(apnRequest.Imp) != len(request.Imp) {
		t.Fatalf("Wrong len(request.Imp). Expected %d, Got %d", len(apnRequest.Imp), len(request.Imp))
	}
	assertImpsEqual(t, &apnRequest.Imp[0], &request.Imp[0])

	var apnExt appnexusImpExt
	if err := json.Unmarshal(apnRequest.Imp[0].Ext, &apnExt); err != nil {
		t.Fatal("Error unmarshalling request.imp[0].ext from the outgoing request.")
	}
	if apnExt.Appnexus.PlacementID != 10433394 {
		t.Errorf("Wrong placement ID. Expected %d, Got %d", 10433394, apnExt.Appnexus.PlacementID)
	}
	if apnExt.Appnexus.TrafficSourceCode != "trafficSource" {
		t.Errorf("Wrong trafficSourceCode. Expected %s, Got %s", "trafficSource", apnExt.Appnexus.TrafficSourceCode)
	}
	keywordsSent := apnExt.Appnexus.Keywords
	if keywordsSent != "foo=bar,foo=baz,valueless" {
		t.Errorf("Wrong keywords. Expected %s, Got %s", "foo=bar,foo=baz,valueless", keywordsSent)
	}
	if apnRequest.Imp[0].BidFloor != float64(20) {
		t.Errorf("The bid floor did not equal the reserve. Expected %f, got %f", float64(20), apnRequest.Imp[0].BidFloor)
	}
	if *apnRequest.Imp[0].Banner.Pos != openrtb.AdPositionBelowTheFold {
		t.Errorf("The banner should be sent as below the fold.")
	}
}

// TestMemberImp makes sure we make outgoing requests properly when given {member, invCode} params.
func TestMemberImp(t *testing.T) {
	bidder := new(AppNexusAdapter)

	request := &openrtb.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb.Imp{{
			ID: "test-imp-id",
			Video: &openrtb.Video{
				MIMEs: []string{"video/mp4"},
			},
			Ext: openrtb.RawJSON(`{"bidder": {
				"member": "someMember",
				"invCode": "someInvCode"
			}}`),
		}},
	}

	reqs, errs := bidder.MakeRequests(request)
	if len(errs) > 0 {
		t.Errorf("Got unexpected errors while building HTTP requests: %v", errs)
	}
	if len(reqs) != 1 {
		t.Fatalf("Unexpected number of HTTP requests. Got %d. Expected %d", len(reqs), 1)
	}
	if reqs[0].Uri != "http://ib.adnxs.com/openrtb2?member_id=someMember" {
		t.Errorf("Bad URI. Expected %s, got %s", "http://ib.adnxs.com/openrtb2?member_id=someMember", reqs[0].Uri)
	}

	var apnRequest openrtb.BidRequest
	if err := json.Unmarshal(reqs[0].Body, &apnRequest); err != nil {
		t.Fatalf("Failed to unmarshal HTTP request: %v", apnRequest)
	}

	if len(apnRequest.Imp) != 1 {
		t.Fatalf("Expected 1 Imp in the outgoing request. Got %d", len(apnRequest.Imp))
	}

	if apnRequest.Imp[0].TagID != "someInvCode" {
		t.Errorf("Bad request.imp.tagid. Expected %s, Got %s", "someInvCode", apnRequest.Imp[0].TagID)
	}

	var apnExt appnexusImpExt
	if err := json.Unmarshal(apnRequest.Imp[0].Ext, &apnExt); err != nil {
		t.Fatal("Error unmarshalling request.imp[0].ext from the outgoing request.")
	}
}

// TestAudioImp makes sure we don't make any http calls for audio imps, which are unsupported for now
func TestAudioImp(t *testing.T) {
	assertNoRequests(t, &openrtb.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb.Imp{{
			ID:    "test-imp-id",
			Audio: &openrtb.Audio{},
			Ext:   openrtb.RawJSON(`{"bidder": {"placementId":10433394}}`),
		}},
	})
}

// TestNativeImp makes sure we don't make any http calls for native imps, which are unsupported for now
func TestNativeImp(t *testing.T) {
	assertNoRequests(t, &openrtb.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb.Imp{{
			ID:     "test-imp-id",
			Native: &openrtb.Native{},
			Ext:    openrtb.RawJSON(`{"bidder": {"placementId":10433394}}`),
		}},
	})
}

// TestInvalidImpParams makes sure we don't make any http calls if the params are invalid.
func TestInvalidImpParams(t *testing.T) {
	assertNoRequests(t, &openrtb.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb.Imp{{
			ID: "test-imp-id",
			Banner: &openrtb.Banner{
				Format: []openrtb.Format{{
					W: 300,
					H: 250,
				}},
			},
			Ext: openrtb.RawJSON(`{"bidder": {}}`), // APN requires PlacementId or {member, invCode}
		}},
	})
}

// TestBestEffortBid makes sure that we send a request for the impressions we can handle, and
// return errors about those we can't.
func TestBestEffortBid(t *testing.T) {
	request := &openrtb.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb.Imp{{
			ID:    "test-imp-id",
			Audio: &openrtb.Audio{},
			Ext:   openrtb.RawJSON(`{"bidder": {"placementId":10433394}}`),
		}, {
			ID: "test-imp-id2",
			Video: &openrtb.Video{
				MIMEs: []string{"video/mp4"},
			},
			Ext: openrtb.RawJSON(`{"bidder": {"placementId":10433394}}`),
		}, {
			ID:    "test-imp-id3",
			Audio: &openrtb.Audio{},
			Ext:   openrtb.RawJSON(`{"bidder": {"placementId":10433394}}`),
		}, {
			ID: "test-imp-id4",
			Video: &openrtb.Video{
				MIMEs: []string{"video/mp4"},
			},
			Ext: openrtb.RawJSON(`{"bidder": {"placementId":10433394}}`),
		}},
	}

	bidder := new(AppNexusAdapter)
	httpReqs, errs := bidder.MakeRequests(request)
	if len(errs) != 2 {
		t.Errorf("We expect two errors for the unsupported impression types.")
	}
	if len(httpReqs) != 1 {
		t.Errorf("A server request is expected for the supported impressions.")
	}

	var apnRequest openrtb.BidRequest
	if err := json.Unmarshal(httpReqs[0].Body, &apnRequest); err != nil {
		t.Fatalf("Failed to unmarshal outgoing request: %v", err)
	}

	if len(apnRequest.Imp) != 2 {
		t.Fatalf("Outgoing request should include two impressions. Got %d", len(apnRequest.Imp))
	}

	seenIds := map[string]bool{
		"test-imp-id2": false,
		"test-imp-id4": false,
	}
	seenIds[apnRequest.Imp[0].ID] = true
	seenIds[apnRequest.Imp[1].ID] = true
	if !seenIds["test-imp-id2"] {
		t.Errorf("Outgoing request didn't contain expected Imp with ID=test-imp-id2")
	}
	if !seenIds["test-imp-id4"] {
		t.Errorf("Outgoing request didn't contain expected Imp with ID=test-imp-id4")
	}
}

func assertNoRequests(t *testing.T, request *openrtb.BidRequest) {
	t.Helper()

	bidder := new(AppNexusAdapter)
	httpReqs, errs := bidder.MakeRequests(request)
	if len(httpReqs) != 0 {
		t.Errorf("Expected no http requests. Got %d", len(httpReqs))
	}
	if len(errs) < 1 {
		t.Errorf("The adapter should have returned at least one error.")
	}
}

func assertFormatsEqual(t *testing.T, actual *openrtb.Format, expected *openrtb.Format) {
	if actual.W != expected.W {
		t.Errorf("Widths don't match. Expected %d, Got %d", expected.W, actual.W)
	}
	if actual.H != expected.H {
		t.Errorf("Heights don't match. Expected %d, Got %d", expected.H, actual.H)
	}
}

func assertImpsEqual(t *testing.T, actual *openrtb.Imp, expected *openrtb.Imp) {
	if actual.ID != expected.ID {
		t.Errorf("IDs don't match. Expected %s, Got %s", expected.ID, actual.ID)
	}
	if (actual.Banner == nil) != (expected.Banner == nil) {
		t.Errorf("Imp types don't match. Expected banner: %t, Got banner: %t", expected.Banner != nil, actual.Banner != nil)
		return
	}
	if actual.Banner != nil {
		if len(actual.Banner.Format) != len(expected.Banner.Format) {
			t.Errorf("Wrong len(imp.banner.format). Expected %d, Got %d", len(expected.Banner.Format), len(actual.Banner.Format))
			return
		}

		for index, actualFormat := range actual.Banner.Format {
			expectedFormat := expected.Banner.Format[index]
			assertFormatsEqual(t, &actualFormat, &expectedFormat)
		}
	}
}

// TestEmptyResponse makes sure we handle NoContent responses properly.
func TestEmptyResponse(t *testing.T) {
	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusNoContent,
	}
	bidder := new(AppNexusAdapter)
	bids, errs := bidder.MakeBids(nil, httpResp)
	if len(bids) != 0 {
		t.Errorf("Expected 0 bids. Got %d", len(bids))
	}
	if len(errs) != 0 {
		t.Errorf("Expected 0 errors. Got %d", len(errs))
	}
}

// TestEmptyResponse makes sure we handle unexpected status codes properly.
func TestSurpriseResponse(t *testing.T) {
	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusAccepted,
	}
	bidder := new(AppNexusAdapter)
	bids, errs := bidder.MakeBids(nil, httpResp)
	if len(bids) != 0 {
		t.Errorf("Expected 0 bids. Got %d", len(bids))
	}
	if len(errs) != 1 {
		t.Errorf("Expected 1 error. Got %d", len(errs))
	}
}

// TestStandardResponse makes sure we handle normal responses properly.
func TestStandardResponse(t *testing.T) {
	request := &openrtb.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb.Imp{{
			ID: "test-imp-id",
			Banner: &openrtb.Banner{
				Format: []openrtb.Format{{
					W: 300,
					H: 250,
				}, {
					W: 300,
					H: 600,
				}},
			},
			Ext: openrtb.RawJSON(`{"bidder": {
				"placementId": 10433394
			}}`),
		}},
	}

	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       []byte(`{"id":"test-request-id","seatbid":[{"bid":[{"id":"1470007324018331644","impid":"test-imp-id","price": 0.500000,"adid":"29681110","adm":"some ad","adomain":["appnexus.com"],"iurl":"http://nym1-ib.adnxs.com/cr?id=29681110","cid":"958","crid":"29681110","h": 250,"w": 300,"ext":{"appnexus":{"brand_id": 1,"auction_id": 2056774007789312974,"bidder_id": 2}}}],"seat":"958"}],"bidid":"8787902579030770524","cur":"USD"}`),
	}

	bidder := new(AppNexusAdapter)
	bids, errs := bidder.MakeBids(request, httpResp)
	if len(bids) != 1 {
		t.Fatalf("Expected 1 bid. Got %d", len(bids))
	}
	if len(errs) != 0 {
		t.Errorf("Expected 0 errors. Got %d", len(errs))
	}
	if bids[0].BidType != openrtb_ext.BidTypeBanner {
		t.Errorf("Expected a banner bid. Got: %s", bids[0].BidType)
	}
	theBid := bids[0].Bid
	if theBid.ID != "1470007324018331644" {
		t.Errorf("Bad bid ID. Expected %s, got %s", "1470007324018331644", theBid.ID)
	}
}

// ----------------------------------------------------------------------------
// Code below this line tests the legacy, non-openrtb code flow. It can be deleted after we
// clean up the existing code and make everything openrtb.

type anTagInfo struct {
	code              string
	invCode           string
	placementID       int
	trafficSourceCode string
	in_keywords       string
	out_keywords      string
	reserve           float64
	position          string
	bid               float64
	content           string
	mediaType         string
}

type anBidInfo struct {
	memberID  string
	domain    string
	page      string
	accountID int
	siteID    int
	tags      []anTagInfo
	deviceIP  string
	deviceUA  string
	buyerUID  string
	delay     time.Duration
}

var andata anBidInfo

func DummyAppNexusServer(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var breq openrtb.BidRequest
	err = json.Unmarshal(body, &breq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	memberID := r.FormValue("member_id")
	if memberID != andata.memberID {
		http.Error(w, fmt.Sprintf("Member ID '%s' doesn't match '%s", memberID, andata.memberID), http.StatusInternalServerError)
		return
	}

	resp := openrtb.BidResponse{
		ID:    breq.ID,
		BidID: "a-random-id",
		Cur:   "USD",
		SeatBid: []openrtb.SeatBid{
			{
				Seat: "Buyer Member ID",
				Bid:  make([]openrtb.Bid, 0, 2),
			},
		},
	}

	for i, imp := range breq.Imp {
		var aix appnexusImpExt
		err = json.Unmarshal(imp.Ext, &aix)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Either placementID or member+invCode must be specified
		has_placement := false
		if aix.Appnexus.PlacementID != 0 {
			if aix.Appnexus.PlacementID != andata.tags[i].placementID {
				http.Error(w, fmt.Sprintf("Placement ID '%d' doesn't match '%d", aix.Appnexus.PlacementID,
					andata.tags[i].placementID), http.StatusInternalServerError)
				return
			}
			has_placement = true
		}
		if memberID != "" && imp.TagID != "" {
			if imp.TagID != andata.tags[i].invCode {
				http.Error(w, fmt.Sprintf("Inv Code '%s' doesn't match '%s", imp.TagID,
					andata.tags[i].invCode), http.StatusInternalServerError)
				return
			}
			has_placement = true
		}
		if !has_placement {
			http.Error(w, fmt.Sprintf("Either placement or member+inv not present"), http.StatusInternalServerError)
			return
		}

		if aix.Appnexus.Keywords != andata.tags[i].out_keywords {
			http.Error(w, fmt.Sprintf("Keywords '%s' doesn't match '%s", aix.Appnexus.Keywords,
				andata.tags[i].out_keywords), http.StatusInternalServerError)
			return
		}

		if aix.Appnexus.TrafficSourceCode != andata.tags[i].trafficSourceCode {
			http.Error(w, fmt.Sprintf("Traffic source code '%s' doesn't match '%s", aix.Appnexus.TrafficSourceCode,
				andata.tags[i].trafficSourceCode), http.StatusInternalServerError)
			return
		}
		if imp.BidFloor != andata.tags[i].reserve {
			http.Error(w, fmt.Sprintf("Bid floor '%.2f' doesn't match '%.2f", imp.BidFloor,
				andata.tags[i].reserve), http.StatusInternalServerError)
			return
		}
		if imp.Banner == nil && imp.Video == nil {
			http.Error(w, fmt.Sprintf("No banner or app object sent"), http.StatusInternalServerError)
			return
		}
		if (imp.Banner == nil && andata.tags[i].mediaType == "banner") || (imp.Banner != nil && andata.tags[i].mediaType != "banner") {
			http.Error(w, fmt.Sprintf("Invalid impression type - banner"), http.StatusInternalServerError)
			return
		}
		if (imp.Video == nil && andata.tags[i].mediaType == "video") || (imp.Video != nil && andata.tags[i].mediaType != "video") {
			http.Error(w, fmt.Sprintf("Invalid impression type - video"), http.StatusInternalServerError)
			return
		}

		if imp.Banner != nil {
			if len(imp.Banner.Format) == 0 {
				http.Error(w, fmt.Sprintf("Empty imp.banner.format array"), http.StatusInternalServerError)
				return
			}
			if andata.tags[i].position == "above" && *imp.Banner.Pos != openrtb.AdPosition(1) {
				http.Error(w, fmt.Sprintf("Mismatch in position - expected 1 for atf"), http.StatusInternalServerError)
				return
			}
			if andata.tags[i].position == "below" && *imp.Banner.Pos != openrtb.AdPosition(3) {
				http.Error(w, fmt.Sprintf("Mismatch in position - expected 3 for btf"), http.StatusInternalServerError)
				return
			}
		}
		if imp.Video != nil {
			// TODO: add more validations
			if len(imp.Video.MIMEs) == 0 {
				http.Error(w, fmt.Sprintf("Empty imp.video.mimes array"), http.StatusInternalServerError)
				return
			}
			if len(imp.Video.Protocols) == 0 {
				http.Error(w, fmt.Sprintf("Empty imp.video.protocols array"), http.StatusInternalServerError)
				return
			}
			for _, protocol := range imp.Video.Protocols {
				if protocol < 1 || protocol > 8 {
					http.Error(w, fmt.Sprintf("Invalid video protocol %d", protocol), http.StatusInternalServerError)
					return
				}
			}
		}

		resBid := openrtb.Bid{
			ID:    "random-id",
			ImpID: imp.ID,
			Price: andata.tags[i].bid,
			AdM:   andata.tags[i].content,
		}

		if imp.Video != nil {
			resBid.Attr = []openrtb.CreativeAttribute{openrtb.CreativeAttribute(6)}
		}
		resp.SeatBid[0].Bid = append(resp.SeatBid[0].Bid, resBid)
	}

	// TODO: are all of these valid for app?
	if breq.Site == nil {
		http.Error(w, fmt.Sprintf("No site object sent"), http.StatusInternalServerError)
		return
	}
	if breq.Site.Domain != andata.domain {
		http.Error(w, fmt.Sprintf("Domain '%s' doesn't match '%s", breq.Site.Domain, andata.domain), http.StatusInternalServerError)
		return
	}
	if breq.Site.Page != andata.page {
		http.Error(w, fmt.Sprintf("Page '%s' doesn't match '%s", breq.Site.Page, andata.page), http.StatusInternalServerError)
		return
	}
	if breq.Device.UA != andata.deviceUA {
		http.Error(w, fmt.Sprintf("UA '%s' doesn't match '%s", breq.Device.UA, andata.deviceUA), http.StatusInternalServerError)
		return
	}
	if breq.Device.IP != andata.deviceIP {
		http.Error(w, fmt.Sprintf("IP '%s' doesn't match '%s", breq.Device.IP, andata.deviceIP), http.StatusInternalServerError)
		return
	}
	if breq.User.BuyerUID != andata.buyerUID {
		http.Error(w, fmt.Sprintf("User ID '%s' doesn't match '%s", breq.User.BuyerUID, andata.buyerUID), http.StatusInternalServerError)
		return
	}

	if andata.delay > 0 {
		<-time.After(andata.delay)
	}

	js, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func TestAppNexusBasicResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(DummyAppNexusServer))
	defer server.Close()

	andata = anBidInfo{
		domain:   "nytimes.com",
		page:     "https://www.nytimes.com/2017/05/04/movies/guardians-of-the-galaxy-2-review-chris-pratt.html?hpw&rref=movies&action=click&pgtype=Homepage&module=well-region&region=bottom-well&WT.nav=bottom-well&_r=0",
		tags:     make([]anTagInfo, 2),
		deviceIP: "25.91.96.36",
		deviceUA: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_4) AppleWebKit/603.1.30 (KHTML, like Gecko) Version/10.1 Safari/603.1.30",
		buyerUID: "23482348223",
		memberID: "958",
	}
	andata.tags[0] = anTagInfo{
		code:              "first-tag",
		placementID:       8394,
		bid:               1.67,
		trafficSourceCode: "ppc-exchange",
		content:           "<html><body>huh</body></html>",
		in_keywords:       "[{ \"key\": \"genre\", \"value\": [\"jazz\", \"pop\"] }, {\"key\": \"myEmptyKey\", \"value\": []}]",
		out_keywords:      "genre=jazz,genre=pop,myEmptyKey",
		reserve:           1.50,
		position:          "below",
		mediaType:         "banner",
	}
	andata.tags[1] = anTagInfo{
		code:              "second-tag",
		invCode:           "leftbottom",
		bid:               3.22,
		trafficSourceCode: "taboola",
		content:           "<html><body>yow!</body></html>",
		in_keywords:       "[{ \"key\": \"genre\", \"value\": [\"rock\", \"pop\"] }, {\"key\": \"myKey\", \"value\": [\"myVal\"]}]",
		out_keywords:      "genre=rock,genre=pop,myKey=myVal",
		reserve:           0.75,
		position:          "above",
		mediaType:         "video",
	}

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewAppNexusAdapter(&conf, server.URL)
	an.URI = server.URL

	pbin := pbs.PBSRequest{
		AdUnits: make([]pbs.AdUnit, 2),
	}
	for i, tag := range andata.tags {
		var params json.RawMessage
		if tag.placementID > 0 {
			params = json.RawMessage(fmt.Sprintf("{\"placementId\": %d, \"member\": \"%s\", \"keywords\": %s, "+
				"\"trafficSourceCode\": \"%s\", \"reserve\": %.3f, \"position\": \"%s\"}",
				tag.placementID, andata.memberID, tag.in_keywords, tag.trafficSourceCode, tag.reserve, tag.position))
		} else {
			params = json.RawMessage(fmt.Sprintf("{\"invCode\": \"%s\", \"member\": \"%s\", \"keywords\": %s, "+
				"\"trafficSourceCode\": \"%s\", \"reserve\": %.3f, \"position\": \"%s\"}",
				tag.invCode, andata.memberID, tag.in_keywords, tag.trafficSourceCode, tag.reserve, tag.position))
		}

		pbin.AdUnits[i] = pbs.AdUnit{
			Code:       tag.code,
			MediaTypes: []string{tag.mediaType},
			Sizes: []openrtb.Format{
				{
					W: 300,
					H: 600,
				},
				{
					W: 300,
					H: 250,
				},
			},
			Bids: []pbs.Bids{
				pbs.Bids{
					BidderCode: "appnexus",
					BidID:      fmt.Sprintf("random-id-from-pbjs-%d", i),
					Params:     params,
				},
			},
		}
		if tag.mediaType == "video" {
			pbin.AdUnits[i].Video = pbs.PBSVideo{
				Mimes:          []string{"video/mp4"},
				Minduration:    15,
				Maxduration:    30,
				Startdelay:     5,
				Skippable:      0,
				PlaybackMethod: 1,
				Protocols:      []int8{2, 3},
			}
		}
	}

	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(pbin)
	if err != nil {
		t.Fatalf("Json encoding failed: %v", err)
	}

	req := httptest.NewRequest("POST", server.URL, body)
	req.Header.Add("Referer", andata.page)
	req.Header.Add("User-Agent", andata.deviceUA)
	req.Header.Add("X-Real-IP", andata.deviceIP)

	pc := pbs.ParsePBSCookieFromRequest(req, &config.Cookie{})
	pc.TrySync("adnxs", andata.buyerUID)
	fakewriter := httptest.NewRecorder()
	pc.SetCookieOnResponse(fakewriter, "")
	req.Header.Add("Cookie", fakewriter.Header().Get("Set-Cookie"))

	cacheClient, _ := dummycache.New()
	hcs := pbs.HostCookieSettings{}

	pbReq, err := pbs.ParsePBSRequest(req, cacheClient, &hcs)
	if err != nil {
		t.Fatalf("ParsePBSRequest failed: %v", err)
	}
	if len(pbReq.Bidders) != 1 {
		t.Fatalf("ParsePBSRequest returned %d bidders instead of 1", len(pbReq.Bidders))
	}
	if pbReq.Bidders[0].BidderCode != "appnexus" {
		t.Fatalf("ParsePBSRequest returned invalid bidder")
	}

	ctx := context.TODO()
	bids, err := an.Call(ctx, pbReq, pbReq.Bidders[0])
	if err != nil {
		t.Fatalf("Should not have gotten an error: %v", err)
	}
	if len(bids) != 2 {
		t.Fatalf("Received %d bids instead of 2", len(bids))
	}
	for _, bid := range bids {
		matched := false
		for _, tag := range andata.tags {
			if bid.AdUnitCode == tag.code {
				matched = true
				if bid.CreativeMediaType != tag.mediaType {
					t.Errorf("Incorrect Creative MediaType '%s'", bid.CreativeMediaType)
				}
				if bid.BidderCode != "appnexus" {
					t.Errorf("Incorrect BidderCode '%s'", bid.BidderCode)
				}
				if bid.Price != tag.bid {
					t.Errorf("Incorrect bid price '%.2f' expected '%.2f'", bid.Price, tag.bid)
				}
				if bid.Adm != tag.content {
					t.Errorf("Incorrect bid markup '%s' expected '%s'", bid.Adm, tag.content)
				}
			}
		}
		if !matched {
			t.Errorf("Received bid for unknown ad unit '%s'", bid.AdUnitCode)
		}
	}

	// same test but with request timing out
	andata.delay = 5 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	bids, err = an.Call(ctx, pbReq, pbReq.Bidders[0])
	if err == nil {
		t.Fatalf("Should have gotten a timeout error: %v", err)
	}
}

func TestAppNexusUserSyncInfo(t *testing.T) {
	an := NewAppNexusAdapter(adapters.DefaultHTTPAdapterConfig, "localhost")
	if an.usersyncInfo.URL != "//ib.adnxs.com/getuid?localhost%2Fsetuid%3Fbidder%3Dadnxs%26uid%3D%24UID" {
		t.Fatalf("should have matched")
	}
	if an.usersyncInfo.Type != "redirect" {
		t.Fatalf("should be redirect")
	}
	if an.usersyncInfo.SupportCORS != false {
		t.Fatalf("should have been false")
	}
}
