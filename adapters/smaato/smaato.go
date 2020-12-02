package smaato

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const clientVersion = "prebid_server_0.1"

type adMarkupType string

const (
	smtAdTypeImg       adMarkupType = "Img"
	smtAdTypeRichmedia adMarkupType = "Richmedia"
	smtAdTypeVideo     adMarkupType = "Video"
)

// SmaatoAdapter describes a Smaato prebid server adapter.
type SmaatoAdapter struct {
	URI string
}

//userExt defines User.Ext object for Smaato
type userExt struct {
	Data userExtData `json:"data"`
}

type userExtData struct {
	Keywords string `json:"keywords"`
	Gender   string `json:"gender"`
	Yob      int64  `json:"yob"`
}

//userExt defines Site.Ext object for Smaato
type siteExt struct {
	Data siteExtData `json:"data"`
}

type siteExtData struct {
	Keywords string `json:"keywords"`
}

// Builder builds a new instance of the Smaato adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &SmaatoAdapter{
		URI: config.Endpoint,
	}
	return bidder, nil
}

// MakeRequests makes the HTTP requests which should be made to fetch bids.
func (a *SmaatoAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	errs := make([]error, 0, len(request.Imp))
	if len(request.Imp) == 0 {
		errs = append(errs, &errortypes.BadInput{Message: "no impressions in bid request"})
		return nil, errs
	}

	// Use bidRequestExt of first imp to retrieve params which are valid for all imps, e.g. publisherId
	publisherID, err := jsonparser.GetString(request.Imp[0].Ext, "bidder", "publisherId")
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	for i := 0; i < len(request.Imp); i++ {
		err := parseImpressionObject(&request.Imp[i])
		// If the parsing is failed, remove imp and add the error.
		if err != nil {
			errs = append(errs, err)
			request.Imp = append(request.Imp[:i], request.Imp[i+1:]...)
			i--
		}
	}
	if request.Site != nil {
		siteCopy := *request.Site
		siteCopy.Publisher = &openrtb.Publisher{ID: publisherID}

		if request.Site.Ext != nil {
			var siteExt siteExt
			err := json.Unmarshal([]byte(request.Site.Ext), &siteExt)
			if err != nil {
				errs = append(errs, err)
				return nil, errs
			}
			siteCopy.Keywords = siteExt.Data.Keywords
			siteCopy.Ext = nil
		}
		request.Site = &siteCopy
	}

	if request.User != nil && request.User.Ext != nil {
		var userExt userExt
		var userExtRaw map[string]json.RawMessage

		rawExtErr := json.Unmarshal(request.User.Ext, &userExtRaw)
		if rawExtErr != nil {
			errs = append(errs, rawExtErr)
			return nil, errs
		}

		userExtErr := json.Unmarshal([]byte(request.User.Ext), &userExt)
		if userExtErr != nil {
			errs = append(errs, userExtErr)
			return nil, errs
		}

		userCopy := *request.User
		extractUserExtAttributes(userExt, &userCopy)
		delete(userExtRaw, "data")
		userCopy.Ext, err = json.Marshal(userExtRaw)
		if err != nil {
			errs = append(errs, err)
			return nil, errs
		}
		request.User = &userCopy
	}

	// Setting ext client info
	type bidRequestExt struct {
		Client string `json:"client"`
	}
	request.Ext, err = json.Marshal(bidRequestExt{Client: clientVersion})
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}
	reqJSON, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	uri := a.URI

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     uri,
		Body:    reqJSON,
		Headers: headers,
	}}, errs
}

