package endpoints

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/cache"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/prebid_cache_client"
	"io"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"
	"time"
)

const (
	AccountParameter   = "a"
	ImpressionCloseTag = "</Impression>"
	ImpressionOpenTag  = "<Impression>"
)

type vtrackEndpoint struct {
	Cfg         *config.Configuration
	DataCache   cache.Cache
	BidderInfos adapters.BidderInfos
	PbsCache    prebid_cache_client.Client
}

type BidCacheRequest struct {
	Puts []prebid_cache_client.Cacheable `json:"puts"`
}

type BidCacheResponse struct {
	Responses []CacheObject `json:"responses"`
}

type CacheObject struct {
	Uuid string `json:"uuid"`
}

func NewVTrackEndpoint(cfg *config.Configuration, dataCache cache.Cache, pbsCache prebid_cache_client.Client, bidderInfos adapters.BidderInfos) httprouter.Handle {
	vte := &vtrackEndpoint{
		Cfg:         cfg,
		DataCache:   dataCache,
		BidderInfos: bidderInfos,
		PbsCache:    pbsCache,
	}

	return vte.Handle
}

/**
 * /vtrack Handler
 */
func (v *vtrackEndpoint) Handle(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	// get account id from request parameter
	accountId := getAccountId(r)

	// account id is required
	if accountId == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Account '%s' is required query parameter and can't be empty", AccountParameter)))
		return
	}

	// parse puts request from request body
	req, err := v.parseVTrackRequest(r)

	// check if there was any error while parsing puts request
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Invalid request: %s\n", err.Error())))
		return
	}

	// get account config
	account, err := v.DataCache.Accounts().Get(accountId)
	if err != nil {
		if err == sql.ErrNoRows {
			account = &cache.Account{
				ID:            accountId,
				EventsEnabled: false,
			}
		} else {
			if glog.V(2) {
				glog.Infof("Invalid account id: %v", err)
			}

			status := http.StatusInternalServerError
			message := fmt.Sprintf("Invalid request: %s\n", err.Error())

			w.WriteHeader(status)
			w.Write([]byte(message))
			return
		}
	}

	// insert impression tracking if account allows events and bidder allows VAST modification
	if account.EventsEnabled && v.PbsCache != nil {
		biddersAllowingVastUpdate := biddersAllowingVastUpdate(req, &v.BidderInfos, v.Cfg.VTrack.AllowUnknownBidder)

		// cache data
		r, errs := v.cachePutObjects(req, biddersAllowingVastUpdate, accountId, v.Cfg.VTrack.TimeoutMs)

		// handle pbs caching errors
		if len(errs) != 0 {
			glog.Errorf("Error(s) updating vast: %v", errs)
			w.WriteHeader(http.StatusInternalServerError)
			for _, err := range errs {
				w.Write([]byte(fmt.Sprintf("Error(s) updating vast: %s\n", err.Error())))
			}

			return
		}

		// build response
		response := &BidCacheResponse{
			Responses: []CacheObject{},
		}

		for _, uuid := range r {
			response.Responses = append(response.Responses, CacheObject{
				Uuid: uuid,
			})
		}

		d, err := json.Marshal(response)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("Error serializing pbs cache response: %s\n", err.Error())))

			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(d)
	}
}

/**
 * Create vast url tracking
 */
func GetVastUrlTracking(externalUrl string, bidid string, bidder string, accountId string, timestamp int64) string {

	eventReq := &EventRequest{
		Type:      IMP,
		Bidid:     bidid,
		AccountId: accountId,
		Bidder:    bidder,
		Timestamp: timestamp,
		Format:    BLANK,
	}

	return EventRequestToUrl(externalUrl, eventReq)
}

/**
 * Parses a BidCacheRequest from an HTTP Request
 */
