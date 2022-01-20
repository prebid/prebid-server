package vastbidder

type macroCallBack struct {
	cached   bool
	callback func(IBidderMacro, string) string
}

//Mapper will map macro with its respective call back function
type Mapper map[string]*macroCallBack

func (obj Mapper) clone() Mapper {
	cloned := make(Mapper, len(obj))
	for k, v := range obj {
		newCallback := *v
		cloned[k] = &newCallback
	}
	return cloned
}

var _defaultMapper = Mapper{
	//Request
	MacroTest:              &macroCallBack{cached: true, callback: IBidderMacro.MacroTest},
	MacroTimeout:           &macroCallBack{cached: true, callback: IBidderMacro.MacroTimeout},
	MacroWhitelistSeat:     &macroCallBack{cached: true, callback: IBidderMacro.MacroWhitelistSeat},
	MacroWhitelistLang:     &macroCallBack{cached: true, callback: IBidderMacro.MacroWhitelistLang},
	MacroBlockedSeat:       &macroCallBack{cached: true, callback: IBidderMacro.MacroBlockedSeat},
	MacroCurrency:          &macroCallBack{cached: true, callback: IBidderMacro.MacroCurrency},
	MacroBlockedCategory:   &macroCallBack{cached: true, callback: IBidderMacro.MacroBlockedCategory},
	MacroBlockedAdvertiser: &macroCallBack{cached: true, callback: IBidderMacro.MacroBlockedAdvertiser},
	MacroBlockedApp:        &macroCallBack{cached: true, callback: IBidderMacro.MacroBlockedApp},

	//Source
	MacroFD:             &macroCallBack{cached: true, callback: IBidderMacro.MacroFD},
	MacroTransactionID:  &macroCallBack{cached: true, callback: IBidderMacro.MacroTransactionID},
	MacroPaymentIDChain: &macroCallBack{cached: true, callback: IBidderMacro.MacroPaymentIDChain},

	//Regs
	MacroCoppa: &macroCallBack{cached: true, callback: IBidderMacro.MacroCoppa},

	//Impression
	MacroDisplayManager:        &macroCallBack{cached: false, callback: IBidderMacro.MacroDisplayManager},
	MacroDisplayManagerVersion: &macroCallBack{cached: false, callback: IBidderMacro.MacroDisplayManagerVersion},
	MacroInterstitial:          &macroCallBack{cached: false, callback: IBidderMacro.MacroInterstitial},
	MacroTagID:                 &macroCallBack{cached: false, callback: IBidderMacro.MacroTagID},
	MacroBidFloor:              &macroCallBack{cached: false, callback: IBidderMacro.MacroBidFloor},
	MacroBidFloorCurrency:      &macroCallBack{cached: false, callback: IBidderMacro.MacroBidFloorCurrency},
	MacroSecure:                &macroCallBack{cached: false, callback: IBidderMacro.MacroSecure},
	MacroPMP:                   &macroCallBack{cached: false, callback: IBidderMacro.MacroPMP},

	//Video
	MacroVideoMIMES:            &macroCallBack{cached: false, callback: IBidderMacro.MacroVideoMIMES},
	MacroVideoMinimumDuration:  &macroCallBack{cached: false, callback: IBidderMacro.MacroVideoMinimumDuration},
	MacroVideoMaximumDuration:  &macroCallBack{cached: false, callback: IBidderMacro.MacroVideoMaximumDuration},
	MacroVideoProtocols:        &macroCallBack{cached: false, callback: IBidderMacro.MacroVideoProtocols},
	MacroVideoPlayerWidth:      &macroCallBack{cached: false, callback: IBidderMacro.MacroVideoPlayerWidth},
	MacroVideoPlayerHeight:     &macroCallBack{cached: false, callback: IBidderMacro.MacroVideoPlayerHeight},
	MacroVideoStartDelay:       &macroCallBack{cached: false, callback: IBidderMacro.MacroVideoStartDelay},
	MacroVideoPlacement:        &macroCallBack{cached: false, callback: IBidderMacro.MacroVideoPlacement},
	MacroVideoLinearity:        &macroCallBack{cached: false, callback: IBidderMacro.MacroVideoLinearity},
	MacroVideoSkip:             &macroCallBack{cached: false, callback: IBidderMacro.MacroVideoSkip},
	MacroVideoSkipMinimum:      &macroCallBack{cached: false, callback: IBidderMacro.MacroVideoSkipMinimum},
	MacroVideoSkipAfter:        &macroCallBack{cached: false, callback: IBidderMacro.MacroVideoSkipAfter},
	MacroVideoSequence:         &macroCallBack{cached: false, callback: IBidderMacro.MacroVideoSequence},
	MacroVideoBlockedAttribute: &macroCallBack{cached: false, callback: IBidderMacro.MacroVideoBlockedAttribute},
	MacroVideoMaximumExtended:  &macroCallBack{cached: false, callback: IBidderMacro.MacroVideoMaximumExtended},
	MacroVideoMinimumBitRate:   &macroCallBack{cached: false, callback: IBidderMacro.MacroVideoMinimumBitRate},
	MacroVideoMaximumBitRate:   &macroCallBack{cached: false, callback: IBidderMacro.MacroVideoMaximumBitRate},
	MacroVideoBoxing:           &macroCallBack{cached: false, callback: IBidderMacro.MacroVideoBoxing},
	MacroVideoPlaybackMethod:   &macroCallBack{cached: false, callback: IBidderMacro.MacroVideoPlaybackMethod},
	MacroVideoDelivery:         &macroCallBack{cached: false, callback: IBidderMacro.MacroVideoDelivery},
	MacroVideoPosition:         &macroCallBack{cached: false, callback: IBidderMacro.MacroVideoPosition},
	MacroVideoAPI:              &macroCallBack{cached: false, callback: IBidderMacro.MacroVideoAPI},

	//Site
	MacroSiteID:       &macroCallBack{cached: true, callback: IBidderMacro.MacroSiteID},
	MacroSiteName:     &macroCallBack{cached: true, callback: IBidderMacro.MacroSiteName},
	MacroSitePage:     &macroCallBack{cached: true, callback: IBidderMacro.MacroSitePage},
	MacroSiteReferrer: &macroCallBack{cached: true, callback: IBidderMacro.MacroSiteReferrer},
	MacroSiteSearch:   &macroCallBack{cached: true, callback: IBidderMacro.MacroSiteSearch},
	MacroSiteMobile:   &macroCallBack{cached: true, callback: IBidderMacro.MacroSiteMobile},

	//App
	MacroAppID:       &macroCallBack{cached: true, callback: IBidderMacro.MacroAppID},
	MacroAppName:     &macroCallBack{cached: true, callback: IBidderMacro.MacroAppName},
	MacroAppBundle:   &macroCallBack{cached: true, callback: IBidderMacro.MacroAppBundle},
	MacroAppStoreURL: &macroCallBack{cached: true, callback: IBidderMacro.MacroAppStoreURL},
	MacroAppVersion:  &macroCallBack{cached: true, callback: IBidderMacro.MacroAppVersion},
	MacroAppPaid:     &macroCallBack{cached: true, callback: IBidderMacro.MacroAppPaid},

	//SiteAppCommon
	MacroCategory:        &macroCallBack{cached: true, callback: IBidderMacro.MacroCategory},
	MacroDomain:          &macroCallBack{cached: true, callback: IBidderMacro.MacroDomain},
	MacroSectionCategory: &macroCallBack{cached: true, callback: IBidderMacro.MacroSectionCategory},
	MacroPageCategory:    &macroCallBack{cached: true, callback: IBidderMacro.MacroPageCategory},
	MacroPrivacyPolicy:   &macroCallBack{cached: true, callback: IBidderMacro.MacroPrivacyPolicy},
	MacroKeywords:        &macroCallBack{cached: true, callback: IBidderMacro.MacroKeywords},

	//Publisher
	MacroPubID:     &macroCallBack{cached: true, callback: IBidderMacro.MacroPubID},
	MacroPubName:   &macroCallBack{cached: true, callback: IBidderMacro.MacroPubName},
	MacroPubDomain: &macroCallBack{cached: true, callback: IBidderMacro.MacroPubDomain},

	//Content
	MacroContentID:                 &macroCallBack{cached: true, callback: IBidderMacro.MacroContentID},
	MacroContentEpisode:            &macroCallBack{cached: true, callback: IBidderMacro.MacroContentEpisode},
	MacroContentTitle:              &macroCallBack{cached: true, callback: IBidderMacro.MacroContentTitle},
	MacroContentSeries:             &macroCallBack{cached: true, callback: IBidderMacro.MacroContentSeries},
	MacroContentSeason:             &macroCallBack{cached: true, callback: IBidderMacro.MacroContentSeason},
	MacroContentArtist:             &macroCallBack{cached: true, callback: IBidderMacro.MacroContentArtist},
	MacroContentGenre:              &macroCallBack{cached: true, callback: IBidderMacro.MacroContentGenre},
	MacroContentAlbum:              &macroCallBack{cached: true, callback: IBidderMacro.MacroContentAlbum},
	MacroContentISrc:               &macroCallBack{cached: true, callback: IBidderMacro.MacroContentISrc},
	MacroContentURL:                &macroCallBack{cached: true, callback: IBidderMacro.MacroContentURL},
	MacroContentCategory:           &macroCallBack{cached: true, callback: IBidderMacro.MacroContentCategory},
	MacroContentProductionQuality:  &macroCallBack{cached: true, callback: IBidderMacro.MacroContentProductionQuality},
	MacroContentVideoQuality:       &macroCallBack{cached: true, callback: IBidderMacro.MacroContentVideoQuality},
	MacroContentContext:            &macroCallBack{cached: true, callback: IBidderMacro.MacroContentContext},
	MacroContentContentRating:      &macroCallBack{cached: true, callback: IBidderMacro.MacroContentContentRating},
	MacroContentUserRating:         &macroCallBack{cached: true, callback: IBidderMacro.MacroContentUserRating},
	MacroContentQAGMediaRating:     &macroCallBack{cached: true, callback: IBidderMacro.MacroContentQAGMediaRating},
	MacroContentKeywords:           &macroCallBack{cached: true, callback: IBidderMacro.MacroContentKeywords},
	MacroContentLiveStream:         &macroCallBack{cached: true, callback: IBidderMacro.MacroContentLiveStream},
	MacroContentSourceRelationship: &macroCallBack{cached: true, callback: IBidderMacro.MacroContentSourceRelationship},
	MacroContentLength:             &macroCallBack{cached: true, callback: IBidderMacro.MacroContentLength},
	MacroContentLanguage:           &macroCallBack{cached: true, callback: IBidderMacro.MacroContentLanguage},
	MacroContentEmbeddable:         &macroCallBack{cached: true, callback: IBidderMacro.MacroContentEmbeddable},

	//Producer
	MacroProducerID:   &macroCallBack{cached: true, callback: IBidderMacro.MacroProducerID},
	MacroProducerName: &macroCallBack{cached: true, callback: IBidderMacro.MacroProducerName},

	//Device
	MacroUserAgent:       &macroCallBack{cached: true, callback: IBidderMacro.MacroUserAgent},
	MacroDNT:             &macroCallBack{cached: true, callback: IBidderMacro.MacroDNT},
	MacroLMT:             &macroCallBack{cached: true, callback: IBidderMacro.MacroLMT},
	MacroIP:              &macroCallBack{cached: true, callback: IBidderMacro.MacroIP},
	MacroDeviceType:      &macroCallBack{cached: true, callback: IBidderMacro.MacroDeviceType},
	MacroMake:            &macroCallBack{cached: true, callback: IBidderMacro.MacroMake},
	MacroModel:           &macroCallBack{cached: true, callback: IBidderMacro.MacroModel},
	MacroDeviceOS:        &macroCallBack{cached: true, callback: IBidderMacro.MacroDeviceOS},
	MacroDeviceOSVersion: &macroCallBack{cached: true, callback: IBidderMacro.MacroDeviceOSVersion},
	MacroDeviceWidth:     &macroCallBack{cached: true, callback: IBidderMacro.MacroDeviceWidth},
	MacroDeviceHeight:    &macroCallBack{cached: true, callback: IBidderMacro.MacroDeviceHeight},
	MacroDeviceJS:        &macroCallBack{cached: true, callback: IBidderMacro.MacroDeviceJS},
	MacroDeviceLanguage:  &macroCallBack{cached: true, callback: IBidderMacro.MacroDeviceLanguage},
	MacroDeviceIFA:       &macroCallBack{cached: true, callback: IBidderMacro.MacroDeviceIFA},
	MacroDeviceDIDSHA1:   &macroCallBack{cached: true, callback: IBidderMacro.MacroDeviceDIDSHA1},
	MacroDeviceDIDMD5:    &macroCallBack{cached: true, callback: IBidderMacro.MacroDeviceDIDMD5},
	MacroDeviceDPIDSHA1:  &macroCallBack{cached: true, callback: IBidderMacro.MacroDeviceDPIDSHA1},
	MacroDeviceDPIDMD5:   &macroCallBack{cached: true, callback: IBidderMacro.MacroDeviceDPIDMD5},
	MacroDeviceMACSHA1:   &macroCallBack{cached: true, callback: IBidderMacro.MacroDeviceMACSHA1},
	MacroDeviceMACMD5:    &macroCallBack{cached: true, callback: IBidderMacro.MacroDeviceMACMD5},

	//Geo
	MacroLatitude:  &macroCallBack{cached: true, callback: IBidderMacro.MacroLatitude},
	MacroLongitude: &macroCallBack{cached: true, callback: IBidderMacro.MacroLongitude},
	MacroCountry:   &macroCallBack{cached: true, callback: IBidderMacro.MacroCountry},
	MacroRegion:    &macroCallBack{cached: true, callback: IBidderMacro.MacroRegion},
	MacroCity:      &macroCallBack{cached: true, callback: IBidderMacro.MacroCity},
	MacroZip:       &macroCallBack{cached: true, callback: IBidderMacro.MacroZip},
	MacroUTCOffset: &macroCallBack{cached: true, callback: IBidderMacro.MacroUTCOffset},

	//User
	MacroUserID:      &macroCallBack{cached: true, callback: IBidderMacro.MacroUserID},
	MacroYearOfBirth: &macroCallBack{cached: true, callback: IBidderMacro.MacroYearOfBirth},
	MacroGender:      &macroCallBack{cached: true, callback: IBidderMacro.MacroGender},

	//Extension
	MacroGDPRConsent: &macroCallBack{cached: true, callback: IBidderMacro.MacroGDPRConsent},
	MacroGDPR:        &macroCallBack{cached: true, callback: IBidderMacro.MacroGDPR},
	MacroUSPrivacy:   &macroCallBack{cached: true, callback: IBidderMacro.MacroUSPrivacy},

	//Additional
	MacroCacheBuster: &macroCallBack{cached: false, callback: IBidderMacro.MacroCacheBuster},
}

//GetDefaultMapper will return clone of default Mapper function
func GetDefaultMapper() Mapper {
	return _defaultMapper.clone()
}
