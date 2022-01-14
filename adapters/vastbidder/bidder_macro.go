package vastbidder

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

//BidderMacro default implementation
type BidderMacro struct {
	IBidderMacro

	//Configuration Parameters
	Conf *config.Adapter

	//OpenRTB Specific Parameters
	Request   *openrtb2.BidRequest
	IsApp     bool
	HasGeo    bool
	Imp       *openrtb2.Imp
	Publisher *openrtb2.Publisher
	Content   *openrtb2.Content

	//Extensions
	ImpBidderExt openrtb_ext.ExtImpVASTBidder
	VASTTag      *openrtb_ext.ExtImpVASTBidderTag
	UserExt      *openrtb_ext.ExtUser
	RegsExt      *openrtb_ext.ExtRegs

	//Impression level Request Headers
	ImpReqHeaders http.Header
}

//NewBidderMacro contains definition for all openrtb macro's
func NewBidderMacro() IBidderMacro {
	obj := &BidderMacro{}
	obj.IBidderMacro = obj
	return obj
}

func (tag *BidderMacro) init() {
	if nil != tag.Request.App {
		tag.IsApp = true
		tag.Publisher = tag.Request.App.Publisher
		tag.Content = tag.Request.App.Content
	} else {
		tag.Publisher = tag.Request.Site.Publisher
		tag.Content = tag.Request.Site.Content
	}
	tag.HasGeo = nil != tag.Request.Device && nil != tag.Request.Device.Geo

	//Read User Extensions
	if nil != tag.Request.User && nil != tag.Request.User.Ext {
		var ext openrtb_ext.ExtUser
		err := json.Unmarshal(tag.Request.User.Ext, &ext)
		if nil == err {
			tag.UserExt = &ext
		}
	}

	//Read Regs Extensions
	if nil != tag.Request.Regs && nil != tag.Request.Regs.Ext {
		var ext openrtb_ext.ExtRegs
		err := json.Unmarshal(tag.Request.Regs.Ext, &ext)
		if nil == err {
			tag.RegsExt = &ext
		}
	}
}

//InitBidRequest will initialise BidRequest
func (tag *BidderMacro) InitBidRequest(request *openrtb2.BidRequest) {
	tag.Request = request
	tag.init()
}

//LoadImpression will set current imp
func (tag *BidderMacro) LoadImpression(imp *openrtb2.Imp) (*openrtb_ext.ExtImpVASTBidder, error) {
	tag.Imp = imp

	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, err
	}

	tag.ImpBidderExt = openrtb_ext.ExtImpVASTBidder{}
	if err := json.Unmarshal(bidderExt.Bidder, &tag.ImpBidderExt); err != nil {
		return nil, err
	}
	return &tag.ImpBidderExt, nil
}

//LoadVASTTag will set current VAST Tag details in bidder keys
func (tag *BidderMacro) LoadVASTTag(vastTag *openrtb_ext.ExtImpVASTBidderTag) {
	tag.VASTTag = vastTag
}

//GetBidderKeys will set bidder level keys
func (tag *BidderMacro) GetBidderKeys() map[string]string {
	//Adding VAST Tag Bidder Parameters
	keys := NormalizeJSON(tag.VASTTag.Params)

	//Adding VAST Tag Standard Params
	keys["dur"] = strconv.Itoa(tag.VASTTag.Duration)

	//Adding Headers as Custom Macros

	//Adding Cookies as Custom Macros

	//Adding Default Empty for standard keys
	for i := range ParamKeys {
		if _, ok := keys[ParamKeys[i]]; !ok {
			keys[ParamKeys[i]] = ""
		}
	}
	return keys
}

//SetAdapterConfig will set Adapter config
func (tag *BidderMacro) SetAdapterConfig(conf *config.Adapter) {
	tag.Conf = conf
}

//GetURI get URL
func (tag *BidderMacro) GetURI() string {

	//check for URI at impression level
	if nil != tag.VASTTag {
		return tag.VASTTag.URL
	}

	//check for URI at config level
	return tag.Conf.Endpoint
}

//GetHeaders returns list of custom request headers
//Override this method if your Vast bidder needs custom  request headers
func (tag *BidderMacro) GetHeaders() http.Header {
	return http.Header{}
}

/********************* Request *********************/

//MacroTest contains definition for Test Parameter
func (tag *BidderMacro) MacroTest(key string) string {
	if tag.Request.Test > 0 {
		return strconv.Itoa(int(tag.Request.Test))
	}
	return ""
}

//MacroTimeout contains definition for Timeout Parameter
func (tag *BidderMacro) MacroTimeout(key string) string {
	if tag.Request.TMax > 0 {
		return strconv.FormatInt(tag.Request.TMax, intBase)
	}
	return ""
}

//MacroWhitelistSeat contains definition for WhitelistSeat Parameter
func (tag *BidderMacro) MacroWhitelistSeat(key string) string {
	return strings.Join(tag.Request.WSeat, comma)
}

//MacroWhitelistLang contains definition for WhitelistLang Parameter
func (tag *BidderMacro) MacroWhitelistLang(key string) string {
	return strings.Join(tag.Request.WLang, comma)
}

