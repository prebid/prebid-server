package floors

import (
	"container/heap"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/alitto/pond"
	validator "github.com/asaskevich/govalidator"
	"github.com/coocood/freecache"
	"github.com/golang/glog"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/timeutil"
)

var refetchCheckInterval = 300

type fetchInfo struct {
	config.AccountFloorFetch
	fetchTime      int64
	refetchRequest bool
	retryCount     int
}

type WorkerPool interface {
	TrySubmit(task func()) bool
	Stop()
}

type FloorFetcher interface {
	Fetch(configs config.AccountPriceFloors) (*openrtb_ext.PriceFloorRules, string)
	Stop()
}

type PriceFloorFetcher struct {
	pool            WorkerPool            // Goroutines worker pool
	fetchQueue      FetchQueue            // Priority Queue to fetch floor data
	fetchInProgress map[string]bool       // Map of URL with fetch status
	configReceiver  chan fetchInfo        // Channel which recieves URLs to be fetched
	done            chan struct{}         // Channel to close fetcher
	cache           *freecache.Cache      // cache
	httpClient      *http.Client          // http client to fetch data from url
	time            timeutil.Time         // time interface to record request timings
	metricEngine    metrics.MetricsEngine // Records malfunctions in dynamic fetch
	maxRetries      int                   // Max number of retries for failing URLs
}

type FetchQueue []*fetchInfo

func (fq FetchQueue) Len() int {
	return len(fq)
}

func (fq FetchQueue) Less(i, j int) bool {
	return fq[i].fetchTime < fq[j].fetchTime
}

func (fq FetchQueue) Swap(i, j int) {
	fq[i], fq[j] = fq[j], fq[i]
}

func (fq *FetchQueue) Push(element interface{}) {
	fetchInfo := element.(*fetchInfo)
	*fq = append(*fq, fetchInfo)
}

func (fq *FetchQueue) Pop() interface{} {
	old := *fq
	n := len(old)
	fetchInfo := old[n-1]
	old[n-1] = nil
	*fq = old[0 : n-1]
	return fetchInfo
}

func (fq *FetchQueue) Top() *fetchInfo {
	old := *fq
	if len(old) == 0 {
		return nil
	}
	return old[0]
}

func workerPanicHandler(p interface{}) {
	glog.Errorf("floor fetcher worker panicked: %v", p)
}

func NewPriceFloorFetcher(config config.PriceFloors, httpClient *http.Client, metricEngine metrics.MetricsEngine) *PriceFloorFetcher {
	if !config.Enabled {
		return nil
	}

	floorFetcher := PriceFloorFetcher{
		pool:            pond.New(config.Fetcher.Worker, config.Fetcher.Capacity, pond.PanicHandler(workerPanicHandler)),
		fetchQueue:      make(FetchQueue, 0, 100),
		fetchInProgress: make(map[string]bool),
		configReceiver:  make(chan fetchInfo, config.Fetcher.Capacity),
		done:            make(chan struct{}),
		cache:           freecache.NewCache(config.Fetcher.CacheSize * 1024 * 1024),
		httpClient:      httpClient,
		time:            &timeutil.RealTime{},
		metricEngine:    metricEngine,
		maxRetries:      config.Fetcher.MaxRetries,
	}

	go floorFetcher.Fetcher()

	return &floorFetcher
}

func (f *PriceFloorFetcher) SetWithExpiry(key string, value json.RawMessage, cacheExpiry int) {
	f.cache.Set([]byte(key), value, cacheExpiry)
}

func (f *PriceFloorFetcher) Get(key string) (json.RawMessage, bool) {
	data, err := f.cache.Get([]byte(key))
	if err != nil {
		return nil, false
	}

	return data, true
}

