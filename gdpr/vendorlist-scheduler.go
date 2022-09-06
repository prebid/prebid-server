package gdpr

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/golang/glog"
)

type vendorListScheduler struct {
	ticker    *time.Ticker
	interval  time.Duration
	done      chan bool
	isRunning bool
	isStarted bool
	lastRun   time.Time

	httpClient *http.Client
	timeout    time.Duration
}

//Only single instance must be created
var _instance *vendorListScheduler
var once sync.Once

func GetVendorListScheduler(interval, timeout string, httpClient *http.Client) (*vendorListScheduler, error) {
	if _instance != nil {
		return _instance, nil
	}

	intervalDuration, err := time.ParseDuration(interval)
	if err != nil {
		return nil, errors.New("error parsing vendor list scheduler interval: " + err.Error())
	}

	timeoutDuration, err := time.ParseDuration(timeout)
	if err != nil {
		return nil, errors.New("error parsing vendor list scheduler timeout: " + err.Error())
	}

	if httpClient == nil {
		return nil, errors.New("http-client can not be nil")
	}

	once.Do(func() {
		_instance = &vendorListScheduler{
			ticker:     nil,
			interval:   intervalDuration,
			done:       make(chan bool),
			httpClient: httpClient,
			timeout:    timeoutDuration,
		}
	})

	return _instance, nil
}

func (scheduler *vendorListScheduler) Start() {
	if scheduler == nil || scheduler.isStarted {
		return
	}

	scheduler.ticker = time.NewTicker(scheduler.interval)
	scheduler.isStarted = true
	go func() {
		for {
			select {
			case <-scheduler.done:
				scheduler.isRunning = false
				scheduler.isStarted = false
				scheduler.ticker = nil
				return
			case t := <-scheduler.ticker.C:
				if !scheduler.isRunning {
					scheduler.isRunning = true

					glog.Info("Running vendor list scheduler at ", t)
					scheduler.runLoadCache()

					scheduler.lastRun = t
					scheduler.isRunning = false
				}
			}
		}
	}()
}

func (scheduler *vendorListScheduler) Stop() {
	if scheduler == nil || !scheduler.isStarted {
		return
	}
	scheduler.ticker.Stop()
	scheduler.done <- true
}

func (scheduler *vendorListScheduler) runLoadCache() {
	if scheduler == nil {
		return
	}

	preloadContext, cancel := context.WithTimeout(context.Background(), scheduler.timeout)
	defer cancel()

	latestVersion := saveOne(preloadContext, scheduler.httpClient, VendorListURLMaker(0), cacheSave)

	// The GVL for TCF2 has no vendors defined in its first version. It's very unlikely to be used, so don't preload it.
	firstVersionToLoad := uint16(2)

	for i := latestVersion; i >= firstVersionToLoad; i-- {
		// Check if version is present in the cache
		if list := cacheLoad(i); list != nil {
			continue
		}
		glog.Infof("Downloading: " + VendorListURLMaker(i))
		saveOne(preloadContext, scheduler.httpClient, VendorListURLMaker(i), cacheSave)
	}
}
