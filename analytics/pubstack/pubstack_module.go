package pubstack

import (
	"fmt"

	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/analytics/pubstack/forwarder"
	"github.com/prebid/prebid-server/analytics/pubstack/parser"
)

type RequestType string

const (
	COOKIE_SYNC RequestType = "/cookie_sync"
	AUCTION     RequestType = "/openrtb2/auction"
	VIDEO       RequestType = "/openrtb2/video"
	SETUID      RequestType = "/set_uid"
	AMP         RequestType = "/openrtb2/amp"
)

//Module that can perform transactional logging
type PubstackModule struct {
	parser    *parser.Parser
	forwarder *forwarder.Forwarder
}

//Writes AuctionObject to file
func (pf *PubstackModule) LogAuctionObject(ao *analytics.AuctionObject) {
	pf.parser.Feed(ao.Request, ao.Response)
	fmt.Println("Receiving an auction")
}

//Writes VideoObject to file
func (pf *PubstackModule) LogVideoObject(vo *analytics.VideoObject) {
	fmt.Println("This function is not yet implemented")
}

//Logs SetUIDObject to file
func (pf *PubstackModule) LogSetUIDObject(so *analytics.SetUIDObject) {
	fmt.Println("This function is not yet implemented")
	//Code to parse the object and log in a way required
}

//Logs CookieSyncObject to file
func (pf *PubstackModule) LogCookieSyncObject(cso *analytics.CookieSyncObject) {
	fmt.Println("This function is not yet implemented")
	//Code to parse the object and log in a way required
}

//Logs AmpObject to file
func (pf *PubstackModule) LogAmpObject(ao *analytics.AmpObject) {
	fmt.Println("This function is not yet implemented")
}

//Method to initialize the analytic module
func NewPubstackModule(scopeid, intake string) (analytics.PBSAnalyticsModule, error) {
	fmt.Println("[PBSTCK] Initializing listener ...")
	return &PubstackModule{
		parser:    parser.NewParser(scopeid),
		forwarder: forwarder.NewForwarder(intake),
	}, nil
}