//MacroBlockedSeat contains definition for Blockedseat Parameter
func (tag *BidderMacro) MacroBlockedSeat(key string) string {
	return strings.Join(tag.Request.BSeat, comma)
}

//MacroCurrency contains definition for Currency Parameter
func (tag *BidderMacro) MacroCurrency(key string) string {
	return strings.Join(tag.Request.Cur, comma)
}

//MacroBlockedCategory contains definition for BlockedCategory Parameter
func (tag *BidderMacro) MacroBlockedCategory(key string) string {
	return strings.Join(tag.Request.BCat, comma)
}

//MacroBlockedAdvertiser contains definition for BlockedAdvertiser Parameter
func (tag *BidderMacro) MacroBlockedAdvertiser(key string) string {
	return strings.Join(tag.Request.BAdv, comma)
}

//MacroBlockedApp contains definition for BlockedApp Parameter
func (tag *BidderMacro) MacroBlockedApp(key string) string {
	return strings.Join(tag.Request.BApp, comma)
}

/********************* Source *********************/

//MacroFD contains definition for FD Parameter
func (tag *BidderMacro) MacroFD(key string) string {
	if nil != tag.Request.Source {
		return strconv.Itoa(int(tag.Request.Source.FD))
	}
	return ""
}

//MacroTransactionID contains definition for TransactionID Parameter
func (tag *BidderMacro) MacroTransactionID(key string) string {
	if nil != tag.Request.Source {
		return tag.Request.Source.TID
	}
	return ""
}

//MacroPaymentIDChain contains definition for PaymentIDChain Parameter
func (tag *BidderMacro) MacroPaymentIDChain(key string) string {
	if nil != tag.Request.Source {
		return tag.Request.Source.PChain
	}
	return ""
}

/********************* Regs *********************/

//MacroCoppa contains definition for Coppa Parameter
func (tag *BidderMacro) MacroCoppa(key string) string {
	if nil != tag.Request.Regs {
		return strconv.Itoa(int(tag.Request.Regs.COPPA))
	}
	return ""
}

/********************* Impression *********************/

//MacroDisplayManager contains definition for DisplayManager Parameter
func (tag *BidderMacro) MacroDisplayManager(key string) string {
	return tag.Imp.DisplayManager
}

//MacroDisplayManagerVersion contains definition for DisplayManagerVersion Parameter
func (tag *BidderMacro) MacroDisplayManagerVersion(key string) string {
	return tag.Imp.DisplayManagerVer
}

//MacroInterstitial contains definition for Interstitial Parameter
func (tag *BidderMacro) MacroInterstitial(key string) string {
	if tag.Imp.Instl > 0 {
		return strconv.Itoa(int(tag.Imp.Instl))
	}
	return ""
}

//MacroTagID contains definition for TagID Parameter
func (tag *BidderMacro) MacroTagID(key string) string {
	return tag.Imp.TagID
}

//MacroBidFloor contains definition for BidFloor Parameter
func (tag *BidderMacro) MacroBidFloor(key string) string {
	if tag.Imp.BidFloor > 0 {
		return fmt.Sprintf("%g", tag.Imp.BidFloor)
	}
	return ""
}

//MacroBidFloorCurrency contains definition for BidFloorCurrency Parameter
func (tag *BidderMacro) MacroBidFloorCurrency(key string) string {
	return tag.Imp.BidFloorCur
}

//MacroSecure contains definition for Secure Parameter
func (tag *BidderMacro) MacroSecure(key string) string {
	if nil != tag.Imp.Secure {
		return strconv.Itoa(int(*tag.Imp.Secure))
	}
	return ""
}

//MacroPMP contains definition for PMP Parameter
func (tag *BidderMacro) MacroPMP(key string) string {
	if nil != tag.Imp.PMP {
		data, _ := json.Marshal(tag.Imp.PMP)
		return string(data)
	}
	return ""
}

/********************* Video *********************/

//MacroVideoMIMES contains definition for VideoMIMES Parameter
func (tag *BidderMacro) MacroVideoMIMES(key string) string {
	if nil != tag.Imp.Video {
		return strings.Join(tag.Imp.Video.MIMEs, comma)
	}
	return ""
}

//MacroVideoMinimumDuration contains definition for VideoMinimumDuration Parameter
func (tag *BidderMacro) MacroVideoMinimumDuration(key string) string {
	if nil != tag.Imp.Video && tag.Imp.Video.MinDuration > 0 {
		return strconv.FormatInt(tag.Imp.Video.MinDuration, intBase)
	}
	return ""
}

//MacroVideoMaximumDuration contains definition for VideoMaximumDuration Parameter
func (tag *BidderMacro) MacroVideoMaximumDuration(key string) string {
	if nil != tag.Imp.Video && tag.Imp.Video.MaxDuration > 0 {
		return strconv.FormatInt(tag.Imp.Video.MaxDuration, intBase)
	}
	return ""
}

//MacroVideoProtocols contains definition for VideoProtocols Parameter
func (tag *BidderMacro) MacroVideoProtocols(key string) string {
	if nil != tag.Imp.Video {
		value := tag.Imp.Video.Protocols
		return ObjectArrayToString(len(value), comma, func(i int) string {
			return strconv.FormatInt(int64(value[i]), intBase)
		})
	}
	return ""
}

