package helpers

type PageViewRecord struct {
	SessionID string `json:"sessionID"`

	Lcp float64 `json:"lcp"`

	Inp float64 `json:"inp"`

	Cls float64 `json:"cls"`

	EventType string `json:"eventType"`

	PageViewID string `json:"pageViewID"`

	Device string `json:"device"`

	Ua string `json:"ua"`

	City string `json:"city"`

	State string `json:"state"`

	Country string `json:"country"`

	Page string `json:"page"`

	AfihbsVersion string `json:"afihbsVersion"`

	YetiSiteID int64 `json:"yetiSiteID"`

	YetiSiteUID string `json:"yetiSiteUID"`

	YetiSiteName string `json:"yetiSiteName"`

	YetiPublisherID int64 `json:"yetiPublisherID"`

	YetiPublisherUID string `json:"yetiPublisherUID"`

	YetiPublisherName string `json:"yetiPublisherName"`

	ServerTimestamp int64 `json:"serverTimestamp"`

	InsertedAt int64 `json:"insertedAt"`

	Uuid string `json:"uuid"`
}
