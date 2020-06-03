package eventchannel

import (
	"bytes"
	"compress/gzip"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/golang/glog"
)

func (c *Channel) resetMetrics() {
	c.metrics.eventCount = 0
	c.metrics.bufferSize = 0
	c.metrics.eventError = 0
}

type ChannelMetrics struct {
	bufferSize int64
	eventCount int64
	eventError int64
}

type Channel struct {
	intake  *url.URL
	gz      *gzip.Writer
	buff    *bytes.Buffer
	ch      chan []byte
	metrics ChannelMetrics
}

// Add : add a new event to be processed
func (c *Channel) Add(event []byte) {
	c.ch <- event
}

func (c *Channel) forward(maxSize, maxCount int64, maxTime time.Duration, termCh chan os.Signal) {
	ticker := time.NewTicker(maxTime)

	for {
		select {
		// termination received
		case <-termCh:
			glog.Info("termination signal received")
			c.flush()
			return
		// event is received
		case event := <-c.ch:
			_, err := c.gz.Write(event)
			if err != nil {
				c.metrics.eventError++
				glog.Warning("fail to compress event")
				continue
			}
			c.metrics.eventCount++
			c.metrics.bufferSize = int64(c.buff.Len())
			if c.metrics.eventCount >= maxCount || c.metrics.bufferSize >= maxSize {
				c.flush()
			}
		// time between flushes has passed
		case <-ticker.C:
			c.flush()
		}
	}
}

func (c *Channel) flush() {
	c.resetMetrics()
	// finish writing gzip header
	c.gz.Close()

	// read gzipped content
	payload := make([]byte, c.buff.Len())
	_, err := c.buff.Read(payload)
	if err != nil {
		glog.Warning("Fail to read gzipped buffer")
	}

	// clean buffers and writers
	c.buff = bytes.NewBufferString("")
	c.gz = gzip.NewWriter(c.buff)

	// send event to intake
	req, err := http.NewRequest(http.MethodPost, c.intake.String(), bytes.NewReader(payload))
	if err != nil {
		glog.Error(err)
		return
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-Encoding", "gzip")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	if resp.StatusCode != http.StatusOK {
		glog.Errorf("wrong code received %d instead of %d", resp.StatusCode, http.StatusOK)
		return
	}
}

func NewChannel(intake, route string, maxSize, maxCount int64, maxTime time.Duration) *Channel {
	u, _ := url.Parse(intake)
	u.Path = path.Join(u.Path, "intake", route)

	b := bytes.NewBufferString("")
	gzw := gzip.NewWriter(b)
	c := Channel{
		intake:  u,
		gz:      gzw,
		buff:    b,
		ch:      make(chan []byte),
		metrics: ChannelMetrics{},
	}

	termCh := make(chan os.Signal)
	signal.Notify(termCh, os.Interrupt, syscall.SIGTERM)

	go c.forward(maxSize, maxCount, maxTime, termCh)
	return &c
}
