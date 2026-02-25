// Use the OS "logrotate" daemon with copytruncate option

package filesystem

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
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

// Module that can perform transactional logging
type fileLogger struct {
	logger *log.Logger
	file   *os.File
	pool   sync.Pool
}

func (f *fileLogger) print(b *bytes.Buffer) {
	timestamp := time.Now().Format(time.DateTime)
	f.logger.Printf("[%s] %s", timestamp, b.String())
}

func (f *fileLogger) release(b *bytes.Buffer) {
	f.pool.Put(b)
}

func (f *fileLogger) getBuffer() *bytes.Buffer {
	buf := f.pool.Get().(*bytes.Buffer)
	buf.Reset()

	return buf
}

func (f *fileLogger) LogAuctionObject(ao *analytics.AuctionObject) {
	b := f.getBuffer()
	defer f.release(b)
	jsonifyAuctionObject(b, ao)
	f.print(b)
}

func (f *fileLogger) LogVideoObject(vo *analytics.VideoObject) {
	b := f.getBuffer()
	defer f.release(b)
	jsonifyVideoObject(b, vo)
	f.print(b)
}

func (f *fileLogger) LogSetUIDObject(so *analytics.SetUIDObject) {
	b := f.getBuffer()
	defer f.release(b)
	jsonifySetUIDObject(b, so)
	f.print(b)
}

func (f *fileLogger) LogCookieSyncObject(cso *analytics.CookieSyncObject) {
	b := f.getBuffer()
	defer f.release(b)
	jsonifyCookieSync(b, cso)
	f.print(b)
}

func (f *fileLogger) LogAmpObject(ao *analytics.AmpObject) {
	if ao == nil {
		return
	}

	b := f.getBuffer()
	defer f.release(b)
	jsonifyAmpObject(b, ao)
	f.print(b)
}

func (f *fileLogger) LogNotificationEventObject(ne *analytics.NotificationEvent) {
	if ne == nil {
		return
	}

	b := f.getBuffer()
	defer f.release(b)
	jsonifyNotificationEventObject(b, ne)
	f.print(b)
}

func (f *fileLogger) Shutdown() {
	_, _ = f.file.Write([]byte("[fileLogger] Shutdown"))
	_ = f.file.Close()
}

func NewFileLogger(filename string) (analytics.Module, error) {
	f, err := os.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("error creating file logger: %w", err)
	}

	return &fileLogger{
		file:   f,
		logger: log.New(f, "", 0),
		pool: sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
	}, err
}

func jsonifyAuctionObject(buffer *bytes.Buffer, ao *analytics.AuctionObject) {
	var logEntry *logAuction
	if ao != nil {
		var request *openrtb2.BidRequest
		if ao.RequestWrapper != nil {
			request = ao.RequestWrapper.BidRequest
		}
		logEntry = &logAuction{
			Status:               ao.Status,
			Errors:               ao.Errors,
			Request:              request,
			Response:             ao.Response,
			Account:              ao.Account,
			StartTime:            ao.StartTime,
			HookExecutionOutcome: ao.HookExecutionOutcome,
		}
	}

	b, err := jsonutil.Marshal(struct {
		Type RequestType `json:"type"`
		*logAuction
	}{
		Type:       AUCTION,
		logAuction: logEntry,
	})

	if err == nil {
		_, _ = buffer.Write(b)
	} else {
		_, _ = fmt.Fprintf(buffer, "Transactional Logs Error: Auction object badly formed %v", err)
	}
}

