package queue

import (
	"bytes"
	"net/http"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/docker/go-units"
	"github.com/golang/glog"
	utils "github.com/prebid/prebid-server/v3/analytics/pubxai/utils"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type QueueService[T any] interface {
	Enqueue(item T)
	UpdateConfig(bufferInterval string, bufferSize string)
}

type GenericQueue[T any] struct {
	QueueType      string
	Queue          []T
	BufferInterval string
	BufferSize     string
	Endpoint       string
	HttpClient     *http.Client
	LastSentTime   time.Time
	Clock          clock.Clock
	Mutex          sync.RWMutex
}

type WinningBidQueue struct {
	QueueService[utils.WinningBid]
}

type AuctionBidsQueue struct {
	QueueService[utils.AuctionBids]
}

type WinningBidQueueInterface interface {
	QueueService[utils.WinningBid]
}

type AuctionBidsQueueInterface interface {
	QueueService[utils.AuctionBids]
}

func NewGenericQueue[T any](queueType string, endpoint string, client *http.Client, clock clock.Clock, bufferInterval string, bufferSize string) QueueService[T] {
	return &GenericQueue[T]{
		Queue:          make([]T, 0),
		QueueType:      queueType,
		Endpoint:       endpoint,
		Mutex:          sync.RWMutex{},
		HttpClient:     client,
		Clock:          clock,
		BufferSize:     bufferSize,
		BufferInterval: bufferInterval,
		LastSentTime:   clock.Now(),
	}
}

func NewWinningBidQueue(endpoint string, client *http.Client, clock clock.Clock, bufferInterval string, bufferSize string) *WinningBidQueue {
	return &WinningBidQueue{
		QueueService: NewGenericQueue[utils.WinningBid]("win", endpoint, client, clock, bufferInterval, bufferSize),
	}
}

func NewAuctionBidQueue(endpoint string, client *http.Client, clock clock.Clock, bufferInterval string, bufferSize string) *AuctionBidsQueue {
	return &AuctionBidsQueue{
		QueueService: NewGenericQueue[utils.AuctionBids]("auction", endpoint, client, clock, bufferInterval, bufferSize),
	}
}

func NewBidQueue(queueType string, endpoint string, client *http.Client, clock clock.Clock, bufferInterval string, bufferSize string) interface{} {
	if queueType == "win" {
		return NewWinningBidQueue(endpoint, client, clock, bufferInterval, bufferSize)
	} else if queueType == "auction" {
		return NewAuctionBidQueue(endpoint, client, clock, bufferInterval, bufferSize)
	}
	glog.Errorf("[pubxai] Invalid Queue initialization")
	return nil
}

func (q *GenericQueue[T]) isTimeToSend() bool {
	timeDifference := q.Clock.Since(q.LastSentTime)
	pDuration, err := time.ParseDuration(q.BufferInterval)
	if err != nil {
		glog.Errorf("[pubxai] Error parsing bufferInterval: %v", err)
		return false
	}
	glog.Infof("[pubxai] Time difference: %v, bufferInterval: %v", timeDifference, pDuration)
	return timeDifference >= pDuration
}

func (q *GenericQueue[T]) flushQueuedData() {

	if len(q.Queue) == 0 {
		glog.Info("[pubxai] No queued data to send.")
		return
	}

	data, err := jsonutil.Marshal(q.Queue)
	if err != nil {
		glog.Errorf("[pubxai] Error marshaling event queue: %v", err)
	}

	resp, err := q.HttpClient.Post(q.Endpoint, "application/json", bytes.NewReader(data))
	if err != nil {
		glog.Errorf("[pubxai] Error sending queued data: %v", err)
		q.Queue = nil
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		glog.Errorf("[pubxai] Unexpected response status: %s", resp.Status)
	} else {
		glog.Infof("[pubxai] Queued data sent successfully.")
	}
	// Clear the queue in any case
	q.Queue = nil
}

func (q *GenericQueue[T]) Enqueue(item T) {
	q.Mutex.Lock()
	defer q.Mutex.Unlock()
	q.Queue = append(q.Queue, item)

	pBufferSize, _ := units.FromHumanSize(q.BufferSize)
	if int64(len(q.Queue)) >= pBufferSize || q.isTimeToSend() {
		q.flushQueuedData()
	}
}

func (q *GenericQueue[T]) UpdateConfig(bufferInterval string, bufferSize string) {
	q.BufferInterval = bufferInterval
	q.BufferSize = bufferSize
}