//MacroVideoPlayerWidth contains definition for VideoPlayerWidth Parameter
func (tag *BidderMacro) MacroVideoPlayerWidth(key string) string {
	if nil != tag.Imp.Video && tag.Imp.Video.W > 0 {
		return strconv.FormatInt(int64(tag.Imp.Video.W), intBase)
	}
	return ""
}

//MacroVideoPlayerHeight contains definition for VideoPlayerHeight Parameter
func (tag *BidderMacro) MacroVideoPlayerHeight(key string) string {
	if nil != tag.Imp.Video && tag.Imp.Video.H > 0 {
		return strconv.FormatInt(int64(tag.Imp.Video.H), intBase)
	}
	return ""
}

//MacroVideoStartDelay contains definition for VideoStartDelay Parameter
func (tag *BidderMacro) MacroVideoStartDelay(key string) string {
	if nil != tag.Imp.Video && nil != tag.Imp.Video.StartDelay {
		return strconv.FormatInt(int64(*tag.Imp.Video.StartDelay), intBase)
	}
	return ""
}

//MacroVideoPlacement contains definition for VideoPlacement Parameter
func (tag *BidderMacro) MacroVideoPlacement(key string) string {
	if nil != tag.Imp.Video && tag.Imp.Video.Placement > 0 {
		return strconv.FormatInt(int64(tag.Imp.Video.Placement), intBase)
	}
	return ""
}

//MacroVideoLinearity contains definition for VideoLinearity Parameter
func (tag *BidderMacro) MacroVideoLinearity(key string) string {
	if nil != tag.Imp.Video && tag.Imp.Video.Linearity > 0 {
		return strconv.FormatInt(int64(tag.Imp.Video.Linearity), intBase)
	}
	return ""
}

//MacroVideoSkip contains definition for VideoSkip Parameter
func (tag *BidderMacro) MacroVideoSkip(key string) string {
	if nil != tag.Imp.Video && nil != tag.Imp.Video.Skip {
		return strconv.FormatInt(int64(*tag.Imp.Video.Skip), intBase)
	}
	return ""
}

//MacroVideoSkipMinimum contains definition for VideoSkipMinimum Parameter
func (tag *BidderMacro) MacroVideoSkipMinimum(key string) string {
	if nil != tag.Imp.Video && tag.Imp.Video.SkipMin > 0 {
		return strconv.FormatInt(tag.Imp.Video.SkipMin, intBase)
	}
	return ""
}

//MacroVideoSkipAfter contains definition for VideoSkipAfter Parameter
func (tag *BidderMacro) MacroVideoSkipAfter(key string) string {
	if nil != tag.Imp.Video && tag.Imp.Video.SkipAfter > 0 {
		return strconv.FormatInt(tag.Imp.Video.SkipAfter, intBase)
	}
	return ""
}

//MacroVideoSequence contains definition for VideoSequence Parameter
func (tag *BidderMacro) MacroVideoSequence(key string) string {
	if nil != tag.Imp.Video && tag.Imp.Video.Sequence > 0 {
		return strconv.FormatInt(int64(tag.Imp.Video.Sequence), intBase)
	}
	return ""
}

//MacroVideoBlockedAttribute contains definition for VideoBlockedAttribute Parameter
func (tag *BidderMacro) MacroVideoBlockedAttribute(key string) string {
	if nil != tag.Imp.Video {
		value := tag.Imp.Video.BAttr
		return ObjectArrayToString(len(value), comma, func(i int) string {
			return strconv.FormatInt(int64(value[i]), intBase)
		})
	}
	return ""
}

//MacroVideoMaximumExtended contains definition for VideoMaximumExtended Parameter
func (tag *BidderMacro) MacroVideoMaximumExtended(key string) string {
	if nil != tag.Imp.Video && tag.Imp.Video.MaxExtended > 0 {
		return strconv.FormatInt(tag.Imp.Video.MaxExtended, intBase)
	}
	return ""
}

//MacroVideoMinimumBitRate contains definition for VideoMinimumBitRate Parameter
func (tag *BidderMacro) MacroVideoMinimumBitRate(key string) string {
	if nil != tag.Imp.Video && tag.Imp.Video.MinBitRate > 0 {
		return strconv.FormatInt(int64(tag.Imp.Video.MinBitRate), intBase)
	}
	return ""
}

//MacroVideoMaximumBitRate contains definition for VideoMaximumBitRate Parameter
func (tag *BidderMacro) MacroVideoMaximumBitRate(key string) string {
	if nil != tag.Imp.Video && tag.Imp.Video.MaxBitRate > 0 {
		return strconv.FormatInt(int64(tag.Imp.Video.MaxBitRate), intBase)
	}
	return ""
}

//MacroVideoBoxing contains definition for VideoBoxing Parameter
func (tag *BidderMacro) MacroVideoBoxing(key string) string {
	if nil != tag.Imp.Video && tag.Imp.Video.BoxingAllowed > 0 {
		return strconv.FormatInt(int64(tag.Imp.Video.BoxingAllowed), intBase)
	}
	return ""
}

