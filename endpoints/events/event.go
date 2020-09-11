package events

import (
	"context"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/stored_requests"
	"net/http"
	"net/url"
	"strconv"
)

const (
	// Required
	TemplateUrl        = "%v/event?t=%v&b=%v&a=%v"
	TypeParameter      = "t"
	BidIdParameter     = "b"
	AccountIdParameter = "a"

	// Optional
	BidderParameter    = "bidder"
	TimestampParameter = "ts"
	FormatParameter    = "f"
	AnalyticsParameter = "x"
)

var trackingPixelPng = &TrackingPixel{
	Content: []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44,
		0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
		0x89, 0x00, 0x00, 0x00, 0x04, 0x73, 0x42, 0x49, 0x54, 0x08, 0x08, 0x08, 0x08, 0x7C, 0x08, 0x64, 0x88,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x44, 0x41, 0x54, 0x08, 0x99, 0x63, 0x60, 0x60, 0x60, 0x60, 0x00, 0x00,
		0x00, 0x05, 0x00, 0x01, 0x87, 0xA1, 0x4E, 0xD4, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
		0x42, 0x60, 0x82},
	ContentType: "image/png",
}

type TrackingPixel struct {
	Content     []byte `json:"content,omitempty"`
	ContentType string `json:"content_type,omitempty"`
}

type eventEndpoint struct {
	Accounts      stored_requests.AccountFetcher
	Analytics     analytics.PBSAnalyticsModule
	Cfg           *config.Configuration
	TrackingPixel *TrackingPixel
}

func NewEventEndpoint(cfg *config.Configuration, accounts stored_requests.AccountFetcher, analytics analytics.PBSAnalyticsModule) httprouter.Handle {
	ee := &eventEndpoint{
		Accounts:      accounts,
		Analytics:     analytics,
		Cfg:           cfg,
		TrackingPixel: trackingPixelPng,
	}

	return ee.Handle
}

func (e *eventEndpoint) Handle(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// parse event request from http req
	eventRequest, err := ParseEventRequest(r)

	// handle possible parsing errors
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("invalid request: %s\n", err.Error())))

		return
	}

	// validate account id
	accountId, err := validateRequiredParameter(r, AccountIdParameter)

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(fmt.Sprintf("Account '%s' is required query parameter and can't be empty", AccountIdParameter)))
		return
	}
	eventRequest.AccountID = accountId

	if eventRequest.Analytics != analytics.Enabled {
		return
	}

	ctx := context.Background()

	// get account details
	account, errs := GetAccount(ctx, e.Cfg, e.Accounts, eventRequest.AccountID)
	if len(errs) > 0 {
		w.WriteHeader(http.StatusInternalServerError)
		for _, err := range errs {
			w.Write([]byte(fmt.Sprintf("Internal Error: %s\n", err.Error())))
		}

		return
	}

	// account does not support events
	if !account.EventsEnabled {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(fmt.Sprintf("Account '%s' doesn't support events", eventRequest.AccountID)))
		return
	}

	// handle notification event
	e.Analytics.LogNotificationEventObject(&analytics.NotificationEvent{
		Request: eventRequest,
		Account: account,
	})

	// OK
	w.WriteHeader(http.StatusOK)
	// Add tracking pixel if format == image
	if eventRequest.Format == analytics.Image {
		w.Header().Add("Content-Type", e.TrackingPixel.ContentType)
		w.Write(e.TrackingPixel.Content)
	}
}

/**
 * Converts an EventRequest to an URL
 */
func EventRequestToUrl(externalUrl string, request *analytics.EventRequest) string {
	s := fmt.Sprintf(TemplateUrl, externalUrl, request.Type, request.Bidid, request.AccountID)

	return s + optionalParameters(request)
}

/**
 * Parses an EventRequest from an Http request
 */
