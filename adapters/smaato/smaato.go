package smaato

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const bidTypeExtKey = "BidType"

type SmaatoAdapter struct {
	http *adapters.HTTPAdapter
	URI  string
}

type ImageAd struct {
	Image Image `json:"image"`
}
type Image struct {
	Img                IMG      `json:"img"`
	Impressiontrackers []string `json:"impressiontrackers"`
	Clicktrackers      []string `json:"clicktrackers"`
}
type IMG struct {
	URL    string `json:"url"`
	W      int    `json:"w"`
	H      int    `json:"h"`
	Ctaurl string `json:"ctaurl"`
}

// used for cookies and such
func (a *SmaatoAdapter) Name() string {
	return "smaato"
}

func (a *SmaatoAdapter) SkipNoCookies() bool {
	return false
}

func (a *SmaatoAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	errs := make([]error, 0, len(request.Imp))

	var err error
	publisherId := ""

	if request.Site != nil {
		siteCopy := *request.Site
		if siteCopy.Publisher != nil {
			publisherCopy := *siteCopy.Publisher
			publisherCopy.ID = publisherId
			siteCopy.Publisher = &publisherCopy
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

		tagId := smaatoExt.AdSpaceId
		instl := smaatoExt.Instl
		secure := smaatoExt.Secure

		if tagId != "" {
			imp.TagID = tagId
			imp.Ext = nil
		}
		if instl >= 0 && secure != nil {
			imp.Instl = instl
			imp.Secure = secure
		}
		return nil
	}
	return fmt.Errorf("invalid MediaType. SMAATO only supports Banner. Ignoring ImpID=%s", imp.ID)
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
		return nil, []error{fmt.Errorf("unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode)}
	}

	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	adType := response.Headers.Get("X-SMT-ADTYPE")

	var errs []error
	for _, sb := range bidResp.SeatBid {
		for i := 0; i < len(sb.Bid); i++ {
			bid := sb.Bid[i]
			adm, _ := getADM(adType, bid.AdM)
			bid.AdM = adm
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

func getADM(adType string, adapterResponseAdm string) (string, bool) {
	imageMarkup, done := extractAdmImage(adType, adapterResponseAdm)
	if done {
		return imageMarkup, done
	}
	return adapterResponseAdm, done
}

func extractAdmImage(adType string, adapterResponseAdm string) (string, bool) {
	var imgMarkup string
	if strings.EqualFold(adType, "img") {

		var imageAd ImageAd
		err := json.Unmarshal([]byte(adapterResponseAdm), &imageAd)
		var image = imageAd.Image

		if err == nil {
			var clickEvent string
			for _, clicktracker := range image.Clicktrackers {
				clickEvent += "fetch(decodeURIComponent(" + url.QueryEscape(clicktracker) + "), {cache: 'no-cache'});"
			}
			imgMarkup = "<div onclick=" + clickEvent + "><a href=" + image.Img.Ctaurl + "><img src=" + image.
				Img.URL + " width=" + strconv.Itoa(image.Img.W) + " height=" + strconv.Itoa(image.Img.
				H) + "/></a></div>"
		}
		return imgMarkup, true
	}
	return adapterResponseAdm, false
}

func NewSmaatoBidder(client *http.Client, uri string) *SmaatoAdapter {
	a := &adapters.HTTPAdapter{Client: client}
	return &SmaatoAdapter{
		http: a,
		URI:  uri,
	}
}
