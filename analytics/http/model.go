package http

import (
	"fmt"
	"time"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/v2/analytics"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/hooks/hookexecution"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"github.com/prebid/prebid-server/v2/util/jsonutil"
)

type EventType string

const (
	EventTypeAuction      EventType = "auction"
	EventTypeAmp          EventType = "amp"
	EventTypeCookieSync   EventType = "cookiesync"
	EventTypeNotification EventType = "notification"
	EventTypeSetUID       EventType = "setuid"
	EventTypeVideo        EventType = "video"
)

type logAuction struct {
	Status               int                          `json:"status,omitempty"`
	Errors               []error                      `json:"errors,omitempty"`
	Request              *openrtb2.BidRequest         `json:"request,omitempty"`
	Response             *openrtb2.BidResponse        `json:"response,omitempty"`
	Account              *config.Account              `json:"account,omitempty"`
	StartTime            time.Time                    `json:"startTime,omitempty"`
	HookExecutionOutcome []hookexecution.StageOutcome `json:"hookExecutionOutcome,omitempty"`
}

type logVideo struct {
	Status        int                           `json:"status,omitempty"`
	Errors        []error                       `json:"errors,omitempty"`
	Request       *openrtb2.BidRequest          `json:"request,omitempty"`
	Response      *openrtb2.BidResponse         `json:"response,omitempty"`
	VideoRequest  *openrtb_ext.BidRequestVideo  `json:"videoRequest,omitempty"`
	VideoResponse *openrtb_ext.BidResponseVideo `json:"videoResponse,omitempty"`
	StartTime     time.Time                     `json:"startTime,omitempty"`
}

type logSetUID struct {
	Status  int     `json:"status,omitempty"`
	Bidder  string  `json:"bidder,omitempty"`
	UID     string  `json:"uid,omitempty"`
	Errors  []error `json:"errors,omitempty"`
	Success bool    `json:"success,omitempty"`
}

type logUserSync struct {
	Status       int                           `json:"Status,omitempty"`
	Errors       []error                       `json:"Errors,omitempty"`
	BidderStatus []*analytics.CookieSyncBidder `json:"BidderStatus,omitempty"`
}

type logAMP struct {
	Status               int                          `json:"status,omitempty"`
	Errors               []error                      `json:"errors,omitempty"`
	Request              *openrtb2.BidRequest         `json:"request,omitempty"`
	AuctionResponse      *openrtb2.BidResponse        `json:"auctionResponse,omitempty"`
	AmpTargetingValues   map[string]string            `json:"ampTargetingValues,omitempty"`
	Origin               string                       `json:"origin,omitempty"`
	StartTime            time.Time                    `json:"startTime,omitempty"`
	HookExecutionOutcome []hookexecution.StageOutcome `json:"hookExecutionOutcome,omitempty"`
}

type logNotificationEvent struct {
	Request *analytics.EventRequest `json:"request,omitempty"`
	Account *config.Account         `json:"account,omitempty"`
}

func serializeAuctionObject(event *analytics.AuctionObject, now time.Time) ([]byte, error) {
	if event == nil {
		return nil, fmt.Errorf("auction object nil")
	}
	var request *openrtb2.BidRequest
	if event.RequestWrapper != nil {
		request = event.RequestWrapper.BidRequest
	}
	logEntry := &logAuction{
		Status:               event.Status,
		Errors:               event.Errors,
		Request:              request,
		Response:             event.Response,
		Account:              event.Account,
		StartTime:            event.StartTime,
		HookExecutionOutcome: event.HookExecutionOutcome,
	}

	return jsonutil.Marshal(&struct {
		Type      EventType `json:"type"`
		CreatedAt time.Time `json:"createdAt"`
		*logAuction
	}{
		Type:       EventTypeAuction,
		CreatedAt:  now,
		logAuction: logEntry,
	})
}

func serializeVideoObject(event *analytics.VideoObject, now time.Time) ([]byte, error) {
	if event == nil {
		return nil, fmt.Errorf("video object nil")
	}
	var request *openrtb2.BidRequest
	if event.RequestWrapper != nil {
		request = event.RequestWrapper.BidRequest
	}
	logEntry := &logVideo{
		Status:        event.Status,
		Errors:        event.Errors,
		Request:       request,
		Response:      event.Response,
		VideoRequest:  event.VideoRequest,
		VideoResponse: event.VideoResponse,
		StartTime:     event.StartTime,
	}

	return jsonutil.Marshal(&struct {
		Type      EventType `json:"type"`
		CreatedAt time.Time `json:"createdAt"`
		*logVideo
	}{
		Type:      EventTypeVideo,
		CreatedAt: now,
		logVideo:  logEntry,
	})
}

func serializeSetUIDObject(event *analytics.SetUIDObject, now time.Time) ([]byte, error) {
	if event == nil {
		return nil, fmt.Errorf("setuid object nil")
	}
	logEntry := &logSetUID{
		Status:  event.Status,
		Bidder:  event.Bidder,
		UID:     event.UID,
		Errors:  event.Errors,
		Success: event.Success,
	}

	return jsonutil.Marshal(&struct {
		Type      EventType `json:"type"`
		CreatedAt time.Time `json:"createdAt"`
		*logSetUID
	}{
		Type:      EventTypeSetUID,
		CreatedAt: now,
		logSetUID: logEntry,
	})
}

func serializeNotificationEvent(event *analytics.NotificationEvent, now time.Time) ([]byte, error) {
	if event == nil {
		return nil, fmt.Errorf("notification event object nil")
	}
	logEntry := &logNotificationEvent{
		Request: event.Request,
		Account: event.Account,
	}

	return jsonutil.Marshal(&struct {
		Type      EventType `json:"type"`
		CreatedAt time.Time `json:"createdAt"`
		*logNotificationEvent
	}{
		Type:                 EventTypeNotification,
		CreatedAt:            now,
		logNotificationEvent: logEntry,
	})
}

func serializeCookieSyncObject(event *analytics.CookieSyncObject, now time.Time) ([]byte, error) {
	if event == nil {
		return nil, fmt.Errorf("cookie sync object nil")
	}
	logEntry := &logUserSync{
		Status:       event.Status,
		Errors:       event.Errors,
		BidderStatus: event.BidderStatus,
	}

	return jsonutil.Marshal(&struct {
		Type      EventType `json:"type"`
		CreatedAt time.Time `json:"createdAt"`
		*logUserSync
	}{
		Type:        EventTypeCookieSync,
		CreatedAt:   now,
		logUserSync: logEntry,
	})
}

func serializeAmpObject(event *analytics.AmpObject, now time.Time) ([]byte, error) {
	if event == nil {
		return nil, fmt.Errorf("amp object nil")
	}
	var request *openrtb2.BidRequest
	if event.RequestWrapper != nil {
		request = event.RequestWrapper.BidRequest
	}
	logEntry := &logAMP{
		Status:               event.Status,
		Errors:               event.Errors,
		Request:              request,
		AuctionResponse:      event.AuctionResponse,
		AmpTargetingValues:   event.AmpTargetingValues,
		Origin:               event.Origin,
		StartTime:            event.StartTime,
		HookExecutionOutcome: event.HookExecutionOutcome,
	}

	return jsonutil.Marshal(&struct {
		Type      EventType `json:"type"`
		CreatedAt time.Time `json:"createdAt"`
		*logAMP
	}{
		Type:      EventTypeAmp,
		CreatedAt: now,
		logAMP:    logEntry,
	})
}