//MacroVideoPlaybackMethod contains definition for VideoPlaybackMethod Parameter
func (tag *BidderMacro) MacroVideoPlaybackMethod(key string) string {
	if nil != tag.Imp.Video {
		value := tag.Imp.Video.PlaybackMethod
		return ObjectArrayToString(len(value), comma, func(i int) string {
			return strconv.FormatInt(int64(value[i]), intBase)
		})
	}
	return ""
}

//MacroVideoDelivery contains definition for VideoDelivery Parameter
func (tag *BidderMacro) MacroVideoDelivery(key string) string {
	if nil != tag.Imp.Video {
		value := tag.Imp.Video.Delivery
		return ObjectArrayToString(len(value), comma, func(i int) string {
			return strconv.FormatInt(int64(value[i]), intBase)
		})
	}
	return ""
}

//MacroVideoPosition contains definition for VideoPosition Parameter
func (tag *BidderMacro) MacroVideoPosition(key string) string {
	if nil != tag.Imp.Video && nil != tag.Imp.Video.Pos {
		return strconv.FormatInt(int64(*tag.Imp.Video.Pos), intBase)
	}
	return ""
}

//MacroVideoAPI contains definition for VideoAPI Parameter
func (tag *BidderMacro) MacroVideoAPI(key string) string {
	if nil != tag.Imp.Video {
		value := tag.Imp.Video.API
		return ObjectArrayToString(len(value), comma, func(i int) string {
			return strconv.FormatInt(int64(value[i]), intBase)
		})
	}
	return ""
}

/********************* Site *********************/

//MacroSiteID contains definition for SiteID Parameter
func (tag *BidderMacro) MacroSiteID(key string) string {
	if !tag.IsApp {
		return tag.Request.Site.ID
	}
	return ""
}

//MacroSiteName contains definition for SiteName Parameter
func (tag *BidderMacro) MacroSiteName(key string) string {
	if !tag.IsApp {
		return tag.Request.Site.Name
	}
	return ""
}

//MacroSitePage contains definition for SitePage Parameter
func (tag *BidderMacro) MacroSitePage(key string) string {
	if !tag.IsApp && nil != tag.Request && nil != tag.Request.Site {
		return tag.Request.Site.Page
	}
	return ""
}

//MacroSiteReferrer contains definition for SiteReferrer Parameter
func (tag *BidderMacro) MacroSiteReferrer(key string) string {
	if !tag.IsApp {
		return tag.Request.Site.Ref
	}
	return ""
}

//MacroSiteSearch contains definition for SiteSearch Parameter
func (tag *BidderMacro) MacroSiteSearch(key string) string {
	if !tag.IsApp {
		return tag.Request.Site.Search
	}
	return ""
}

//MacroSiteMobile contains definition for SiteMobile Parameter
func (tag *BidderMacro) MacroSiteMobile(key string) string {
	if !tag.IsApp && tag.Request.Site.Mobile > 0 {
		return strconv.FormatInt(int64(tag.Request.Site.Mobile), intBase)
	}
	return ""
}

/********************* App *********************/

//MacroAppID contains definition for AppID Parameter
func (tag *BidderMacro) MacroAppID(key string) string {
	if tag.IsApp {
		return tag.Request.App.ID
	}
	return ""
}

//MacroAppName contains definition for AppName Parameter
func (tag *BidderMacro) MacroAppName(key string) string {
	if tag.IsApp {
		return tag.Request.App.Name
	}
	return ""
}

//MacroAppBundle contains definition for AppBundle Parameter
func (tag *BidderMacro) MacroAppBundle(key string) string {
	if tag.IsApp {
		return tag.Request.App.Bundle
	}
	return ""
}

//MacroAppStoreURL contains definition for AppStoreURL Parameter
func (tag *BidderMacro) MacroAppStoreURL(key string) string {
	if tag.IsApp {
		return tag.Request.App.StoreURL
	}
	return ""
}

//MacroAppVersion contains definition for AppVersion Parameter
func (tag *BidderMacro) MacroAppVersion(key string) string {
	if tag.IsApp {
		return tag.Request.App.Ver
	}
	return ""
}

//MacroAppPaid contains definition for AppPaid Parameter
func (tag *BidderMacro) MacroAppPaid(key string) string {
	if tag.IsApp && tag.Request.App.Paid != 0 {
		return strconv.FormatInt(int64(tag.Request.App.Paid), intBase)
	}
	return ""
}

/********************* Site/App Common *********************/

//MacroCategory contains definition for Category Parameter
func (tag *BidderMacro) MacroCategory(key string) string {
	if tag.IsApp {
		return strings.Join(tag.Request.App.Cat, comma)
	}
	return strings.Join(tag.Request.Site.Cat, comma)
}

//MacroDomain contains definition for Domain Parameter
func (tag *BidderMacro) MacroDomain(key string) string {
	if tag.IsApp {
		return tag.Request.App.Domain
	}
	return tag.Request.Site.Domain
}

