package analytics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/chasex/glog"
)

type RequestType string

const (
	COOKIE_SYNC RequestType = "/cookie_sync"
	AUCTION     RequestType = "/openrtb2/auction"
	SETUID      RequestType = "/set_uid"
	AMP         RequestType = "/openrtb2/amp"
)

//Module that can perform transactional logging
type FileLogger struct {
	Logger *glog.Logger
}

//Writes AuctionObject to file
func (f *FileLogger) LogAuctionObject(ao *AuctionObject) {
	//Code to parse the object and log in a way required
	var b bytes.Buffer
	b.WriteString(ao.ToJson())
	f.Logger.Debug(b.String())
	f.Logger.Flush()
}

//Logs SetUIDObject to file
func (f *FileLogger) LogSetUIDObject(so *SetUIDObject) {
	//Code to parse the object and log in a way required
	var b bytes.Buffer
	b.WriteString(so.ToJson())
	f.Logger.Debug(b.String())
	f.Logger.Flush()
}

//Logs CookieSyncObject to file
func (f *FileLogger) LogCookieSyncObject(cso *CookieSyncObject) {
	//Code to parse the object and log in a way required
	var b bytes.Buffer
	b.WriteString(cso.ToJson())
	f.Logger.Debug(b.String())
	f.Logger.Flush()
}

//Logs AmpObject to file
func (f *FileLogger) LogAmpObject(ao *AmpObject) {
	//Code to parse the object and log in a way required
	var b bytes.Buffer
	b.WriteString(ao.ToJson())
	f.Logger.Debug(b.String())
	f.Logger.Flush()
}

//Method to initialize the analytic module
func NewFileLogger(filename string) (PBSAnalyticsModule, error) {
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

func (ao *AuctionObject) ToJson() string {
	if content, err := json.Marshal(ao); err != nil {
		return fmt.Sprintf("Transactional Logs Error: Auction object badly formed %v", err)
	} else {
		return string(content)
	}
}

func (cso *CookieSyncObject) ToJson() string {
	if content, err := json.Marshal(cso); err != nil {
		return fmt.Sprintf("Transactional Logs Error: CookieSync object badly formed %v", err)
	} else {
		return string(content)
	}
}

func (so *SetUIDObject) ToJson() string {
	if content, err := json.Marshal(so); err != nil {
		return fmt.Sprintf("Transactional Logs Error: Set UID object badly formed %v", err)
	} else {
		return string(content)
	}
}

func (ao *AmpObject) ToJson() string {
	if content, err := json.Marshal(ao); err != nil {
		return fmt.Sprintf("Transactional Logs Error: Amp object badly formed %v", err)
	} else {
		return string(content)
	}
}

func (ao *AuctionObject) MarshalJSON() ([]byte, error) {
	type alias AuctionObject
	return json.Marshal(&struct {
		Type RequestType `json:"type"`
		*alias
	}{
		Type:  AUCTION,
		alias: (*alias)(ao),
	})
}

func (ao *AmpObject) MarshalJSON() ([]byte, error) {
	type alias AmpObject
	return json.Marshal(&struct {
		Type RequestType `json:"type"`
		*alias
	}{
		Type:  AMP,
		alias: (*alias)(ao),
	})
}

func (so *SetUIDObject) MarshalJSON() ([]byte, error) {
	type alias SetUIDObject
	return json.Marshal(&struct {
		Type RequestType `json:"type"`
		*alias
	}{
		Type:  SETUID,
		alias: (*alias)(so),
	})
}

func (cso *CookieSyncObject) MarshalJSON() ([]byte, error) {
	type alias CookieSyncObject
	return json.Marshal(&struct {
		Type RequestType `json:"type"`
		*alias
	}{
		Type:  COOKIE_SYNC,
		alias: (*alias)(cso),
	})
}
