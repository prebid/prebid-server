package appnexus

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/stretchr/testify/assert"

	"github.com/prebid/prebid-server/cache/dummycache"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
	"github.com/prebid/prebid-server/usersync"

	"fmt"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAppnexus, config.Adapter{
		Endpoint: "http://ib.adnxs.com/openrtb2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "appnexustest", bidder)
}

func TestVideoSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAppnexus, config.Adapter{
		Endpoint:   "http://ib.adnxs.com/openrtb2",
		PlatformID: "8"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "appnexusplatformtest", bidder)
}

func TestMemberQueryParam(t *testing.T) {
	uriWithMember := appendMemberId("http://ib.adnxs.com/openrtb2?query_param=true", "102")
	expected := "http://ib.adnxs.com/openrtb2?query_param=true&member_id=102"
	if uriWithMember != expected {
		t.Errorf("appendMemberId() failed on URI with query string. Expected %s, got %s", expected, uriWithMember)
	}
}

func TestVideoSinglePod(t *testing.T) {
	var a AppNexusAdapter
	a.URI = "http://test.com/openrtb2"
	a.hbSource = 5

	var reqInfo adapters.ExtraRequestInfo
	reqInfo.PbsEntryPoint = "video"

	var req openrtb2.BidRequest
	req.ID = "test_id"

	reqExt := `{"prebid":{}}`
	impExt := `{"bidder":{"placementId":123, "generate_ad_pod_id":true}}`
	req.Ext = []byte(reqExt)

	req.Imp = append(req.Imp, openrtb2.Imp{ID: "1_0", Ext: []byte(impExt)})
	req.Imp = append(req.Imp, openrtb2.Imp{ID: "1_1", Ext: []byte(impExt)})
	req.Imp = append(req.Imp, openrtb2.Imp{ID: "1_2", Ext: []byte(impExt)})

	result, err := a.MakeRequests(&req, &reqInfo)

	assert.Empty(t, err, "Errors array should be empty")
	assert.Len(t, result, 1, "Only one request should be returned")

	var error error
	var reqData *openrtb2.BidRequest
	error = json.Unmarshal(result[0].Body, &reqData)
	assert.NoError(t, error, "Response body unmarshalling error should be nil")

	var reqDataExt *appnexusReqExt
	error = json.Unmarshal(reqData.Ext, &reqDataExt)
	assert.NoError(t, error, "Response ext unmarshalling error should be nil")

	regMatch, matchErr := regexp.Match(`^[0-9]+$`, []byte(reqDataExt.Appnexus.AdPodId))
	assert.NoError(t, matchErr, "Regex match error should be nil")
	assert.True(t, regMatch, "AdPod id doesn't present in Appnexus extension or has incorrect format")
}

