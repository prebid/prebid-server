package appnexus

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
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

func TestVideoSinglePod(t *testing.T) {
	var a adapter
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

func TestVideoTwoPods(t *testing.T) {
	var a adapter
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
	var a adapter
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
