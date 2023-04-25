package enums

// RequestType : Request type enumeration
type RequestType string

// The request types (endpoints)
const (
	ReqTypeORTB2Web RequestType = "openrtb2-web"
	ReqTypeORTB2App RequestType = "openrtb2-app"
	ReqTypeAMP      RequestType = "amp"
	ReqTypeVideo    RequestType = "video"
)

func RequestTypes() []RequestType {
	return []RequestType{
		ReqTypeORTB2Web,
		ReqTypeORTB2App,
		ReqTypeAMP,
		ReqTypeVideo,
	}
}
