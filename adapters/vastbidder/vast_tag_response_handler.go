package vastbidder

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/beevik/etree"
	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

var durationRegExp = regexp.MustCompile(`^([01]?\d|2[0-3]):([0-5]?\d):([0-5]?\d)(\.(\d{1,3}))?$`)

//IVASTTagResponseHandler to parse VAST Tag
type IVASTTagResponseHandler interface {
	ITagResponseHandler
	ParseExtension(version string, tag *etree.Element, bid *adapters.TypedBid) []error
	GetStaticPrice(ext json.RawMessage) float64
}

//VASTTagResponseHandler to parse VAST Tag
type VASTTagResponseHandler struct {
	IVASTTagResponseHandler
	ImpBidderExt *openrtb_ext.ExtImpVASTBidder
	VASTTag      *openrtb_ext.ExtImpVASTBidderTag
}

//NewVASTTagResponseHandler returns new object
func NewVASTTagResponseHandler() *VASTTagResponseHandler {
	obj := &VASTTagResponseHandler{}
	obj.IVASTTagResponseHandler = obj
	return obj
}

//Validate will return bids
func (handler *VASTTagResponseHandler) Validate(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) []error {
	if response.StatusCode != http.StatusOK {
		return []error{errors.New(`validation failed`)}
	}

	if len(internalRequest.Imp) < externalRequest.Params.ImpIndex {
		return []error{errors.New(`validation failed invalid impression index`)}
	}

	impExt, err := readImpExt(internalRequest.Imp[externalRequest.Params.ImpIndex].Ext)
	if nil != err {
		return []error{err}
	}

	if len(impExt.Tags) < externalRequest.Params.VASTTagIndex {
		return []error{errors.New(`validation failed invalid vast tag index`)}
	}

	//Initialise Extensions
	handler.ImpBidderExt = impExt
	handler.VASTTag = impExt.Tags[externalRequest.Params.VASTTagIndex]
	return nil
}

//MakeBids will return bids
func (handler *VASTTagResponseHandler) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if err := handler.IVASTTagResponseHandler.Validate(internalRequest, externalRequest, response); len(err) > 0 {
		return nil, err[:]
	}

	bidResponses, err := handler.vastTagToBidderResponse(internalRequest, externalRequest, response)
	return bidResponses, err
}

//ParseExtension will parse VAST XML extension object
func (handler *VASTTagResponseHandler) ParseExtension(version string, ad *etree.Element, bid *adapters.TypedBid) []error {
	return nil
}

func (handler *VASTTagResponseHandler) vastTagToBidderResponse(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var errs []error

	doc := etree.NewDocument()

	//Read Document
	if err := doc.ReadFromBytes(response.Body); err != nil {
		errs = append(errs, err)
		return nil, errs[:]
	}

	//Check VAST Tag
	vast := doc.Element.FindElement(`./VAST`)
	if vast == nil {
		errs = append(errs, errors.New("VAST Tag Not Found"))
		return nil, errs[:]
	}

	//Check VAST/Ad Tag
	adElement := getAdElement(vast)
	if nil == adElement {
		errs = append(errs, errors.New("VAST/Ad Tag Not Found"))
		return nil, errs[:]
	}

	typedBid := &adapters.TypedBid{
		Bid:     &openrtb2.Bid{},
		BidType: openrtb_ext.BidTypeVideo,
		BidVideo: &openrtb_ext.ExtBidPrebidVideo{
			VASTTagID: handler.VASTTag.TagID,
		},
	}

	creatives := adElement.FindElements("Creatives/Creative")
	if nil != creatives {
		for _, creative := range creatives {
			// get creative id
			typedBid.Bid.CrID = getCreativeID(creative)

			// get duration from vast creative
			dur, err := getDuration(creative)
			if nil != err {
				// get duration from input bidder vast tag
				dur = getStaticDuration(handler.VASTTag)
			}
			if dur > 0 {
				typedBid.BidVideo.Duration = int(dur) // prebid expects int value
			}
		}
	}

	bidResponse := &adapters.BidderResponse{
		Bids:     []*adapters.TypedBid{typedBid},
		Currency: `USD`, //TODO: Need to check how to get currency value
	}

	//GetVersion
	version := vast.SelectAttrValue(`version`, `2.0`)

	if err := handler.IVASTTagResponseHandler.ParseExtension(version, adElement, typedBid); len(err) > 0 {
		errs = append(errs, err...)
		return nil, errs[:]
	}

	//if bid.price is not set in ParseExtension
	if typedBid.Bid.Price <= 0 {
		price, currency := getPricingDetails(version, adElement)
		if price <= 0 {
			price, currency = getStaticPricingDetails(handler.VASTTag)
			if price <= 0 {
				errs = append(errs, &errortypes.NoBidPrice{Message: "Bid Price Not Present"})
				return nil, errs[:]
			}
		}
		typedBid.Bid.Price = price
		if len(currency) > 0 {
			bidResponse.Currency = currency
		}
	}

	typedBid.Bid.ADomain = getAdvertisers(version, adElement)

	//if bid.id is not set in ParseExtension
	if len(typedBid.Bid.ID) == 0 {
		typedBid.Bid.ID = GetRandomID()
	}

	//if bid.impid is not set in ParseExtension
	if len(typedBid.Bid.ImpID) == 0 {
		typedBid.Bid.ImpID = internalRequest.Imp[externalRequest.Params.ImpIndex].ID
	}

	//if bid.adm is not set in ParseExtension
	if len(typedBid.Bid.AdM) == 0 {
		typedBid.Bid.AdM = string(response.Body)
	}

	//if bid.CrID is not set in ParseExtension
	if len(typedBid.Bid.CrID) == 0 {
		typedBid.Bid.CrID = "cr_" + GetRandomID()
	}

	return bidResponse, nil
}

