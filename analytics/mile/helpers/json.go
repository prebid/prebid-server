package helpers

import (
	"github.com/prebid/prebid-server/v2/analytics"
)

func JsonifyAuctionObject(ao *analytics.AuctionObject, scope string) (*MileAnalyticsEvent, error) {
	var logEntry *MileAnalyticsEvent
	if ao != nil {
		//var request *openrtb2.BidRequest
		//if ao.RequestWrapper != nil {
		//	request = ao.RequestWrapper.BidRequest
		//}

		//siteID, err := strconv.Atoi(ao.RequestWrapper.Site.ID)
		//if err != nil {
		//	return nil, err
		//}

		logEntry = &MileAnalyticsEvent{
			//SessionID: ao.RequestWrapper
			Ip:                ao.RequestWrapper.Device.IP,
			Ua:                ao.RequestWrapper.Device.UA,
			CityName:          ao.RequestWrapper.Device.Geo.City,
			StateName:         ao.RequestWrapper.Device.Geo.Region,
			CountryName:       ao.RequestWrapper.Device.Geo.Country,
			ArbitraryData:     "",
			Device:            ao.RequestWrapper.Device.Model,
			Publisher:         ao.RequestWrapper.Site.Publisher.ID,
			Site:              ao.RequestWrapper.Site.ID,
			ReferrerURL:       ao.RequestWrapper.Site.Ref,
			AdvertiserName:    "",
			AuctionID:         ao.RequestWrapper.ID, // TODO
			Page:              ao.RequestWrapper.Site.Page,
			YetiSiteID:        ao.RequestWrapper.Site.ID,
			YetiPublisherID:   ao.RequestWrapper.Site.Publisher.ID,
			SessionID:         "",
			EventType:         "",
			Section:           "",
			BidBidders:        []string{},
			ConfiguredBidders: []string{},
			//Viewability: ao.RequestWrapper.
			//WinningSize: ao.Response.SeatBi

			//Status:               ao.Status,
			//Errors:               ao.Errors,
			//Request:              request,
			//Response:             ao.Response,
			//Account:              ao.Account,
			//StartTime:            ao.StartTime,
			//HookExecutionOutcome: ao.HookExecutionOutcome,
		}
	}
	return logEntry, nil

}

func JsonifyVideoObject(vo *analytics.VideoObject, scope string) (*MileAnalyticsEvent, error) {
	var logEntry *MileAnalyticsEvent
	if vo != nil {
		//var request *openrtb2.BidRequest
		//if ao.RequestWrapper != nil {
		//	request = ao.RequestWrapper.BidRequest
		//}
		logEntry = &MileAnalyticsEvent{
			//Status:               ao.Status,
			//Errors:               ao.Errors,
			//Request:              request,
			//Response:             ao.Response,
			//Account:              ao.Account,
			//StartTime:            ao.StartTime,
			//HookExecutionOutcome: ao.HookExecutionOutcome,
		}
	}
	return logEntry, nil

}

func JsonifyAmpObject(ao *analytics.AmpObject, scope string) (*MileAnalyticsEvent, error) {
	var logEntry *MileAnalyticsEvent
	if ao != nil {
		//var request *openrtb2.BidRequest
		//if ao.RequestWrapper != nil {
		//	request = ao.RequestWrapper.BidRequest
		//}
		logEntry = &MileAnalyticsEvent{
			//Status:               ao.Status,
			//Errors:               ao.Errors,
			//Request:              request,
			//Response:             ao.Response,
			//Account:              ao.Account,
			//StartTime:            ao.StartTime,
			//HookExecutionOutcome: ao.HookExecutionOutcome,
		}
	}
	return logEntry, nil

}
func JsonifyCookieSync(cso *analytics.CookieSyncObject, scope string) (*MileAnalyticsEvent, error) {
	var logEntry *MileAnalyticsEvent
	if cso != nil {
		//var request *openrtb2.BidRequest
		//if ao.RequestWrapper != nil {
		//	request = ao.RequestWrapper.BidRequest
		//}
		logEntry = &MileAnalyticsEvent{
			//Status:               ao.Status,
			//Errors:               ao.Errors,
			//Request:              request,
			//Response:             ao.Response,
			//Account:              ao.Account,
			//StartTime:            ao.StartTime,
			//HookExecutionOutcome: ao.HookExecutionOutcome,
		}
	}
	return logEntry, nil

}
func JsonifySetUIDObject(so *analytics.SetUIDObject, scope string) (*MileAnalyticsEvent, error) {
	var logEntry *MileAnalyticsEvent
	if so != nil {
		//var request *openrtb2.BidRequest
		//if ao.RequestWrapper != nil {
		//	request = ao.RequestWrapper.BidRequest
		//}
		logEntry = &MileAnalyticsEvent{
			//Status:               ao.Status,
			//Errors:               ao.Errors,
			//Request:              request,
			//Response:             ao.Response,
			//Account:              ao.Account,
			//StartTime:            ao.StartTime,
			//HookExecutionOutcome: ao.HookExecutionOutcome,
		}
	}
	return logEntry, nil

}
