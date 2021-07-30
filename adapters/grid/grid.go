package grid

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/util/maputil"
)

type GridAdapter struct {
	endpoint string
}

type GridBidExt struct {
	Bidder ExtBidder `json:"bidder"`
}

type ExtBidder struct {
	Grid ExtBidderGrid `json:"grid"`
}

type ExtBidderGrid struct {
	DemandSource string `json:"demandSource"`
}

type ExtImpDataAdServer struct {
	Name   string `json:"name"`
	AdSlot string `json:"adslot"`
}

type ExtImpData struct {
	PbAdslot string              `json:"pbadslot,omitempty"`
	AdServer *ExtImpDataAdServer `json:"adserver,omitempty"`
}

type ExtImp struct {
	Prebid *openrtb_ext.ExtImpPrebid `json:"prebid,omitempty"`
	Bidder json.RawMessage           `json:"bidder"`
	Data   *ExtImpData               `json:"data,omitempty"`
	Gpid   string                    `json:"gpid,omitempty"`
}

type KeywordSegment struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type KeywordsPublisherItem struct {
	Name     string           `json:"name"`
	Segments []KeywordSegment `json:"segments"`
}

type KeywordsPublisher map[string][]KeywordsPublisherItem

type Keywords map[string]KeywordsPublisher

