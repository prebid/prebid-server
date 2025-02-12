package events

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
	"unicode"

	"github.com/prebid/prebid-server/v3/openrtb_ext"

	"github.com/julienschmidt/httprouter"
	accountService "github.com/prebid/prebid-server/v3/account"
	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/privacy"
	"github.com/prebid/prebid-server/v3/stored_requests"
	"github.com/prebid/prebid-server/v3/util/httputil"
)

const (
	// Required
	TemplateUrl        = "%v/event?t=%v&b=%v&a=%v"
	TypeParameter      = "t"
	VTypeParameter     = "vtype"
	BidIdParameter     = "b"
	AccountIdParameter = "a"

	// Optional
	BidderParameter          = "bidder"
	TimestampParameter       = "ts"
	FormatParameter          = "f"
	AnalyticsParameter       = "x"
	IntegrationTypeParameter = "int"
)

const integrationParamMaxLength = 64

type eventEndpoint struct {
	Accounts      stored_requests.AccountFetcher
	Analytics     analytics.Runner
	Cfg           *config.Configuration
	TrackingPixel *httputil.Pixel
	MetricsEngine metrics.MetricsEngine
}

func NewEventEndpoint(cfg *config.Configuration, accounts stored_requests.AccountFetcher, analytics analytics.Runner, me metrics.MetricsEngine) httprouter.Handle {
	ee := &eventEndpoint{
		Accounts:      accounts,
		Analytics:     analytics,
		Cfg:           cfg,
		TrackingPixel: &httputil.Pixel1x1PNG,
		MetricsEngine: me,
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
			fmt.Fprintf(w, "invalid request: %s\n", err.Error())
		}

		return
	}

	// validate account id
	accountId, err := checkRequiredParameter(r, AccountIdParameter)

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "Account '%s' is required query parameter and can't be empty", AccountIdParameter)
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
	account, errs := accountService.GetAccount(ctx, e.Cfg, e.Accounts, eventRequest.AccountID, e.MetricsEngine)
	if len(errs) > 0 {
		status, messages := HandleAccountServiceErrors(errs)
		w.WriteHeader(status)

		for _, message := range messages {
			fmt.Fprintf(w, "Invalid request: %s\n", message)
		}
		return
	}

	// Check if events are enabled for the account
	if !account.Events.Enabled {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "Account '%s' doesn't support events", eventRequest.AccountID)
		return
	}

	activities := privacy.NewActivityControl(&account.Privacy)

	// handle notification event
	e.Analytics.LogNotificationEventObject(&analytics.NotificationEvent{
		Request: eventRequest,
		Account: account,
	}, activities)

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

	if event.Type == analytics.Vast {
		if err := readVType(event, r); err != nil {
			errs = append(errs, err)
		}
	} else {
		if t := r.URL.Query().Get(VTypeParameter); t != "" {
			errs = append(errs, &errortypes.BadInput{Message: "parameter 'vtype' is only required for t=vast"})
		}
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

	if err := readIntegrationType(event, r); err != nil {
		errs = append(errs, err)
	}

	// Bidder
	bidderName := r.URL.Query().Get(BidderParameter)
	if normalisedBidderName, ok := openrtb_ext.NormalizeBidderName(bidderName); ok {
		bidderName = normalisedBidderName.String()
	}

	event.Bidder = bidderName

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

		if errCode == errortypes.BlockedAppErrorCode || errCode == errortypes.AccountDisabledErrorCode {
			status = http.StatusServiceUnavailable
		}
		if errCode == errortypes.MalformedAcctErrorCode {
			status = http.StatusInternalServerError
		}
		if errCode == errortypes.TimeoutErrorCode && status == http.StatusBadRequest {
			status = http.StatusGatewayTimeout
		}
	}

	return
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

	if request.Integration != "" {
		r.Add(IntegrationTypeParameter, request.Integration)
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
	case string(analytics.Vast):
		er.Type = analytics.Vast
		return nil
	default:
		return &errortypes.BadInput{Message: fmt.Sprintf("unknown type: '%s'", t)}
	}
}

// readVType validates analytics.EventRequest vtype
func readVType(er *analytics.EventRequest, httpRequest *http.Request) error {
	vtype, err := checkRequiredParameter(httpRequest, VTypeParameter)

	if err != nil {
		return err
	}

	switch vtype {
	case string(analytics.Start):
		er.VType = analytics.Start
	case string(analytics.FirstQuartile):
		er.VType = analytics.FirstQuartile
	case string(analytics.MidPoint):
		er.VType = analytics.MidPoint
	case string(analytics.ThirdQuartile):
		er.VType = analytics.ThirdQuartile
	case string(analytics.Complete):
		er.VType = analytics.Complete
	default:
		return &errortypes.BadInput{Message: fmt.Sprintf("unknown vtype: '%s'", vtype)}
	}

	return nil
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

func readIntegrationType(er *analytics.EventRequest, httpRequest *http.Request) error {
	integrationType := httpRequest.URL.Query().Get(IntegrationParameter)
	err := validateIntegrationType(integrationType)
	if err != nil {
		return err
	}
	er.Integration = integrationType
	return nil
}

func validateIntegrationType(integrationType string) error {
	if len(integrationType) > integrationParamMaxLength {
		return errors.New("integration type length is too long")
	}
	for _, char := range integrationType {
		if !unicode.IsDigit(char) && !unicode.IsLetter(char) && char != '-' && char != '_' {
			return errors.New("integration type can only contain numbers, letters and these characters '-', '_'")
		}
	}
	return nil
}
