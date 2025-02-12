package events

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	accountService "github.com/prebid/prebid-server/v3/account"
	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/prebid_cache_client"
	"github.com/prebid/prebid-server/v3/stored_requests"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

const (
	AccountParameter     = "a"
	IntegrationParameter = "int"
	ImpressionCloseTag   = "</Impression>"
	ImpressionOpenTag    = "<Impression>"
)

type vtrackEndpoint struct {
	Cfg                 *config.Configuration
	Accounts            stored_requests.AccountFetcher
	BidderInfos         config.BidderInfos
	Cache               prebid_cache_client.Client
	MetricsEngine       metrics.MetricsEngine
	normalizeBidderName openrtb_ext.BidderNameNormalizer
}

type BidCacheRequest struct {
	Puts []prebid_cache_client.Cacheable `json:"puts"`
}

type BidCacheResponse struct {
	Responses []CacheObject `json:"responses"`
}

type CacheObject struct {
	UUID string `json:"uuid"`
}

func NewVTrackEndpoint(cfg *config.Configuration, accounts stored_requests.AccountFetcher, cache prebid_cache_client.Client, bidderInfos config.BidderInfos, me metrics.MetricsEngine) httprouter.Handle {
	vte := &vtrackEndpoint{
		Cfg:                 cfg,
		Accounts:            accounts,
		BidderInfos:         bidderInfos,
		Cache:               cache,
		MetricsEngine:       me,
		normalizeBidderName: openrtb_ext.NormalizeBidderName,
	}

	return vte.Handle
}

// /vtrack Handler
func (v *vtrackEndpoint) Handle(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	// get account id from request parameter
	accountId := getAccountId(r)

	// account id is required
	if accountId == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Account '%s' is required query parameter and can't be empty", AccountParameter)
		return
	}

	// get integration value from request parameter
	integrationType, err := getIntegrationType(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Invalid integration type: %s\n", err.Error())
		return
	}

	// parse puts request from request body
	req, err := ParseVTrackRequest(r, v.Cfg.MaxRequestSize+1)

	// check if there was any error while parsing puts request
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Invalid request: %s\n", err.Error())
		return
	}

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Duration(v.Cfg.VTrack.TimeoutMS)*time.Millisecond))
	defer cancel()

	// get account details
	account, errs := accountService.GetAccount(ctx, v.Cfg, v.Accounts, accountId, v.MetricsEngine)
	if len(errs) > 0 {
		status, messages := HandleAccountServiceErrors(errs)
		w.WriteHeader(status)

		for _, message := range messages {
			fmt.Fprintf(w, "Invalid request: %s\n", message)
		}
		return
	}

	// insert impression tracking if account allows events and bidder allows VAST modification
	if v.Cache != nil {
		cachingResponse, errs := v.handleVTrackRequest(ctx, req, account, integrationType)

		if len(errs) > 0 {
			w.WriteHeader(http.StatusInternalServerError)
			for _, err := range errs {
				fmt.Fprintf(w, "Error(s) updating vast: %s\n", err.Error())

				return
			}
		}

		d, err := jsonutil.Marshal(*cachingResponse)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Error serializing pbs cache response: %s\n", err.Error())

			return
		}

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(d)

		return
	}

	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("PBS Cache client is not configured"))
}

// GetVastUrlTracking creates a vast url tracking
func GetVastUrlTracking(externalUrl string, bidid string, bidder string, accountId string, timestamp int64, integration string) string {

	eventReq := &analytics.EventRequest{
		Type:        analytics.Imp,
		BidID:       bidid,
		AccountID:   accountId,
		Bidder:      bidder,
		Timestamp:   timestamp,
		Format:      analytics.Blank,
		Integration: integration,
	}

	return EventRequestToUrl(externalUrl, eventReq)
}

// ParseVTrackRequest parses a BidCacheRequest from an HTTP Request
func ParseVTrackRequest(httpRequest *http.Request, maxRequestSize int64) (req *BidCacheRequest, err error) {
	req = &BidCacheRequest{}
	err = nil

	// Pull the request body into a buffer, so we have it for later usage.
	lr := &io.LimitedReader{
		R: httpRequest.Body,
		N: maxRequestSize,
	}

	defer httpRequest.Body.Close()
	requestJson, err := io.ReadAll(lr)
	if err != nil {
		return req, err
	}

	// Check if the request size was too large
	if lr.N <= 0 {
		err = &errortypes.BadInput{Message: fmt.Sprintf("request size exceeded max size of %d bytes", maxRequestSize-1)}
		return req, err
	}

	if len(requestJson) == 0 {
		err = &errortypes.BadInput{Message: "request body is empty"}
		return req, err
	}

	if err := jsonutil.UnmarshalValid(requestJson, req); err != nil {
		return req, err
	}

	for _, bcr := range req.Puts {
		if bcr.BidID == "" {
			err = error(&errortypes.BadInput{Message: "'bidid' is required field and can't be empty"})
			return req, err
		}

		if bcr.Bidder == "" {
			err = error(&errortypes.BadInput{Message: "'bidder' is required field and can't be empty"})
			return req, err
		}
	}

	return req, nil
}