func ParseEventRequest(r *http.Request) (*analytics.EventRequest, error) {
	event := &analytics.EventRequest{}

	// validate type
	if err := validateType(event, r); err != nil {
		return event, err
	}

	// validate bidid
	if bidid, err := validateRequiredParameter(r, BidIdParameter); err != nil {
		return event, err
	} else {
		event.Bidid = bidid
	}

	// validate timestamp (optional)
	if err := validateTimestamp(event, r); err != nil {
		return event, err
	}

	// validate format (optional)
	if err := validateFormat(event, r); err != nil {
		return event, err
	}

	// validate analytics (optional)
	if err := validateAnalytics(event, r); err != nil {
		return event, err
	}

	// Bidder
	bidder := r.FormValue(BidderParameter)

	if bidder != "" {
		event.Bidder = bidder
	}

	return event, nil
}

func optionalParameters(request *analytics.EventRequest) string {
	r := url.Values{}

	// timestamp
	if request.Timestamp > 0 {
		r.Add(TimestampParameter, strconv.FormatInt(request.Timestamp, 10))
	}

	// bidder
	if request.Bidder != "" {
		r.Add(BidderParameter, request.Bidder)
	}

	// format
	switch request.Format {
	case analytics.Blank:
		r.Add(FormatParameter, string(analytics.Blank))
	case analytics.Image:
		r.Add(FormatParameter, string(analytics.Image))
	}

	//analytics
	switch request.Analytics {
	case analytics.Enabled:
		r.Add(AnalyticsParameter, string(analytics.Enabled))
	case analytics.Disabled:
		r.Add(AnalyticsParameter, string(analytics.Disabled))
	}

	opt := r.Encode()

	if opt != "" {
		return "&" + opt
	}

	return opt
}

/**
 * validate type
 */
func validateType(er *analytics.EventRequest, httpRequest *http.Request) error {
	t, err := validateRequiredParameter(httpRequest, TypeParameter)

	if err != nil {
		return err
	}

	switch t {
	case string(analytics.Imp):
		er.Type = analytics.Imp
		return nil
	case string(analytics.Win):
		er.Type = analytics.Win
		return nil
	default:
		return &errortypes.BadInput{Message: fmt.Sprintf("unknown type: '%s'", t)}
	}
}

/**
 * validate format
 */
func validateFormat(er *analytics.EventRequest, httpRequest *http.Request) error {
	f := httpRequest.FormValue(FormatParameter)

	if f != "" {
		switch f {
		case string(analytics.Blank):
			er.Format = analytics.Blank
			return nil
		case string(analytics.Image):
			er.Format = analytics.Image
			return nil
		default:
			return &errortypes.BadInput{Message: fmt.Sprintf("unknown format: '%s'", f)}
		}
	}

	return nil
}

/**
 * validate analytics
 */
func validateAnalytics(er *analytics.EventRequest, httpRequest *http.Request) error {
	a := httpRequest.FormValue(AnalyticsParameter)

	if a != "" {
		switch a {
		case string(analytics.Enabled):
			er.Analytics = analytics.Enabled
			return nil
		case string(analytics.Disabled):
			er.Analytics = analytics.Disabled
			return nil
		default:
			return &errortypes.BadInput{Message: fmt.Sprintf("unknown analytics: '%s'", a)}
		}
	}

	return nil
}

/**
 * validate timestamp
 */
func validateTimestamp(er *analytics.EventRequest, httpRequest *http.Request) error {
	t := httpRequest.FormValue(TimestampParameter)

	if t != "" {
		ts, err := strconv.ParseInt(t, 10, 64)

		if err != nil {
			return &errortypes.BadInput{Message: fmt.Sprintf("invalid request: error parsing timestamp '%s'", t)}
		}

		er.Timestamp = ts
		return nil
	}

	return nil
}

/**
 * validate required parameter
 */
func validateRequiredParameter(httpRequest *http.Request, parameter string) (string, error) {
	t := httpRequest.FormValue(parameter)

	if t == "" {
		return "", &errortypes.BadInput{Message: fmt.Sprintf("parameter '%s' is required", parameter)}
	}

	return t, nil
}

/**
 * Check if []error contains a NotFoundError
 */
func accountNotFound(errs []error) bool {
	for _, el := range errs {
		if _, ok := el.(stored_requests.NotFoundError); ok {
			return true
		}
	}
	return false
}
