package openrtb_ext_test

import (
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/stretchr/testify/assert"
)

func TestInvalidSiteExt(t *testing.T) {
	var s openrtb_ext.ExtSite
	assert.EqualError(t, jsonutil.UnmarshalValid([]byte(`{"amp":-1}`), &s), "request.site.ext.amp must be either 1, 0, or undefined")
	assert.EqualError(t, jsonutil.UnmarshalValid([]byte(`{"amp":2}`), &s), "request.site.ext.amp must be either 1, 0, or undefined")
	assert.EqualError(t, jsonutil.UnmarshalValid([]byte(`{"amp":true}`), &s), "request.site.ext.amp must be either 1, 0, or undefined")
	assert.EqualError(t, jsonutil.UnmarshalValid([]byte(`{"amp":null}`), &s), "request.site.ext.amp must be either 1, 0, or undefined")
	assert.EqualError(t, jsonutil.UnmarshalValid([]byte(`{"amp":"1"}`), &s), "request.site.ext.amp must be either 1, 0, or undefined")
}

func TestValidSiteExt(t *testing.T) {
	var s openrtb_ext.ExtSite
	assert.NoError(t, jsonutil.UnmarshalValid([]byte(`{"amp":0}`), &s))
	assert.EqualValues(t, 0, s.AMP)
	assert.NoError(t, jsonutil.UnmarshalValid([]byte(`{"amp":1}`), &s))
	assert.EqualValues(t, 1, s.AMP)
	assert.NoError(t, jsonutil.UnmarshalValid([]byte(`{"amp":      1   }`), &s))
	assert.EqualValues(t, 1, s.AMP)
}
