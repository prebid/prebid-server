package mediasquare

import (
	"encoding/json"
	"fmt"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
)

// msqResponse: Bid-Response sent by mediasquare.
type msqResponse struct {
	Infos struct {
		Version     string `json:"version"`
		Description string `json:"description"`
		Hostname    string `json:"hostname,omitempty"`
	} `json:"infos"`
	Responses []msqResponseBids `json:"responses"`
}

// msqParameters: Bid-Request sent to mediasquare.
type msqParameters struct {
	Codes []msqParametersCodes `json:"codes"`
	Gdpr  struct {
		ConsentRequired bool   `json:"consent_required"`
		ConsentString   string `json:"consent_string"`
	} `json:"gdpr"`
	Type    string      `json:"type"`
	DSA     interface{} `json:"dsa,omitempty"`
	Support msqSupport  `json:"tech"`
	Test    bool        `json:"test"`
}

type msqResponseBidsVideo struct {
	Xml string `json:"xml"`
	Url string `json:"url"`
}

type nativeResponseImg struct {
	Url    string `json:"url"`
	Width  *int   `json:"width,omitempty"`
	Height *int   `json:"height,omitempty"`
}

type msqResponseBidsNative struct {
	ClickUrl           string             `json:"clickUrl,omitempty"`
	ClickTrackers      []string           `json:"clickTrackers,omitempty"`
	ImpressionTrackers []string           `json:"impressionTrackers,omitempty"`
	JavascriptTrackers []string           `json:"javascriptTrackers,omitempty"`
	Privacy            *string            `json:"privacy,omitempty"`
	Title              *string            `json:"title,omitempty"`
	Icon               *nativeResponseImg `json:"icon,omitempty"`
	Image              *nativeResponseImg `json:"image,omitempty"`
	Cta                *string            `json:"cta,omitempty"`
	Rating             *string            `json:"rating,omitempty"`
	Downloads          *string            `json:"downloads,omitempty"`
	Likes              *string            `json:"likes,omitempty"`
	Price              *string            `json:"price,omitempty"`
	SalePrice          *string            `json:"saleprice,omitempty"`
	Address            *string            `json:"address,omitempty"`
	Phone              *string            `json:"phone,omitempty"`
	Body               *string            `json:"body,omitempty"`
	Body2              *string            `json:"body2,omitempty"`
	SponsoredBy        *string            `json:"sponsoredBy,omitempty"`
	DisplayUrl         *string            `json:"displayUrl,omitempty"`
}

type msqResponseBids struct {
	ID            string                 `json:"id"`
	Ad            string                 `json:"ad,omitempty"`
	BidId         string                 `json:"bid_id,omitempty"`
	Bidder        string                 `json:"bidder,omitempty"`
	Cpm           float64                `json:"cpm,omitempty"`
	Currency      string                 `json:"currency,omitempty"`
	CreativeId    string                 `json:"creative_id,omitempty"`
	Height        int64                  `json:"height,omitempty"`
	Width         int64                  `json:"width,omitempty"`
	NetRevenue    bool                   `json:"net_revenue,omitempty"`
	TransactionId string                 `json:"transaction_id,omitempty"`
	Ttl           int                    `json:"ttl,omitempty"`
	Video         *msqResponseBidsVideo  `json:"video,omitempty"`
	Native        *msqResponseBidsNative `json:"native,omitempty"`
	ADomain       []string               `json:"adomain,omitempty"`
	Dsa           interface{}            `json:"dsa,omitempty"`
	BURL          string                 `json:"burl,omitempty"`
}

type msqSupport struct {
	Device interface{} `json:"device"`
	App    interface{} `json:"app"`
}

type msqParametersCodes struct {
	AdUnit     string              `json:"adunit"`
	AuctionId  string              `json:"auctionid"`
	BidId      string              `json:"bidid"`
	Code       string              `json:"code"`
	Owner      string              `json:"owner"`
	Mediatypes mediaTypes          `json:"mediatypes,omitempty"`
	Floor      map[string]msqFloor `json:"floor,omitempty"`
}

type msqFloor struct {
	Price    float64 `json:"floor,omitempty"`
	Currency string  `json:"currency,omitempty"`
}

