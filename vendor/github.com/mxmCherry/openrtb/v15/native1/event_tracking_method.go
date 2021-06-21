package native1

// 7.7 Event Tracking Methods Table
type EventTrackingMethod int64

const (
	EventTrackingMethodImage EventTrackingMethod = 1 // Image-pixel tracking - URL provided will be inserted as a 1x1 pixel at the time of the event.
	EventTrackingMethodJS    EventTrackingMethod = 2 // Javascript-based tracking - URL provided will be inserted as a js tag at the time of the event.

	// 500+ exchangespecific
	// Could include custom measurement companies such as moat, doubleverify, IAS, etc - in this case additional elements will often be passed
)
