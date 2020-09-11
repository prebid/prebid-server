package analytics

// EventType enumerates the values of events Prebid Server can receive for an ad.
type EventType string

// Possible values of  vents Prebid Server can receive for an ad.
const (
	Win EventType = "win"
	Imp EventType = "imp"
)

// EventFormat enumerates the values of a Prebid Server event.
type ResponseFormat string

const (
	// BlankEventFormat describes an event which returns an HTTP 200 with an empty body.
	Blank ResponseFormat = "b"
	// ImageEventFormat describes an event which returns an HTTP 200 with a PNG body.
	Image ResponseFormat = "i"
)

// Indicates if the notification event should be handled or not
type Analytics string

const (
	Enabled  Analytics = "1"
	Disabled Analytics = "0"
)

type EventRequest struct {
	Type      EventType      `json:"type,omitempty"`
	Format    ResponseFormat `json:"format,omitempty"`
	Analytics Analytics      `json:"analytics,omitempty"`
	Bidid     string         `json:"bidid,omitempty"`
	AccountID string         `json:"account_id,omitempty"`
	Bidder    string         `json:"bidder,omitempty"`
	Timestamp int64          `json:"timestamp,omitempty"`
}
