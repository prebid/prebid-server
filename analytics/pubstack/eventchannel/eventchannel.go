package eventchannel

import (
	"bytes"
	"compress/gzip"
	"github.com/prebid/prebid-server/analytics/clients"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/golang/glog"
)

func (c *EventChannel) resetMetrics() {
	c.metrics.eventCount = 0
	c.metrics.bufferSize = 0
	c.metrics.eventError = 0
}

type Metrics struct {
	bufferSize int64
	eventCount int64
	eventError int64
}

type EventChannel struct {
	endpoint *url.URL
	gz       *gzip.Writer
	buff     *bytes.Buffer
	ch       chan []byte
	metrics  Metrics
	mux      sync.Mutex
}

func NewEventChannel(endpoint *url.URL, maxSize, maxCount int64, maxTime time.Duration) *EventChannel {
	b := bytes.NewBufferString("")
	gzw := gzip.NewWriter(b)

	c := EventChannel{
		endpoint: endpoint,
		gz:       gzw,
		buff:     b,
		ch:       make(chan []byte),
		metrics:  Metrics{},
	}

	termCh := make(chan os.Signal)
	signal.Notify(termCh, os.Interrupt, syscall.SIGTERM)

	go c.batchAndSendEvents(maxSize, maxCount, maxTime, termCh)
	return &c
}

func (c *EventChannel) Add(event []byte) {
	c.ch <- event
}

func (c *EventChannel) batchAndSendEvents(maxSize, maxCount int64, maxTime time.Duration, termCh chan os.Signal) {
	ticker := time.NewTicker(maxTime)

	for {
		select {
		// termination received
		case <-termCh:
			glog.Info("[pubstack] termination signal received")
			c.flush()
			return
		// event is received
		case event := <-c.ch:
			c.mux.Lock()
			_, err := c.gz.Write(event)
			c.mux.Unlock()

			if err != nil {
				c.metrics.eventError++
				glog.Warning("[pubstack] fail to compress, skip the event")
				continue
			}
			c.metrics.eventCount++
			c.metrics.bufferSize = int64(c.buff.Len())
			if c.metrics.eventCount >= maxCount || c.metrics.bufferSize >= maxSize {
				c.flush()
			}
		// time between 2 flushes has passed
		case <-ticker.C:
			c.flush()
		}
	}
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
	c.resetMetrics()
	c.gz.Reset(c.buff)

	// send event to intake (async)
	go post(c.endpoint.String(), payload)

}

func post(endpoint string, payload []byte) {
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		glog.Error(err)
		return
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-Encoding", "gzip")

	resp, err := clients.GetDefaultInstance().Do(req)
	if err != nil {
		return
	}
	if resp.StatusCode != http.StatusOK {
		glog.Errorf("[pubstack] Wrong code received %d instead of %d", resp.StatusCode, http.StatusOK)
		return
	}
}
