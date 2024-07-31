package helpers

import (
	"github.com/prebid/prebid-server/v2/analytics"
)

func JsonifyAuctionObject(ao *analytics.AuctionObject, scope string) (*PageViewRecord, error) {
	var logEntry *PageViewRecord
	if ao != nil {
		//var request *openrtb2.BidRequest
		//if ao.RequestWrapper != nil {
		//	request = ao.RequestWrapper.BidRequest
		//}
		logEntry = &PageViewRecord{
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

func JsonifyVideoObject(vo *analytics.VideoObject, scope string) (*PageViewRecord, error) {
	var logEntry *PageViewRecord
	if vo != nil {
		//var request *openrtb2.BidRequest
		//if ao.RequestWrapper != nil {
		//	request = ao.RequestWrapper.BidRequest
		//}
		logEntry = &PageViewRecord{
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

func JsonifyAmpObject(ao *analytics.AmpObject, scope string) (*PageViewRecord, error) {
	var logEntry *PageViewRecord
	if ao != nil {
		//var request *openrtb2.BidRequest
		//if ao.RequestWrapper != nil {
		//	request = ao.RequestWrapper.BidRequest
		//}
		logEntry = &PageViewRecord{
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
func JsonifyCookieSync(cso *analytics.CookieSyncObject, scope string) (*PageViewRecord, error) {
	var logEntry *PageViewRecord
	if cso != nil {
		//var request *openrtb2.BidRequest
		//if ao.RequestWrapper != nil {
		//	request = ao.RequestWrapper.BidRequest
		//}
		logEntry = &PageViewRecord{
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
func JsonifySetUIDObject(so *analytics.SetUIDObject, scope string) (*PageViewRecord, error) {
	var logEntry *PageViewRecord
	if so != nil {
		//var request *openrtb2.BidRequest
		//if ao.RequestWrapper != nil {
		//	request = ao.RequestWrapper.BidRequest
		//}
		logEntry = &PageViewRecord{
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