// handleVTrackRequest handles a VTrack request
func (v *vtrackEndpoint) handleVTrackRequest(ctx context.Context, req *BidCacheRequest, account *config.Account, integration string) (*BidCacheResponse, []error) {
	biddersAllowingVastUpdate := getBiddersAllowingVastUpdate(req, &v.BidderInfos, v.Cfg.VTrack.AllowUnknownBidder, v.normalizeBidderName)
	// cache data
	r, errs := v.cachePutObjects(ctx, req, biddersAllowingVastUpdate, account.ID, integration)

	// handle pbs caching errors
	if len(errs) != 0 {
		glog.Errorf("Error(s) updating vast: %v", errs)
		return nil, errs
	}

	// build response
	response := &BidCacheResponse{
		Responses: []CacheObject{},
	}

	for _, uuid := range r {
		response.Responses = append(response.Responses, CacheObject{
			UUID: uuid,
		})
	}

	return response, nil
}

// cachePutObjects caches BidCacheRequest data
func (v *vtrackEndpoint) cachePutObjects(ctx context.Context, req *BidCacheRequest, biddersAllowingVastUpdate map[string]struct{}, accountId string, integration string) ([]string, []error) {
	var cacheables []prebid_cache_client.Cacheable

	for _, c := range req.Puts {

		nc := &prebid_cache_client.Cacheable{
			Type:       c.Type,
			Data:       c.Data,
			TTLSeconds: c.TTLSeconds,
			Key:        c.Key,
		}

		if _, ok := biddersAllowingVastUpdate[c.Bidder]; ok && nc.Data != nil {
			nc.Data = ModifyVastXmlJSON(v.Cfg.ExternalURL, nc.Data, c.BidID, c.Bidder, accountId, c.Timestamp, integration)
		}

		cacheables = append(cacheables, *nc)
	}

	return v.Cache.PutJson(ctx, cacheables)
}

// getBiddersAllowingVastUpdate returns a list of bidders that allow VAST XML modification
func getBiddersAllowingVastUpdate(req *BidCacheRequest, bidderInfos *config.BidderInfos, allowUnknownBidder bool, normalizeBidderName openrtb_ext.BidderNameNormalizer) map[string]struct{} {
	bl := map[string]struct{}{}

	for _, bcr := range req.Puts {
		if _, ok := bl[bcr.Bidder]; isAllowVastForBidder(bcr.Bidder, bidderInfos, allowUnknownBidder, normalizeBidderName) && !ok {
			bl[bcr.Bidder] = struct{}{}
		}
	}

	return bl
}

// isAllowVastForBidder checks if a bidder is active and allowed to modify vast xml data
func isAllowVastForBidder(bidder string, bidderInfos *config.BidderInfos, allowUnknownBidder bool, normalizeBidderName openrtb_ext.BidderNameNormalizer) bool {
	// if bidder is active and isModifyingVastXmlAllowed is true
	// check if bidder is configured
	if normalizedBidder, ok := normalizeBidderName(bidder); ok {
		if bidderInfos != nil {
			if b, ok := (*bidderInfos)[normalizedBidder.String()]; ok {
				return b.IsEnabled() && b.ModifyingVastXmlAllowed
			}
		}
	}

	return allowUnknownBidder
}

// getAccountId extracts an account id from an HTTP Request
func getAccountId(httpRequest *http.Request) string {
	return httpRequest.URL.Query().Get(AccountParameter)
}

func getIntegrationType(httpRequest *http.Request) (string, error) {
	integrationType := httpRequest.URL.Query().Get(IntegrationParameter)
	err := validateIntegrationType(integrationType)
	if err != nil {
		return "", err
	}
	return integrationType, nil
}

// ModifyVastXmlString rewrites and returns the string vastXML and a flag indicating if it was modified
func ModifyVastXmlString(externalUrl, vast, bidid, bidder, accountID string, timestamp int64, integrationType string) (string, bool) {
	ci := strings.Index(vast, ImpressionCloseTag)

	// no impression tag - pass it as it is
	if ci == -1 {
		return vast, false
	}

	vastUrlTracking := GetVastUrlTracking(externalUrl, bidid, bidder, accountID, timestamp, integrationType)
	impressionUrl := "<![CDATA[" + vastUrlTracking + "]]>"
	oi := strings.Index(vast, ImpressionOpenTag)

	if ci-oi == len(ImpressionOpenTag) {
		return strings.Replace(vast, ImpressionOpenTag, ImpressionOpenTag+impressionUrl, 1), true
	}

	return strings.Replace(vast, ImpressionCloseTag, ImpressionCloseTag+ImpressionOpenTag+impressionUrl+ImpressionCloseTag, 1), true
}

// ModifyVastXmlJSON modifies BidCacheRequest element Vast XML data
func ModifyVastXmlJSON(externalUrl string, data json.RawMessage, bidid, bidder, accountId string, timestamp int64, integrationType string) json.RawMessage {
	var vast string
	if err := jsonutil.Unmarshal(data, &vast); err != nil {
		// failed to decode json, fall back to string
		vast = string(data)
	}
	vast, ok := ModifyVastXmlString(externalUrl, vast, bidid, bidder, accountId, timestamp, integrationType)
	if !ok {
		return data
	}
	return json.RawMessage(vast)
}
