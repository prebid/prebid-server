package appnexus

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/stretchr/testify/assert"
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

func TestVideoSinglePodManyImps(t *testing.T) {
	var a adapter
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
