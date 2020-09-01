package endpoints

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"fmt"
	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/cache"
	"net/http"
	"strconv"
	"strings"
)

type Type string
type Format string
type Analytics string

const (
	WIN Type = "win"
	IMP Type = "imp"

	BLANK Format = "b"
	IMAGE Format = "i"

	ENABLED  Analytics = "1"
	DISABLED Analytics = "0"

	// Required
	TEMPLATE_URL         = "%v/event?t=%v&b=%v&a=%v"
	TYPE_PARAMETER       = "t"
	BID_ID_PARAMETER     = "b"
	ACCOUNT_ID_PARAMETER = "a"

	// Optional
	BIDDER_PARAMETER    = "bidder"
	TIMESTAMP_PARAMETER = "ts"
	FORMAT_PARAMETER    = "f"
	ANALYTICS_PARAMETER = "x"

	TRACKING_PIXEL_PNG              = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAABHNCSVQICAgIfAhkiAAAAA1JREFUCJljYGBgYAAAAAUAAYehTtQAAAAASUVORK5CYII="
	TRACKING_PIXEL_PNG_CONTENT_TYPE = "image/png"
)

type EventRequest struct {
	Type      Type      `json:"type,omitempty"`
	Bidid     string    `json:"bidid,omitempty"`
	AccountId string    `json:"account_id,omitempty"`
	Bidder    string    `json:"bidder,omitempty"`
	Timestamp int64     `json:"timestamp,omitempty"`
	Format    Format    `json:"format,omitempty"`
	Analytics Analytics `json:"analytics,omitempty"`
}

type TrackingPixel struct {
	Content     []byte `json:"content,omitempty"`
	ContentType string `json:"content_type,omitempty"`
}

type eventEndpoint struct {
	DataCache     cache.Cache
	Analytics     analytics.PBSAnalyticsModule
	TrackingPixel *TrackingPixel
}

func NewEventEndpoint(dataCache cache.Cache, analytics analytics.PBSAnalyticsModule) httprouter.Handle {
	ee := &eventEndpoint{
		DataCache:     dataCache,
		Analytics:     analytics,
		TrackingPixel: loadTrackingPixel(),
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
	accountId, err := validateRequiredParameter(r, ACCOUNT_ID_PARAMETER)

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(fmt.Sprintf("Account '%s' is required query parameter and can't be empty", ACCOUNT_ID_PARAMETER)))
		return
	}
	eventRequest.AccountId = accountId

	if eventRequest.Analytics == ENABLED {

		// get account details
		account, err := e.DataCache.Accounts().Get(eventRequest.AccountId)
		if err != nil {
			if err == sql.ErrNoRows {
				account = &cache.Account{
					ID:            accountId,
					EventsEnabled: false,
				}
			} else {
				if glog.V(2) {
					glog.Infof("Invalid account id: %v", err)
				}

				status := http.StatusInternalServerError
				message := fmt.Sprintf("Invalid request: %s\n", err.Error())

				w.WriteHeader(status)
				w.Write([]byte(message))
				return
			}
		}

		// account does not support events
		if !account.EventsEnabled {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(fmt.Sprintf("Account '%s' doesn't support events", eventRequest.AccountId)))
			return
		}

		// handle notification event
		e.Analytics.LogNotificationEventObject(&analytics.NotificationEvent{
			Type:      string(eventRequest.Type),
			Timestamp: eventRequest.Timestamp,
			Bidid:     eventRequest.Bidid,
			Bidder:    eventRequest.Bidder,
			Account:   account,
		})

		// OK
		w.WriteHeader(http.StatusOK)
		// Add tracking pixel if format == image
		if eventRequest.Format == IMAGE {
			w.Header().Add("Content-Type", TRACKING_PIXEL_PNG_CONTENT_TYPE)
			w.Write(e.TrackingPixel.Content)
		}

		return
	}
}

/**
 * Converts an EventRequest to an URL
 */
