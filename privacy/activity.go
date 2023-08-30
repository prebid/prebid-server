package privacy

// Activity defines privileges which can be controlled directly by the publisher or via privacy policies.
type Activity int

const (
	ActivitySyncUser Activity = iota + 1
	ActivityFetchBids
	ActivityEnrichUserFPD
	ActivityReportAnalytics
	ActivityTransmitUserFPD
	ActivityTransmitPreciseGeo
	ActivityTransmitUniqueRequestIds
	ActivityTransmitTids
)

func (a Activity) String() string {
	switch a {
	case ActivitySyncUser:
		return "syncUser"
	case ActivityFetchBids:
		return "fetchBids"
	case ActivityEnrichUserFPD:
		return "enrichUfpd"
	case ActivityReportAnalytics:
		return "reportAnalytics"
	case ActivityTransmitUserFPD:
		return "transmitUfpd"
	case ActivityTransmitPreciseGeo:
		return "transmitPreciseGeo"
	case ActivityTransmitUniqueRequestIds:
		return "transmitUniqueRequestIds"
	case ActivityTransmitTids:
		return "transmitTid"
	}

	return ""
}