func (f *PriceFloorFetcher) Fetch(config config.AccountPriceFloors) (*openrtb_ext.PriceFloorRules, string) {
	if f == nil || !config.UseDynamicData || len(config.Fetcher.URL) == 0 || !validator.IsURL(config.Fetcher.URL) {
		return nil, openrtb_ext.FetchNone
	}

	// Check for floors JSON in cache
	if result, found := f.Get(config.Fetcher.URL); found {
		var fetchedFloorData openrtb_ext.PriceFloorRules
		if err := json.Unmarshal(result, &fetchedFloorData); err != nil || fetchedFloorData.Data == nil {
			return nil, openrtb_ext.FetchError
		}
		return &fetchedFloorData, openrtb_ext.FetchSuccess
	}

	//miss: push to channel to fetch and return empty response
	if config.Enabled && config.Fetcher.Enabled && config.Fetcher.Timeout > 0 {
		fetchConfig := fetchInfo{AccountFloorFetch: config.Fetcher, fetchTime: f.time.Now().Unix(), refetchRequest: false, retryCount: 0}
		f.configReceiver <- fetchConfig
	}

	return nil, openrtb_ext.FetchInprogress
}

func (f *PriceFloorFetcher) worker(fetchConfig fetchInfo) {
	floorData, fetchedMaxAge := f.fetchAndValidate(fetchConfig.AccountFloorFetch)
	if floorData != nil {
		// Reset retry count when data is successfully fetched
		fetchConfig.retryCount = 0

		// Update cache with new floor rules
		cacheExpiry := fetchConfig.AccountFloorFetch.MaxAge
		if fetchedMaxAge != 0 {
			cacheExpiry = fetchedMaxAge
		}
		floorData, err := json.Marshal(floorData)
		if err != nil {
			glog.Errorf("Error while marshaling fetched floor data for url %s", fetchConfig.AccountFloorFetch.URL)
		} else {
			f.SetWithExpiry(fetchConfig.AccountFloorFetch.URL, floorData, cacheExpiry)
		}
	} else {
		fetchConfig.retryCount++
	}

	// Send to refetch channel
	if fetchConfig.retryCount < f.maxRetries {
		fetchConfig.fetchTime = f.time.Now().Add(time.Duration(fetchConfig.AccountFloorFetch.Period) * time.Second).Unix()
		fetchConfig.refetchRequest = true
		f.configReceiver <- fetchConfig
	}
}

// Stop terminates price floor fetcher
func (f *PriceFloorFetcher) Stop() {
	if f == nil {
		return
	}

	close(f.done)
	f.pool.Stop()
	close(f.configReceiver)
}

func (f *PriceFloorFetcher) submit(fetchConfig *fetchInfo) {
	status := f.pool.TrySubmit(func() {
		f.worker(*fetchConfig)
	})
	if !status {
		heap.Push(&f.fetchQueue, fetchConfig)
	}
}

func (f *PriceFloorFetcher) Fetcher() {
	//Create Ticker of 5 minutes
	ticker := time.NewTicker(time.Duration(refetchCheckInterval) * time.Second)

	for {
		select {
		case fetchConfig := <-f.configReceiver:
			if fetchConfig.refetchRequest {
				heap.Push(&f.fetchQueue, &fetchConfig)
			} else {
				if _, ok := f.fetchInProgress[fetchConfig.URL]; !ok {
					f.fetchInProgress[fetchConfig.URL] = true
					f.submit(&fetchConfig)
				}
			}
		case <-ticker.C:
			currentTime := f.time.Now().Unix()
			for top := f.fetchQueue.Top(); top != nil && top.fetchTime <= currentTime; top = f.fetchQueue.Top() {
				nextFetch := heap.Pop(&f.fetchQueue)
				f.submit(nextFetch.(*fetchInfo))
			}
		case <-f.done:
			ticker.Stop()
			glog.Info("Price Floor fetcher terminated")
			return
		}
	}
}