//MacroSectionCategory contains definition for SectionCategory Parameter
func (tag *BidderMacro) MacroSectionCategory(key string) string {
	if tag.IsApp {
		return strings.Join(tag.Request.App.SectionCat, comma)
	}
	return strings.Join(tag.Request.Site.SectionCat, comma)
}

//MacroPageCategory contains definition for PageCategory Parameter
func (tag *BidderMacro) MacroPageCategory(key string) string {
	if tag.IsApp {
		return strings.Join(tag.Request.App.PageCat, comma)
	}
	return strings.Join(tag.Request.Site.PageCat, comma)
}

//MacroPrivacyPolicy contains definition for PrivacyPolicy Parameter
func (tag *BidderMacro) MacroPrivacyPolicy(key string) string {
	var value int8 = 0
	if tag.IsApp {
		value = tag.Request.App.PrivacyPolicy
	} else {
		value = tag.Request.Site.PrivacyPolicy
	}
	if value > 0 {
		return strconv.FormatInt(int64(value), intBase)
	}
	return ""
}

//MacroKeywords contains definition for Keywords Parameter
func (tag *BidderMacro) MacroKeywords(key string) string {
	if tag.IsApp {
		return tag.Request.App.Keywords
	}
	return tag.Request.Site.Keywords
}

/********************* Publisher *********************/

//MacroPubID contains definition for PubID Parameter
func (tag *BidderMacro) MacroPubID(key string) string {
	if nil != tag.Publisher {
		return tag.Publisher.ID
	}
	return ""
}

//MacroPubName contains definition for PubName Parameter
func (tag *BidderMacro) MacroPubName(key string) string {
	if nil != tag.Publisher {
		return tag.Publisher.Name
	}
	return ""
}

//MacroPubDomain contains definition for PubDomain Parameter
func (tag *BidderMacro) MacroPubDomain(key string) string {
	if nil != tag.Publisher {
		return tag.Publisher.Domain
	}
	return ""
}

/********************* Content *********************/

//MacroContentID contains definition for ContentID Parameter
func (tag *BidderMacro) MacroContentID(key string) string {
	if nil != tag.Content {
		return tag.Content.ID
	}
	return ""
}

//MacroContentEpisode contains definition for ContentEpisode Parameter
func (tag *BidderMacro) MacroContentEpisode(key string) string {
	if nil != tag.Content {
		return strconv.FormatInt(int64(tag.Content.Episode), intBase)
	}
	return ""
}

//MacroContentTitle contains definition for ContentTitle Parameter
func (tag *BidderMacro) MacroContentTitle(key string) string {
	if nil != tag.Content {
		return tag.Content.Title
	}
	return ""
}

//MacroContentSeries contains definition for ContentSeries Parameter
func (tag *BidderMacro) MacroContentSeries(key string) string {
	if nil != tag.Content {
		return tag.Content.Series
	}
	return ""
}

//MacroContentSeason contains definition for ContentSeason Parameter
func (tag *BidderMacro) MacroContentSeason(key string) string {
	if nil != tag.Content {
		return tag.Content.Season
	}
	return ""
}

//MacroContentArtist contains definition for ContentArtist Parameter
func (tag *BidderMacro) MacroContentArtist(key string) string {
	if nil != tag.Content {
		return tag.Content.Artist
	}
	return ""
}

//MacroContentGenre contains definition for ContentGenre Parameter
func (tag *BidderMacro) MacroContentGenre(key string) string {
	if nil != tag.Content {
		return tag.Content.Genre
	}
	return ""
}

//MacroContentAlbum contains definition for ContentAlbum Parameter
func (tag *BidderMacro) MacroContentAlbum(key string) string {
	if nil != tag.Content {
		return tag.Content.Album
	}
	return ""
}

//MacroContentISrc contains definition for ContentISrc Parameter
func (tag *BidderMacro) MacroContentISrc(key string) string {
	if nil != tag.Content {
		return tag.Content.ISRC
	}
	return ""
}

//MacroContentURL contains definition for ContentURL Parameter
func (tag *BidderMacro) MacroContentURL(key string) string {
	if nil != tag.Content {
		return tag.Content.URL
	}
	return ""
}

//MacroContentCategory contains definition for ContentCategory Parameter
func (tag *BidderMacro) MacroContentCategory(key string) string {
	if nil != tag.Content {
		return strings.Join(tag.Content.Cat, comma)
	}
	return ""
}

//MacroContentProductionQuality contains definition for ContentProductionQuality Parameter
func (tag *BidderMacro) MacroContentProductionQuality(key string) string {
	if nil != tag.Content && nil != tag.Content.ProdQ {
		return strconv.FormatInt(int64(*tag.Content.ProdQ), intBase)
	}
	return ""
}

//MacroContentVideoQuality contains definition for ContentVideoQuality Parameter
func (tag *BidderMacro) MacroContentVideoQuality(key string) string {
	if nil != tag.Content && nil != tag.Content.VideoQuality {
		return strconv.FormatInt(int64(*tag.Content.VideoQuality), intBase)
	}
	return ""
}

