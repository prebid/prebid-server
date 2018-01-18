package analytics

import (
	"bytes"
	"github.com/chasex/glog"
	"log"
)

type FileLogger struct {
	FileName string `mapstructure:"filename"`
	Logger   *glog.Logger
}

func (f *FileLogger) logAuctionObject(ao *AuctionObject) {
	//Code to parse the object and log in a way required
	var b bytes.Buffer
	b.Write(ao.log())
	f.Logger.Debug(b.String())
	f.Logger.Flush()
}

func (f *FileLogger) logSetUIDObject(so *SetUIDObject) {
	//Code to parse the object and log in a way required
}
func (f *FileLogger) logCookieSyncObject(cso *CookieSyncObject) {
	//Code to parse the object and log in a way required
}

func (f *FileLogger) setupFileLogger() *FileLogger {
	//Any other settings can be configured here
	options := glog.LogOptions{
		File:  f.FileName,
		Flag:  glog.LstdFlags,
		Level: glog.Ldebug,
		Mode:  glog.R_Day,
	}
	var err error
	f.Logger, err = glog.New(options)
	if err != nil {
		log.Printf("File Logger could not be initialized. Error: %v", err)
	}
	return f
}
