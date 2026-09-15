package eskimi

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

const testEndpoint = "https://ittr.eskimi.com/prebidjs"

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderEskimi, config.Adapter{
		Endpoint: testEndpoint},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 814, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "eskimitest", bidder)
}

func newAdapter() *adapter {
	return &adapter{endpoint: testEndpoint}
}

func bannerImp(ext string) openrtb2.Imp {
	return openrtb2.Imp{
		ID:     "imp-1",
		Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}},
		Ext:    json.RawMessage(ext),
	}
}

func TestMalformedSiteExt(t *testing.T) {
	req := &openrtb2.BidRequest{
		ID:   "req-1",
		Imp:  []openrtb2.Imp{bannerImp(`{"bidder":{"placementId":625}}`)},
		Site: &openrtb2.Site{ID: "271", Ext: json.RawMessage(`{invalid`)},
	}
	_, errs := newAdapter().MakeRequests(req, &adapters.ExtraRequestInfo{})
	assert.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "site.ext")
}

func TestMalformedAppExt(t *testing.T) {
	req := &openrtb2.BidRequest{
		ID:  "req-1",
		Imp: []openrtb2.Imp{bannerImp(`{"bidder":{"placementId":625}}`)},
		App: &openrtb2.App{ID: "app-1", Ext: json.RawMessage(`{invalid`)},
	}
	_, errs := newAdapter().MakeRequests(req, &adapters.ExtraRequestInfo{})
	assert.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "app.ext")
}

func TestImpSecureDefaultedToOne(t *testing.T) {
	req := &openrtb2.BidRequest{
		ID:   "req-1",
		Imp:  []openrtb2.Imp{bannerImp(`{"bidder":{"placementId":625}}`)},
		Site: &openrtb2.Site{ID: "271"},
	}
	reqData, errs := newAdapter().MakeRequests(req, &adapters.ExtraRequestInfo{})
	assert.Empty(t, errs)
	var out openrtb2.BidRequest
	assert.NoError(t, json.Unmarshal(reqData[0].Body, &out))
	assert.NotNil(t, out.Imp[0].Secure)
	assert.EqualValues(t, 1, *out.Imp[0].Secure)
}

func TestImpSecureNotOverridden(t *testing.T) {
	zero := int8(0)
	imp := bannerImp(`{"bidder":{"placementId":625}}`)
	imp.Secure = &zero
	req := &openrtb2.BidRequest{
		ID:   "req-1",
		Imp:  []openrtb2.Imp{imp},
		Site: &openrtb2.Site{ID: "271"},
	}
	reqData, errs := newAdapter().MakeRequests(req, &adapters.ExtraRequestInfo{})
	assert.Empty(t, errs)
	var out openrtb2.BidRequest
	assert.NoError(t, json.Unmarshal(reqData[0].Body, &out))
	assert.EqualValues(t, 0, *out.Imp[0].Secure)
}

// TestCallerRequestNotMutated asserts every copy-on-write surface in MakeRequests:
// request.{BCat,BAdv,BApp}, request.{Site,App}.Ext, request.Imp[i].Secure,
// request.Imp[i].{Banner,Video}.BAttr, and request.Imp[i].{BidFloor,BidFloorCur}.
func TestCallerRequestNotMutated(t *testing.T) {
	imp := openrtb2.Imp{
		ID:     "imp-1",
		Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}},
		Video:  &openrtb2.Video{MIMEs: []string{"video/mp4"}, W: openrtb2.Int64Ptr(640), H: openrtb2.Int64Ptr(480)},
		Ext: json.RawMessage(`{"bidder":{
			"placementId":625,
			"bcat":["IAB1"],"badv":["bad.com"],"bapp":["com.bad"],
			"bidFloor":1.5,"bidFloorCur":"EUR",
			"battr":[1,2]
		}}`),
	}
	req := &openrtb2.BidRequest{
		ID:   "req-1",
		Imp:  []openrtb2.Imp{imp},
		Site: &openrtb2.Site{ID: "271", Ext: json.RawMessage(`{"amp":0}`)},
	}

	_, errs := newAdapter().MakeRequests(req, &adapters.ExtraRequestInfo{})
	assert.Empty(t, errs)

	assert.Empty(t, req.BCat, "caller's request.BCat must not be mutated")
	assert.Empty(t, req.BAdv, "caller's request.BAdv must not be mutated")
	assert.Empty(t, req.BApp, "caller's request.BApp must not be mutated")
	assert.JSONEq(t, `{"amp":0}`, string(req.Site.Ext), "caller's request.Site.Ext must not be mutated")
	assert.Nil(t, req.Imp[0].Secure, "caller's imp[0].Secure must not be mutated")
	assert.EqualValues(t, 0, req.Imp[0].BidFloor, "caller's imp[0].BidFloor must not be mutated")
	assert.Empty(t, req.Imp[0].BidFloorCur, "caller's imp[0].BidFloorCur must not be mutated")
	assert.Empty(t, req.Imp[0].Banner.BAttr, "caller's imp[0].Banner.BAttr must not be mutated")
	assert.Empty(t, req.Imp[0].Video.BAttr, "caller's imp[0].Video.BAttr must not be mutated")
}

func TestCallerAppRequestNotMutated(t *testing.T) {
	req := &openrtb2.BidRequest{
		ID:  "req-1",
		Imp: []openrtb2.Imp{bannerImp(`{"bidder":{"placementId":625}}`)},
		App: &openrtb2.App{ID: "app-1", Ext: json.RawMessage(`{"installed":1}`)},
	}
	_, errs := newAdapter().MakeRequests(req, &adapters.ExtraRequestInfo{})
	assert.Empty(t, errs)
	assert.JSONEq(t, `{"installed":1}`, string(req.App.Ext), "caller's request.App.Ext must not be mutated")
}

func TestLaterImpWithInvalidExtSurfacesError(t *testing.T) {
	req := &openrtb2.BidRequest{
		ID: "req-1",
		Imp: []openrtb2.Imp{
			bannerImp(`{"bidder":{"placementId":625,"bidFloor":1.5,"bidFloorCur":"EUR"}}`),
			{ID: "imp-2", Banner: &openrtb2.Banner{}, Ext: json.RawMessage(`"not-an-object"`)},
		},
		Site: &openrtb2.Site{ID: "271"},
	}
	reqData, errs := newAdapter().MakeRequests(req, &adapters.ExtraRequestInfo{})
	// First-imp request still goes out, but the bad later imp surfaces an error.
	assert.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "imp.ext")
	assert.Len(t, reqData, 1)

	var out openrtb2.BidRequest
	assert.NoError(t, json.Unmarshal(reqData[0].Body, &out))
	assert.EqualValues(t, 1.5, out.Imp[0].BidFloor)
}
