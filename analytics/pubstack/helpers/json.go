package helpers

import (
	"fmt"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

func JsonifyAuctionObject(ao *analytics.AuctionObject, scope string) ([]byte, error) {
	var logEntry *logAuction
	if ao != nil {
		var request *openrtb2.BidRequest
		if ao.RequestWrapper != nil {
			request = ao.RequestWrapper.BidRequest
		}
		logEntry = &logAuction{
			Status:               ao.Status,
			Errors:               ao.Errors,
			Request:              request,
			Response:             ao.Response,
			Account:              ao.Account,
			StartTime:            ao.StartTime,
			HookExecutionOutcome: ao.HookExecutionOutcome,
		}
	}

	b, err := jsonutil.Marshal(&struct {
		Scope string `json:"scope"`
		*logAuction
	}{
		Scope:      scope,
		logAuction: logEntry,
	})

	if err == nil {
		b = append(b, byte('\n'))
		return b, nil
	}
	return nil, fmt.Errorf("auction object badly formed %v", err)
}

func JsonifyVideoObject(vo *analytics.VideoObject, scope string) ([]byte, error) {
	var logEntry *logVideo
	if vo != nil {
		var request *openrtb2.BidRequest
		if vo.RequestWrapper != nil {
			request = vo.RequestWrapper.BidRequest
		}
		logEntry = &logVideo{
			Status:        vo.Status,
			Errors:        vo.Errors,
			Request:       request,
			Response:      vo.Response,
			VideoRequest:  vo.VideoRequest,
			VideoResponse: vo.VideoResponse,
			StartTime:     vo.StartTime,
		}
	}

	b, err := jsonutil.Marshal(&struct {
		Scope string `json:"scope"`
		*logVideo
	}{
		Scope:    scope,
		logVideo: logEntry,
	})

	if err == nil {
		b = append(b, byte('\n'))
		return b, nil
	}
	return nil, fmt.Errorf("video object badly formed %v", err)
}

func JsonifyCookieSync(cso *analytics.CookieSyncObject, scope string) ([]byte, error) {
	var logEntry *logUserSync
	if cso != nil {
		logEntry = &logUserSync{
			Status:       cso.Status,
			Errors:       cso.Errors,
			BidderStatus: cso.BidderStatus,
		}
	}

	b, err := jsonutil.Marshal(&struct {
		Scope string `json:"scope"`
		*logUserSync
	}{
		Scope:       scope,
		logUserSync: logEntry,
	})

	if err == nil {
		b = append(b, byte('\n'))
		return b, nil
	}
	return nil, fmt.Errorf("cookie sync object badly formed %v", err)
}

func JsonifySetUIDObject(so *analytics.SetUIDObject, scope string) ([]byte, error) {
	var logEntry *logSetUID
	if so != nil {
		logEntry = &logSetUID{
			Status:  so.Status,
			Bidder:  so.Bidder,
			UID:     so.UID,
			Errors:  so.Errors,
			Success: so.Success,
		}
	}

	b, err := jsonutil.Marshal(&struct {
		Scope string `json:"scope"`
		*logSetUID
	}{
		Scope:     scope,
		logSetUID: logEntry,
	})

	if err == nil {
		b = append(b, byte('\n'))
		return b, nil
	}
	return nil, fmt.Errorf("set UID object badly formed %v", err)
}

func JsonifyAmpObject(ao *analytics.AmpObject, scope string) ([]byte, error) {
	var logEntry *logAMP
	if ao != nil {
		var request *openrtb2.BidRequest
		if ao.RequestWrapper != nil {
			request = ao.RequestWrapper.BidRequest
		}
		logEntry = &logAMP{
			Status:               ao.Status,
			Errors:               ao.Errors,
			Request:              request,
			AuctionResponse:      ao.AuctionResponse,
			AmpTargetingValues:   ao.AmpTargetingValues,
			Origin:               ao.Origin,
			StartTime:            ao.StartTime,
			HookExecutionOutcome: ao.HookExecutionOutcome,
		}
	}

	b, err := jsonutil.Marshal(&struct {
		Scope string `json:"scope"`
		*logAMP
	}{
		Scope:  scope,
		logAMP: logEntry,
	})

	if err == nil {
		b = append(b, byte('\n'))
		return b, nil
	}
	return nil, fmt.Errorf("amp object badly formed %v", err)
}
