package events

import (
	"context"
	"errors"
	"fmt"
	"github.com/julienschmidt/httprouter"
	accountService "github.com/prebid/prebid-server/account"
	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/stored_requests"
	"net/http"
	"net/url"
	"strconv"
	"time"
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

var trackingPixelPng = &trackingPixel{
	Content: []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44,
		0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
		0x89, 0x00, 0x00, 0x00, 0x04, 0x73, 0x42, 0x49, 0x54, 0x08, 0x08, 0x08, 0x08, 0x7C, 0x08, 0x64, 0x88,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x44, 0x41, 0x54, 0x08, 0x99, 0x63, 0x60, 0x60, 0x60, 0x60, 0x00, 0x00,
		0x00, 0x05, 0x00, 0x01, 0x87, 0xA1, 0x4E, 0xD4, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
		0x42, 0x60, 0x82},
	ContentType: "image/png",
}

type trackingPixel struct {
	Content     []byte `json:"content,omitempty"`
	ContentType string `json:"content_type,omitempty"`
}

type eventEndpoint struct {
	Accounts      stored_requests.AccountFetcher
	Analytics     analytics.PBSAnalyticsModule
	Cfg           *config.Configuration
	TrackingPixel *trackingPixel
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
	eventRequest, errs := ParseEventRequest(r)

	// handle possible parsing errors
	if len(errs) > 0 {
		w.WriteHeader(http.StatusBadRequest)

		for _, err := range errs {
			w.Write([]byte(fmt.Sprintf("invalid request: %s\n", err.Error())))
		}

		return
	}

	// validate account id
	accountId, err := checkRequiredParameter(r, AccountIdParameter)

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(fmt.Sprintf("Account '%s' is required query parameter and can't be empty", AccountIdParameter)))
		return
	}
	eventRequest.AccountID = accountId

	if eventRequest.Analytics != analytics.Enabled {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	ctx := context.Background()
	if e.Cfg.Event.TimeoutMS > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(e.Cfg.Event.TimeoutMS)*time.Millisecond)
		defer cancel()
	}

	// get account details
	account, errs := accountService.GetAccount(ctx, e.Cfg, e.Accounts, eventRequest.AccountID)
	if len(errs) > 0 {
		status, messages := HandleAccountServiceErrors(errs)
		w.WriteHeader(status)

		for _, message := range messages {
			w.Write([]byte(fmt.Sprintf("Invalid request: %s\n", message)))
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

	// Add tracking pixel if format == image
	if eventRequest.Format == analytics.Image {
		w.WriteHeader(http.StatusOK)
		w.Header().Add("Content-Type", e.TrackingPixel.ContentType)
		w.Write(e.TrackingPixel.Content)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// EventRequestToUrl converts an analytics.EventRequest to an URL
func EventRequestToUrl(externalUrl string, request *analytics.EventRequest) string {
	s := fmt.Sprintf(TemplateUrl, externalUrl, request.Type, request.BidID, request.AccountID)

	return s + optionalParameters(request)
}

// ParseEventRequest parses an analytics.EventRequest from an Http request
func ParseEventRequest(r *http.Request) (*analytics.EventRequest, []error) {
	event := &analytics.EventRequest{}
	var errs []error
	// validate type
	if err := readType(event, r); err != nil {
		errs = append(errs, err)
	}

	// validate bidid
	if bidid, err := checkRequiredParameter(r, BidIdParameter); err != nil {
		errs = append(errs, err)
	} else {
		event.BidID = bidid
	}

	// validate timestamp (optional)
	if err := readTimestamp(event, r); err != nil {
		errs = append(errs, err)
	}

	// validate format (optional)
	if err := readFormat(event, r); err != nil {
		errs = append(errs, err)
	}

	// validate analytics (optional)
	if err := readAnalytics(event, r); err != nil {
		errs = append(errs, err)
	}

	// Bidder
	event.Bidder = r.URL.Query().Get(BidderParameter)

	return event, errs
}

// HandleAccountServiceErrors handles account.GetAccount errors
func HandleAccountServiceErrors(errs []error) (status int, messages []string) {
	messages = []string{}
	status = http.StatusBadRequest

	for _, er := range errs {
		if errors.Is(er, context.DeadlineExceeded) {
			er = &errortypes.Timeout{
				Message: er.Error(),
			}
		}

		messages = append(messages, er.Error())

		errCode := errortypes.ReadCode(er)

		if errCode == errortypes.BlacklistedAppErrorCode || errCode == errortypes.BlacklistedAcctErrorCode {
			status = http.StatusServiceUnavailable
		}

		if errCode == errortypes.TimeoutErrorCode && status == http.StatusBadRequest {
			status = http.StatusGatewayTimeout
		}
	}

	return status, messages
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

// readType validates analytics.EventRequest type
func readType(er *analytics.EventRequest, httpRequest *http.Request) error {
	t, err := checkRequiredParameter(httpRequest, TypeParameter)

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

// readFormat validates analytics.EventRequest format attribute
func readFormat(er *analytics.EventRequest, httpRequest *http.Request) error {
	f := httpRequest.URL.Query().Get(FormatParameter)

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

// readAnalytics validates analytics.EventRequest analytics attribute
func readAnalytics(er *analytics.EventRequest, httpRequest *http.Request) error {
	a := httpRequest.URL.Query().Get(AnalyticsParameter)

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

	er.Analytics = analytics.Enabled
	return nil
}

// readTimestamp validates analytics.EventRequest timestamp attribute
func readTimestamp(er *analytics.EventRequest, httpRequest *http.Request) error {
	t := httpRequest.URL.Query().Get(TimestampParameter)

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

// checkRequiredParameter checks if http.Request contains all required parameters
func checkRequiredParameter(httpRequest *http.Request, parameter string) (string, error) {
	t := httpRequest.URL.Query().Get(parameter)

	if t == "" {
		return "", &errortypes.BadInput{Message: fmt.Sprintf("parameter '%s' is required", parameter)}
	}

	return t, nil
}
