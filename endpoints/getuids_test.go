package endpoints

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/stretchr/testify/assert"
)

func TestGetUIDs(t *testing.T) {
	req := makeRequest("/getuids", map[string]string{"adnxs": "123", "audienceNetwork": "456"})
	endpoint := NewGetUIDsEndpoint(config.HostCookie{})
	res := httptest.NewRecorder()
	endpoint(res, req, nil)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.JSONEq(t, `{"buyeruids": {"adnxs": "123", "audienceNetwork": "456"}}`,
		res.Body.String(), "GetUIDs endpoint should return the correct user ID for each bidder")
}

func TestGetUIDsWithNoSyncs(t *testing.T) {
	req := makeRequest("/getuids", map[string]string{})
	endpoint := NewGetUIDsEndpoint(config.HostCookie{})
	res := httptest.NewRecorder()
	endpoint(res, req, nil)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.JSONEq(t, `{}`, res.Body.String(), "GetUIDs endpoint shouldn't return anything if there don't exist any user syncs")
}

func TestGetUIDWIthNoCookie(t *testing.T) {
	req := httptest.NewRequest("GET", "/getuids", nil)
	endpoint := NewGetUIDsEndpoint(config.HostCookie{})
	res := httptest.NewRecorder()
	endpoint(res, req, nil)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.JSONEq(t, `{}`, res.Body.String(), "GetUIDs endpoint shouldn't return anything if there doesn't exist a PBS cookie")
}
