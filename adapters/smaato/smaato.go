package smaato

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const client = "prebid_server_0.1"

type smaatoParams openrtb_ext.ExtImpSmaato
type adMarkupType string

const (
	smtAdTypeImg       adMarkupType = "Img"
	smtAdTypeRichmedia adMarkupType = "Richmedia"
)

// SmaatoAdapter describes a Smaato prebid server adapter.
type SmaatoAdapter struct {
	http *adapters.HTTPAdapter
	URI  string
}

// NewSmaatoBidder creates a Smaato bid adapter.
func NewSmaatoBidder(client *http.Client, uri string) *SmaatoAdapter {
	a := &adapters.HTTPAdapter{Client: client}
	return &SmaatoAdapter{
		http: a,
		URI:  uri,
	}
}

// MakeRequests makes the HTTP requests which should be made to fetch bids.
func (a *SmaatoAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	errs := make([]error, 0, len(request.Imp))
	if len(request.Imp) == 0 {
		errs = append(errs, &errortypes.BadInput{Message: "no impressions in bid request"})
		return nil, errs
	}

	// Use bidRequestExt of first imp to retrieve params which are valid for all imps, e.g. publisherId
	smaatoParams, err := parseSmaatoParams(&request.Imp[0])
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
		siteCopy.Publisher = &openrtb.Publisher{ID: smaatoParams.PublisherID}
		request.Site = &siteCopy
	}

	// Setting ext client info
	type bidRequestExt struct {
		Client string `json:"client"`
	}
	request.Ext, _ = json.Marshal(bidRequestExt{Client: client})

	reqJSON, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	uri := a.URI
	if smaatoParams.Endpoint != "" {
		uri = smaatoParams.Endpoint
	}

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

			var markupError error
			bid.AdM, markupError = renderAdMarkup(getAdMarkupType(response, bid.AdM), bid.AdM)
			if markupError != nil {
				continue // no bid when broken ad markup
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: openrtb_ext.BidTypeBanner,
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
	default:
		return "", fmt.Errorf("Unknown markup type %s", adMarkupType)
	}
	return adm, markupError
}

func getAdMarkupType(response *adapters.ResponseData, adMarkup string) adMarkupType {
	admType := adMarkupType(response.Headers.Get("X-SMT-ADTYPE"))
	if admType == "" && strings.HasPrefix(adMarkup, `{"image":`) {
		admType = smtAdTypeImg
	}
	if admType == "" && strings.HasPrefix(adMarkup, `{"richmedia":`) {
		admType = smtAdTypeRichmedia
	}
	return admType
}

func assignBannerSize(banner *openrtb.Banner) error {
	if banner == nil {
		return nil
	}
	if banner.W != nil && banner.H != nil {
		return nil
	}
	if len(banner.Format) == 0 {
		return fmt.Errorf("No sizes provided for Banner %v", banner.Format)
	}

	banner.W = new(uint64)
	*banner.W = banner.Format[0].W
	banner.H = new(uint64)
	*banner.H = banner.Format[0].H
	return nil
}

// parseImpressionObject parse the imp to get it ready to send to smaato
func parseImpressionObject(imp *openrtb.Imp) error {
	smaatoParams, err := parseSmaatoParams(imp)
	if err != nil {
		return err
	}

	// SMAATO supports banner impressions.
	if imp.Banner != nil {
		if err := assignBannerSize(imp.Banner); err != nil {
			return err
		}

		imp.TagID = smaatoParams.AdSpaceID
		imp.Ext = nil

		return nil
	}
	return fmt.Errorf("invalid MediaType. SMAATO only supports Banner. Ignoring ImpID=%s", imp.ID)
}

func parseSmaatoParams(imp *openrtb.Imp) (smaatoParams, error) {
	var bidderExt adapters.ExtImpBidder
	var smaatoExt smaatoParams

	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return smaatoExt, err
	}
	if err := json.Unmarshal(bidderExt.Bidder, &smaatoExt); err != nil {
		return smaatoExt, err
	}
	return smaatoExt, nil
}
