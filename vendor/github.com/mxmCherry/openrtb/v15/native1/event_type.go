package native1

// 7.6 Event Types Table
type EventType int64

const (
	EventTypeImpression      EventType = 1 // Impression
	EventTypeViewableMRC50   EventType = 2 // Visible impression using MRC definition at 50% in view for 1 second
	EventTypeViewableMRC100  EventType = 3 // 100% in view for 1 second (ie GroupM standard)
	EventTypeViewableVideo50 EventType = 4 // Visible impression for video using MRC definition at 50% in view for 2 seconds

	// 500+ exchange-specific
)