func getAdElement(vast *etree.Element) *etree.Element {
	if ad := vast.FindElement(`./Ad/Wrapper`); nil != ad {
		return ad
	}
	if ad := vast.FindElement(`./Ad/InLine`); nil != ad {
		return ad
	}
	return nil
}

func getAdvertisers(vastVer string, ad *etree.Element) []string {
	version, err := strconv.ParseFloat(vastVer, 64)
	if err != nil {
		version = 2.0
	}

	advertisers := make([]string, 0)

	switch int(version) {
	case 2, 3:
		for _, ext := range ad.FindElements(`./Extensions/Extension/`) {
			for _, attr := range ext.Attr {
				if attr.Key == "type" && attr.Value == "advertiser" {
					for _, ele := range ext.ChildElements() {
						if ele.Tag == "Advertiser" {
							if strings.TrimSpace(ele.Text()) != "" {
								advertisers = append(advertisers, ele.Text())
							}
						}
					}
				}
			}
		}
	case 4:
		if ad.FindElement("./Advertiser") != nil {
			adv := strings.TrimSpace(ad.FindElement("./Advertiser").Text())
			if adv != "" {
				advertisers = append(advertisers, adv)
			}
		}
	default:
		glog.V(3).Infof("Handle getAdvertisers for VAST version %d", int(version))
	}

	if len(advertisers) == 0 {
		return nil
	}
	return advertisers
}

func getStaticPricingDetails(vastTag *openrtb_ext.ExtImpVASTBidderTag) (float64, string) {
	if nil == vastTag {
		return 0.0, ""
	}
	return vastTag.Price, "USD"
}

func getPricingDetails(version string, ad *etree.Element) (float64, string) {
	var currency string
	var node *etree.Element

	if version == `2.0` {
		node = ad.FindElement(`./Extensions/Extension/Price`)
	} else {
		node = ad.FindElement(`./Pricing`)
	}

	if node == nil {
		return 0.0, currency
	}

	priceValue, err := strconv.ParseFloat(node.Text(), 64)
	if nil != err {
		return 0.0, currency
	}

	currencyNode := node.SelectAttr(`currency`)
	if nil != currencyNode {
		currency = currencyNode.Value
	}

	return priceValue, currency
}

// getDuration extracts the duration of the bid from input creative of Linear type.
// The lookup may vary from vast version provided in the input
// returns duration in seconds or error if failed to obtained the duration.
// If multple Linear tags are present, onlyfirst one will be used
//
// It will lookup for duration only in case of creative type is Linear.
// If creative type other than Linear then this function will return error
// For Linear Creative it will lookup for Duration attribute.Duration value will be in hh:mm:ss.mmm format as per VAST specifications
// If Duration attribute not present this will return error
//
// After extracing the duration it will convert it into seconds
//
// The ad server uses the <Duration> element to denote
// the intended playback duration for the video or audio component of the ad.
// Time value may be in the format HH:MM:SS.mmm where .mmm indicates milliseconds.
// Providing milliseconds is optional.
//
// Reference
// 1.https://iabtechlab.com/wp-content/uploads/2019/06/VAST_4.2_final_june26.pdf
// 2.https://iabtechlab.com/wp-content/uploads/2018/11/VAST4.1-final-Nov-8-2018.pdf
// 3.https://iabtechlab.com/wp-content/uploads/2016/05/VAST4.0_Updated_April_2016.pdf
// 4.https://iabtechlab.com/wp-content/uploads/2016/04/VASTv3_0.pdf
func getDuration(creative *etree.Element) (int, error) {
	if nil == creative {
		return 0, errors.New("Invalid Creative")
	}
	node := creative.FindElement("./Linear/Duration")
	if nil == node {
		return 0, errors.New("Invalid Duration")
	}
	duration := node.Text()
	// check if milliseconds is provided
	match := durationRegExp.FindStringSubmatch(duration)
	if nil == match {
		return 0, errors.New("Invalid Duration")
	}
	repl := "${1}h${2}m${3}s"
	ms := match[5]
	if "" != ms {
		repl += "${5}ms"
	}
	duration = durationRegExp.ReplaceAllString(duration, repl)
	dur, err := time.ParseDuration(duration)
	if err != nil {
		return 0, err
	}
	return int(dur.Seconds()), nil
}

func getStaticDuration(vastTag *openrtb_ext.ExtImpVASTBidderTag) int {
	if nil == vastTag {
		return 0
	}
	return vastTag.Duration
}

//getCreativeID looks for ID inside input creative tag
func getCreativeID(creative *etree.Element) string {
	if nil == creative {
		return ""
	}
	return creative.SelectAttrValue("id", "")
}