func jsonifyVideoObject(buffer *bytes.Buffer, vo *analytics.VideoObject) {
	var logEntry *logVideo
	if vo != nil {
		var request *openrtb2.BidRequest
		if vo.RequestWrapper != nil {
			request = vo.RequestWrapper.BidRequest
		}
		logEntry = &logVideo{
			Status:        vo.Status,
			Errors:        vo.Errors,
			Request:       request,
			Response:      vo.Response,
			VideoRequest:  vo.VideoRequest,
			VideoResponse: vo.VideoResponse,
			StartTime:     vo.StartTime,
		}
	}

	b, err := jsonutil.Marshal(&struct {
		Type RequestType `json:"type"`
		*logVideo
	}{
		Type:     VIDEO,
		logVideo: logEntry,
	})

	if err == nil {
		_, _ = buffer.Write(b)
	} else {
		_, _ = fmt.Fprintf(buffer, "Transactional Logs Error: Video object badly formed %v", err)
	}
}

func jsonifyCookieSync(buffer *bytes.Buffer, cso *analytics.CookieSyncObject) {
	var logEntry *logUserSync
	if cso != nil {
		logEntry = &logUserSync{
			Status:       cso.Status,
			Errors:       cso.Errors,
			BidderStatus: cso.BidderStatus,
		}
	}

	b, err := jsonutil.Marshal(&struct {
		Type RequestType `json:"type"`
		*logUserSync
	}{
		Type:        COOKIE_SYNC,
		logUserSync: logEntry,
	})

	if err == nil {
		_, _ = buffer.Write(b)
	} else {
		_, _ = fmt.Fprintf(buffer, "Transactional Logs Error: Cookie sync object badly formed %v", err)
	}
}

func jsonifySetUIDObject(buffer *bytes.Buffer, so *analytics.SetUIDObject) {
	var logEntry *logSetUID
	if so != nil {
		logEntry = &logSetUID{
			Status:  so.Status,
			Bidder:  so.Bidder,
			UID:     so.UID,
			Errors:  so.Errors,
			Success: so.Success,
		}
	}

	b, err := jsonutil.Marshal(&struct {
		Type RequestType `json:"type"`
		*logSetUID
	}{
		Type:      SETUID,
		logSetUID: logEntry,
	})

	if err == nil {
		_, _ = buffer.Write(b)
	} else {
		_, _ = fmt.Fprintf(buffer, "Transactional Logs Error: Set UID object badly formed %v", err)
	}
}

func jsonifyAmpObject(buffer *bytes.Buffer, ao *analytics.AmpObject) {
	var logEntry *logAMP
	if ao != nil {
		var request *openrtb2.BidRequest
		if ao.RequestWrapper != nil {
			request = ao.RequestWrapper.BidRequest
		}
		logEntry = &logAMP{
			Status:               ao.Status,
			Errors:               ao.Errors,
			Request:              request,
			AuctionResponse:      ao.AuctionResponse,
			AmpTargetingValues:   ao.AmpTargetingValues,
			Origin:               ao.Origin,
			StartTime:            ao.StartTime,
			HookExecutionOutcome: ao.HookExecutionOutcome,
		}
	}

	b, err := jsonutil.Marshal(&struct {
		Type RequestType `json:"type"`
		*logAMP
	}{
		Type:   AMP,
		logAMP: logEntry,
	})

	if err == nil {
		_, _ = buffer.Write(b)
	} else {
		_, _ = fmt.Fprintf(buffer, "Transactional Logs Error: AMP object badly formed %v", err)
	}
}

func jsonifyNotificationEventObject(buffer *bytes.Buffer, ne *analytics.NotificationEvent) {
	var logEntry *logNotificationEvent
	if ne != nil {
		logEntry = &logNotificationEvent{
			Request: ne.Request,
			Account: ne.Account,
		}
	}

	b, err := jsonutil.Marshal(&struct {
		Type RequestType `json:"type"`
		*logNotificationEvent
	}{
		Type:                 NOTIFICATION_EVENT,
		logNotificationEvent: logEntry,
	})

	if err == nil {
		_, _ = buffer.Write(b)
	} else {
		_, _ = fmt.Fprintf(buffer, "Transactional Logs Error: NotificationEvent object badly formed %v", err)
	}
}
