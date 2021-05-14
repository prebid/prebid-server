package filesystem

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/chasex/glog"
	"github.com/prebid/prebid-server/analytics"
)

type RequestType string

const (
	COOKIE_SYNC        RequestType = "/cookie_sync"
	AUCTION            RequestType = "/openrtb2/auction"
	VIDEO              RequestType = "/openrtb2/video"
	SETUID             RequestType = "/set_uid"
	AMP                RequestType = "/openrtb2/amp"
	NOTIFICATION_EVENT RequestType = "/event"
)

//Module that can perform transactional logging
type FileLogger struct {
	Logger *glog.Logger
}

//Writes AuctionObject to file
func (f *FileLogger) LogAuctionObject(ao *analytics.AuctionObject) {
	//Code to parse the object and log in a way required
	var b bytes.Buffer
	b.WriteString(jsonifyAuctionObject(ao))
	f.Logger.Debug(b.String())
	f.Logger.Flush()
}

//Writes VideoObject to file
func (f *FileLogger) LogVideoObject(vo *analytics.VideoObject) {
	//Code to parse the object and log in a way required
	var b bytes.Buffer
	b.WriteString(jsonifyVideoObject(vo))
	f.Logger.Debug(b.String())
	f.Logger.Flush()
}

//Logs SetUIDObject to file
func (f *FileLogger) LogSetUIDObject(so *analytics.SetUIDObject) {
	//Code to parse the object and log in a way required
	var b bytes.Buffer
	b.WriteString(jsonifySetUIDObject(so))
	f.Logger.Debug(b.String())
	f.Logger.Flush()
}

//Logs CookieSyncObject to file
func (f *FileLogger) LogCookieSyncObject(cso *analytics.CookieSyncObject) {
	//Code to parse the object and log in a way required
	var b bytes.Buffer
	b.WriteString(jsonifyCookieSync(cso))
	f.Logger.Debug(b.String())
	f.Logger.Flush()
}

//Logs AmpObject to file
func (f *FileLogger) LogAmpObject(ao *analytics.AmpObject) {
	if ao == nil {
		return
	}
	//Code to parse the object and log in a way required
	var b bytes.Buffer
	b.WriteString(jsonifyAmpObject(ao))
	f.Logger.Debug(b.String())
	f.Logger.Flush()
}

//Logs NotificationEvent to file
func (f *FileLogger) LogNotificationEventObject(ne *analytics.NotificationEvent) {
	if ne == nil {
		return
	}
	//Code to parse the object and log in a way required
	var b bytes.Buffer
	b.WriteString(jsonifyNotificationEventObject(ne))
	f.Logger.Debug(b.String())
	f.Logger.Flush()
}

//Method to initialize the analytic module
func NewFileLogger(filename string) (analytics.PBSAnalyticsModule, error) {
	options := glog.LogOptions{
		File:  filename,
		Flag:  glog.LstdFlags,
		Level: glog.Ldebug,
		Mode:  glog.R_Day,
	}
	if logger, err := glog.New(options); err == nil {
		return &FileLogger{
			logger,
		}, nil
	} else {
		return nil, err
	}
}

type fileAuctionObject analytics.AuctionObject

func jsonifyAuctionObject(ao *analytics.AuctionObject) string {
	type alias analytics.AuctionObject
	b, err := json.Marshal(&struct {
		Type RequestType `json:"type"`
		*alias
	}{
		Type:  AUCTION,
		alias: (*alias)(ao),
	})

	if err == nil {
		return string(b)
	} else {
		return fmt.Sprintf("Transactional Logs Error: Auction object badly formed %v", err)
	}
}

func jsonifyVideoObject(vo *analytics.VideoObject) string {
	type alias analytics.VideoObject
	b, err := json.Marshal(&struct {
		Type RequestType `json:"type"`
		*alias
	}{
		Type:  VIDEO,
		alias: (*alias)(vo),
	})

	if err == nil {
		return string(b)
	} else {
		return fmt.Sprintf("Transactional Logs Error: Video object badly formed %v", err)
	}
}

func jsonifyCookieSync(cso *analytics.CookieSyncObject) string {
	type alias analytics.CookieSyncObject

	b, err := json.Marshal(&struct {
		Type RequestType `json:"type"`
		*alias
	}{
		Type:  COOKIE_SYNC,
		alias: (*alias)(cso),
	})

	if err == nil {
		return string(b)
	} else {
		return fmt.Sprintf("Transactional Logs Error: Cookie sync object badly formed %v", err)
	}
}

func jsonifySetUIDObject(so *analytics.SetUIDObject) string {
	type alias analytics.SetUIDObject
	b, err := json.Marshal(&struct {
		Type RequestType `json:"type"`
		*alias
	}{
		Type:  SETUID,
		alias: (*alias)(so),
	})

	if err == nil {
		return string(b)
	} else {
		return fmt.Sprintf("Transactional Logs Error: Set UID object badly formed %v", err)
	}
}

func jsonifyAmpObject(ao *analytics.AmpObject) string {
	type alias analytics.AmpObject
	b, err := json.Marshal(&struct {
		Type RequestType `json:"type"`
		*alias
	}{
		Type:  AMP,
		alias: (*alias)(ao),
	})

	if err == nil {
		return string(b)
	} else {
		return fmt.Sprintf("Transactional Logs Error: Amp object badly formed %v", err)
	}
}

func jsonifyNotificationEventObject(ne *analytics.NotificationEvent) string {
	type alias analytics.NotificationEvent
	b, err := json.Marshal(&struct {
		Type RequestType `json:"type"`
		*alias
	}{
		Type:  NOTIFICATION_EVENT,
		alias: (*alias)(ne),
	})

	if err == nil {
		return string(b)
	} else {
		return fmt.Sprintf("Transactional Logs Error: NotificationEvent object badly formed %v", err)
	}
}