//MacroContentContext contains definition for ContentContext Parameter
func (tag *BidderMacro) MacroContentContext(key string) string {
	if nil != tag.Content && tag.Content.Context > 0 {
		return strconv.FormatInt(int64(tag.Content.Context), intBase)
	}
	return ""
}

//MacroContentContentRating contains definition for ContentContentRating Parameter
func (tag *BidderMacro) MacroContentContentRating(key string) string {
	if nil != tag.Content {
		return tag.Content.ContentRating
	}
	return ""
}

//MacroContentUserRating contains definition for ContentUserRating Parameter
func (tag *BidderMacro) MacroContentUserRating(key string) string {
	if nil != tag.Content {
		return tag.Content.UserRating
	}
	return ""
}

//MacroContentQAGMediaRating contains definition for ContentQAGMediaRating Parameter
func (tag *BidderMacro) MacroContentQAGMediaRating(key string) string {
	if nil != tag.Content && tag.Content.QAGMediaRating > 0 {
		return strconv.FormatInt(int64(tag.Content.QAGMediaRating), intBase)
	}
	return ""
}

//MacroContentKeywords contains definition for ContentKeywords Parameter
func (tag *BidderMacro) MacroContentKeywords(key string) string {
	if nil != tag.Content {
		return tag.Content.Keywords
	}
	return ""
}

//MacroContentLiveStream contains definition for ContentLiveStream Parameter
func (tag *BidderMacro) MacroContentLiveStream(key string) string {
	if nil != tag.Content {
		return strconv.FormatInt(int64(tag.Content.LiveStream), intBase)
	}
	return ""
}

//MacroContentSourceRelationship contains definition for ContentSourceRelationship Parameter
func (tag *BidderMacro) MacroContentSourceRelationship(key string) string {
	if nil != tag.Content {
		return strconv.FormatInt(int64(tag.Content.SourceRelationship), intBase)
	}
	return ""
}

//MacroContentLength contains definition for ContentLength Parameter
func (tag *BidderMacro) MacroContentLength(key string) string {
	if nil != tag.Content {
		return strconv.FormatInt(int64(tag.Content.Len), intBase)
	}
	return ""
}

//MacroContentLanguage contains definition for ContentLanguage Parameter
func (tag *BidderMacro) MacroContentLanguage(key string) string {
	if nil != tag.Content {
		return tag.Content.Language
	}
	return ""
}

//MacroContentEmbeddable contains definition for ContentEmbeddable Parameter
func (tag *BidderMacro) MacroContentEmbeddable(key string) string {
	if nil != tag.Content {
		return strconv.FormatInt(int64(tag.Content.Embeddable), intBase)
	}
	return ""
}

/********************* Producer *********************/

//MacroProducerID contains definition for ProducerID Parameter
func (tag *BidderMacro) MacroProducerID(key string) string {
	if nil != tag.Content && nil != tag.Content.Producer {
		return tag.Content.Producer.ID
	}
	return ""
}

//MacroProducerName contains definition for ProducerName Parameter
func (tag *BidderMacro) MacroProducerName(key string) string {
	if nil != tag.Content && nil != tag.Content.Producer {
		return tag.Content.Producer.Name
	}
	return ""
}

/********************* Device *********************/

//MacroUserAgent contains definition for UserAgent Parameter
func (tag *BidderMacro) MacroUserAgent(key string) string {
	if nil != tag.Request && nil != tag.Request.Device {
		return tag.Request.Device.UA
	}
	return ""
}

//MacroDNT contains definition for DNT Parameter
func (tag *BidderMacro) MacroDNT(key string) string {
	if nil != tag.Request.Device && nil != tag.Request.Device.DNT {
		return strconv.FormatInt(int64(*tag.Request.Device.DNT), intBase)
	}
	return ""
}

//MacroLMT contains definition for LMT Parameter
func (tag *BidderMacro) MacroLMT(key string) string {
	if nil != tag.Request.Device && nil != tag.Request.Device.Lmt {
		return strconv.FormatInt(int64(*tag.Request.Device.Lmt), intBase)
	}
	return ""
}

//MacroIP contains definition for IP Parameter
func (tag *BidderMacro) MacroIP(key string) string {
	if nil != tag.Request && nil != tag.Request.Device {
		if len(tag.Request.Device.IP) > 0 {
			return tag.Request.Device.IP
		} else if len(tag.Request.Device.IPv6) > 0 {
			return tag.Request.Device.IPv6
		}
	}
	return ""
}

//MacroDeviceType contains definition for DeviceType Parameter
func (tag *BidderMacro) MacroDeviceType(key string) string {
	if nil != tag.Request.Device && tag.Request.Device.DeviceType > 0 {
		return strconv.FormatInt(int64(tag.Request.Device.DeviceType), intBase)
	}
	return ""
}

//MacroMake contains definition for Make Parameter
func (tag *BidderMacro) MacroMake(key string) string {
	if nil != tag.Request.Device {
		return tag.Request.Device.Make
	}
	return ""
}

//MacroModel contains definition for Model Parameter
func (tag *BidderMacro) MacroModel(key string) string {
	if nil != tag.Request.Device {
		return tag.Request.Device.Model
	}
	return ""
}