// MakeBids unpacks the server's response into Bids.
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
		return nil, []error{fmt.Errorf("unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode)}
	}

	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	for _, sb := range bidResp.SeatBid {
		for i := 0; i < len(sb.Bid); i++ {
			bid := sb.Bid[i]

			markupType, markupTypeErr := getAdMarkupType(response, bid.AdM)
			if markupTypeErr != nil {
				return nil, []error{markupTypeErr}
			}

			var markupError error
			bid.AdM, markupError = renderAdMarkup(markupType, bid.AdM)
			if markupError != nil {
				return nil, []error{markupError}
			}

			bidType, bidTypeErr := markupTypeToBidType(markupType)
			if bidTypeErr != nil {
				return nil, []error{bidTypeErr}
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: bidType,
			})
		}
	}
	return bidResponse, nil
}

func renderAdMarkup(adMarkupType adMarkupType, adMarkup string) (string, error) {
	var markupError error
	var adm string
	switch adMarkupType {
	case smtAdTypeImg:
		adm, markupError = extractAdmImage(adMarkup)
	case smtAdTypeRichmedia:
		adm, markupError = extractAdmRichMedia(adMarkup)
	case smtAdTypeVideo:
		adm, markupError = adMarkup, nil
	default:
		return "", fmt.Errorf("Unknown markup type %s", adMarkupType)
	}
	return adm, markupError
}

func markupTypeToBidType(markupType adMarkupType) (openrtb_ext.BidType, error) {
	switch markupType {
	case smtAdTypeImg:
		return openrtb_ext.BidTypeBanner, nil
	case smtAdTypeRichmedia:
		return openrtb_ext.BidTypeBanner, nil
	case smtAdTypeVideo:
		return openrtb_ext.BidTypeVideo, nil
	default:
		return "", fmt.Errorf("Invalid markupType %s", markupType)
	}
}

func getAdMarkupType(response *adapters.ResponseData, adMarkup string) (adMarkupType, error) {
	if admType := adMarkupType(response.Headers.Get("X-SMT-ADTYPE")); admType != "" {
		return admType, nil
	}
	if strings.HasPrefix(adMarkup, `{"image":`) {
		return smtAdTypeImg, nil
	}
	if strings.HasPrefix(adMarkup, `{"richmedia":`) {
		return smtAdTypeRichmedia, nil
	}
	if strings.HasPrefix(adMarkup, `<?xml`) {
		return smtAdTypeVideo, nil
	}
	return "", fmt.Errorf("Invalid ad markup %s", adMarkup)
}

func assignBannerSize(banner *openrtb.Banner) (*openrtb.Banner, error) {
	if banner.W != nil && banner.H != nil {
		return banner, nil
	}
	if len(banner.Format) == 0 {
		return banner, fmt.Errorf("No sizes provided for Banner %v", banner.Format)
	}
	bannerCopy := *banner
	bannerCopy.W = new(uint64)
	*bannerCopy.W = banner.Format[0].W
	bannerCopy.H = new(uint64)
	*bannerCopy.H = banner.Format[0].H

	return &bannerCopy, nil
}

// parseImpressionObject parse the imp to get it ready to send to smaato
func parseImpressionObject(imp *openrtb.Imp) error {
	adSpaceID, err := jsonparser.GetString(imp.Ext, "bidder", "adspaceId")
	if err != nil {
		return err
	}

	// SMAATO supports banner impressions.
	if imp.Banner != nil {
		bannerCopy, err := assignBannerSize(imp.Banner)
		if err != nil {
			return err
		}
		imp.Banner = bannerCopy
		imp.TagID = adSpaceID
		imp.Ext = nil
		return nil
	}

	if imp.Video != nil {
		imp.TagID = adSpaceID
		imp.Ext = nil
		return nil
	}

	return fmt.Errorf("invalid MediaType. SMAATO only supports Banner and Video. Ignoring ImpID=%s", imp.ID)
}

func extractUserExtAttributes(userExt userExt, userCopy *openrtb.User) {
	gender := userExt.Data.Gender
	if gender != "" {
		userCopy.Gender = gender
	}

	yob := userExt.Data.Yob
	if yob != 0 {
		userCopy.Yob = yob
	}

	keywords := userExt.Data.Keywords
	if keywords != "" {
		userCopy.Keywords = keywords
	}
}
