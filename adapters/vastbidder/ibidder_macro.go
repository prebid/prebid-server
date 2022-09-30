package vastbidder

import (
	"net/http"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

//IBidderMacro interface will capture all macro definition
type IBidderMacro interface {
	//Helper Function
	InitBidRequest(request *openrtb2.BidRequest)
	LoadImpression(imp *openrtb2.Imp) (*openrtb_ext.ExtImpVASTBidder, error)
	LoadVASTTag(tag *openrtb_ext.ExtImpVASTBidderTag)
	GetBidderKeys() map[string]string
	SetAdapterConfig(*config.Adapter)
	GetURI() string
	GetHeaders() http.Header
	//getAllHeaders returns default and custom heades
	getAllHeaders() http.Header

	//Request
	MacroTest(string) string
	MacroTimeout(string) string
	MacroWhitelistSeat(string) string
	MacroWhitelistLang(string) string
	MacroBlockedSeat(string) string
	MacroCurrency(string) string
	MacroBlockedCategory(string) string
	MacroBlockedAdvertiser(string) string
	MacroBlockedApp(string) string

	//Source
	MacroFD(string) string
	MacroTransactionID(string) string
	MacroPaymentIDChain(string) string

	//Regs
	MacroCoppa(string) string

	//Impression
	MacroDisplayManager(string) string
	MacroDisplayManagerVersion(string) string
	MacroInterstitial(string) string
	MacroTagID(string) string
	MacroBidFloor(string) string
	MacroBidFloorCurrency(string) string
	MacroSecure(string) string
	MacroPMP(string) string

	//Video
	MacroVideoMIMES(string) string
	MacroVideoMinimumDuration(string) string
	MacroVideoMaximumDuration(string) string
	MacroVideoProtocols(string) string
	MacroVideoPlayerWidth(string) string
	MacroVideoPlayerHeight(string) string
	MacroVideoStartDelay(string) string
	MacroVideoPlacement(string) string
	MacroVideoLinearity(string) string
	MacroVideoSkip(string) string
	MacroVideoSkipMinimum(string) string
	MacroVideoSkipAfter(string) string
	MacroVideoSequence(string) string
	MacroVideoBlockedAttribute(string) string
	MacroVideoMaximumExtended(string) string
	MacroVideoMinimumBitRate(string) string
	MacroVideoMaximumBitRate(string) string
	MacroVideoBoxing(string) string
	MacroVideoPlaybackMethod(string) string
	MacroVideoDelivery(string) string
	MacroVideoPosition(string) string
	MacroVideoAPI(string) string

	//Site
	MacroSiteID(string) string
	MacroSiteName(string) string
	MacroSitePage(string) string
	MacroSiteReferrer(string) string
	MacroSiteSearch(string) string
	MacroSiteMobile(string) string

	//App
	MacroAppID(string) string
	MacroAppName(string) string
	MacroAppBundle(string) string
	MacroAppStoreURL(string) string
	MacroAppVersion(string) string
	MacroAppPaid(string) string

	//SiteAppCommon
	MacroCategory(string) string
	MacroDomain(string) string
	MacroSectionCategory(string) string
	MacroPageCategory(string) string
	MacroPrivacyPolicy(string) string
	MacroKeywords(string) string

	//Publisher
	MacroPubID(string) string
	MacroPubName(string) string
	MacroPubDomain(string) string

	//Content
	MacroContentID(string) string
	MacroContentEpisode(string) string
	MacroContentTitle(string) string
	MacroContentSeries(string) string
	MacroContentSeason(string) string
	MacroContentArtist(string) string
	MacroContentGenre(string) string
	MacroContentAlbum(string) string
	MacroContentISrc(string) string
	MacroContentURL(string) string
	MacroContentCategory(string) string
	MacroContentProductionQuality(string) string
	MacroContentVideoQuality(string) string
	MacroContentContext(string) string
	MacroContentContentRating(string) string
	MacroContentUserRating(string) string
	MacroContentQAGMediaRating(string) string
	MacroContentKeywords(string) string
	MacroContentLiveStream(string) string
	MacroContentSourceRelationship(string) string
	MacroContentLength(string) string
	MacroContentLanguage(string) string
	MacroContentEmbeddable(string) string

	//Producer
	MacroProducerID(string) string
	MacroProducerName(string) string

	//Device
	MacroUserAgent(string) string
	MacroDNT(string) string
	MacroLMT(string) string
	MacroIP(string) string
	MacroDeviceType(string) string
	MacroMake(string) string
	MacroModel(string) string
	MacroDeviceOS(string) string
	MacroDeviceOSVersion(string) string
	MacroDeviceWidth(string) string
	MacroDeviceHeight(string) string
	MacroDeviceJS(string) string
	MacroDeviceLanguage(string) string
	MacroDeviceIFA(string) string
	MacroDeviceIFAType(string) string
	MacroDeviceDIDSHA1(string) string
	MacroDeviceDIDMD5(string) string
	MacroDeviceDPIDSHA1(string) string
	MacroDeviceDPIDMD5(string) string
	MacroDeviceMACSHA1(string) string
	MacroDeviceMACMD5(string) string

	//Geo
	MacroLatitude(string) string
	MacroLongitude(string) string
	MacroCountry(string) string
	MacroRegion(string) string
	MacroCity(string) string
	MacroZip(string) string
	MacroUTCOffset(string) string

	//User
	MacroUserID(string) string
	MacroYearOfBirth(string) string
	MacroGender(string) string

	//Extension
	MacroGDPRConsent(string) string
	MacroGDPR(string) string
	MacroUSPrivacy(string) string

	//Additional
	MacroCacheBuster(string) string
}

var bidderMacroMap = map[openrtb_ext.BidderName]func() IBidderMacro{}

//RegisterNewBidderMacro will be used by each bidder to set its respective macro IBidderMacro
func RegisterNewBidderMacro(bidder openrtb_ext.BidderName, macro func() IBidderMacro) {
	bidderMacroMap[bidder] = macro
}

//GetNewBidderMacro will return IBidderMacro of specific bidder
func GetNewBidderMacro(bidder openrtb_ext.BidderName) IBidderMacro {
	callback, ok := bidderMacroMap[bidder]
	if ok {
		return callback()
	}
	return NewBidderMacro()
}