//MacroDeviceOS contains definition for DeviceOS Parameter
func (tag *BidderMacro) MacroDeviceOS(key string) string {
	if nil != tag.Request.Device {
		return tag.Request.Device.OS
	}
	return ""
}

//MacroDeviceOSVersion contains definition for DeviceOSVersion Parameter
func (tag *BidderMacro) MacroDeviceOSVersion(key string) string {
	if nil != tag.Request.Device {
		return tag.Request.Device.OSV
	}
	return ""
}

//MacroDeviceWidth contains definition for DeviceWidth Parameter
func (tag *BidderMacro) MacroDeviceWidth(key string) string {
	if nil != tag.Request.Device {
		return strconv.FormatInt(int64(tag.Request.Device.W), intBase)
	}
	return ""
}

//MacroDeviceHeight contains definition for DeviceHeight Parameter
func (tag *BidderMacro) MacroDeviceHeight(key string) string {
	if nil != tag.Request.Device {
		return strconv.FormatInt(int64(tag.Request.Device.H), intBase)
	}
	return ""
}

//MacroDeviceJS contains definition for DeviceJS Parameter
func (tag *BidderMacro) MacroDeviceJS(key string) string {
	if nil != tag.Request.Device {
		return strconv.FormatInt(int64(tag.Request.Device.JS), intBase)
	}
	return ""
}

//MacroDeviceLanguage contains definition for DeviceLanguage Parameter
func (tag *BidderMacro) MacroDeviceLanguage(key string) string {
	if nil != tag.Request && nil != tag.Request.Device {
		return tag.Request.Device.Language
	}
	return ""
}

//MacroDeviceIFA contains definition for DeviceIFA Parameter
func (tag *BidderMacro) MacroDeviceIFA(key string) string {
	if nil != tag.Request.Device {
		return tag.Request.Device.IFA
	}
	return ""
}

//MacroDeviceDIDSHA1 contains definition for DeviceDIDSHA1 Parameter
func (tag *BidderMacro) MacroDeviceDIDSHA1(key string) string {
	if nil != tag.Request.Device {
		return tag.Request.Device.DIDSHA1
	}
	return ""
}

//MacroDeviceDIDMD5 contains definition for DeviceDIDMD5 Parameter
func (tag *BidderMacro) MacroDeviceDIDMD5(key string) string {
	if nil != tag.Request.Device {
		return tag.Request.Device.DIDMD5
	}
	return ""
}

//MacroDeviceDPIDSHA1 contains definition for DeviceDPIDSHA1 Parameter
func (tag *BidderMacro) MacroDeviceDPIDSHA1(key string) string {
	if nil != tag.Request.Device {
		return tag.Request.Device.DPIDSHA1
	}
	return ""
}

//MacroDeviceDPIDMD5 contains definition for DeviceDPIDMD5 Parameter
func (tag *BidderMacro) MacroDeviceDPIDMD5(key string) string {
	if nil != tag.Request.Device {
		return tag.Request.Device.DPIDMD5
	}
	return ""
}

//MacroDeviceMACSHA1 contains definition for DeviceMACSHA1 Parameter
func (tag *BidderMacro) MacroDeviceMACSHA1(key string) string {
	if nil != tag.Request.Device {
		return tag.Request.Device.MACSHA1
	}
	return ""
}

//MacroDeviceMACMD5 contains definition for DeviceMACMD5 Parameter
func (tag *BidderMacro) MacroDeviceMACMD5(key string) string {
	if nil != tag.Request.Device {
		return tag.Request.Device.MACMD5
	}
	return ""
}

/********************* Geo *********************/

//MacroLatitude contains definition for Latitude Parameter
func (tag *BidderMacro) MacroLatitude(key string) string {
	if tag.HasGeo {
		return fmt.Sprintf("%g", tag.Request.Device.Geo.Lat)
	}
	return ""
}

//MacroLongitude contains definition for Longitude Parameter
func (tag *BidderMacro) MacroLongitude(key string) string {
	if tag.HasGeo {
		return fmt.Sprintf("%g", tag.Request.Device.Geo.Lon)
	}
	return ""
}

//MacroCountry contains definition for Country Parameter
func (tag *BidderMacro) MacroCountry(key string) string {
	if tag.HasGeo {
		return tag.Request.Device.Geo.Country
	}
	return ""
}

//MacroRegion contains definition for Region Parameter
func (tag *BidderMacro) MacroRegion(key string) string {
	if tag.HasGeo {
		return tag.Request.Device.Geo.Region
	}
	return ""
}

//MacroCity contains definition for City Parameter
func (tag *BidderMacro) MacroCity(key string) string {
	if tag.HasGeo {
		return tag.Request.Device.Geo.City
	}
	return ""
}

//MacroZip contains definition for Zip Parameter
func (tag *BidderMacro) MacroZip(key string) string {
	if tag.HasGeo {
		return tag.Request.Device.Geo.ZIP
	}
	return ""
}

//MacroUTCOffset contains definition for UTCOffset Parameter
func (tag *BidderMacro) MacroUTCOffset(key string) string {
	if tag.HasGeo {
		return strconv.FormatInt(tag.Request.Device.Geo.UTCOffset, intBase)
	}
	return ""
}