// buildConsolidatedKeywordsReqExt builds a new request.ext json incorporating request.site.keywords, request.user.keywords,
// and request.imp[0].ext.keywords, and request.ext.keywords. Invalid keywords in request.imp[0].ext.keywords are not incorporated.
// Invalid keywords in request.ext.keywords.site and request.ext.keywords.user are dropped.
func buildConsolidatedKeywordsReqExt(openRTBUser, openRTBSite string, firstImpExt, requestExt json.RawMessage) (json.RawMessage, error) {
	// unmarshal ext to object map
	requestExtMap := parseExtToMap(requestExt)
	firstImpExtMap := parseExtToMap(firstImpExt)
	// extract `keywords` field
	requestExtKeywordsMap := extractKeywordsMap(requestExtMap)
	firstImpExtKeywordsMap := extractBidderKeywordsMap(firstImpExtMap)
	// parse + merge keywords
	keywords := parseKeywordsFromMap(requestExtKeywordsMap)                // request.ext.keywords
	mergeKeywords(keywords, parseKeywordsFromMap(firstImpExtKeywordsMap))  // request.imp[0].ext.bidder.keywords
	mergeKeywords(keywords, parseKeywordsFromOpenRTB(openRTBUser, "user")) // request.user.keywords
	mergeKeywords(keywords, parseKeywordsFromOpenRTB(openRTBSite, "site")) // request.site.keywords

	// overlay site + user keywords
	if site, exists := keywords["site"]; exists && len(site) > 0 {
		requestExtKeywordsMap["site"] = site
	} else {
		delete(requestExtKeywordsMap, "site")
	}
	if user, exists := keywords["user"]; exists && len(user) > 0 {
		requestExtKeywordsMap["user"] = user
	} else {
		delete(requestExtKeywordsMap, "user")
	}
	// reconcile keywords with request.ext
	if len(requestExtKeywordsMap) > 0 {
		requestExtMap["keywords"] = requestExtKeywordsMap
	} else {
		delete(requestExtMap, "keywords")
	}
	// marshal final result
	if len(requestExtMap) > 0 {
		return json.Marshal(requestExtMap)
	}
	return nil, nil
}
func parseExtToMap(ext json.RawMessage) map[string]interface{} {
	var root map[string]interface{}
	if err := json.Unmarshal(ext, &root); err != nil {
		return make(map[string]interface{})
	}
	return root
}
func extractKeywordsMap(ext map[string]interface{}) map[string]interface{} {
	if keywords, exists := maputil.ReadEmbeddedMap(ext, "keywords"); exists {
		return keywords
	}
	return make(map[string]interface{})
}
func extractBidderKeywordsMap(ext map[string]interface{}) map[string]interface{} {
	if bidder, exists := maputil.ReadEmbeddedMap(ext, "bidder"); exists {
		return extractKeywordsMap(bidder)
	}
	return make(map[string]interface{})
}
func parseKeywordsFromMap(extKeywords map[string]interface{}) Keywords {
	keywords := make(Keywords)
	for k, v := range extKeywords {
		// keywords may only be provided in the site and user sections
		if k != "site" && k != "user" {
			continue
		}
		// the site or user sections must be an object
		if section, ok := v.(map[string]interface{}); ok {
			keywords[k] = parseKeywordsFromSection(section)
		}
	}
	return keywords
}
func parseKeywordsFromSection(section map[string]interface{}) KeywordsPublisher {
	keywordsPublishers := make(KeywordsPublisher)
	for publisherKey, publisherValue := range section {
		// publisher value must be a slice
		publisherValueSlice, ok := publisherValue.([]interface{})
		if !ok {
			continue
		}
		for _, publisherValueItem := range publisherValueSlice {
			// item must be an object
			publisherItem, ok := publisherValueItem.(map[string]interface{})
			if !ok {
				continue
			}
			// publisher item must have a name
			publisherName, ok := maputil.ReadEmbeddedString(publisherItem, "name")
			if !ok {
				continue
			}
			var segments []KeywordSegment
			// extract valid segments
			if segmentsSlice, exists := maputil.ReadEmbeddedSlice(publisherItem, "segments"); exists {
				for _, segment := range segmentsSlice {
					if segmentMap, ok := segment.(map[string]interface{}); ok {
						name, hasName := maputil.ReadEmbeddedString(segmentMap, "name")
						value, hasValue := maputil.ReadEmbeddedString(segmentMap, "value")
						if hasName && hasValue {
							segments = append(segments, KeywordSegment{Name: name, Value: value})
						}
					}
				}
			}
			// ensure consistent ordering for publisher item map
			publisherItemKeys := make([]string, 0, len(publisherItem))
			for v := range publisherItem {
				publisherItemKeys = append(publisherItemKeys, v)
			}
			sort.Strings(publisherItemKeys)
			// compose compatible alternate segment format
			for _, potentialSegmentName := range publisherItemKeys {
				potentialSegmentValues := publisherItem[potentialSegmentName]
				// values must be an array
				if valuesSlice, ok := potentialSegmentValues.([]interface{}); ok {
					for _, value := range valuesSlice {
						if valueAsString, ok := value.(string); ok {
							segments = append(segments, KeywordSegment{Name: potentialSegmentName, Value: valueAsString})
						}
					}
				}
			}
			if len(segments) > 0 {
				keywordsPublishers[publisherKey] = append(keywordsPublishers[publisherKey], KeywordsPublisherItem{Name: publisherName, Segments: segments})
			}
		}
	}
	return keywordsPublishers
}
func parseKeywordsFromOpenRTB(keywords, section string) Keywords {
	keywordsSplit := strings.Split(keywords, ",")
	segments := make([]KeywordSegment, 0, len(keywordsSplit))
	for _, v := range keywordsSplit {
		if v != "" {
			segments = append(segments, KeywordSegment{Name: "keywords", Value: v})
		}
	}
	if len(segments) > 0 {
		return map[string]KeywordsPublisher{section: map[string][]KeywordsPublisherItem{"ortb2": {{Name: "keywords", Segments: segments}}}}
	}
	return make(Keywords)
}
func mergeKeywords(a, b Keywords) {
	for key, values := range b {
		if _, sectionExists := a[key]; !sectionExists {
			a[key] = KeywordsPublisher{}
		}
		for publisherKey, publisherValues := range values {
			a[key][publisherKey] = append(publisherValues, a[key][publisherKey]...)
		}
	}
}

