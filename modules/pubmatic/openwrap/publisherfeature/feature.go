package publisherfeature

import "github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"

type Feature interface {
	IsFscApplicable(pubId int, seat string, dspId int) bool
	IsAmpMultiformatEnabled(pubId int) bool
	IsMaxFloorsEnabled(pubId int) bool
	IsTBFFeatureEnabled(pubid int, profid int) bool
	IsAnalyticsTrackingThrottled(pubID, profileID int) (bool, bool)
	IsBidRecoveryEnabled(pubID int, profileID int) bool
	IsApplovinMultiFloorsEnabled(pubID int, profileID string) bool
	GetApplovinMultiFloors(pubID int, profileID string) models.ApplovinAdUnitFloors
	GetImpCountingMethodEnabledBidders() map[string]struct{}
	IsCountryGDPREnabled(countryCode string) bool
	IsMBMFCountryForPublisher(countryCode string, pubID int) bool
	IsMBMFPublisherEnabled(pubID int) bool
	IsMBMFEnabledForAdUnitFormat(pubID int, adUnitFormat string) bool
	GetMBMFFloorsForAdUnitFormat(pubID int, adUnitFormat string) *models.MultiFloors
	GetProfileAdUnitMultiFloors(profileID int) map[string]*models.MultiFloors
	IsDynamicFloorEnabledPublisher(pubID int) bool
	GetEnabledPerformanceDSPs() map[int]struct{}
	GetInViewEnabledPublishers() map[int]struct{}
	IsActApplicable(pubId int, seat string, dspId int) bool
	IsEDSBlockedCountry(countryCode string) bool
}
