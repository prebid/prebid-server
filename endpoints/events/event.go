package events

import (
	"context"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/stored_requests"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// EventType enumerates the values of events Prebid Server can receive for an ad.
type EventType string

// Possible values of  vents Prebid Server can receive for an ad.
const (
	Win EventType = "win"
	Imp EventType = "imp"
)

// EventFormat enumerates the values of a Prebid Server event.
type Format string

const (
	// BlankEventFormat describes an event which returns an HTTP 200 with an empty body.
	Blank Format = "b"
	// ImageEventFormat describes an event which returns an HTTP 200 with a PNG body.
	Image Format = "i"
)

// Indicates if the notification event should be handled or not
type Analytics string

const (
	Enabled  Analytics = "1"
	Disabled Analytics = "0"
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

type EventRequest struct {
	Type      EventType `json:"type,omitempty"`
	Format    Format    `json:"format,omitempty"`
	Analytics Analytics `json:"analytics,omitempty"`
	Bidid     string    `json:"bidid,omitempty"`
	AccountID string    `json:"account_id,omitempty"`
	Bidder    string    `json:"bidder,omitempty"`
	Timestamp int64     `json:"timestamp,omitempty"`
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

	if eventRequest.Analytics != Enabled {
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
		Type:      string(eventRequest.Type),
		Timestamp: time.Unix(eventRequest.Timestamp, 0),
		Bidid:     eventRequest.Bidid,
		Bidder:    eventRequest.Bidder,
		Account:   account,
	})

	// OK
	w.WriteHeader(http.StatusOK)
	// Add tracking pixel if format == image
	if eventRequest.Format == Image {
		w.Header().Add("Content-Type", e.TrackingPixel.ContentType)
		w.Write(e.TrackingPixel.Content)
	}
}

/**
 * Converts an EventRequest to an URL
 */
func EventRequestToUrl(externalUrl string, request *EventRequest) string {
	s := fmt.Sprintf(TemplateUrl, externalUrl, request.Type, request.Bidid, request.AccountID)

	return s + optionalParameters(request)
}

/**
 * Parses an EventRequest from an Http request
 */
func ParseEventRequest(r *http.Request) (*EventRequest, error) {
	er := &EventRequest{}

	// validate type
	if err := validateType(er, r); err != nil {
		return er, err
	}

	// validate bidid
	if bidid, err := validateRequiredParameter(r, BidIdParameter); err != nil {
		return er, err
	} else {
		er.Bidid = bidid
	}

	// validate timestamp (optional)
	if err := validateTimestamp(er, r); err != nil {
		return er, err
	}

	// validate format (optional)
	if err := validateFormat(er, r); err != nil {
		return er, err
	}

	// validate analytics (optional)
	if err := validateAnalytics(er, r); err != nil {
		return er, err
	}

	// Bidder
	bidder := r.FormValue(BidderParameter)

	if bidder != "" {
		er.Bidder = bidder
	}

	return er, nil
}

func optionalParameters(request *EventRequest) string {
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
	case Blank:
		r.Add(FormatParameter, string(Blank))
	case Image:
		r.Add(FormatParameter, string(Image))
	}

	//analytics
	switch request.Analytics {
	case Enabled:
		r.Add(AnalyticsParameter, string(Enabled))
	case Disabled:
		r.Add(AnalyticsParameter, string(Disabled))
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
func validateType(er *EventRequest, httpRequest *http.Request) error {
	t, err := validateRequiredParameter(httpRequest, TypeParameter)

	if err != nil {
		return err
	}

	switch t {
	case string(Imp):
		er.Type = Imp
		return nil
	case string(Win):
		er.Type = Win
		return nil
	default:
		return fmt.Errorf("unknown type: '%s'", t)
	}
}

/**
 * validate format
 */
func validateFormat(er *EventRequest, httpRequest *http.Request) error {
	f := httpRequest.FormValue(FormatParameter)

	if f != "" {
		switch f {
		case string(Blank):
			er.Format = Blank
			return nil
		case string(Image):
			er.Format = Image
			return nil
		default:
			return fmt.Errorf("unknown format: '%s'", f)
		}
	}

	return nil
}

/**
 * validate analytics
 */
func validateAnalytics(er *EventRequest, httpRequest *http.Request) error {
	a := httpRequest.FormValue(AnalyticsParameter)

	if a != "" {
		switch a {
		case string(Enabled):
			er.Analytics = Enabled
			return nil
		case string(Disabled):
			er.Analytics = Disabled
			return nil
		default:
			return fmt.Errorf("unknown analytics: '%s'", a)
		}
	}

	return nil
}

/**
 * validate timestamp
 */
func validateTimestamp(er *EventRequest, httpRequest *http.Request) error {
	t := httpRequest.FormValue(TimestampParameter)

	if t != "" {
		ts, err := strconv.ParseInt(t, 10, 64)

		if err != nil {
			return fmt.Errorf("invalid request: error parsing timestamp '%s'", t)
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
		return "", fmt.Errorf("parameter '%s' is required", parameter)
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