type mediaTypeNativeBasis struct {
	Required bool
	Len      *int
}

type mediaTypeNativeImage struct {
	Required     bool
	Sizes        []*int
	Aspect_ratio *struct {
		Min_width    *int
		Min_height   *int
		Ratio_width  *int
		Ratio_height *int
	}
}

type mediaTypeNativeTitle struct {
	Required bool
	Len      int
}

type mediaTypeNative struct {
	Title       *mediaTypeNativeTitle `json:"title"`
	Icon        *mediaTypeNativeImage `json:"icon"`
	Image       *mediaTypeNativeImage `json:"image"`
	Clickurl    *mediaTypeNativeBasis `json:"clickUrl"`
	Displayurl  *mediaTypeNativeBasis `json:"displayUrl"`
	Privacylink *mediaTypeNativeBasis `json:"privacyLink"`
	Privacyicon *mediaTypeNativeBasis `json:"privacyIcon"`
	Cta         *mediaTypeNativeBasis `json:"cta"`
	Rating      *mediaTypeNativeBasis `json:"rating"`
	Downloads   *mediaTypeNativeBasis `json:"downloads"`
	Likes       *mediaTypeNativeBasis `json:"likes"`
	Price       *mediaTypeNativeBasis `json:"price"`
	Saleprice   *mediaTypeNativeBasis `json:"saleprice"`
	Address     *mediaTypeNativeBasis `json:"address"`
	Phone       *mediaTypeNativeBasis `json:"phone"`
	Body        *mediaTypeNativeBasis `json:"body"`
	Body2       *mediaTypeNativeBasis `json:"body2"`
	Sponsoredby *mediaTypeNativeBasis `json:"sponsoredBy"`
	Sizes       [][]int               `json:"sizes"`
	Type        string                `json:"type"`
}

type mediaTypeVideo struct {
	Mimes          []string `json:"mimes"`
	Minduration    *int     `json:"minduration"`
	Maxduration    *int     `json:"maxduration"`
	Protocols      []*int   `json:"protocols"`
	Startdelay     *int     `json:"startdelay"`
	Placement      *int     `json:"placement"`
	Skip           *int     `json:"skip"`
	Skipafter      *int     `json:"skipafter"`
	Minbitrate     *int     `json:"minbitrate"`
	Maxbitrate     *int     `json:"maxbitrate"`
	Delivery       []*int   `json:"delivery"`
	Playbackmethod []*int   `json:"playbackmethod"`
	Api            []*int   `json:"api"`
	Linearity      *int     `json:"linearity"`
	W              *int     `json:"w"`
	H              *int     `json:"h"`
	Boxingallowed  *int     `json:"boxingallower"`
	PlayerSize     [][]int  `json:"playersize"`
	Context        string   `json:"context"`
	Plcmt          *int     `json:"plcmt,omitempty"`
}

type mediaTypes struct {
	Banner *mediaTypeBanner `json:"banner"`
	Video  *mediaTypeVideo  `json:"video"`
	Native *mediaTypeNative `json:"native"`
}

type mediaTypeBanner struct {
	Sizes [][]*int `json:"sizes"`
}

func initMsqParams(request *openrtb2.BidRequest) (msqParams msqParameters) {
	msqParams.Type = "pbs"
	msqParams.Support = msqSupport{
		Device: request.Device,
		App:    request.App,
	}
	msqParams.Gdpr = struct {
		ConsentRequired bool   `json:"consent_required"`
		ConsentString   string `json:"consent_string"`
	}{
		ConsentRequired: (parserGDPR{}).getValue("consent_requirement", request) == "true",
		ConsentString:   (parserGDPR{}).getValue("consent_string", request),
	}
	msqParams.DSA = (parserDSA{}).getValue(request)

	return
}

