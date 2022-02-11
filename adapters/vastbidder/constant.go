package vastbidder

const (
	intBase = 10
	comma   = `,`
)

//List of Tag Bidder Macros
const (
	//Request
	MacroTest              = `test`
	MacroTimeout           = `timeout`
	MacroWhitelistSeat     = `wseat`
	MacroWhitelistLang     = `wlang`
	MacroBlockedSeat       = `bseat`
	MacroCurrency          = `cur`
	MacroBlockedCategory   = `bcat`
	MacroBlockedAdvertiser = `badv`
	MacroBlockedApp        = `bapp`

	//Source
	MacroFD             = `fd`
	MacroTransactionID  = `tid`
	MacroPaymentIDChain = `pchain`

	//Regs
	MacroCoppa = `coppa`

	//Impression
	MacroDisplayManager        = `displaymanager`
	MacroDisplayManagerVersion = `displaymanagerver`
	MacroInterstitial          = `instl`
	MacroTagID                 = `tagid`
	MacroBidFloor              = `bidfloor`
	MacroBidFloorCurrency      = `bidfloorcur`
	MacroSecure                = `secure`
	MacroPMP                   = `pmp`

	//Video
	MacroVideoMIMES            = `mimes`
	MacroVideoMinimumDuration  = `minduration`
	MacroVideoMaximumDuration  = `maxduration`
	MacroVideoProtocols        = `protocols`
	MacroVideoPlayerWidth      = `playerwidth`
	MacroVideoPlayerHeight     = `playerheight`
	MacroVideoStartDelay       = `startdelay`
	MacroVideoPlacement        = `placement`
	MacroVideoLinearity        = `linearity`
	MacroVideoSkip             = `skip`
	MacroVideoSkipMinimum      = `skipmin`
	MacroVideoSkipAfter        = `skipafter`
	MacroVideoSequence         = `sequence`
	MacroVideoBlockedAttribute = `battr`
	MacroVideoMaximumExtended  = `maxextended`
	MacroVideoMinimumBitRate   = `minbitrate`
	MacroVideoMaximumBitRate   = `maxbitrate`
	MacroVideoBoxing           = `boxingallowed`
	MacroVideoPlaybackMethod   = `playbackmethod`
	MacroVideoDelivery         = `delivery`
	MacroVideoPosition         = `position`
	MacroVideoAPI              = `api`

	//Site
	MacroSiteID       = `siteid`
	MacroSiteName     = `sitename`
	MacroSitePage     = `page`
	MacroSiteReferrer = `ref`
	MacroSiteSearch   = `search`
	MacroSiteMobile   = `mobile`

	//App
	MacroAppID       = `appid`
	MacroAppName     = `appname`
	MacroAppBundle   = `bundle`
	MacroAppStoreURL = `storeurl`
	MacroAppVersion  = `appver`
	MacroAppPaid     = `paid`

	//SiteAppCommon
	MacroCategory        = `cat`
	MacroDomain          = `domain`
	MacroSectionCategory = `sectioncat`
	MacroPageCategory    = `pagecat`
	MacroPrivacyPolicy   = `privacypolicy`
	MacroKeywords        = `keywords`

	//Publisher
	MacroPubID     = `pubid`
	MacroPubName   = `pubname`
	MacroPubDomain = `pubdomain`

	//Content
	MacroContentID                 = `contentid`
	MacroContentEpisode            = `episode`
	MacroContentTitle              = `title`
	MacroContentSeries             = `series`
	MacroContentSeason             = `season`
	MacroContentArtist             = `artist`
	MacroContentGenre              = `genre`
	MacroContentAlbum              = `album`
	MacroContentISrc               = `isrc`
	MacroContentURL                = `contenturl`
	MacroContentCategory           = `contentcat`
	MacroContentProductionQuality  = `contentprodq`
	MacroContentVideoQuality       = `contentvideoquality`
	MacroContentContext            = `context`
	MacroContentContentRating      = `contentrating`
	MacroContentUserRating         = `userrating`
	MacroContentQAGMediaRating     = `qagmediarating`
	MacroContentKeywords           = `contentkeywords`
	MacroContentLiveStream         = `livestream`
	MacroContentSourceRelationship = `sourcerelationship`
	MacroContentLength             = `contentlen`
	MacroContentLanguage           = `contentlanguage`
	MacroContentEmbeddable         = `contentembeddable`

	//Producer
	MacroProducerID   = `prodid`
	MacroProducerName = `prodname`

	//Device
	MacroUserAgent       = `useragent`
	MacroDNT             = `dnt`
	MacroLMT             = `lmt`
	MacroIP              = `ip`
	MacroDeviceType      = `devicetype`
	MacroMake            = `make`
	MacroModel           = `model`
	MacroDeviceOS        = `os`
	MacroDeviceOSVersion = `osv`
	MacroDeviceWidth     = `devicewidth`
	MacroDeviceHeight    = `deviceheight`
	MacroDeviceJS        = `js`
	MacroDeviceLanguage  = `lang`
	MacroDeviceIFA       = `ifa`
	MacroDeviceIFAType   = `ifa_type`
	MacroDeviceDIDSHA1   = `didsha1`
	MacroDeviceDIDMD5    = `didmd5`
	MacroDeviceDPIDSHA1  = `dpidsha1`
	MacroDeviceDPIDMD5   = `dpidmd5`
	MacroDeviceMACSHA1   = `macsha1`
	MacroDeviceMACMD5    = `macmd5`

	//Geo
	MacroLatitude  = `lat`
	MacroLongitude = `lon`
	MacroCountry   = `country`
	MacroRegion    = `region`
	MacroCity      = `city`
	MacroZip       = `zip`
	MacroUTCOffset = `utcoffset`

	//User
	MacroUserID      = `uid`
	MacroYearOfBirth = `yob`
	MacroGender      = `gender`

	//Extension
	MacroGDPRConsent = `consent`
	MacroGDPR        = `gdpr`
	MacroUSPrivacy   = `usprivacy`

	//Additional
	MacroCacheBuster = `cachebuster`
)

var ParamKeys = []string{"param1", "param2", "param3", "param4", "param5"}