func setImpExtKeywords(request *openrtb2.BidRequest) error {
	userKeywords := ""
	if request.User != nil {
		userKeywords = request.User.Keywords
	}
	siteKeywords := ""
	if request.Site != nil {
		siteKeywords = request.Site.Keywords
	}
	var err error
	request.Ext, err = buildConsolidatedKeywordsReqExt(userKeywords, siteKeywords, request.Imp[0].Ext, request.Ext)
	return err
}

func processImp(imp *openrtb2.Imp) error {
	// get the grid extension
	var ext adapters.ExtImpBidder
	var gridExt openrtb_ext.ExtImpGrid
	if err := json.Unmarshal(imp.Ext, &ext); err != nil {
		return err
	}
	if err := json.Unmarshal(ext.Bidder, &gridExt); err != nil {
		return err
	}

	if gridExt.Uid == 0 {
		err := &errortypes.BadInput{
			Message: "uid is empty",
		}
		return err
	}
	// no error
	return nil
}

func setImpExtData(imp openrtb2.Imp) openrtb2.Imp {
	var ext ExtImp
	if err := json.Unmarshal(imp.Ext, &ext); err != nil {
		return imp
	}
	if ext.Data != nil && ext.Data.AdServer != nil && ext.Data.AdServer.AdSlot != "" {
		ext.Gpid = ext.Data.AdServer.AdSlot
		extJSON, err := json.Marshal(ext)
		if err == nil {
			imp.Ext = extJSON
		}
	}
	return imp
}

// MakeRequests makes the HTTP requests which should be made to fetch bids.
func (a *GridAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errors = make([]error, 0)

	// this will contain all the valid impressions
	var validImps []openrtb2.Imp
	// pre-process the imps
	for _, imp := range request.Imp {
		if err := processImp(&imp); err == nil {
			validImps = append(validImps, setImpExtData(imp))
		} else {
			errors = append(errors, err)
		}
	}
	if len(validImps) == 0 {
		err := &errortypes.BadInput{
			Message: "No valid impressions for grid",
		}
		errors = append(errors, err)
		return nil, errors
	}

	if err := setImpExtKeywords(request); err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	request.Imp = validImps

	reqJSON, err := json.Marshal(request)
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    reqJSON,
		Headers: headers,
	}}, errors
}

// MakeBids unpacks the server's response into Bids.
func (a *GridAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bidMeta, err := getBidMeta(sb.Bid[i].Ext)
			bidType, err := getMediaTypeForImp(sb.Bid[i].ImpID, internalRequest.Imp)
			if err != nil {
				return nil, []error{err}
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: bidType,
				BidMeta: bidMeta,
			})
		}
	}
	return bidResponse, nil

}

// Builder builds a new instance of the Grid adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &GridAdapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func getBidMeta(ext json.RawMessage) (*openrtb_ext.ExtBidPrebidMeta, error) {
	var bidExt GridBidExt

	if err := json.Unmarshal(ext, &bidExt); err != nil {
		return nil, err
	}
	var bidMeta *openrtb_ext.ExtBidPrebidMeta
	if bidExt.Bidder.Grid.DemandSource != "" {
		bidMeta = &openrtb_ext.ExtBidPrebidMeta{
			NetworkName: bidExt.Bidder.Grid.DemandSource,
		}
	}
	return bidMeta, nil
}

func getMediaTypeForImp(impID string, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	for _, imp := range imps {
		if imp.ID == impID {
			if imp.Banner != nil {
				return openrtb_ext.BidTypeBanner, nil
			}

			if imp.Video != nil {
				return openrtb_ext.BidTypeVideo, nil
			}

			return "", &errortypes.BadServerResponse{
				Message: fmt.Sprintf("Unknown impression type for ID: \"%s\"", impID),
			}
		}
	}

	// This shouldnt happen. Lets handle it just incase by returning an error.
	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("Failed to find impression for ID: \"%s\"", impID),
	}
}
