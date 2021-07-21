package grid

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
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

type ReqExt struct {
	Prebid   json.RawMessage `json:"prebid,omitempty"`
	Keywords json.RawMessage `json:"keywords,omitempty"`
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

func mixKeywords(keywordsString string, keywords map[string]interface{}) {
	var ortb2Array []interface{}

	keywordsArr := strings.Split(keywordsString, ",")

	if len(keywordsArr) > 0 {
		keywordsInt := make([]interface{}, len(keywordsArr))
		for i, v := range keywordsArr {
			keywordsInt[i] = v
		}
		ortb2Keywords := map[string]interface{}{
			"name":     "keywords",
			"keywords": keywordsInt,
		}
		if keywords["ortb2"] == nil {
			ortb2Array = make([]interface{}, 0)
		} else {
			ortb2Array = keywords["ortb2"].([]interface{})
		}
		ortb2Array = append(ortb2Array, ortb2Keywords)
		keywords["ortb2"] = ortb2Array
	}
}

func mergeWithReqExtKeywords(extKeywords map[string]interface{}, request *openrtb2.BidRequest) error {
	var reqExt ReqExt
	var reqExtKeywords map[string]interface{}

	if err := json.Unmarshal(request.Ext, &reqExt); err != nil {
		return err
	}
	if reqExt.Keywords != nil {
		json.Unmarshal(reqExt.Keywords, &reqExtKeywords)

		for key, keyword := range reqExtKeywords {
			if extKeywords[key] == nil {
				extKeywords[key] = keyword
			} else {
				if key == "site" || key == "user" {
					target := extKeywords[key].(map[string]interface{})
					from := keyword.(map[string]interface{})

					for name, value := range from {
						if target[name] == nil {
							target[name] = value
						} else {
							valueArr := value.([]interface{})
							targetArr := target[name].([]interface{})
							target[name] = append(targetArr, valueArr...)
						}
					}
				} else {
					extKeywords[key] = keyword
				}
			}
		}
	}
	return nil
}

func fillSegments(publisherItem map[string]interface{}) []KeywordSegment {
	segments := make([]KeywordSegment, 0)

	for key, value := range publisherItem {
		if key != "name" {
			if keywords, ok := value.([]interface{}); ok {
				for _, keyword := range keywords {
					switch keyword.(type) {
					case map[string]interface{}:
						keywordSegment := keyword.(map[string]interface{})
						if key == "segments" && keywordSegment["name"] != nil && keywordSegment["value"] != nil {
							segment := KeywordSegment{
								Name:  keywordSegment["name"].(string),
								Value: keywordSegment["value"].(string),
							}
							segments = append(segments, segment)
						}
					case string:
						segment := KeywordSegment{
							Name:  key,
							Value: keyword.(string),
						}
						segments = append(segments, segment)
					}
				}
			}
		}
	}

	return segments
}

func reformatExtKeywords(extKeywords map[string]interface{}) {
	for name, pubData := range extKeywords {
		if pubArr, ok := pubData.([]interface{}); ok {
			formatedPubArr := make([]KeywordsPublisherItem, 0)
			for _, item := range pubArr {
				if publisherItem, itemOk := item.(map[string]interface{}); itemOk {
					segments := fillSegments(publisherItem)

					if len(segments) > 0 {
						formatedPublisher := KeywordsPublisherItem{
							Name:     publisherItem["name"].(string),
							Segments: segments,
						}
						formatedPubArr = append(formatedPubArr, formatedPublisher)
					}
				}
			}
			if len(formatedPubArr) > 0 {
				extKeywords[name] = formatedPubArr
			} else {
				delete(extKeywords, name)
			}
		} else {
			delete(extKeywords, name)
		}
	}
}

func updateExtKeywords(keywords json.RawMessage, request *openrtb2.BidRequest) json.RawMessage {
	var extKeywords map[string]interface{}
	var extKWSite map[string]interface{}
	var extKWUser map[string]interface{}

	json.Unmarshal(keywords, &extKeywords)

	if request.Ext != nil {
		if extKeywords == nil {
			extKeywords = make(map[string]interface{})
		}
		mergeWithReqExtKeywords(extKeywords, request)
	}

	if request.Site != nil && request.Site.Keywords != "" {
		if extKeywords == nil {
			extKeywords = make(map[string]interface{})
		}

		if extKeywords["site"] != nil {
			extKWSite = extKeywords["site"].(map[string]interface{})
		} else {
			extKWSite = make(map[string]interface{})
		}

		mixKeywords(request.Site.Keywords, extKWSite)
		extKeywords["site"] = extKWSite
	}

	if request.User != nil && request.User.Keywords != "" {
		if extKeywords == nil {
			extKeywords = make(map[string]interface{})
		}

		if extKeywords["user"] != nil {
			extKWUser = extKeywords["user"].(map[string]interface{})
		} else {
			extKWUser = make(map[string]interface{})
		}

		mixKeywords(request.User.Keywords, extKWUser)
		extKeywords["user"] = extKWUser
	}

	if extKeywords != nil {
		if extKeywords["site"] != nil {
			if extKeywordsSite, ok := extKeywords["site"].(map[string]interface{}); ok {
				reformatExtKeywords(extKeywordsSite)
			}
		}
		if extKeywords["user"] != nil {
			if extKeywordsUser, ok := extKeywords["user"].(map[string]interface{}); ok {
				reformatExtKeywords(extKeywordsUser)
			}
		}

		if extKeywordsJSON, err := json.Marshal(extKeywords); err == nil {
			return extKeywordsJSON
		}
	}

	return nil
}

func setImpExtKeywords(imp openrtb2.Imp, request *openrtb2.BidRequest) error {
	var ext adapters.ExtImpBidder
	var gridExt openrtb_ext.ExtImpGrid
	var reqExt ReqExt
	if err := json.Unmarshal(imp.Ext, &ext); err != nil {
		return err
	}
	if err := json.Unmarshal(ext.Bidder, &gridExt); err != nil {
		return err
	}

	keywords := updateExtKeywords(gridExt.Keywords, request)

	if keywords != nil {
		reqExt.Keywords = keywords
		extJSON, err := json.Marshal(reqExt)
		if err != nil {
			return err
		}
		request.Ext = extJSON
	}
	return nil
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

	err := setImpExtKeywords(validImps[0], request)
	if err != nil {
		errors = append(errors, err)
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
