package analytics

// EventType enumerates the values of events Prebid Server can receive for an ad.
type EventType string

// Possible values of events Prebid Server can receive for an ad.
const (
	Win  EventType = "win"
	Imp  EventType = "imp"
	Vast EventType = "vast"
)

// ResponseFormat enumerates the values of a Prebid Server event.
type ResponseFormat string

const (
	// Blank describes an event which returns an HTTP 200 with an empty body.
	Blank ResponseFormat = "b"
	// Image describes an event which returns an HTTP 200 with a PNG body.
	Image ResponseFormat = "i"
)

// VastType enumerates the values of vast type events Prebid Server can receive
type VastType string

// Possible value of VastType event prebid server can receive.
const (
	Start         VastType = "start"
	FirstQuartile VastType = "firstQuartile"
	MidPoint      VastType = "midPoint"
	ThirdQuartile VastType = "thirdQuartile"
	Complete      VastType = "complete"
)

// Analytics indicates if the notification event should be handled or not
type Analytics string

const (
	Enabled  Analytics = "1"
	Disabled Analytics = "0"
)

type EventRequest struct {
	Type        EventType      `json:"type,omitempty"`
	Format      ResponseFormat `json:"format,omitempty"`
	Analytics   Analytics      `json:"analytics,omitempty"`
	BidID       string         `json:"bidid,omitempty"`
	AccountID   string         `json:"account_id,omitempty"`
	Bidder      string         `json:"bidder,omitempty"`
	Timestamp   int64          `json:"timestamp,omitempty"`
	Integration string         `json:"integration,omitempty"`
	VType       VastType       `json:"vtype,omitempty"`
}