func TestVideoSinglePodManyImps(t *testing.T) {
	var a AppNexusAdapter
	a.URI = "http://test.com/openrtb2"
	a.hbSource = 5

	var reqInfo adapters.ExtraRequestInfo
	reqInfo.PbsEntryPoint = "video"

	var req openrtb2.BidRequest
	req.ID = "test_id"

	reqExt := `{"prebid":{}}`
	impExt := `{"bidder":{"placementId":123}}`
	req.Ext = []byte(reqExt)

	req.Imp = append(req.Imp, openrtb2.Imp{ID: "1_0", Ext: []byte(impExt)})
	req.Imp = append(req.Imp, openrtb2.Imp{ID: "1_1", Ext: []byte(impExt)})
	req.Imp = append(req.Imp, openrtb2.Imp{ID: "1_2", Ext: []byte(impExt)})
	req.Imp = append(req.Imp, openrtb2.Imp{ID: "1_3", Ext: []byte(impExt)})
	req.Imp = append(req.Imp, openrtb2.Imp{ID: "1_4", Ext: []byte(impExt)})
	req.Imp = append(req.Imp, openrtb2.Imp{ID: "1_5", Ext: []byte(impExt)})
	req.Imp = append(req.Imp, openrtb2.Imp{ID: "1_6", Ext: []byte(impExt)})
	req.Imp = append(req.Imp, openrtb2.Imp{ID: "1_7", Ext: []byte(impExt)})
	req.Imp = append(req.Imp, openrtb2.Imp{ID: "1_8", Ext: []byte(impExt)})
	req.Imp = append(req.Imp, openrtb2.Imp{ID: "1_9", Ext: []byte(impExt)})
	req.Imp = append(req.Imp, openrtb2.Imp{ID: "1_10", Ext: []byte(impExt)})
	req.Imp = append(req.Imp, openrtb2.Imp{ID: "1_11", Ext: []byte(impExt)})
	req.Imp = append(req.Imp, openrtb2.Imp{ID: "1_12", Ext: []byte(impExt)})
	req.Imp = append(req.Imp, openrtb2.Imp{ID: "1_13", Ext: []byte(impExt)})
	req.Imp = append(req.Imp, openrtb2.Imp{ID: "1_14", Ext: []byte(impExt)})

	res, err := a.MakeRequests(&req, &reqInfo)

	assert.Empty(t, err, "Errors array should be empty")
	assert.Len(t, res, 2, "Two requests should be returned")

	var error error
	var reqData1 *openrtb2.BidRequest
	error = json.Unmarshal(res[0].Body, &reqData1)
	assert.NoError(t, error, "Response body unmarshalling error should be nil")

	var reqDataExt1 *appnexusReqExt
	error = json.Unmarshal(reqData1.Ext, &reqDataExt1)
	assert.NoError(t, error, "Response ext unmarshalling error should be nil")

	assert.Equal(t, "", reqDataExt1.Appnexus.AdPodId, "AdPod id should not be present in first request")

	var reqData2 *openrtb2.BidRequest
	error = json.Unmarshal(res[1].Body, &reqData2)
	assert.NoError(t, error, "Response body unmarshalling error should be nil")

	var reqDataExt2 *appnexusReqExt
	error = json.Unmarshal(reqData2.Ext, &reqDataExt2)
	assert.NoError(t, error, "Response ext unmarshalling error should be nil")

	assert.Equal(t, "", reqDataExt2.Appnexus.AdPodId, "AdPod id should not be present in second request")
}

func TestVideoTwoPods(t *testing.T) {
	var a AppNexusAdapter
	a.URI = "http://test.com/openrtb2"
	a.hbSource = 5

	var reqInfo adapters.ExtraRequestInfo
	reqInfo.PbsEntryPoint = "video"

	var req openrtb2.BidRequest
	req.ID = "test_id"

	reqExt := `{"prebid":{}}`
	impExt := `{"bidder":{"placementId":123, "generate_ad_pod_id": true}}`
	req.Ext = []byte(reqExt)

	req.Imp = append(req.Imp, openrtb2.Imp{ID: "1_0", Ext: []byte(impExt)})
	req.Imp = append(req.Imp, openrtb2.Imp{ID: "1_1", Ext: []byte(impExt)})
	req.Imp = append(req.Imp, openrtb2.Imp{ID: "1_2", Ext: []byte(impExt)})

	req.Imp = append(req.Imp, openrtb2.Imp{ID: "2_0", Ext: []byte(impExt)})
	req.Imp = append(req.Imp, openrtb2.Imp{ID: "2_1", Ext: []byte(impExt)})
	req.Imp = append(req.Imp, openrtb2.Imp{ID: "2_2", Ext: []byte(impExt)})

	res, err := a.MakeRequests(&req, &reqInfo)

	assert.Empty(t, err, "Errors array should be empty")
	assert.Len(t, res, 2, "Two request should be returned")

	var error error
	var reqData1 *openrtb2.BidRequest
	error = json.Unmarshal(res[0].Body, &reqData1)
	assert.NoError(t, error, "Response body unmarshalling error should be nil")

	var reqDataExt1 *appnexusReqExt
	error = json.Unmarshal(reqData1.Ext, &reqDataExt1)
	assert.NoError(t, error, "Response ext unmarshalling error should be nil")

	adPodId1 := reqDataExt1.Appnexus.AdPodId

	var reqData2 *openrtb2.BidRequest
	error = json.Unmarshal(res[1].Body, &reqData2)
	assert.NoError(t, error, "Response body unmarshalling error should be nil")

	var reqDataExt2 *appnexusReqExt
	error = json.Unmarshal(reqData2.Ext, &reqDataExt2)
	assert.NoError(t, error, "Response ext unmarshalling error should be nil")

	adPodId2 := reqDataExt2.Appnexus.AdPodId

	assert.NotEqual(t, adPodId1, adPodId2, "AdPod id should be different for different pods")
}

