package analytics

import (
	"bytes"
	"github.com/chasex/glog"
)

const FILE_LOGGER = "file_logger"

type FileLogger struct {
	Logger *glog.Logger
}

func (f *FileLogger) LogAuctionObject(ao *AuctionObject) {
	//Code to parse the object and log in a way required
	var b bytes.Buffer
	b.WriteString(ao.ToJson())
	f.Logger.Debug(b.String())
	f.Logger.Flush()
}

func (f *FileLogger) LogSetUIDObject(so *SetUIDObject) {
	//Code to parse the object and log in a way required
	var b bytes.Buffer
	b.WriteString(so.ToJson())
	f.Logger.Debug(b.String())
	f.Logger.Flush()
}

func (f *FileLogger) LogCookieSyncObject(cso *CookieSyncObject) {
	//Code to parse the object and log in a way required
	var b bytes.Buffer
	b.WriteString(cso.ToJson())
	f.Logger.Debug(b.String())
	f.Logger.Flush()
}

func (f *FileLogger) LogAmpObject(ao *AmpObject) {
	//Code to parse the object and log in a way required
	var b bytes.Buffer
	b.WriteString(ao.ToJson())
	f.Logger.Debug(b.String())
	f.Logger.Flush()
}

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
