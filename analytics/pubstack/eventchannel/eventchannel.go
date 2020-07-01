package eventchannel

import (
	"bytes"
	"compress/gzip"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/golang/glog"
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
	gz   *gzip.Writer
	buff *bytes.Buffer

	ch      chan []byte
	endCh   chan int
	metrics Metrics
	mux     sync.Mutex
	send    Sender
	limit   Limit
}

func NewEventChannel(sender Sender, maxByteSize, maxEventCount int64, maxTime time.Duration) *EventChannel {
	b := &bytes.Buffer{}
	gzw := gzip.NewWriter(b)

	c := EventChannel{
		gz:      gzw,
		buff:    b,
		ch:      make(chan []byte),
		endCh:   make(chan int),
		metrics: Metrics{},
		send:    sender,
		limit:   Limit{maxByteSize, maxEventCount, maxTime},
	}

	termCh := make(chan os.Signal)
	signal.Notify(termCh, os.Interrupt, syscall.SIGTERM)

	go c.start(termCh)
	return &c
}

func (c *EventChannel) Push(event []byte) {
	c.ch <- event
}

func (c *EventChannel) Close() {
	c.endCh <- 1
}

func (c *EventChannel) buffer(event []byte) {
	c.mux.Lock()
	defer c.mux.Unlock()

	_, err := c.gz.Write(event)
	if err != nil {
		glog.Warning("[pubstack] fail to compress, skip the event")
		return
	}

	c.metrics.eventCount++
	c.metrics.bufferSize += int64(len(event))
}

func (c *EventChannel) isBufferFull() bool {
	if c.metrics.eventCount >= c.limit.maxEventCount || c.metrics.bufferSize >= c.limit.maxByteSize {
		return true
	}
	return false
}

func (c *EventChannel) reset() {
	// reset buffer
	c.gz.Reset(c.buff)
	c.buff.Reset()

	// reset metrics
	c.metrics.eventCount = 0
	c.metrics.bufferSize = 0
}

func (c *EventChannel) flush() {
	c.mux.Lock()
	defer c.mux.Unlock()

	// finish writing gzip header
	err := c.gz.Flush()
	if err != nil {
		glog.Warning("[pubstack] fail to flush gzipped buffer")
		return
	}

	// copy the current buffer to send the payload in a new thread
	payload := make([]byte, c.buff.Len())
	_, err = c.buff.Read(payload)
	if err != nil {
		glog.Warning("[pubstack] fail to copy the buffer")
		return
	}

	// reset buffers and writers
	c.reset()

	// send events (async)
	go c.send(payload)
}

func (c *EventChannel) start(termCh chan os.Signal) {
	ticker := time.NewTicker(c.limit.maxTime)

	for {
		select {
		case <-c.endCh:
			c.flush()
			return
		// termination received
		case <-termCh:
			glog.Info("[pubstack] termination signal received")
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
