package files

import (
	"context"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/stored_requests"
	"github.com/prebid/prebid-server/stored_requests/backends/file_fetcher"
	"github.com/prebid/prebid-server/stored_requests/events"
)

type fileEventProducer struct {
	invalidations chan events.Invalidation
	saves         chan events.Save
}

func (f *fileEventProducer) Saves() <-chan events.Save {
	return f.saves
}
func (f *fileEventProducer) Invalidations() <-chan events.Invalidation {
	return nil
}

// NewFilesLoader returns an EventProducer preloaded with all the stored reqs+imps
func NewFilesLoader(cfg config.FileFetcherConfig) events.EventProducer {
	fp := &fileEventProducer{
		saves: make(chan events.Save, 1),
	}
	if fetcher, err := file_fetcher.NewFileFetcher(cfg.Path); err == nil {
		reqData, impData, errs := fetcher.(stored_requests.Fetcher).FetchAllRequests(context.Background())
		if len(reqData) > 0 || len(impData) > 0 {
			fp.saves <- events.Save{
				Requests: reqData,
				Imps:     impData,
			}
		}
		for _, err := range errs {
			glog.Warning(err.Error())
		}
	} else {
		glog.Warningf("Failed to prefetch files from %s: %v", cfg.Path, err)
		close(fp.saves)
	}
	return fp
}