/********************* User *********************/

//MacroUserID contains definition for UserID Parameter
func (tag *BidderMacro) MacroUserID(key string) string {
	if nil != tag.Request.User {
		return tag.Request.User.ID
	}
	return ""
}

//MacroYearOfBirth contains definition for YearOfBirth Parameter
func (tag *BidderMacro) MacroYearOfBirth(key string) string {
	if nil != tag.Request.User && tag.Request.User.Yob > 0 {
		return strconv.FormatInt(tag.Request.User.Yob, intBase)
	}
	return ""
}

//MacroGender contains definition for Gender Parameter
func (tag *BidderMacro) MacroGender(key string) string {
	if nil != tag.Request.User {
		return tag.Request.User.Gender
	}
	return ""
}

/********************* Extension *********************/

//MacroGDPRConsent contains definition for GDPRConsent Parameter
func (tag *BidderMacro) MacroGDPRConsent(key string) string {
	if nil != tag.UserExt {
		return tag.UserExt.Consent
	}
	return ""
}

//MacroGDPR contains definition for GDPR Parameter
func (tag *BidderMacro) MacroGDPR(key string) string {
	if nil != tag.RegsExt && nil != tag.RegsExt.GDPR {
		return strconv.FormatInt(int64(*tag.RegsExt.GDPR), intBase)
	}
	return ""
}

//MacroUSPrivacy contains definition for USPrivacy Parameter
func (tag *BidderMacro) MacroUSPrivacy(key string) string {
	if nil != tag.RegsExt {
		return tag.RegsExt.USPrivacy
	}
	return ""
}

/********************* Additional *********************/

//MacroCacheBuster contains definition for CacheBuster Parameter
func (tag *BidderMacro) MacroCacheBuster(key string) string {
	//change implementation
	return strconv.FormatInt(time.Now().UnixNano(), intBase)
}

/********************* Request Headers *********************/

// setDefaultHeaders sets following default headers based on VAST protocol version
//  X-device-IP; end users IP address, per VAST 4.x
//  X-Forwarded-For; end users IP address, prior VAST versions
//  X-Device-User-Agent; End users user agent, per VAST 4.x
//  User-Agent; End users user agent, prior VAST versions
//  X-Device-Referer; Referer value from the original request, per VAST 4.x
//  X-device-Accept-Language, Accept-language value from the original request, per VAST 4.x
func setDefaultHeaders(tag *BidderMacro) {
	// openrtb2. auction.go setDeviceImplicitly
	// already populates OpenRTB bid request based on http request headers
	// reusing the same information to set these headers via Macro* methods
	headers := http.Header{}
	ip := tag.IBidderMacro.MacroIP("")
	userAgent := tag.IBidderMacro.MacroUserAgent("")
	referer := tag.IBidderMacro.MacroSitePage("")
	language := tag.IBidderMacro.MacroDeviceLanguage("")

	// 1 - vast 1 - 3 expected, 2 - vast 4 expected
	expectedVastTags := 0
	if nil != tag.Imp && nil != tag.Imp.Video && nil != tag.Imp.Video.Protocols && len(tag.Imp.Video.Protocols) > 0 {
		for _, protocol := range tag.Imp.Video.Protocols {
			if protocol == openrtb2.ProtocolVAST40 || protocol == openrtb2.ProtocolVAST40Wrapper {
				expectedVastTags |= 1 << 1
			}
			if protocol <= openrtb2.ProtocolVAST30Wrapper {
				expectedVastTags |= 1 << 0
			}
		}
	} else {
		// not able to detect protocols. set all headers
		expectedVastTags = 3
	}

	if expectedVastTags == 1 || expectedVastTags == 3 {
		// vast prior to version 3 headers
		setHeaders(headers, "X-Forwarded-For", ip)
		setHeaders(headers, "User-Agent", userAgent)
	}

	if expectedVastTags == 2 || expectedVastTags == 3 {
		// vast 4 specific headers
		setHeaders(headers, "X-device-Ip", ip)
		setHeaders(headers, "X-Device-User-Agent", userAgent)
		setHeaders(headers, "X-Device-Referer", referer)
		setHeaders(headers, "X-Device-Accept-Language", language)
	}
	tag.ImpReqHeaders = headers
}

func setHeaders(headers http.Header, key, value string) {
	if len(value) > 0 {
		headers.Set(key, value)
	}
}

//getAllHeaders combines default and custom headers and returns common list
//It internally calls GetHeaders() method for obtaining list of custom headers
func (tag *BidderMacro) getAllHeaders() http.Header {
	setDefaultHeaders(tag)
	customHeaders := tag.IBidderMacro.GetHeaders()
	if nil != customHeaders {
		for k, v := range customHeaders {
			// custom header may contains default header key with value
			// in such case custom value will be prefered
			if nil != v && len(v) > 0 {
				tag.ImpReqHeaders.Set(k, v[0])
				for i := 1; i < len(v); i++ {
					tag.ImpReqHeaders.Add(k, v[i])
				}
			}
		}
	}
	return tag.ImpReqHeaders
}
