package openrtb

// 5.6 API Frameworks
//
// List of API frameworks supported by the publisher.
type APIFramework int8

const (
	APIFrameworkVPAID10 APIFramework = 1 // VPAID 1.0
	APIFrameworkVPAID20 APIFramework = 2 // VPAID 2.0
	APIFrameworkMRAID1  APIFramework = 3 // MRAID-1
	APIFrameworkORMMA   APIFramework = 4 // ORMMA
	APIFrameworkMRAID2  APIFramework = 5 // MRAID-2
	APIFrameworkMRAID3  APIFramework = 6 // MRAID-3
)