func TestVideoTwoPodsManyImps(t *testing.T) {
	var a AppNexusAdapter
	a.URI = "http://test.com/openrtb2"
	a.hbSource = 5

	var reqInfo adapters.ExtraRequestInfo
	reqInfo.PbsEntryPoint = "video"

	var req openrtb2.BidRequest
	req.ID = "test_id"

	reqExt := `{"prebid":{}}`
	impExt := `{"bidder":{"placementId":123, "generate_ad_pod_id":true}}`
	req.Ext = []byte(reqExt)

	req.Imp = append(req.Imp, openrtb2.Imp{ID: "1_0", Ext: []byte(impExt)})
	req.Imp = append(req.Imp, openrtb2.Imp{ID: "1_1", Ext: []byte(impExt)})
	req.Imp = append(req.Imp, openrtb2.Imp{ID: "1_2", Ext: []byte(impExt)})

	req.Imp = append(req.Imp, openrtb2.Imp{ID: "2_0", Ext: []byte(impExt)})
	req.Imp = append(req.Imp, openrtb2.Imp{ID: "2_1", Ext: []byte(impExt)})
	req.Imp = append(req.Imp, openrtb2.Imp{ID: "2_2", Ext: []byte(impExt)})
	req.Imp = append(req.Imp, openrtb2.Imp{ID: "2_3", Ext: []byte(impExt)})
	req.Imp = append(req.Imp, openrtb2.Imp{ID: "2_4", Ext: []byte(impExt)})
	req.Imp = append(req.Imp, openrtb2.Imp{ID: "2_5", Ext: []byte(impExt)})
	req.Imp = append(req.Imp, openrtb2.Imp{ID: "2_6", Ext: []byte(impExt)})
	req.Imp = append(req.Imp, openrtb2.Imp{ID: "2_7", Ext: []byte(impExt)})
	req.Imp = append(req.Imp, openrtb2.Imp{ID: "2_8", Ext: []byte(impExt)})
	req.Imp = append(req.Imp, openrtb2.Imp{ID: "2_9", Ext: []byte(impExt)})
	req.Imp = append(req.Imp, openrtb2.Imp{ID: "2_10", Ext: []byte(impExt)})
	req.Imp = append(req.Imp, openrtb2.Imp{ID: "2_11", Ext: []byte(impExt)})
	req.Imp = append(req.Imp, openrtb2.Imp{ID: "2_12", Ext: []byte(impExt)})
	req.Imp = append(req.Imp, openrtb2.Imp{ID: "2_13", Ext: []byte(impExt)})
	req.Imp = append(req.Imp, openrtb2.Imp{ID: "2_14", Ext: []byte(impExt)})

	res, err := a.MakeRequests(&req, &reqInfo)

	assert.Empty(t, err, "Errors array should be empty")
	assert.Len(t, res, 3, "Three requests should be returned")

	var error error
	var reqData1 *openrtb2.BidRequest
	error = json.Unmarshal(res[0].Body, &reqData1)
	assert.NoError(t, error, "Response body unmarshalling error should be nil")

	var reqDataExt1 *appnexusReqExt
	error = json.Unmarshal(reqData1.Ext, &reqDataExt1)
	assert.NoError(t, error, "Response ext unmarshalling error should be nil")

	var reqData2 *openrtb2.BidRequest
	error = json.Unmarshal(res[1].Body, &reqData2)
	assert.NoError(t, error, "Response body unmarshalling error should be nil")

	var reqDataExt2 *appnexusReqExt
	error = json.Unmarshal(reqData2.Ext, &reqDataExt2)
	assert.NoError(t, error, "Response ext unmarshalling error should be nil")

	var reqData3 *openrtb2.BidRequest
	error = json.Unmarshal(res[2].Body, &reqData3)
	assert.NoError(t, error, "Response body unmarshalling error should be nil")

	var reqDataExt3 *appnexusReqExt
	error = json.Unmarshal(reqData3.Ext, &reqDataExt3)
	assert.NoError(t, error, "Response ext unmarshalling error should be nil")

	adPodId1 := reqDataExt1.Appnexus.AdPodId
	adPodId2 := reqDataExt2.Appnexus.AdPodId
	adPodId3 := reqDataExt3.Appnexus.AdPodId

	podIds := make(map[string]int)
	podIds[adPodId1] = podIds[adPodId1] + 1
	podIds[adPodId2] = podIds[adPodId2] + 1
	podIds[adPodId3] = podIds[adPodId3] + 1

	assert.Len(t, podIds, 2, "Incorrect number of unique pod ids")
}