func (v *vtrackEndpoint) parseVTrackRequest(httpRequest *http.Request) (req *BidCacheRequest, err error) {
	req = &BidCacheRequest{}
	err = nil

	// Pull the request body into a buffer, so we have it for later usage.
	lr := &io.LimitedReader{
		R: httpRequest.Body,
		N: v.Cfg.MaxRequestSize,
	}

	requestJson, err := ioutil.ReadAll(lr)
	if err != nil {
		return req, err
	}

	// If the request size was too large, read through the rest of the request body so that the connection can be reused.
	if lr.N <= 0 {
		if written, err := io.Copy(ioutil.Discard, httpRequest.Body); written > 0 || err != nil {
			err = fmt.Errorf("request size exceeded max size of %d bytes", v.Cfg.MaxRequestSize)
			return req, err
		}
	}

	if len(requestJson) == 0 {
		err = fmt.Errorf("request body is empty")
		return req, err
	}

	if err := json.Unmarshal(requestJson, req); err != nil {
		return req, err
	}

	// validate request
	if len(req.Puts) == 0 {
		return req, err
	}

	for _, bcr := range req.Puts {
		if bcr.BidID == "" {
			err = error(&errortypes.BadInput{Message: fmt.Sprint("'bidid' is required field and can't be empty")})
			return req, err
		}

		if bcr.Bidder == "" {
			err = error(&errortypes.BadInput{Message: fmt.Sprint("'bidder' is required field and can't be empty")})
			return req, err
		}
	}

	return req, nil
}

/**
 * Cache BidCacheRequest data
 */
func (v *vtrackEndpoint) cachePutObjects(req *BidCacheRequest, biddersAllowingVastUpdate []string, accountId string, timeout int64) ([]string, []error) {
	var nputs []prebid_cache_client.Cacheable
	sort.Strings(biddersAllowingVastUpdate)

	for _, c := range req.Puts {

		nc := &prebid_cache_client.Cacheable{
			Type:       c.Type,
			Data:       c.Data,
			TTLSeconds: c.TTLSeconds,
			Key:        c.Key,
		}

		if contains(biddersAllowingVastUpdate, c.Bidder) && nc.Data != nil {
			nc.Data = modifyVastXml(v.Cfg.ExternalURL, nc.Data, c.BidID, c.Bidder, accountId, c.Timestamp)
		}

		nputs = append(nputs, *nc)
	}

	t := time.Now().Add(time.Duration(timeout) * time.Millisecond)
	ctx, cancel := context.WithDeadline(context.Background(), t)
	defer cancel()

	return v.PbsCache.PutJson(ctx, nputs)
}

/**
 * Returns list of bidders that allow VAST XML modification.
 */
func biddersAllowingVastUpdate(req *BidCacheRequest, bidderInfos *adapters.BidderInfos, allowUnknownBidder bool) []string {
	var bl []string

	for _, bcr := range req.Puts {
		if isAllowVastForBidder(&bcr, bidderInfos, allowUnknownBidder) {
			bl = append(bl, bcr.Bidder)
		}
	}

	return dedupe(bl)
}

/**
 * Checks if Bidder is active and allowed to modify vast xml data
 */
func isAllowVastForBidder(r *prebid_cache_client.Cacheable, bidderInfos *adapters.BidderInfos, allowUnknownBidder bool) bool {
	//if bidder is active and isModifyingVastXmlAllowed is true
	// check if bidder is configured
	if b, ok := (*bidderInfos)[r.Bidder]; bidderInfos != nil && ok {
		// check if bidder is enabled
		return b.Status == adapters.StatusActive && b.ModifyingVastXmlAllowed
	}

	return allowUnknownBidder
}

/**
 * Extracts account id from a HTTP Request
 */
func getAccountId(httpRequest *http.Request) string {
	return httpRequest.FormValue(AccountParameter)
}

/**
 * Modify BidCacheRequest element Vast XML data
 */
func modifyVastXml(externalUrl string, data json.RawMessage, bidid string, bidder string, accountId string, timestamp int64) json.RawMessage {
	c := string(data)
	ci := strings.Index(c, ImpressionCloseTag)

	// no impression tag - pass it as it is
	if ci == -1 {
		return json.RawMessage(c)
	}

	vastUrlTracking := GetVastUrlTracking(externalUrl, bidid, bidder, accountId, timestamp)
	impressionUrl := "<![CDATA[" + vastUrlTracking + "]]>"

	oi := strings.Index(c, ImpressionOpenTag)

	if ci-oi == len(ImpressionOpenTag) {
		return json.RawMessage(strings.Replace(c, ImpressionOpenTag, ImpressionOpenTag+impressionUrl, 1))
	}

	return json.RawMessage(strings.Replace(c, ImpressionCloseTag, ImpressionCloseTag+ImpressionOpenTag+impressionUrl+ImpressionCloseTag, 1))
}

/**
 * Util
 */

func dedupe(s []string) []string {
	seen := make(map[string]struct{}, len(s))
	j := 0
	for _, v := range s {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		s[j] = v
		j++
	}
	return s[:j]
}

func contains(s []string, e string) bool {
	i := sort.SearchStrings(s, e)
	return i < len(s) && s[i] == e
}