func (f *PriceFloorFetcher) fetchAndValidate(config config.AccountFloorFetch) (*openrtb_ext.PriceFloorRules, int) {
	floorResp, maxAge, err := f.fetchFloorRulesFromURL(config)
	if floorResp == nil || err != nil {
		glog.Errorf("Error while fetching floor data from URL: %s, reason : %s", config.URL, err.Error())
		return nil, 0
	}

	if len(floorResp) > (config.MaxFileSizeKB * 1024) {
		glog.Errorf("Recieved invalid floor data from URL: %s, reason : floor file size is greater than MaxFileSize", config.URL)
		return nil, 0
	}

	var priceFloors openrtb_ext.PriceFloorRules
	if err = json.Unmarshal(floorResp, &priceFloors.Data); err != nil {
		glog.Errorf("Recieved invalid price floor json from URL: %s", config.URL)
		return nil, 0
	}

	if err := validateRules(config, &priceFloors); err != nil {
		glog.Errorf("Validation failed for floor JSON from URL: %s, reason: %s", config.URL, err.Error())
		return nil, 0
	}

	return &priceFloors, maxAge
}

// fetchFloorRulesFromURL returns a price floor JSON and time for which this JSON is valid
// from provided URL with timeout constraints
func (f *PriceFloorFetcher) fetchFloorRulesFromURL(config config.AccountFloorFetch) ([]byte, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.Timeout)*time.Millisecond)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, config.URL, nil)
	if err != nil {
		return nil, 0, errors.New("error while forming http fetch request : " + err.Error())
	}

	httpResp, err := f.httpClient.Do(httpReq)
	if err != nil {
		return nil, 0, errors.New("error while getting response from url : " + err.Error())
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, 0, errors.New("no response from server")
	}

	var maxAge int
	if maxAgeStr := httpResp.Header.Get("max-age"); maxAgeStr != "" {
		maxAge, err = strconv.Atoi(maxAgeStr)
		if err != nil {
			glog.Errorf("max-age in header is malformed for url %s", config.URL)
		}
		if maxAge <= config.Period || maxAge > math.MaxInt32 {
			glog.Errorf("Invalid max-age = %s provided, value should be valid integer and should be within (%v, %v)", maxAgeStr, config.Period, math.MaxInt32)
		}
	}

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, 0, errors.New("unable to read response")
	}
	defer httpResp.Body.Close()

	return respBody, maxAge, nil
}

func validateRules(config config.AccountFloorFetch, priceFloors *openrtb_ext.PriceFloorRules) error {
	if priceFloors.Data == nil {
		return errors.New("empty data in floor JSON")
	}

	if len(priceFloors.Data.ModelGroups) == 0 {
		return errors.New("no model groups found in price floor data")
	}

	if priceFloors.Data.SkipRate < 0 || priceFloors.Data.SkipRate > 100 {
		return errors.New("skip rate should be greater than or equal to 0 and less than 100")
	}

	if priceFloors.Data.UseFetchDataRate != nil && (*priceFloors.Data.UseFetchDataRate < dataRateMin || *priceFloors.Data.UseFetchDataRate > dataRateMax) {
		return errors.New("usefetchdatarate should be greater than or equal to 0 and less than or equal to 100")
	}

	for _, modelGroup := range priceFloors.Data.ModelGroups {
		if len(modelGroup.Values) == 0 || len(modelGroup.Values) > config.MaxRules {
			return errors.New("invalid number of floor rules, floor rules should be greater than zero and less than MaxRules specified in account config")
		}

		if modelGroup.ModelWeight != nil && (*modelGroup.ModelWeight < 1 || *modelGroup.ModelWeight > 100) {
			return errors.New("modelGroup[].modelWeight should be greater than or equal to 1 and less than 100")
		}

		if modelGroup.SkipRate < 0 || modelGroup.SkipRate > 100 {
			return errors.New("model group skip rate should be greater than or equal to 0 and less than 100")
		}

		if modelGroup.Default < 0 {
			return errors.New("modelGroup.Default should be greater than 0")
		}
	}

	return nil
}
