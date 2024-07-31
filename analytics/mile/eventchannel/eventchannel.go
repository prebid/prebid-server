package eventchannel

import (
	"github.com/prebid/prebid-server/v2/analytics/mile/helpers"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
)

type Metrics struct {
	bufferSize int64
	eventCount int64
}

type Limit struct {
	maxByteSize   int64
	maxEventCount int64
	maxTime       time.Duration
}

type EventChannel struct {
	//gz   *gzip.Writer
	buff []*helpers.PageViewRecord

	ch          chan *helpers.PageViewRecord
	endCh       chan int
	metrics     Metrics
	muxGzBuffer sync.RWMutex
	send        Sender
	limit       Limit
	clock       clock.Clock
}

func NewEventChannel(sender Sender, clock clock.Clock, maxByteSize, maxEventCount int64, maxTime time.Duration) *EventChannel {
	b := []*helpers.PageViewRecord{}
	//gzw := gzip.NewWriter(b)

	c := EventChannel{
		//gz:      gzw,
		buff:    b,
		ch:      make(chan *helpers.PageViewRecord),
		endCh:   make(chan int),
		metrics: Metrics{},
		send:    sender,
		limit:   Limit{maxByteSize, maxEventCount, maxTime},
		clock:   clock,
	}
	go c.start()
	return &c
}

func (c *EventChannel) Push(event *helpers.PageViewRecord) {
	c.ch <- event
}

func (c *EventChannel) Close() {
	c.endCh <- 1
}

func (c *EventChannel) buffer(event *helpers.PageViewRecord) {
	c.muxGzBuffer.Lock()
	defer c.muxGzBuffer.Unlock()

	//_, err := c.gz.Write(event)
	//if err != nil {
	//	glog.Warning("[pubstack] fail to compress, skip the event")
	//	return
	c.buff = append(c.buff, event)

	c.metrics.eventCount++
	//c.metrics.bufferSize += int64(len(event))
}

func (c *EventChannel) isBufferFull() bool {
	c.muxGzBuffer.RLock()
	defer c.muxGzBuffer.RUnlock()
	return c.metrics.eventCount >= c.limit.maxEventCount || c.metrics.bufferSize >= c.limit.maxByteSize
}

func (c *EventChannel) reset() {
	// reset buffer
	//c.gz.Reset(c.buff)
	//c.buff.Reset()

	c.buff = []*helpers.PageViewRecord{}

	// reset metrics
	c.metrics.eventCount = 0
	c.metrics.bufferSize = 0
}

func (c *EventChannel) flush() {
	c.muxGzBuffer.Lock()
	defer c.muxGzBuffer.Unlock()

	if c.metrics.eventCount == 0 || c.metrics.bufferSize == 0 {
		return
	}

	// reset buffers and writers
	defer c.reset()

	// finish writing gzip header
	//err := c.gz.Close()
	//if err != nil {
	//	glog.Warning("[mile] fail to close gzipped buffer")
	//	return
	//}

	// copy the current buffer to send the payload in a new thread
	//payload := make([]byte, c.buff.Len())
	//_, err = c.buff.Read(payload)
	//if err != nil {
	//	glog.Warning("[mile] fail to copy the buffer")
	//	return
	//}

	// send events (async)
	go c.send(c.buff)
}

func (c *EventChannel) start() {
	ticker := c.clock.Ticker(c.limit.maxTime)

	for {
		select {
		case <-c.endCh:
			c.flush()
			return

		// event is received
		case event := <-c.ch:
			c.buffer(event)
			if c.isBufferFull() {
				c.flush()
			}

		// time between 2 flushes has passed
		case <-ticker.C:
			c.flush()
		}
	}
}
