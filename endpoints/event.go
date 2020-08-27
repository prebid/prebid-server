package endpoints

import (
	"fmt"
	"strconv"
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
)

type EventRequest struct {
	Type      Type      `json:"type"`
	Bidid     string    `json:"bidid"`
	AccountId string    `json:"account_id"`
	Bidder    string    `json:"bidder"`
	Timestamp int64     `json:"timestamp"`
	Format    Format    `json:"format"`
	Analytics Analytics `json:"analytics"`
}

func EventRequestToUrl(externalUrl string, request *EventRequest) string {
	s := fmt.Sprintf(TEMPLATE_URL, externalUrl, request.Type, request.Bidid, request.AccountId)

	return s + optionalParameters(request)
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
