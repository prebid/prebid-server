package analytics

import (
	"bytes"
	"github.com/chasex/glog"
	"errors"
)

const FILE_LOGGER = "file_logger"

type FileLogger struct {
	Logger *glog.Logger
}

func (f *FileLogger) LogAuctionObject(ao *AuctionObject) {
	//Code to parse the object and log in a way required
	var b bytes.Buffer
	b.WriteString(ao.String())
	f.Logger.Debug(b.String())
	f.Logger.Flush()
}

func (f *FileLogger) LogSetUIDObject(so *SetUIDObject) {
	//Code to parse the object and log in a way required
	var b bytes.Buffer
	b.WriteString(so.String())
	f.Logger.Debug(b.String())
	f.Logger.Flush()
}
func (f *FileLogger) LogCookieSyncObject(cso *CookieSyncObject) {
	//Code to parse the object and log in a way required
	var b bytes.Buffer
	b.WriteString(cso.String())
	f.Logger.Debug(b.String())
	f.Logger.Flush()
}

func NewFileLogger(conf map[string]string) (PBSAnalyticsModule, error) {
	fileName, ok := conf[FILE_LOGGER]
	if !ok {
		return nil, errors.New("FileLogger not configured")
	}
	options := glog.LogOptions{
		File:  fileName,
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
