package agma

import (
	"fmt"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type EventType string

const (
	EventTypeAuction EventType = "auction"
	EventTypeAmp     EventType = "amp"
	EventTypeVideo   EventType = "video"
)

type logObject struct {
	EventType   EventType        `json:"type"`
	RequestId   string           `json:"id"`
	AccountCode string           `json:"code"`
	Site        *openrtb2.Site   `json:"site,omitempty"`
	App         *openrtb2.App    `json:"app,omitempty"`
	Device      *openrtb2.Device `json:"device,omitempty"`
	User        *openrtb2.User   `json:"user,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
}

func serializeAnayltics(
	requestwrapper *openrtb_ext.RequestWrapper,
	eventType EventType,
	code string,
	createdAt time.Time,
) ([]byte, error) {
	if requestwrapper == nil || requestwrapper.BidRequest == nil {
		return nil, fmt.Errorf("requestwrapper or BidRequest object nil")
	}
	return jsonutil.Marshal(&logObject{
		EventType:   eventType,
		RequestId:   requestwrapper.ID,
		AccountCode: code,
		Site:        requestwrapper.BidRequest.Site,
		App:         requestwrapper.BidRequest.App,
		Device:      requestwrapper.BidRequest.Device,
		User:        requestwrapper.BidRequest.User,
		CreatedAt:   createdAt,
	})
}
