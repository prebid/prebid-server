package analytics

/*
  	PBSAnalyticsModule must be implemented by any analytics module that does transactional logging.

	New modules can use the /analytics/endpoint_data_objects, extract the
	information required and are responsible for handling all their logging activities inside LogAuctionObject, LogAmpObject
	LogCookieSyncObject and LogSetUIDObject method implementations.
*/

type PBSAnalyticsModule interface {
	LogAuctionObject(*AuctionObject)
	LogCookieSyncObject(*CookieSyncObject)
	LogSetUIDObject(*SetUIDObject)
	LogAmpObject(*AmpObject)
}
