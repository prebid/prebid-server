package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/beevik/etree"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	accountService "github.com/prebid/prebid-server/account"
	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/prebid_cache_client"
	"github.com/prebid/prebid-server/stored_requests"
)

const (
	AccountParameter   = "a"
	ImpressionCloseTag = "</Impression>"
	ImpressionOpenTag  = "<Impression>"
)

type vtrackEndpoint struct {
	Cfg         *config.Configuration
	Accounts    stored_requests.AccountFetcher
	BidderInfos config.BidderInfos
	Cache       prebid_cache_client.Client
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

// standard VAST macros
// https://interactiveadvertisingbureau.github.io/vast/vast4macros/vast4-macros-latest.html#macro-spec-adcount
const (
	VASTAdTypeMacro    = "[ADTYPE]"
	VASTAppBundleMacro = "[APPBUNDLE]"
	VASTDomainMacro    = "[DOMAIN]"
	VASTPageURLMacro   = "[PAGEURL]"

	// PBS specific macros
	PBSEventIDMacro = "[EVENT_ID]" // macro for injecting PBS defined  video event tracker id
	//[PBS-ACCOUNT] represents publisher id / account id
	PBSAccountMacro = "[PBS-ACCOUNT]"
	// [PBS-BIDDER] represents bidder name
	PBSBidderMacro = "[PBS-BIDDER]"
	// [PBS-BIDID] represents bid id. If auction.generate-bid-id config is on, then resolve with response.seatbid.bid.ext.prebid.bidid. Else replace with response.seatbid.bid.id
	PBSBidIDMacro = "[PBS-BIDID]"
	// [ADERVERTISER_NAME] represents advertiser name
	PBSAdvertiserNameMacro = "[ADVERTISER_NAME]"
	// Pass imp.tagId using this macro
	PBSAdUnitIDMacro = "[AD_UNIT]"
	//PBSBidderCodeMacro represents an alias id or core bidder id.
	PBSBidderCodeMacro = "[BIDDER_CODE]"
)

var trackingEvents = []string{"start", "firstQuartile", "midpoint", "thirdQuartile", "complete"}

// PubMatic specific event IDs
// This will go in event-config once PreBid modular design is in place
var eventIDMap = map[string]string{
	"start":         "2",
	"firstQuartile": "4",
	"midpoint":      "3",
	"thirdQuartile": "5",
	"complete":      "6",
}

func NewVTrackEndpoint(cfg *config.Configuration, accounts stored_requests.AccountFetcher, cache prebid_cache_client.Client, bidderInfos config.BidderInfos) httprouter.Handle {
	vte := &vtrackEndpoint{
		Cfg:         cfg,
		Accounts:    accounts,
		BidderInfos: bidderInfos,
		Cache:       cache,
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
		w.Write([]byte(fmt.Sprintf("Account '%s' is required query parameter and can't be empty", AccountParameter)))
		return
	}

	// parse puts request from request body
	req, err := ParseVTrackRequest(r, v.Cfg.MaxRequestSize+1)

	// check if there was any error while parsing puts request
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Invalid request: %s\n", err.Error())))
		return
	}

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Duration(v.Cfg.VTrack.TimeoutMS)*time.Millisecond))
	defer cancel()

	// get account details
	account, errs := accountService.GetAccount(ctx, v.Cfg, v.Accounts, accountId)
	if len(errs) > 0 {
		status, messages := HandleAccountServiceErrors(errs)
		w.WriteHeader(status)

		for _, message := range messages {
			w.Write([]byte(fmt.Sprintf("Invalid request: %s\n", message)))
		}
		return
	}

	// insert impression tracking if account allows events and bidder allows VAST modification
	if v.Cache != nil {
		cachingResponse, errs := v.handleVTrackRequest(ctx, req, account)

		if len(errs) > 0 {
			w.WriteHeader(http.StatusInternalServerError)
			for _, err := range errs {
				w.Write([]byte(fmt.Sprintf("Error(s) updating vast: %s\n", err.Error())))

				return
			}
		}

		d, err := json.Marshal(*cachingResponse)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("Error serializing pbs cache response: %s\n", err.Error())))

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
func GetVastUrlTracking(externalUrl string, bidid string, bidder string, accountId string, timestamp int64) string {

	eventReq := &analytics.EventRequest{
		Type:      analytics.Imp,
		BidID:     bidid,
		AccountID: accountId,
		Bidder:    bidder,
		Timestamp: timestamp,
		Format:    analytics.Blank,
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
	requestJson, err := ioutil.ReadAll(lr)
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

	if err := json.Unmarshal(requestJson, req); err != nil {
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

// handleVTrackRequest handles a VTrack request
func (v *vtrackEndpoint) handleVTrackRequest(ctx context.Context, req *BidCacheRequest, account *config.Account) (*BidCacheResponse, []error) {
	biddersAllowingVastUpdate := getBiddersAllowingVastUpdate(req, &v.BidderInfos, v.Cfg.VTrack.AllowUnknownBidder)
	// cache data
	r, errs := v.cachePutObjects(ctx, req, biddersAllowingVastUpdate, account.ID)

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
func (v *vtrackEndpoint) cachePutObjects(ctx context.Context, req *BidCacheRequest, biddersAllowingVastUpdate map[string]struct{}, accountId string) ([]string, []error) {
	var cacheables []prebid_cache_client.Cacheable

	for _, c := range req.Puts {

		nc := &prebid_cache_client.Cacheable{
			Type:       c.Type,
			Data:       c.Data,
			TTLSeconds: c.TTLSeconds,
			Key:        c.Key,
		}

		if _, ok := biddersAllowingVastUpdate[c.Bidder]; ok && nc.Data != nil {
			nc.Data = ModifyVastXmlJSON(v.Cfg.ExternalURL, nc.Data, c.BidID, c.Bidder, accountId, c.Timestamp)
		}

		cacheables = append(cacheables, *nc)
	}

	return v.Cache.PutJson(ctx, cacheables)
}

// getBiddersAllowingVastUpdate returns a list of bidders that allow VAST XML modification
func getBiddersAllowingVastUpdate(req *BidCacheRequest, bidderInfos *config.BidderInfos, allowUnknownBidder bool) map[string]struct{} {
	bl := map[string]struct{}{}

	for _, bcr := range req.Puts {
		if _, ok := bl[bcr.Bidder]; isAllowVastForBidder(bcr.Bidder, bidderInfos, allowUnknownBidder) && !ok {
			bl[bcr.Bidder] = struct{}{}
		}
	}

	return bl
}

// isAllowVastForBidder checks if a bidder is active and allowed to modify vast xml data
func isAllowVastForBidder(bidder string, bidderInfos *config.BidderInfos, allowUnknownBidder bool) bool {
	//if bidder is active and isModifyingVastXmlAllowed is true
	// check if bidder is configured
	if b, ok := (*bidderInfos)[bidder]; bidderInfos != nil && ok {
		// check if bidder is enabled
		return b.Enabled && b.ModifyingVastXmlAllowed
	}

	return allowUnknownBidder
}

// getAccountId extracts an account id from an HTTP Request
func getAccountId(httpRequest *http.Request) string {
	return httpRequest.URL.Query().Get(AccountParameter)
}

// ModifyVastXmlString rewrites and returns the string vastXML and a flag indicating if it was modified
func ModifyVastXmlString(externalUrl, vast, bidid, bidder, accountID string, timestamp int64) (string, bool) {
	ci := strings.Index(vast, ImpressionCloseTag)

	// no impression tag - pass it as it is
	if ci == -1 {
		return vast, false
	}

	vastUrlTracking := GetVastUrlTracking(externalUrl, bidid, bidder, accountID, timestamp)
	impressionUrl := "<![CDATA[" + vastUrlTracking + "]]>"
	oi := strings.Index(vast, ImpressionOpenTag)

	if ci-oi == len(ImpressionOpenTag) {
		return strings.Replace(vast, ImpressionOpenTag, ImpressionOpenTag+impressionUrl, 1), true
	}

	return strings.Replace(vast, ImpressionCloseTag, ImpressionCloseTag+ImpressionOpenTag+impressionUrl+ImpressionCloseTag, 1), true
}

// ModifyVastXmlJSON modifies BidCacheRequest element Vast XML data
func ModifyVastXmlJSON(externalUrl string, data json.RawMessage, bidid, bidder, accountId string, timestamp int64) json.RawMessage {
	var vast string
	if err := json.Unmarshal(data, &vast); err != nil {
		// failed to decode json, fall back to string
		vast = string(data)
	}
	vast, ok := ModifyVastXmlString(externalUrl, vast, bidid, bidder, accountId, timestamp)
	if !ok {
		return data
	}
	return json.RawMessage(vast)
}

//InjectVideoEventTrackers injects the video tracking events
//Returns VAST xml contains as first argument. Second argument indicates whether the trackers are injected and last argument indicates if there is any error in injecting the trackers
func InjectVideoEventTrackers(trackerURL, vastXML string, bid *openrtb2.Bid, requestingBidder, bidderCoreName, accountID string, timestamp int64, bidRequest *openrtb2.BidRequest) ([]byte, bool, error) {
	// parse VAST
	doc := etree.NewDocument()
	err := doc.ReadFromString(vastXML)
	if nil != err {
		err = fmt.Errorf("Error parsing VAST XML. '%v'", err.Error())
		glog.Errorf(err.Error())
		return []byte(vastXML), false, err // false indicates events trackers are not injected
	}

	//Maintaining BidRequest Impression Map (Copied from exchange.go#applyCategoryMapping)
	//TODO: It should be optimized by forming once and reusing
	impMap := make(map[string]*openrtb2.Imp)
	for i := range bidRequest.Imp {
		impMap[bidRequest.Imp[i].ID] = &bidRequest.Imp[i]
	}

	eventURLMap := GetVideoEventTracking(trackerURL, bid, requestingBidder, bidderCoreName, accountID, timestamp, bidRequest, doc, impMap)
	trackersInjected := false
	// return if if no tracking URL
	if len(eventURLMap) == 0 {
		return []byte(vastXML), false, errors.New("Event URLs are not found")
	}

	creatives := FindCreatives(doc)

	if adm := strings.TrimSpace(bid.AdM); adm == "" || strings.HasPrefix(adm, "http") {
		// determine which creative type to be created based on linearity
		if imp, ok := impMap[bid.ImpID]; ok && nil != imp.Video {
			// create creative object
			creatives = doc.FindElements("VAST/Ad/Wrapper/Creatives")
			// var creative *etree.Element
			// if len(creatives) > 0 {
			// 	creative = creatives[0] // consider only first creative
			// } else {
			creative := doc.CreateElement("Creative")
			creatives[0].AddChild(creative)

			// }

			switch imp.Video.Linearity {
			case openrtb2.VideoLinearityLinearInStream:
				creative.AddChild(doc.CreateElement("Linear"))
			case openrtb2.VideoLinearityNonLinearOverlay:
				creative.AddChild(doc.CreateElement("NonLinearAds"))
			default: // create both type of creatives
				creative.AddChild(doc.CreateElement("Linear"))
				creative.AddChild(doc.CreateElement("NonLinearAds"))
			}
			creatives = creative.ChildElements() // point to actual cratives
		}
	}
	for _, creative := range creatives {
		trackingEvents := creative.SelectElement("TrackingEvents")
		if nil == trackingEvents {
			trackingEvents = creative.CreateElement("TrackingEvents")
			creative.AddChild(trackingEvents)
		}
		// Inject
		for event, url := range eventURLMap {
			trackingEle := trackingEvents.CreateElement("Tracking")
			trackingEle.CreateAttr("event", event)
			trackingEle.SetText(fmt.Sprintf("%s", url))
			trackersInjected = true
		}
	}

	out := []byte(vastXML)
	var wErr error
	if trackersInjected {
		out, wErr = doc.WriteToBytes()
		trackersInjected = trackersInjected && nil == wErr
		if nil != wErr {
			glog.Errorf("%v", wErr.Error())
		}
	}
	return out, trackersInjected, wErr
}

// GetVideoEventTracking returns map containing key as event name value as associaed video event tracking URL
// By default PBS will expect [EVENT_ID] macro in trackerURL to inject event information
// [EVENT_ID] will be injected with one of the following values
//    firstQuartile, midpoint, thirdQuartile, complete
// If your company can not use [EVENT_ID] and has its own macro. provide config.TrackerMacros implementation
// and ensure that your macro is part of trackerURL configuration
func GetVideoEventTracking(trackerURL string, bid *openrtb2.Bid, requestingBidder string, bidderCoreName string, accountId string, timestamp int64, req *openrtb2.BidRequest, doc *etree.Document, impMap map[string]*openrtb2.Imp) map[string]string {
	eventURLMap := make(map[string]string)
	if "" == strings.TrimSpace(trackerURL) {
		return eventURLMap
	}

	// lookup custom macros
	var customMacroMap map[string]string
	if nil != req.Ext {
		reqExt := new(openrtb_ext.ExtRequest)
		err := json.Unmarshal(req.Ext, &reqExt)
		if err == nil {
			customMacroMap = reqExt.Prebid.Macros
		} else {
			glog.Warningf("Error in unmarshling req.Ext.Prebid.Vast: [%s]", err.Error())
		}
	}

	for _, event := range trackingEvents {
		eventURL := trackerURL
		// lookup in custom macros
		if nil != customMacroMap {
			for customMacro, value := range customMacroMap {
				eventURL = replaceMacro(eventURL, customMacro, value)
			}
		}
		// replace standard macros
		eventURL = replaceMacro(eventURL, VASTAdTypeMacro, string(openrtb_ext.BidTypeVideo))
		if nil != req && nil != req.App {
			// eventURL = replaceMacro(eventURL, VASTAppBundleMacro, req.App.Bundle)
			eventURL = replaceMacro(eventURL, VASTDomainMacro, req.App.Bundle)
			if nil != req.App.Publisher {
				eventURL = replaceMacro(eventURL, PBSAccountMacro, req.App.Publisher.ID)
			}
		}
		if nil != req && nil != req.Site {
			eventURL = replaceMacro(eventURL, VASTDomainMacro, getDomain(req.Site))
			eventURL = replaceMacro(eventURL, VASTPageURLMacro, req.Site.Page)
			if nil != req.Site.Publisher {
				eventURL = replaceMacro(eventURL, PBSAccountMacro, req.Site.Publisher.ID)
			}
		}

		domain := ""
		if len(bid.ADomain) > 0 {
			var err error
			//eventURL = replaceMacro(eventURL, PBSAdvertiserNameMacro, strings.Join(bid.ADomain, ","))
			domain, err = extractDomain(bid.ADomain[0])
			if err != nil {
				glog.Warningf("Unable to extract domain from '%s'. [%s]", bid.ADomain[0], err.Error())
			}
		}

		eventURL = replaceMacro(eventURL, PBSAdvertiserNameMacro, domain)

		eventURL = replaceMacro(eventURL, PBSBidderMacro, bidderCoreName)
		eventURL = replaceMacro(eventURL, PBSBidderCodeMacro, requestingBidder)

		eventURL = replaceMacro(eventURL, PBSBidIDMacro, bid.ID)
		// replace [EVENT_ID] macro with PBS defined event ID
		eventURL = replaceMacro(eventURL, PBSEventIDMacro, eventIDMap[event])

		if imp, ok := impMap[bid.ImpID]; ok {
			eventURL = replaceMacro(eventURL, PBSAdUnitIDMacro, imp.TagID)
		} else {
			glog.Warningf("Setting empty value for %s macro, as failed to determine imp.TagID for bid.ImpID: %s", PBSAdUnitIDMacro, bid.ImpID)
			eventURL = replaceMacro(eventURL, PBSAdUnitIDMacro, "")
		}

		eventURLMap[event] = eventURL
	}
	return eventURLMap
}

func replaceMacro(trackerURL, macro, value string) string {
	macro = strings.TrimSpace(macro)
	trimmedValue := strings.TrimSpace(value)

	if strings.HasPrefix(macro, "[") && strings.HasSuffix(macro, "]") && len(trimmedValue) > 0 {
		trackerURL = strings.ReplaceAll(trackerURL, macro, url.QueryEscape(value))
	} else if strings.HasPrefix(macro, "[") && strings.HasSuffix(macro, "]") && len(trimmedValue) == 0 {
		trackerURL = strings.ReplaceAll(trackerURL, macro, url.QueryEscape(""))
	} else {
		glog.Warningf("Invalid macro '%v'. Either empty or missing prefix '[' or suffix ']", macro)
	}
	return trackerURL
}

//FindCreatives finds Linear, NonLinearAds fro InLine and Wrapper Type of creatives
//from input doc - VAST Document
//NOTE: This function is temporarily seperated to reuse in ctv_auction.go. Because, in case of ctv
//we generate bid.id
func FindCreatives(doc *etree.Document) []*etree.Element {
	// Find Creatives of Linear and NonLinear Type
	// Injecting Tracking Events for Companion is not supported here
	creatives := doc.FindElements("VAST/Ad/InLine/Creatives/Creative/Linear")
	creatives = append(creatives, doc.FindElements("VAST/Ad/Wrapper/Creatives/Creative/Linear")...)
	creatives = append(creatives, doc.FindElements("VAST/Ad/InLine/Creatives/Creative/NonLinearAds")...)
	creatives = append(creatives, doc.FindElements("VAST/Ad/Wrapper/Creatives/Creative/NonLinearAds")...)
	return creatives
}

func extractDomain(rawURL string) (string, error) {
	if !strings.HasPrefix(rawURL, "http") {
		rawURL = "http://" + rawURL
	}
	// decode rawURL
	rawURL, err := url.QueryUnescape(rawURL)
	if nil != err {
		return "", err
	}
	url, err := url.Parse(rawURL)
	if nil != err {
		return "", err
	}
	// remove www if present
	return strings.TrimPrefix(url.Hostname(), "www."), nil
}

func getDomain(site *openrtb2.Site) string {
	if site.Domain != "" {
		return site.Domain
	}

	hostname := ""

	if site.Page != "" {
		pageURL, err := url.Parse(site.Page)
		if err == nil && pageURL != nil {
			hostname = pageURL.Host
		}
	}
	return hostname
}