func EventRequestToUrl(externalUrl string, request *EventRequest) string {
	s := fmt.Sprintf(TEMPLATE_URL, externalUrl, request.Type, request.Bidid, request.AccountId)

	return s + optionalParameters(request)
}

/**
 * Parses an EventRequest from an Http request
 */
func ParseEventRequest(r *http.Request) (*EventRequest, error) {
	er := &EventRequest{}

	// validate type
	err := validateType(er, r)

	if err != nil {
		return er, err
	}

	// validate bidid
	bidid, err := validateRequiredParameter(r, BID_ID_PARAMETER)

	if err != nil {
		return er, err
	}
	er.Bidid = bidid

	// validate timestamp (optional)
	err = validateTimestamp(er, r)

	if err != nil {
		return er, err
	}

	// validate format (optional)
	err = validateFormat(er, r)

	if err != nil {
		return er, err
	}

	// validate analytics (optional)
	err = validateAnalytics(er, r)

	if err != nil {
		return er, err
	}

	// Bidder
	bidder := r.FormValue(BIDDER_PARAMETER)

	if bidder != "" {
		er.Bidder = bidder
	}

	return er, nil
}

func optionalParameters(request *EventRequest) string {
	r := ""

	// timestamp
	if request.Timestamp > 0 {
		r = r + nameValueAsQueryString(TIMESTAMP_PARAMETER, strconv.FormatInt(request.Timestamp, 10))
	}

	// bidder
	if request.Bidder != "" {
		r = r + nameValueAsQueryString(BIDDER_PARAMETER, request.Bidder)
	}

	// format
	switch request.Format {
	case BLANK:
		r = r + nameValueAsQueryString(FORMAT_PARAMETER, string(BLANK))
	case IMAGE:
		r = r + nameValueAsQueryString(FORMAT_PARAMETER, string(IMAGE))
	}

	//analytics
	switch request.Analytics {
	case ENABLED:
		r = r + nameValueAsQueryString(ANALYTICS_PARAMETER, string(ENABLED))
	case DISABLED:
		r = r + nameValueAsQueryString(ANALYTICS_PARAMETER, string(DISABLED))
	}

	return r
}

func nameValueAsQueryString(name string, value string) string {
	if value == "" {
		return ""
	}

	return "&" + name + "=" + value
}

/**
 * validate type
 */
func validateType(er *EventRequest, httpRequest *http.Request) error {
	t, err := validateRequiredParameter(httpRequest, TYPE_PARAMETER)

	if err != nil {
		return err
	}

	switch t {
	case string(IMP):
		er.Type = IMP
		return nil
	case string(WIN):
		er.Type = WIN
		return nil
	default:
		return fmt.Errorf("unknown type: '%s'", t)
	}
}

/**
 * validate format
 */
func validateFormat(er *EventRequest, httpRequest *http.Request) error {
	f := httpRequest.FormValue(FORMAT_PARAMETER)

	if f != "" {
		switch f {
		case string(BLANK):
			er.Format = BLANK
			return nil
		case string(IMAGE):
			er.Format = IMAGE
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
	a := httpRequest.FormValue(ANALYTICS_PARAMETER)

	if a != "" {
		switch a {
		case string(ENABLED):
			er.Analytics = ENABLED
			return nil
		case string(DISABLED):
			er.Analytics = DISABLED
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
	t := httpRequest.FormValue(TIMESTAMP_PARAMETER)

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
 * Load tracking pixel from Base64 string
 */
func loadTrackingPixel() *TrackingPixel {
	reader := base64.NewDecoder(base64.StdEncoding, strings.NewReader(TRACKING_PIXEL_PNG))
	buff := bytes.Buffer{}
	_, err := buff.ReadFrom(reader)
	if err != nil {
		panic(err)
	}

	return &TrackingPixel{
		Content:     buff.Bytes(),
		ContentType: TRACKING_PIXEL_PNG_CONTENT_TYPE,
	}
}