// ----------------------------------------------------------------------------
// Code below this line tests the legacy, non-openrtb code flow. It can be deleted after we
// clean up the existing code and make everything openrtb2.

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

	var breq openrtb2.BidRequest
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

	resp := openrtb2.BidResponse{
		ID:    breq.ID,
		BidID: "a-random-id",
		Cur:   "USD",
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "Buyer Member ID",
				Bid:  make([]openrtb2.Bid, 0, 2),
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
			if andata.tags[i].position == "above" && *imp.Banner.Pos != openrtb2.AdPosition(1) {
				http.Error(w, fmt.Sprintf("Mismatch in position - expected 1 for atf"), http.StatusInternalServerError)
				return
			}
			if andata.tags[i].position == "below" && *imp.Banner.Pos != openrtb2.AdPosition(3) {
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

		resBid := openrtb2.Bid{
			ID:    "random-id",
			ImpID: imp.ID,
			Price: andata.tags[i].bid,
			AdM:   andata.tags[i].content,
			Ext:   json.RawMessage(fmt.Sprintf(`{"appnexus":{"bid_ad_type":%d}}`, bidTypeToInt(andata.tags[i].mediaType))),
		}

		if imp.Video != nil {
			resBid.Attr = []openrtb2.CreativeAttribute{openrtb2.CreativeAttribute(6)}
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

func bidTypeToInt(bidType string) int {
	switch bidType {
	case "banner":
		return 0
	case "video":
		return 1
	case "audio":
		return 2
	case "native":
		return 3
	default:
		return -1
	}
}
func TestAppNexusLegacyBasicResponse(t *testing.T) {
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
	an := NewAppNexusLegacyAdapter(&conf, server.URL, "")

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
			Sizes: []openrtb2.Format{
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
				{
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

	pc := usersync.ParsePBSCookieFromRequest(req, &config.HostCookie{})
	pc.TrySync("adnxs", andata.buyerUID)
	fakewriter := httptest.NewRecorder()

	pc.SetCookieOnResponse(fakewriter, false, &config.HostCookie{Domain: ""}, 90*24*time.Hour)
	req.Header.Add("Cookie", fakewriter.Header().Get("Set-Cookie"))

	cacheClient, _ := dummycache.New()
	hcc := config.HostCookie{}

	pbReq, err := pbs.ParsePBSRequest(req, &config.AuctionTimeouts{
		Default: 2000,
		Max:     2000,
	}, cacheClient, &hcc)
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
					t.Errorf("Incorrect Creative MediaType '%s'. Expected '%s'", bid.CreativeMediaType, tag.mediaType)
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