// setContent: Loads currentImp into msqParams (*msqParametersCodes),
// returns (errs []error, ok bool) where `ok` express if mandatory content had been loaded.
func (msqParams *msqParametersCodes) setContent(currentImp openrtb2.Imp) (ok bool) {
	var (
		currentMapFloors = make(map[string]msqFloor, 0)
		currentFloor     = msqFloor{
			Price:    currentImp.BidFloor,
			Currency: currentImp.BidFloorCur,
		}
	)

	if currentImp.Video != nil {
		ok = true
		var video mediaTypeVideo
		currentVideoBytes, _ := json.Marshal(currentImp.Video)
		json.Unmarshal(currentVideoBytes, &video)
		json.Unmarshal(currentImp.Video.Ext, &video)

		msqParams.Mediatypes.Video = &video
		if msqParams.Mediatypes.Video != nil {
			if currentImp.Video.W != nil && currentImp.Video.H != nil {
				currentMapFloors[fmt.Sprintf("%dx%d", *(currentImp.Video.W), *(currentImp.Video.H))] = currentFloor
			}
		}
		currentMapFloors["*"] = currentFloor
	}

	if currentImp.Banner != nil {
		ok = true
		var banner mediaTypeBanner
		json.Unmarshal(currentImp.Banner.Ext, &banner)

		msqParams.Mediatypes.Banner = &banner
		switch {
		case len(currentImp.Banner.Format) > 0:
			for _, bannerFormat := range currentImp.Banner.Format {
				currentMapFloors[fmt.Sprintf("%dx%d", bannerFormat.W, bannerFormat.H)] = currentFloor
				msqParams.Mediatypes.Banner.Sizes = append(msqParams.Mediatypes.Banner.Sizes,
					[]*int{intToPtrInt(int(bannerFormat.W)), intToPtrInt(int(bannerFormat.H))})
			}
		case currentImp.Banner.W != nil && currentImp.Banner.H != nil:
			currentMapFloors[fmt.Sprintf("%dx%d", *(currentImp.Banner.W), *(currentImp.Banner.H))] = currentFloor
			msqParams.Mediatypes.Banner.Sizes = append(msqParams.Mediatypes.Banner.Sizes,
				[]*int{intToPtrInt(int(*currentImp.Banner.W)), intToPtrInt(int(*currentImp.Banner.H))})
		}

		if msqParams.Mediatypes.Banner != nil {
			for _, bannerSizes := range msqParams.Mediatypes.Banner.Sizes {
				if len(bannerSizes) == 2 && bannerSizes[0] != nil && bannerSizes[1] != nil {
					currentMapFloors[fmt.Sprintf("%dx%d", *(bannerSizes[0]), *(bannerSizes[1]))] = currentFloor
				}
			}
		}
	}

	if currentImp.Native != nil {
		ok = true
		var native = mediaTypeNative{Type: "native"}
		json.Unmarshal(currentImp.Native.Ext, &native)

		msqParams.Mediatypes.Native = &native
		for _, nativeSizes := range msqParams.Mediatypes.Native.Sizes {
			if len(nativeSizes) == 2 {
				currentMapFloors[fmt.Sprintf("%dx%d", nativeSizes[0], nativeSizes[1])] = currentFloor
			}
		}
		currentMapFloors["*"] = currentFloor
	}

	if len(currentMapFloors) > 0 {
		msqParams.Floor = currentMapFloors
	}
	return
}

// getContent: Loads msqResp content into the bidderResponse (*adapters.BidderResponse).
func (msqResp *msqResponse) getContent(bidderResponse *adapters.BidderResponse) {
	var tmpBids []*adapters.TypedBid
	for _, resp := range msqResp.Responses {
		tmpTBid := adapters.TypedBid{
			BidType: resp.bidType(),
			Bid: &openrtb2.Bid{
				ID:      resp.ID,
				ImpID:   resp.BidId,
				Price:   resp.Cpm,
				AdM:     resp.Ad,
				ADomain: resp.ADomain,
				W:       resp.Width,
				H:       resp.Height,
				CrID:    resp.CreativeId,
				MType:   resp.mType(),
				BURL:    resp.BURL,
				Ext:     resp.extBid(),
			},
			BidMeta: resp.extBidPrebidMeta(),
		}
		tmpBids = append(tmpBids, &tmpTBid)
		bidderResponse.Currency = resp.Currency
	}

	if len(tmpBids) > 0 {
		bidderResponse.Bids = tmpBids
	}
}
