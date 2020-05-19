package smaato

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
)

const MAX_IMPRESSIONS_SMAATO = 30
const bidTypeExtKey = "BidType"

type SmaatoAdapter struct {
	http *adapters.HTTPAdapter
	URI  string
}

// used for cookies and such
func (a *SmaatoAdapter) Name() string {
	return "smaato"
}

func (a *SmaatoAdapter) SkipNoCookies() bool {
	return false
}

const (
	INVALID_PARAMS = "Invalid BidParam"
)

type smaatoSize struct {
	w uint16
	h uint16
}

var smaatoSizeMap = map[smaatoSize]int{
	{w: 320, h: 50}:  1,
	{w: 320, h: 250}: 2,
}

func PrepareLogMessage(tID, pubId, adUnitId, bidID, details string, args ...interface{}) string {
	return fmt.Sprintf("[SMAATO] ReqID [%s] PubID [%s] AdUnit [%s] BidID [%s] %s \n",
		tID, pubId, adUnitId, bidID, details)
}

func (a *SmaatoAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	errs := make([]error, 0, len(request.Imp))

	var err error
	publisherId := ""
	//instl :="

	if request.Site != nil {
		siteCopy := *request.Site
		if siteCopy.Publisher != nil {
			publisherCopy := *siteCopy.Publisher
			publisherCopy.ID = publisherId
			siteCopy.Publisher = &publisherCopy
		} else {
			siteCopy.Publisher = &openrtb.Publisher{ID: publisherId}
		}
		request.Site = &siteCopy
	} else if request.App != nil {
		appCopy := *request.App
		if appCopy.Publisher == nil {
			appCopy.Publisher = &openrtb.Publisher{ID: publisherId}
		}
		request.App = &appCopy
	}

	for i := 0; i < len(request.Imp); i++ {
		err = parseImpressionObject(&request.Imp[i])
		// If the parsing is failed, remove imp and add the error.
		if err != nil {
			errs = append(errs, err)
			request.Imp = append(request.Imp[:i], request.Imp[i+1:]...)
			i--
		}
	}

	thisURI := a.URI

	// If all the requests are invalid, Call to adaptor is skipped
	if len(request.Imp) == 0 {
		return nil, errs
	}

	reqJSON, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     thisURI,
		Body:    reqJSON,
		Headers: headers,
	}}, errs
}

func assignBannerSize(banner *openrtb.Banner) error {
	if banner == nil {
		return nil
	}
	if banner.W != nil && banner.H != nil {
		return nil
	}
	if len(banner.Format) == 0 {
		return errors.New(fmt.Sprintf("No sizes provided for Banner %v", banner.Format))
	}

	banner.W = new(uint64)
	*banner.W = banner.Format[0].W
	banner.H = new(uint64)
	*banner.H = banner.Format[0].H
	return nil
}

// parseImpressionObject parse the imp to get it ready to send to smaato
func parseImpressionObject(imp *openrtb.Imp) error {
	// SMAATO supports banner impressions.

	if imp.Banner != nil {
		if err := assignBannerSize(imp.Banner); err != nil {
			return err
		}

		var bidderExt adapters.ExtImpBidder
		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			return err
		}

		var smaatoExt openrtb_ext.ExtImpSmaato
		if err := json.Unmarshal(bidderExt.Bidder, &smaatoExt); err != nil {
			return err
		}

		tagId := smaatoExt.TagId
		id := smaatoExt.Id
		instl := smaatoExt.Instl
		secure := smaatoExt.Secure

		if tagId != "" && id != "" && instl >= 0 && secure != nil {
			imp.ID = id
			imp.TagID = tagId
			imp.Instl = instl
			imp.Secure = secure

			imp.Banner.Format = nil
			imp.Ext = nil
		} else {
			return fmt.Errorf("Invalid MediaType. SMAATO only supports Banner. Ignoring ImpID=%s", imp.ID)
		}
		return nil
	}
	return fmt.Errorf("Invalid MediaType. SMAATO only supports Banner. Ignoring ImpID=%s", imp.ID)
}

func (a *SmaatoAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{fmt.Errorf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode)}
	}

	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	var errs []error
	for _, sb := range bidResp.SeatBid {
		for i := 0; i < len(sb.Bid); i++ {
			bid := sb.Bid[i]
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: getBidType(bid.Ext),
			})

		}
	}
	return bidResponse, errs
}

// getBidType returns the bid type specified in the response bid.ext
func getBidType(bidExt json.RawMessage) openrtb_ext.BidType {
	// setting "banner" as the default bid type
	bidType := openrtb_ext.BidTypeBanner
	if bidExt != nil {
		bidExtMap := make(map[string]interface{})
		extbyte, err := json.Marshal(bidExt)
		if err == nil {
			err = json.Unmarshal(extbyte, &bidExtMap)
			if err == nil && bidExtMap[bidTypeExtKey] != nil {
				bidTypeVal := int(bidExtMap[bidTypeExtKey].(float64))
				switch bidTypeVal {
				case 0:
					bidType = openrtb_ext.BidTypeBanner
				case 1:
					bidType = openrtb_ext.BidTypeVideo
				case 2:
					bidType = openrtb_ext.BidTypeNative
				default:
					// default value is banner
					bidType = openrtb_ext.BidTypeBanner
				}
			}
		}
	}
	return bidType
}

func logf(msg string, args ...interface{}) {
	if glog.V(2) {
		glog.Infof(msg, args...)
	}
}

func NewSmaatoAdapter(config *adapters.HTTPAdapterConfig, uri string) *SmaatoAdapter {
	a := adapters.NewHTTPAdapter(config)
	return &SmaatoAdapter{
		http: a,
		URI:  uri,
	}
}

func NewSmaatoBidder(client *http.Client, uri string) *SmaatoAdapter {
	a := &adapters.HTTPAdapter{Client: client}
	return &SmaatoAdapter{
		http: a,
		URI:  uri,
	}
}
