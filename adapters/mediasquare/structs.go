package mediasquare

import (
	"fmt"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
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
	Codes   []msqParametersCodes `json:"codes"`
	Gdpr    msqParametersGdpr    `json:"gdpr"`
	Type    string               `json:"type"`
	DSA     interface{}          `json:"dsa,omitempty"`
	Support msqSupport           `json:"tech"`
	Test    bool                 `json:"test"`
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

type msqParametersGdpr struct {
	ConsentRequired bool   `json:"consent_required"`
	ConsentString   string `json:"consent_string"`
}

type msqFloor struct {
	Price    float64 `json:"floor,omitempty"`
	Currency string  `json:"currency,omitempty"`
}

type mediaTypes struct {
	Banner        *mediaTypeBanner `json:"banner,omitempty"`
	Video         *openrtb2.Video  `json:"video,omitempty"`
	NativeRequest *string          `json:"native_request,omitempty"`
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
	msqParams.Gdpr = msqParametersGdpr{
		ConsentRequired: (parserGDPR{}).getValue("consent_requirement", request) == "true",
		ConsentString:   (parserGDPR{}).getValue("consent_string", request),
	}
	msqParams.DSA = (parserDSA{}).getValue(request)

	return
}

// setContent: Loads currentImp into msqParams (*msqParametersCodes),
// returns (ok bool) where `ok` express if mandatory content had been loaded.
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
		msqParams.Mediatypes.Video = currentImp.Video
		if currentImp.Video.W != nil && currentImp.Video.H != nil {
			currentMapFloors[fmt.Sprintf("%dx%d", *(currentImp.Video.W), *(currentImp.Video.H))] = currentFloor
		}
		currentMapFloors["*"] = currentFloor
	}

	if currentImp.Banner != nil {
		switch {
		case len(currentImp.Banner.Format) > 0:
			ok = true
			msqParams.Mediatypes.Banner = new(mediaTypeBanner)
			for _, bannerFormat := range currentImp.Banner.Format {
				currentMapFloors[fmt.Sprintf("%dx%d", bannerFormat.W, bannerFormat.H)] = currentFloor
				msqParams.Mediatypes.Banner.Sizes = append(msqParams.Mediatypes.Banner.Sizes,
					[]*int{ptrutil.ToPtr(int(bannerFormat.W)), ptrutil.ToPtr(int(bannerFormat.H))})
			}
		case currentImp.Banner.W != nil && currentImp.Banner.H != nil:
			ok = true
			msqParams.Mediatypes.Banner = new(mediaTypeBanner)
			currentMapFloors[fmt.Sprintf("%dx%d", *(currentImp.Banner.W), *(currentImp.Banner.H))] = currentFloor
			msqParams.Mediatypes.Banner.Sizes = append(msqParams.Mediatypes.Banner.Sizes,
				[]*int{ptrutil.ToPtr(int(*currentImp.Banner.W)), ptrutil.ToPtr(int(*currentImp.Banner.H))})
		}

		if msqParams.Mediatypes.Banner != nil {
			for _, bannerSizes := range msqParams.Mediatypes.Banner.Sizes {
				if len(bannerSizes) == 2 && bannerSizes[0] != nil && bannerSizes[1] != nil {
					currentMapFloors[fmt.Sprintf("%dx%d", *(bannerSizes[0]), *(bannerSizes[1]))] = currentFloor
				}
			}
		}
	}

	if currentImp.Native != nil && len(currentImp.Native.Request) > 0 {
		ok = true
		msqParams.Mediatypes.NativeRequest = ptrutil.ToPtr(currentImp.Native.Request)
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
